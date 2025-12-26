package helpers

import (
	"bytes"
	"devops-cd/internal/model"
	"fmt"
	"text/template"

	"gorm.io/gorm"
)

func ParseDeploymentName(app *model.Application, projectConfig *model.ProjectEnvConfig, appEnvConfig *model.AppEnvConfig) (string, error) {
	if appEnvConfig.DeploymentNameOverride != nil && *appEnvConfig.DeploymentNameOverride != "" {
		// 注意：DeploymentNameOverride 应该直接作为 deployment name 返回（历史实现有误）
		return *appEnvConfig.DeploymentNameOverride, nil
	}

	// 已迁移到 deployment 层：release_name_template 的解析与回填由 deployment Pending 阶段负责
	_ = projectConfig
	return app.Name, nil
}

// ParseValues 从 project_env_config的artifacts中解析values, 模拟helm的values.yaml解析逻辑
func ParseValues(db *gorm.DB, app *model.Application, build *model.Build, projectConfig *model.ProjectEnvConfig, appEnvConfig *model.AppEnvConfig) (map[string]interface{}, error) {
	// 已迁移到 deployment 层：values layers 由对应 driver 在执行时解析与加载
	m, err := ParseValuesV1(db, app, build, projectConfig, appEnvConfig, nil)
	if err != nil {
		return nil, err
	}
	return marshalMeta(m)
}

func Diff() {

}

func parseTemplate(tpl string, data map[string]interface{}) (string, error) {
	tmpl, err := template.New("").Parse(tpl)
	if err != nil {
		return "", fmt.Errorf("parse template Failed: %w", err)
	}

	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template Failed: %w", err)
	}

	return buf.String(), nil
}

// ParseTemplateForInternal 仅用于内部模块复用模板渲染逻辑（v1 artifacts_json）
func ParseTemplateForInternal(tpl string, data map[string]interface{}) (string, error) {
	return parseTemplate(tpl, data)
}

func notEmpty(ptr *string) bool {
	return ptr != nil && *ptr != ""
}
