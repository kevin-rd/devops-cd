package helm

import (
	"bytes"
	"devops-cd/internal/core/deployment/helpers/tpl"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"devops-cd/internal/model"
	"devops-cd/internal/pkg/crypto"

	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

// ParseValuesV1 根据 artifacts_json 中 values[] 生成最终 values map（后者覆盖前者）
func ParseValuesV1(db *gorm.DB, app *model.Application, build *model.Build, env string, cluster string, layers []model.ValuesLayer) (map[string]interface{}, error) {
	ctx := tpl.RenderTemplateContext(app, build, env, cluster)

	merged := map[string]interface{}{}
	for idx, layer := range layers {
		content, err := loadValuesLayerContent(db, ctx, layer)
		if err != nil {
			return nil, fmt.Errorf("values[%d] 加载失败: %w", idx, err)
		}
		if strings.TrimSpace(string(content)) == "" {
			continue
		}

		var obj interface{}
		if err := yaml.Unmarshal(content, &obj); err != nil {
			return nil, fmt.Errorf("values[%d] YAML 解析失败: %w", idx, err)
		}
		m, ok := normalizeYAMLToStringMap(obj).(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("values[%d] YAML 顶层必须是 map/object", idx)
		}
		merged = deepMerge(merged, m)
	}

	// 运行时注入 image.tag（保持旧逻辑）
	if build != nil && build.ImageTag != "" {
		merged = deepMerge(merged, map[string]interface{}{
			"image": map[string]interface{}{
				"tag": build.ImageTag,
			},
		})
	}

	return marshalMeta(merged)
}

// loadValuesLayerContent 加载某一层 values 的 YAML 内容
func loadValuesLayerContent(db *gorm.DB, ctx map[string]interface{}, layer model.ValuesLayer) ([]byte, error) {
	cred, err := resolveCredentialData(db, layer.CredentialRef)
	if err != nil {
		return nil, err
	}

	switch layer.Type {
	case "inline":
		return []byte(layer.Content), nil
	case "http_file", "pipeline_artifact":
		urlTpl := strings.TrimSpace(layer.URLTemplate)
		if urlTpl == "" {
			return nil, fmt.Errorf("url_template 为空")
		}
		url, err := tpl.ParseTemplate(urlTpl, ctx)
		if err != nil {
			return nil, err
		}
		return httpGet(url, cred)
	case "git":
		repo := strings.TrimSpace(layer.RepoURL)
		if repo == "" {
			return nil, fmt.Errorf("repo_url 为空")
		}
		ref := strings.TrimSpace(layer.RefTemplate)
		if ref == "" {
			ref = "main"
		} else {
			r, err := tpl.ParseTemplate(ref, ctx)
			if err != nil {
				return nil, err
			}
			ref = r
		}
		pathTpl := strings.TrimSpace(layer.PathTemplate)
		if pathTpl == "" {
			return nil, fmt.Errorf("path_template 为空")
		}
		relPath, err := tpl.ParseTemplate(pathTpl, ctx)
		if err != nil {
			return nil, err
		}

		repoURL, env, cleanupKey, err := prepareGitAuth(repo, cred)
		if err != nil {
			return nil, err
		}
		defer cleanupKey()

		dir, cleanup, err := gitCheckoutToTemp(repoURL, ref, env)
		if err != nil {
			return nil, err
		}
		defer cleanup()

		abs := filepath.Join(dir, filepath.Clean(relPath))
		// 防止路径穿越：要求最终路径仍在 dir 下
		if !strings.HasPrefix(abs, dir+string(os.PathSeparator)) && abs != dir {
			return nil, fmt.Errorf("path_template 非法: %s", relPath)
		}
		return os.ReadFile(abs)
	default:
		return nil, fmt.Errorf("不支持的 values type: %s", layer.Type)
	}
}

func httpGet(url string, cred map[string]string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	applyHTTPAuth(req, cred)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return io.ReadAll(resp.Body)
}

// gitCheckoutToTemp 以最小依赖方式调用系统 git 拉取指定 ref 到临时目录
func gitCheckoutToTemp(repoURL, ref string, extraEnv []string) (dir string, cleanup func(), err error) {
	base, err := os.MkdirTemp("", "devops-cd-values-*")
	if err != nil {
		return "", nil, err
	}
	cleanup = func() { _ = os.RemoveAll(base) }

	run := func(args ...string) error {
		cmd := exec.Command("git", args...)
		cmd.Dir = base
		// 避免 git 交互
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		if len(extraEnv) > 0 {
			cmd.Env = append(cmd.Env, extraEnv...)
		}
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("git %s failed: %w; output=%s", strings.Join(args, " "), err, strings.TrimSpace(out.String()))
		}
		return nil
	}

	if err := run("init"); err != nil {
		return "", cleanup, err
	}
	if err := run("remote", "add", "origin", repoURL); err != nil {
		return "", cleanup, err
	}
	// depth=1 拉取 ref（branch/tag/commit 都尝试）
	if err := run("fetch", "--depth", "1", "origin", ref); err != nil {
		// fallback: ref 可能是分支名，尝试 refs/heads/
		if err2 := run("fetch", "--depth", "1", "origin", "refs/heads/"+ref); err2 != nil {
			return "", cleanup, err
		}
	}
	if err := run("checkout", "--detach", "FETCH_HEAD"); err != nil {
		return "", cleanup, err
	}
	return base, cleanup, nil
}

func normalizeYAMLToStringMap(v interface{}) interface{} {
	switch t := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(t))
		for k, vv := range t {
			out[k] = normalizeYAMLToStringMap(vv)
		}
		return out
	case map[interface{}]interface{}:
		out := make(map[string]interface{}, len(t))
		for k, vv := range t {
			out[fmt.Sprint(k)] = normalizeYAMLToStringMap(vv)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(t))
		for i := range t {
			out[i] = normalizeYAMLToStringMap(t[i])
		}
		return out
	default:
		return v
	}
}

// deepMerge：map 递归合并；非 map 类型（含数组）直接覆盖
func deepMerge(dst, src map[string]interface{}) map[string]interface{} {
	if dst == nil {
		dst = map[string]interface{}{}
	}
	for k, v := range src {
		if existing, ok := dst[k]; ok {
			em, eok := existing.(map[string]interface{})
			vm, vok := v.(map[string]interface{})
			if eok && vok {
				dst[k] = deepMerge(em, vm)
				continue
			}
		}
		dst[k] = v
	}
	return dst
}

func marshalMeta(m map[string]interface{}) (map[string]interface{}, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	var out map[string]interface{}
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// resolveCredentialData 根据 credential_ref 取出明文（map），不回传给 API，仅内部使用
// 当前支持 credential_ref=纯数字（id）或 "id:123"
func resolveCredentialData(db *gorm.DB, ref string) (map[string]string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, nil
	}
	if db == nil {
		return nil, fmt.Errorf("db is nil, cannot resolve credential_ref")
	}
	idStr := ref
	if strings.HasPrefix(ref, "id:") {
		idStr = strings.TrimPrefix(ref, "id:")
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("credential_ref 非法: %s", ref)
	}

	var c model.Credential
	if err := db.First(&c, id).Error; err != nil {
		return nil, fmt.Errorf("credential_ref=%s 查询失败: %w", ref, err)
	}
	plain, err := crypto.Decrypt(c.EncryptedData)
	if err != nil {
		return nil, fmt.Errorf("credential_ref=%s 解密失败: %w", ref, err)
	}
	var raw map[string]string
	if err := json.Unmarshal([]byte(plain), &raw); err != nil {
		return nil, fmt.Errorf("credential_ref=%s JSON 解析失败: %w", ref, err)
	}
	raw["_type"] = c.Type
	return raw, nil
}

func applyHTTPAuth(req *http.Request, cred map[string]string) {
	if req == nil || cred == nil {
		return
	}
	switch cred["_type"] {
	case "token":
		if t := strings.TrimSpace(cred["token"]); t != "" {
			req.Header.Set("Authorization", "Bearer "+t)
		}
	case "basic_auth":
		u := cred["username"]
		p := cred["password"]
		if u != "" || p != "" {
			req.SetBasicAuth(u, p)
		}
	}
}

// prepareGitAuth 基于凭据生成 git clone 所需的 repoURL/env（v1：basic_auth/token/ssh_key）
func prepareGitAuth(repoURL string, cred map[string]string) (finalURL string, env []string, cleanup func(), err error) {
	finalURL = repoURL
	cleanup = func() {}
	if cred == nil {
		return
	}

	switch cred["_type"] {
	case "token":
		tok := strings.TrimSpace(cred["token"])
		if tok != "" && strings.HasPrefix(repoURL, "https://") {
			finalURL = injectUserInfo(repoURL, tok, "")
		}
	case "basic_auth":
		u := cred["username"]
		p := cred["password"]
		if (u != "" || p != "") && strings.HasPrefix(repoURL, "https://") {
			finalURL = injectUserInfo(repoURL, u, p)
		}
	case "ssh_key":
		key := strings.TrimSpace(cred["private_key"])
		if key == "" {
			return finalURL, env, cleanup, fmt.Errorf("ssh_key.private_key 为空")
		}
		dir, err := os.MkdirTemp("", "devops-cd-ssh-*")
		if err != nil {
			return finalURL, env, cleanup, err
		}
		cleanup = func() { _ = os.RemoveAll(dir) }
		keyPath := filepath.Join(dir, "id_rsa")
		if err := os.WriteFile(keyPath, []byte(key), 0600); err != nil {
			return finalURL, env, cleanup, err
		}
		env = append(env, fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null", keyPath))
	}
	return
}

func injectUserInfo(rawURL, user, pass string) string {
	s := strings.TrimPrefix(rawURL, "https://")
	if pass == "" {
		return "https://" + urlEscapeUser(user) + "@" + s
	}
	return "https://" + urlEscapeUser(user) + ":" + urlEscapeUser(pass) + "@" + s
}

func urlEscapeUser(s string) string {
	s = strings.ReplaceAll(s, "@", "%40")
	s = strings.ReplaceAll(s, ":", "%3A")
	s = strings.ReplaceAll(s, "/", "%2F")
	return s
}
