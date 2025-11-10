package dependency

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"devops-cd/internal/model"
	"devops-cd/pkg/constants"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Config 依赖解析配置
type Config struct {
	// AppTypeDepends 定义 app_type 级依赖，比如 {"static": ["java", "go"]}
	AppTypeDepends map[string][]string
}

// Resolver 应用发布依赖解析器
type Resolver struct {
	db     *gorm.DB
	logger *zap.Logger
	cfg    Config
}

// NewResolver 创建依赖解析器
func NewResolver(db *gorm.DB, logger *zap.Logger, cfg Config) *Resolver {
	if cfg.AppTypeDepends == nil {
		cfg.AppTypeDepends = make(map[string][]string)
	}
	return &Resolver{db: db, logger: logger, cfg: cfg}
}

// Result 依赖检查结果
type Result struct {
	Pending []Status
	Failed  []Status
}

// Status 单个依赖状态
type Status struct {
	AppID         int64
	AppName       string
	AppType       string
	ReleaseID     *int64
	CurrentStatus int8
	RequiredStage string
	Sources       []string
	Message       string
}

// IsReady 依赖是否全部满足
func (r *Result) IsReady() bool {
	return r != nil && len(r.Pending) == 0 && len(r.Failed) == 0
}

// HasPending 是否存在等待中的依赖
func (r *Result) HasPending() bool {
	return r != nil && len(r.Pending) > 0
}

// HasFailed 是否存在失败的依赖
func (r *Result) HasFailed() bool {
	return r != nil && len(r.Failed) > 0
}

// Summary 汇总阻塞原因
func (r *Result) Summary() string {
	if r == nil {
		return ""
	}

	var parts []string
	for _, item := range append(r.Failed, r.Pending...) {
		if item.Message != "" {
			parts = append(parts, item.Message)
		}
	}

	return strings.Join(parts, "; ")
}

// CheckRelease 检查 ReleaseApp 在指定阶段的依赖是否满足
func (r *Resolver) CheckRelease(ctx context.Context, release *model.ReleaseApp, stage string) (*Result, error) {
	var app model.Application
	if err := r.db.WithContext(ctx).First(&app, release.AppID).Error; err != nil {
		return nil, fmt.Errorf("查询应用信息失败: %w", err)
	}

	deps, err := r.collectDependencies(ctx, release, &app)
	if err != nil {
		return nil, err
	}

	if len(deps) == 0 {
		return &Result{}, nil
	}

	depIDs := make([]int64, 0, len(deps))
	for id := range deps {
		depIDs = append(depIDs, id)
	}

	appInfos := make(map[int64]*model.Application, len(depIDs))
	var depApps []model.Application
	if err := r.db.WithContext(ctx).Where("id IN ?", depIDs).Find(&depApps).Error; err != nil {
		return nil, fmt.Errorf("查询依赖应用失败: %w", err)
	}
	for i := range depApps {
		app := depApps[i]
		appInfos[app.ID] = &app
	}

	releaseInfos := make(map[int64]*model.ReleaseApp, len(depIDs))
	var depReleases []model.ReleaseApp
	if err := r.db.WithContext(ctx).
		Where("batch_id = ? AND app_id IN ?", release.BatchID, depIDs).
		Find(&depReleases).Error; err != nil {
		return nil, fmt.Errorf("查询依赖的 ReleaseApp 失败: %w", err)
	}
	for i := range depReleases {
		rel := depReleases[i]
		releaseInfos[rel.AppID] = &rel
	}

	result := &Result{}
	for id, entry := range deps {
		status := Status{
			AppID:         id,
			RequiredStage: stage,
			Sources:       entry.sources,
		}

		if info, ok := appInfos[id]; ok {
			status.AppName = info.Name
			status.AppType = info.AppType
		}

		rel, ok := releaseInfos[id]
		if !ok {
			// 依赖不在本批次，视为已满足
			continue
		}

		releaseID := rel.ID
		status.ReleaseID = &releaseID
		status.CurrentStatus = rel.Status

		ready, failed := evaluateStage(stage, rel.Status)
		status.Message = formatStatusMessage(status, entry, ready, failed)

		if failed {
			result.Failed = append(result.Failed, status)
			continue
		}
		if !ready {
			result.Pending = append(result.Pending, status)
		}
	}

	return result, nil
}

// CheckDeployment 检查 Deployment 的依赖是否满足
func (r *Resolver) CheckDeployment(ctx context.Context, dep *model.Deployment) (*Result, error) {
	var release model.ReleaseApp
	if err := r.db.WithContext(ctx).First(&release, dep.ReleaseID).Error; err != nil {
		return nil, fmt.Errorf("查询部署关联的 ReleaseApp 失败: %w", err)
	}

	return r.CheckRelease(ctx, &release, dep.Environment)
}

type dependencyEntry struct {
	sources []string
}

func (e *dependencyEntry) addSource(source string) {
	for _, existing := range e.sources {
		if existing == source {
			return
		}
	}
	e.sources = append(e.sources, source)
}

func (r *Resolver) collectDependencies(ctx context.Context, release *model.ReleaseApp, app *model.Application) (map[int64]*dependencyEntry, error) {
	deps := make(map[int64]*dependencyEntry)

	add := func(appID int64, source string) {
		if appID == release.AppID || appID == 0 {
			return
		}
		entry, ok := deps[appID]
		if !ok {
			entry = &dependencyEntry{}
			deps[appID] = entry
		}
		entry.addSource(source)
	}

	defaultIDs := app.DefaultDependsOn

	for _, id := range defaultIDs {
		add(id, "default")
	}

	tempIDs := release.TempDependsOn

	for _, id := range tempIDs {
		add(id, "temporary")
	}

	if types, ok := r.cfg.AppTypeDepends[app.AppType]; ok {
		if len(types) > 0 {
			var typeDeps []struct {
				AppID   int64
				AppType string
			}
			if err := r.db.WithContext(ctx).
				Table("release_apps").
				Select("release_apps.app_id as app_id, applications.app_type as app_type").
				Joins("JOIN applications ON release_apps.app_id = applications.id").
				Where("release_apps.batch_id = ? AND applications.app_type IN ?", release.BatchID, types).
				Find(&typeDeps).Error; err != nil {
				return nil, fmt.Errorf("查询 app_type 依赖失败: %w", err)
			}
			for _, dep := range typeDeps {
				add(dep.AppID, fmt.Sprintf("app_type:%s", dep.AppType))
			}
		}
	}

	return deps, nil
}

func decodeIDs(raw []byte) ([]int64, error) {
	if len(raw) == 0 {
		return []int64{}, nil
	}

	var ids []int64
	if err := json.Unmarshal(raw, &ids); err != nil {
		return nil, err
	}

	ids = normalizeIDs(ids)
	return ids, nil
}

func normalizeIDs(ids []int64) []int64 {
	if len(ids) == 0 {
		return []int64{}
	}

	seen := make(map[int64]struct{}, len(ids))
	result := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}

	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func evaluateStage(stage string, status int8) (ready bool, failed bool) {
	switch stage {
	case constants.EnvTypePre:
		if status == constants.ReleaseAppStatusPreFailed {
			return false, true
		}
		return status >= constants.ReleaseAppStatusPreDeployed || status >= constants.ReleaseAppStatusProdWaiting, false
	case constants.EnvTypeProd:
		if status == constants.ReleaseAppStatusProdFailed {
			return false, true
		}
		return status >= constants.ReleaseAppStatusProdDeployed, false
	default:
		return true, false
	}
}

func formatStatusMessage(status Status, entry *dependencyEntry, ready bool, failed bool) string {
	var builder strings.Builder

	if status.AppName != "" {
		builder.WriteString(fmt.Sprintf("依赖应用 %s", status.AppName))
	} else {
		builder.WriteString(fmt.Sprintf("依赖应用 ID=%d", status.AppID))
	}

	if status.AppType != "" {
		builder.WriteString(fmt.Sprintf("(类型:%s)", status.AppType))
	}

	if len(entry.sources) > 0 {
		builder.WriteString(fmt.Sprintf(" 来源:%s", strings.Join(entry.sources, ",")))
	}

	stageName := stageDisplayName(status.RequiredStage)
	statusName := releaseStatusText(status.CurrentStatus)

	if failed {
		builder.WriteString(fmt.Sprintf(" %s阶段失败，当前状态:%s", stageName, statusName))
	} else if !ready {
		builder.WriteString(fmt.Sprintf(" %s阶段未完成，当前状态:%s", stageName, statusName))
	}

	return builder.String()
}

func stageDisplayName(stage string) string {
	switch stage {
	case constants.EnvTypePre:
		return "预发布"
	case constants.EnvTypeProd:
		return "生产"
	default:
		return stage
	}
}

func releaseStatusText(status int8) string {
	switch status {
	case constants.ReleaseAppStatusPending:
		return "Pending"
	case constants.ReleaseAppStatusTagged:
		return "Tagged"
	case constants.ReleaseAppStatusPreWaiting:
		return "PreWaiting"
	case constants.ReleaseAppStatusPreCanTrigger:
		return "PreCanTrigger"
	case constants.ReleaseAppStatusPreTriggered:
		return "PreTriggered"
	case constants.ReleaseAppStatusPreDeployed:
		return "PreDeployed"
	case constants.ReleaseAppStatusPreFailed:
		return "PreFailed"
	case constants.ReleaseAppStatusProdWaiting:
		return "ProdWaiting"
	case constants.ReleaseAppStatusProdCanTrigger:
		return "ProdCanTrigger"
	case constants.ReleaseAppStatusProdTriggered:
		return "ProdTriggered"
	case constants.ReleaseAppStatusProdDeployed:
		return "ProdDeployed"
	case constants.ReleaseAppStatusProdFailed:
		return "ProdFailed"
	default:
		return fmt.Sprintf("Unknown(%d)", status)
	}
}
