package tpl

import (
	"bytes"
	"fmt"
	"text/template"

	"devops-cd/internal/model"
)

type ContextOptions struct {
	// Repo：用于注入 repo.* 模板变量；若为空则尝试从 app.Repository 读取
	Repo *model.Repository
	// RepoAppCount：repo 在“当前 project”下关联的应用数（按 repo_id + project_id 统计）
	RepoAppCount *int64
}

// RenderTemplateContext 构建模板变量上下文（白名单字段）
func RenderTemplateContext(app *model.Application, build *model.Build, env string, cluster string, opts *ContextOptions) map[string]interface{} {
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

	// repo.*：用于 artifacts 中大量模板渲染（namespace/release_name/values layers 等）
	var repo *model.Repository
	if opts != nil && opts.Repo != nil {
		repo = opts.Repo
	} else if app != nil && app.Repository != nil {
		repo = app.Repository
	}
	var repoAppCount int64
	if opts != nil && opts.RepoAppCount != nil {
		repoAppCount = *opts.RepoAppCount
	}
	repoNamespace := ""
	repoName := ""
	if repo != nil {
		repoNamespace = repo.Namespace
		repoName = repo.Name
	}
	fullName := ""
	if repoNamespace != "" && repoName != "" {
		fullName = repoNamespace + "/" + repoName
	} else if repoName != "" {
		fullName = repoName
	}
	ctx["repo"] = map[string]interface{}{
		"namespace": repoNamespace,
		"name":      repoName,
		"full_name": fullName,
		"app_count": repoAppCount,
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
