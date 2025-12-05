package release_app

import (
	"bytes"
	"devops-cd/internal/model"
	"fmt"
	"text/template"
)

func mustDeploymentName(app *model.Application, projectConfig *model.ProjectEnvConfig, appEnvConfig *model.AppEnvConfig) (string, error) {
	if appEnvConfig.DeploymentNameOverride != nil && *appEnvConfig.DeploymentNameOverride != "" {
		return projectConfig.Namespace, nil
	}

	if projectConfig.DeploymentNameTemplate == "" {
		return app.Name, nil
	}

	tmpl, err := template.New("").Parse(projectConfig.DeploymentNameTemplate)
	if err != nil {
		return "", fmt.Errorf("parse DeploymentName Failed: %w", err)
	}

	data := map[string]interface{}{
		"app_name": app.Name,
		"project":  app.Project.Name,
		"env":      appEnvConfig.Env,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("parse DeploymentName Failed: %w", err)
	}

	return buf.String(), nil
}
