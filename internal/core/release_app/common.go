package release_app

import (
	"bytes"
	"devops-cd/internal/model"
	"devops-cd/internal/pkg/logger"
	"text/template"
)

func mustDeploymentName(app *model.Application, projectConfig *model.ProjectEnvConfig, appEnvConfig *model.AppEnvConfig) string {
	//if appEnvConfig.DeploymentName != "" {
	//	return projectConfig.Namespace
	//}

	if projectConfig.DeploymentNameTemplate == "" {
		return app.Name
	}

	tmpl := template.Must(template.New("").Parse(projectConfig.DeploymentNameTemplate))
	data := map[string]interface{}{
		"app_name": app.Name,
		"project":  app.Project.Name,
		"env":      appEnvConfig.Env,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		logger.Sugar().Fatalf("execute deployment name template error: %v", err)
		panic(err)
	}

	return buf.String()
}
