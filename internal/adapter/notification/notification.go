package notification

import (
	"bytes"
	"context"
	"devops-cd/internal/model"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"net/http"
	"time"
)

// NotificationType 通知类型
type NotificationType string

const (
	NotifyBatchStart       NotificationType = "batch_start"        // 批次开始
	NotifyBatchComplete    NotificationType = "batch_complete"     // 批次完成
	NotifyBatchFailed      NotificationType = "batch_failed"       // 批次失败
	NotifyDeployStart      NotificationType = "deploy_start"       // 部署开始
	NotifyDeploySuccess    NotificationType = "deploy_success"     // 部署成功
	NotifyDeployFailed     NotificationType = "deploy_failed"      // 部署失败
	NotifyAppDeploySuccess NotificationType = "app_deploy_success" // 应用部署成功
	NotifyAppDeployFailed  NotificationType = "app_deploy_failed"  // 应用部署失败
	NotifyStateTransition  NotificationType = "state_transition"   // 状态转换
)

// NotificationMessage 通知消息
type NotificationMessage struct {
	Type      NotificationType       `json:"type"`
	Title     string                 `json:"title"`
	Content   string                 `json:"content"`
	Timestamp time.Time              `json:"timestamp"`
	Extra     map[string]interface{} `json:"extra,omitempty"` // 额外信息
}

// Notifier 通知器接口
type Notifier interface {
	// Send 发送通知
	Send(ctx context.Context, msg *NotificationMessage) error

	// SendBatchNotification 发送批次通知
	SendBatchNotification(ctx context.Context, batch *model.Batch, notifyType NotificationType, message string) error

	// SendAppDeployNotification 发送应用部署通知
	SendAppDeployNotification(ctx context.Context, batchID int64, appID int64, appName string, notifyType NotificationType, message string) error
}

// ============= Lark 通知适配器 =============

// LarkNotifier Lark通知器
type LarkNotifier struct {
	webhookURL string
	enabled    bool
	logger     *zap.Logger
	client     *http.Client
}

// NewLarkNotifier 创建Lark通知器
func NewLarkNotifier(webhookURL string, enabled bool, logger *zap.Logger) *LarkNotifier {
	return &LarkNotifier{
		webhookURL: webhookURL,
		enabled:    enabled,
		logger:     logger,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Send 发送通知
func (n *LarkNotifier) Send(ctx context.Context, msg *NotificationMessage) error {
	if !n.enabled {
		n.logger.Debug("通知已禁用,跳过发送")
		return nil
	}

	if n.webhookURL == "" {
		n.logger.Warn("Lark Webhook URL未配置")
		return nil
	}

	// 构建Lark消息格式
	larkMsg := n.buildLarkMessage(msg)

	// 发送HTTP请求
	jsonData, err := json.Marshal(larkMsg)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", n.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Lark API返回错误状态码: %d", resp.StatusCode)
	}

	n.logger.Info("Lark通知发送成功",
		zap.String("type", string(msg.Type)),
		zap.String("title", msg.Title))

	return nil
}

// SendBatchNotification 发送批次通知
func (n *LarkNotifier) SendBatchNotification(ctx context.Context, batch *model.Batch, notifyType NotificationType, message string) error {
	var title, content string
	var color string

	switch notifyType {
	case NotifyBatchStart:
		title = "🚀 批次部署开始"
		color = "blue"
	case NotifyBatchComplete:
		title = "✅ 批次部署完成"
		color = "green"
	case NotifyBatchFailed:
		title = "❌ 批次部署失败"
		color = "red"
	case NotifyDeployStart:
		title = "🔄 开始部署"
		color = "blue"
	default:
		title = "📢 批次通知"
		color = "grey"
	}

	content = fmt.Sprintf("**批次编号**: %s\n**发起人**: %s\n**消息**: %s",
		batch.BatchNumber, batch.Initiator, message)

	msg := &NotificationMessage{
		Type:      notifyType,
		Title:     title,
		Content:   content,
		Timestamp: time.Now(),
		Extra: map[string]interface{}{
			"batch_id":     batch.ID,
			"batch_number": batch.BatchNumber,
			"color":        color,
		},
	}

	return n.Send(ctx, msg)
}

// SendAppDeployNotification 发送应用部署通知
func (n *LarkNotifier) SendAppDeployNotification(ctx context.Context, batchID int64, appID int64, appName string, notifyType NotificationType, message string) error {
	var title string
	var color string

	switch notifyType {
	case NotifyAppDeploySuccess:
		title = "✅ 应用部署成功"
		color = "green"
	case NotifyAppDeployFailed:
		title = "❌ 应用部署失败"
		color = "red"
	default:
		title = "📢 应用部署通知"
		color = "grey"
	}

	content := fmt.Sprintf("**应用**: %s (ID: %d)\n**批次ID**: %d\n**消息**: %s",
		appName, appID, batchID, message)

	msg := &NotificationMessage{
		Type:      notifyType,
		Title:     title,
		Content:   content,
		Timestamp: time.Now(),
		Extra: map[string]interface{}{
			"batch_id": batchID,
			"app_id":   appID,
			"app_name": appName,
			"color":    color,
		},
	}

	return n.Send(ctx, msg)
}

// buildLarkMessage 构建Lark消息格式
func (n *LarkNotifier) buildLarkMessage(msg *NotificationMessage) map[string]interface{} {
	color := "grey"
	if c, ok := msg.Extra["color"].(string); ok {
		color = c
	}

	// Lark富文本消息格式
	return map[string]interface{}{
		"msg_type": "interactive",
		"card": map[string]interface{}{
			"header": map[string]interface{}{
				"title": map[string]interface{}{
					"tag":     "plain_text",
					"content": msg.Title,
				},
				"template": color,
			},
			"elements": []interface{}{
				map[string]interface{}{
					"tag": "div",
					"text": map[string]interface{}{
						"tag":     "lark_md",
						"content": msg.Content,
					},
				},
				map[string]interface{}{
					"tag": "div",
					"text": map[string]interface{}{
						"tag":     "plain_text",
						"content": fmt.Sprintf("时间: %s", msg.Timestamp.Format("2006-01-02 15:04:05")),
					},
				},
			},
		},
	}
}

// ============= 多通知器 =============

// MultiNotifier 多通知器(支持同时发送到多个渠道)
type MultiNotifier struct {
	notifiers []Notifier
	logger    *zap.Logger
}

// NewMultiNotifier 创建多通知器
func NewMultiNotifier(logger *zap.Logger, notifiers ...Notifier) *MultiNotifier {
	return &MultiNotifier{
		notifiers: notifiers,
		logger:    logger,
	}
}

// Send 发送到所有通知器
func (m *MultiNotifier) Send(ctx context.Context, msg *NotificationMessage) error {
	var lastErr error
	for _, notifier := range m.notifiers {
		if err := notifier.Send(ctx, msg); err != nil {
			m.logger.Error("发送通知失败", zap.Error(err))
			lastErr = err
			// 继续发送其他通知器
		}
	}
	return lastErr
}

// SendBatchNotification 发送批次通知到所有通知器
func (m *MultiNotifier) SendBatchNotification(ctx context.Context, batch *model.Batch, notifyType NotificationType, message string) error {
	var lastErr error
	for _, notifier := range m.notifiers {
		if err := notifier.SendBatchNotification(ctx, batch, notifyType, message); err != nil {
			m.logger.Error("发送批次通知失败", zap.Error(err))
			lastErr = err
		}
	}
	return lastErr
}

// SendAppDeployNotification 发送应用部署通知到所有通知器
func (m *MultiNotifier) SendAppDeployNotification(ctx context.Context, batchID int64, appID int64, appName string, notifyType NotificationType, message string) error {
	var lastErr error
	for _, notifier := range m.notifiers {
		if err := notifier.SendAppDeployNotification(ctx, batchID, appID, appName, notifyType, message); err != nil {
			m.logger.Error("发送应用部署通知失败", zap.Error(err))
			lastErr = err
		}
	}
	return lastErr
}

// ============= 日志通知器(仅记录日志,不发送实际通知) =============

// LogNotifier 日志通知器
type LogNotifier struct {
	logger *zap.Logger
}

// NewLogNotifier 创建日志通知器
func NewLogNotifier(logger *zap.Logger) *LogNotifier {
	return &LogNotifier{
		logger: logger,
	}
}

// Send 记录通知到日志
func (n *LogNotifier) Send(ctx context.Context, msg *NotificationMessage) error {
	n.logger.Info("📢 通知",
		zap.String("type", string(msg.Type)),
		zap.String("title", msg.Title),
		zap.String("content", msg.Content),
		zap.Any("extra", msg.Extra))
	return nil
}

// SendBatchNotification 记录批次通知到日志
func (n *LogNotifier) SendBatchNotification(ctx context.Context, batch *model.Batch, notifyType NotificationType, message string) error {
	n.logger.Info("📢 批次通知",
		zap.String("type", string(notifyType)),
		zap.Int64("batch_id", batch.ID),
		zap.String("batch_number", batch.BatchNumber),
		zap.String("message", message))
	return nil
}

// SendAppDeployNotification 记录应用部署通知到日志
func (n *LogNotifier) SendAppDeployNotification(ctx context.Context, batchID int64, appID int64, appName string, notifyType NotificationType, message string) error {
	n.logger.Info("📢 应用部署通知",
		zap.String("type", string(notifyType)),
		zap.Int64("batch_id", batchID),
		zap.Int64("app_id", appID),
		zap.String("app_name", appName),
		zap.String("message", message))
	return nil
}
