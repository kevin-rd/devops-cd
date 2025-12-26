package model

import (
	"encoding/json"
	"fmt"
)

// ArtifactsV1 是 project_env_configs.artifacts_json 的 v1 结构
//
// 约束：
// - namespace_template 放在顶层，pre/main 阶段共用 namespace
// - config_chart/app_chart 表示两个固定阶段（pre/main），由 enabled 控制是否执行
type ArtifactsV1 struct {
	SchemaVersion     int    `json:"schema_version"`
	NamespaceTemplate string `json:"namespace_template"`

	ConfigChart *StageSpecV1 `json:"config_chart,omitempty"`
	AppChart    *StageSpecV1 `json:"app_chart,omitempty"`
}

// StageSpecV1 表示某一阶段的 driver 配置（当前固定为 pre/main 两阶段）。
//
// 设计目标：artifacts 层不耦合具体 driver 字段，driver 自行解析 data。
// - type: driver 名称（例如 "helm"）
// - data: driver 私有配置（例如 helm 的 repo_url/values 等）
type StageSpecV1 struct {
	Enabled bool            `json:"enabled"`
	Type    string          `json:"type"` // driver type，例如 "helm"
	Data    json.RawMessage `json:"data,omitempty"`
}

type ValuesLayer struct {
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

func LoadArtifactsV1(artifactsJSON *string) (*ArtifactsV1, error) {
	if artifactsJSON != nil && *artifactsJSON != "" {
		var a ArtifactsV1
		if err := json.Unmarshal([]byte(*artifactsJSON), &a); err != nil {
			return nil, fmt.Errorf("artifacts_json 解析失败: %w", err)
		}
		if a.SchemaVersion == 0 {
			a.SchemaVersion = 1
		}
		return &a, nil
	}
	return DefaultArtifactsV1(), nil
}

func NormalizeArtifactsJSON(raw []byte, schemaVersion int) (normalized string, finalSchemaVersion int, err error) {
	if len(raw) == 0 {
		return "", schemaVersion, nil
	}

	var a ArtifactsV1
	if err := json.Unmarshal(raw, &a); err != nil {
		return "", schemaVersion, fmt.Errorf("artifacts_json 解析失败: %w", err)
	}
	if a.SchemaVersion == 0 {
		a.SchemaVersion = schemaVersion
	}
	if a.SchemaVersion == 0 {
		a.SchemaVersion = 1
	}
	b, err := json.Marshal(&a)
	if err != nil {
		return "", a.SchemaVersion, err
	}
	return string(b), a.SchemaVersion, nil
}

func DefaultArtifactsV1() *ArtifactsV1 {
	return &ArtifactsV1{
		SchemaVersion:     1,
		NamespaceTemplate: "",
		ConfigChart: &StageSpecV1{
			Enabled: false,
			Type:    "helm",
			Data:    json.RawMessage([]byte(`{}`)),
		},
		AppChart: &StageSpecV1{
			Enabled: true,
			Type:    "helm",
			Data:    json.RawMessage([]byte(`{}`)),
		},
	}
}
