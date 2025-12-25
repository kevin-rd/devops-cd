package release_app

import (
	"devops-cd/internal/model"
	"encoding/json"
	"fmt"
)

// ArtifactsV1 是 project_env_configs.artifacts_json 的 v1 结构
//
// 约束：
// - namespace_template 放在顶层，app_chart/config_chart 必须共用 namespace
// - sources 不单独抽象，所有来源配置内联在对应结构内（保持简洁）
type ArtifactsV1 struct {
	SchemaVersion     int          `json:"schema_version"`
	NamespaceTemplate string       `json:"namespace_template"`
	ConfigChart       *ChartSpecV1 `json:"config_chart,omitempty"`
	AppChart          *ChartSpecV1 `json:"app_chart,omitempty"`
}

type ChartSpecV1 struct {
	Enabled              bool              `json:"enabled"`
	ReleaseNameTemplate  string            `json:"release_name_template,omitempty"`
	DependsOnConfigChart bool              `json:"depends_on_config_chart,omitempty"` // 仅 app_chart 使用
	Chart                ChartSourceSpecV1 `json:"chart"`
	Values               []ValuesLayerV1   `json:"values,omitempty"`
}

type ChartSourceSpecV1 struct {
	Type                 string `json:"type"`               // helm_repo | oci | pipeline_artifact
	RepoURL              string `json:"repo_url,omitempty"` // helm_repo
	CredentialRef        string `json:"credential_ref,omitempty"`
	ChartNameTemplate    string `json:"chart_name_template,omitempty"`
	ChartVersionTemplate string `json:"chart_version_template,omitempty"`

	// pipeline_artifact：允许 chart 从流水线产物拉取（v1 先支持 HTTP 下载）
	ArtifactURLTemplate string `json:"artifact_url_template,omitempty"`
}

type ValuesLayerV1 struct {
	Type          string `json:"type"` // git | http_file | inline | pipeline_artifact
	CredentialRef string `json:"credential_ref,omitempty"`

	// git
	RepoURL      string `json:"repo_url,omitempty"`
	RefTemplate  string `json:"ref_template,omitempty"`
	PathTemplate string `json:"path_template,omitempty"`

	// http_file / pipeline_artifact
	URLTemplate string `json:"url_template,omitempty"`

	// inline
	Content string `json:"content,omitempty"`
}

func LoadArtifactsV1(projectConfig *model.ProjectEnvConfig) (*ArtifactsV1, error) {
	// 1) 优先读 artifacts_json
	if projectConfig != nil && projectConfig.ArtifactsJSON != nil && *projectConfig.ArtifactsJSON != "" {
		var a ArtifactsV1
		if err := json.Unmarshal([]byte(*projectConfig.ArtifactsJSON), &a); err != nil {
			return nil, fmt.Errorf("artifacts_json 解析失败: %w", err)
		}
		if a.SchemaVersion == 0 {
			a.SchemaVersion = projectConfig.SchemaVersion
		}
		if a.SchemaVersion == 0 {
			a.SchemaVersion = 1
		}
		return &a, nil
	}

	// 2) 否则从旧字段生成默认 v1 结构（兼容）
	return DefaultArtifactsV1FromLegacy(projectConfig), nil
}

func DefaultArtifactsV1FromLegacy(projectConfig *model.ProjectEnvConfig) *ArtifactsV1 {
	a := &ArtifactsV1{
		SchemaVersion:     1,
		NamespaceTemplate: "",
		ConfigChart: &ChartSpecV1{
			Enabled: false,
			Chart: ChartSourceSpecV1{
				Type: "helm_repo",
			},
		},
		AppChart: &ChartSpecV1{
			Enabled: true,
			Chart: ChartSourceSpecV1{
				Type:              "helm_repo",
				RepoURL:           "",
				ChartNameTemplate: "{{.app_type}}",
			},
			Values: []ValuesLayerV1{},
		},
	}

	if projectConfig == nil {
		return a
	}

	// namespace: 旧字段是固定字符串；v1 允许 template，这里直接透传为 template（无变量时也成立）
	if projectConfig.Namespace != "" {
		a.NamespaceTemplate = projectConfig.Namespace
	}

	// chart repo url
	if projectConfig.ChartRepoURL != "" {
		a.AppChart.Chart.RepoURL = projectConfig.ChartRepoURL
	}

	// values repo + path（旧字段）
	if projectConfig.ValuesRepoURL != nil && *projectConfig.ValuesRepoURL != "" &&
		projectConfig.ValuesPathTemplate != nil && *projectConfig.ValuesPathTemplate != "" {
		a.AppChart.Values = append(a.AppChart.Values, ValuesLayerV1{
			Type:         "git",
			RepoURL:      *projectConfig.ValuesRepoURL,
			RefTemplate:  "main",
			PathTemplate: *projectConfig.ValuesPathTemplate,
		})
	}

	return a
}
