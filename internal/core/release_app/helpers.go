package release_app

import (
	"bytes"
	"devops-cd/internal/model"
	"fmt"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/getter"
	"text/template"
)

var helmProviders = getter.All(cli.New())

func ParseDeploymentName(app *model.Application, projectConfig *model.ProjectEnvConfig, appEnvConfig *model.AppEnvConfig) (string, error) {
	if appEnvConfig.DeploymentNameOverride != nil && *appEnvConfig.DeploymentNameOverride != "" {
		return projectConfig.Namespace, nil
	}

	if projectConfig.DeploymentNameTemplate == "" {
		return app.Name, nil
	}

	data := map[string]interface{}{
		"app_name": app.Name,
		"app_type": app.AppType,
		"project":  app.Project.Name,
		"env":      appEnvConfig.Env,
		"cluster":  appEnvConfig.Cluster,
	}

	return parseTemplate(projectConfig.DeploymentNameTemplate, data)
}

// 生成 values.yaml 文件
func ParseValues(app *model.Application, build *model.Build, projectConfig *model.ProjectEnvConfig, appEnvConfig *model.AppEnvConfig) (map[string]interface{}, error) {
	options := values.Options{}

	// 1. 从cd仓库文件中获取values.yaml
	if notEmpty(projectConfig.ValuesRepoURL) && notEmpty(projectConfig.ValuesPathTemplate) {
		data := map[string]interface{}{
			"app_name": app.Name,
			"app_type": app.AppType,
			"project":  app.Project.Name,
			"env":      appEnvConfig.Env,
			"cluster":  appEnvConfig.Cluster,
		}
		path, err := parseTemplate(*projectConfig.ValuesPathTemplate, data)
		if err != nil {
			return nil, err
		}

		options.ValueFiles = append(options.ValueFiles, fmt.Sprintf("%s/%s", *projectConfig.ValuesRepoURL, path))
	}

	// 2. 从appEnvConfigs中获取配置
	// todo: 暂无

	// 3. 获取image tag
	if build.ImageTag != "" {
		// 默认使用 image.tag
		options.Values = append(options.Values, fmt.Sprintf("image.tag=%s", build.ImageTag))
	}
	return options.MergeValues(helmProviders)
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

func notEmpty(ptr *string) bool {
	return ptr != nil && *ptr != ""
}
