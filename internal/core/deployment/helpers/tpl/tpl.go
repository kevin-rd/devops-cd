package tpl

import (
	"bytes"
	"fmt"
	"text/template"

	"devops-cd/internal/model"
)

// RenderTemplateContext 构建模板变量上下文（白名单字段）
func RenderTemplateContext(app *model.Application, build *model.Build, env string, cluster string) map[string]interface{} {
	ctx := map[string]interface{}{}
	if app != nil {
		ctx["app_name"] = app.Name
		ctx["app_type"] = app.AppType
		if app.Project != nil {
			ctx["project"] = app.Project.Name
		}
	}
	if env != "" {
		ctx["env"] = env
	}
	if cluster != "" {
		ctx["cluster"] = cluster
	}
	if build != nil {
		ctx["build"] = map[string]interface{}{
			"image_tag": build.ImageTag,
		}
	}
	return ctx
}

func ParseTemplate(tplStr string, data map[string]interface{}) (string, error) {
	tmpl, err := template.New("").Parse(tplStr)
	if err != nil {
		return "", fmt.Errorf("parse template failed: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template failed: %w", err)
	}
	return buf.String(), nil
}
