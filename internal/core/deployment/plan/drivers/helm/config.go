package helm

import (
	artifacts "devops-cd/internal/model"
	"encoding/json"
	"fmt"
	"strings"
)

// Config 是 helm driver 的私有配置（对应 artifacts_json.*_chart.data）。
type Config struct {
	ReleaseNameTemplate string `json:"release_name_template,omitempty"`

	RepoURL              string `json:"repo_url,omitempty"`
	CredentialRef        string `json:"credential_ref,omitempty"`
	ChartNameTemplate    string `json:"chart_name_template,omitempty"`
	ChartVersionTemplate string `json:"chart_version_template,omitempty"`

	// helm values layers（helm driver 特有）
	Values []artifacts.ValuesLayer `json:"values,omitempty"`
}

func DecodeConfig(raw json.RawMessage) (*Config, error) {
	if len(raw) == 0 || strings.TrimSpace(string(raw)) == "" || strings.TrimSpace(string(raw)) == "null" {
		return &Config{}, nil
	}
	var c Config
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("helm config decode failed: %w", err)
	}
	return &c, nil
}

type DeploymentParam struct {
	AppName string // Usually: Chart name is AppName or AppType
	AppType string
	Values  map[string]interface{}

	ReleaseName string
	Env         string
	Namespace   string

	Kubeconfig string

	// Chart 相关（v1：从 project_env_configs.artifacts_json 解析而来；为空时 fallback 到全局 helm.repo.*）
	ChartName     string // 已解析后的 chart 名称
	ChartVersion  string // 可选
	ChartRepoURL  string // helm_repo
	ChartUsername string // 可选（basic auth）
	ChartPassword string // 可选（basic auth）

}
