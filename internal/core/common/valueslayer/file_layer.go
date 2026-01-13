package valueslayer

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const DefaultCacheTTL = 30 * time.Minute

// LoadFileLayer 支持两种模式：
// 1) 本地文件：baseURL==""，path 为相对路径（相对本包的 local root）
// 2) URL 压缩包：baseURL!=""，baseURL 为压缩包 URL，path 为压缩包内目标文件路径
//
// 注意：render 是已绑定 ctx 的模板渲染函数。
func LoadFileLayer(baseURLTemplate string, pathTemplate string, render func(string) (string, error), applyAuth func(*http.Request)) ([]byte, error) {
	baseURLTemplate = strings.TrimSpace(baseURLTemplate)
	pathTemplate = strings.TrimSpace(pathTemplate)

	cacheRoot, err := cacheRootDir()
	if err != nil {
		return nil, err
	}

	if baseURLTemplate == "" {
		if pathTemplate == "" {
			return nil, fmt.Errorf("path_template 为空")
		}
		p, err := render(pathTemplate)
		if err != nil {
			return nil, err
		}
		p = strings.TrimSpace(p)
		if p == "" {
			return nil, fmt.Errorf("path_template 解析结果为空")
		}
		// 安全：只允许相对路径，避免读任意系统文件
		if filepath.IsAbs(p) {
			return nil, fmt.Errorf("file.path_template 不允许使用绝对路径: %s", p)
		}
		if strings.HasPrefix(p, "~") {
			return nil, fmt.Errorf("file.path_template 不支持 ~: %s", p)
		}
		clean := filepath.Clean(p)
		if clean == "." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) || clean == ".." {
			return nil, fmt.Errorf("file.path_template 非法: %s", p)
		}
		localRoot := filepath.Join(cacheRoot, "local")
		abs := filepath.Join(localRoot, clean)
		// 防止路径穿越
		if !strings.HasPrefix(abs, localRoot+string(os.PathSeparator)) && abs != localRoot {
			return nil, fmt.Errorf("file.path_template 非法: %s", p)
		}
		return os.ReadFile(abs)
	}

	// URL 压缩包模式
	if pathTemplate == "" {
		return nil, fmt.Errorf("path_template 为空（file + url 模式需要指定压缩包内路径）")
	}
	u, err := render(baseURLTemplate)
	if err != nil {
		return nil, err
	}
	u = strings.TrimSpace(u)
	if u == "" {
		return nil, fmt.Errorf("base_url_template 解析结果为空")
	}
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		return nil, fmt.Errorf("base_url_template 必须是 http(s) URL: %s", u)
	}

	inner, err := render(pathTemplate)
	if err != nil {
		return nil, err
	}
	inner = strings.TrimSpace(inner)
	if inner == "" {
		return nil, fmt.Errorf("path_template 解析结果为空（file + url 模式需要指定压缩包内路径）")
	}

	innerClean, err := sanitizeArchivePath(inner)
	if err != nil {
		return nil, err
	}

	archiveKind, err := detectArchiveKind(u)
	if err != nil {
		return nil, err
	}

	downloadsDir := filepath.Join(cacheRoot, "downloads")
	extractedDir := filepath.Join(cacheRoot, "extracted")
	if err := os.MkdirAll(downloadsDir, 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(extractedDir, 0o755); err != nil {
		return nil, err
	}

	archiveKey := sha(u)
	archivePath := filepath.Join(downloadsDir, archiveKey+archiveKind.fileExt())
	archiveMod, err := ensureCachedDownload(u, archivePath, applyAuth)
	if err != nil {
		return nil, err
	}

	exKey := sha(u + "|" + innerClean)
	extractedPath := filepath.Join(extractedDir, exKey)
	if ok, err := canUseCache(extractedPath, archiveMod); err != nil {
		return nil, err
	} else if ok {
		return os.ReadFile(extractedPath)
	}

	if err := extractSingleFile(archivePath, archiveKind, innerClean, extractedPath); err != nil {
		return nil, err
	}
	return os.ReadFile(extractedPath)
}

type archiveKind int

const (
	archiveZip archiveKind = iota + 1
	archiveTar
	archiveTgz
)

func (k archiveKind) fileExt() string {
	switch k {
	case archiveZip:
		return ".zip"
	case archiveTar:
		return ".tar"
	case archiveTgz:
		return ".tgz"
	default:
		return ""
	}
}

func detectArchiveKind(url string) (archiveKind, error) {
	u := strings.ToLower(strings.TrimSpace(url))
	switch {
	case strings.HasSuffix(u, ".zip"):
		return archiveZip, nil
	case strings.HasSuffix(u, ".tar"):
		return archiveTar, nil
	case strings.HasSuffix(u, ".tgz"), strings.HasSuffix(u, ".tar.gz"):
		return archiveTgz, nil
	default:
		return 0, fmt.Errorf("不支持的压缩包格式（请使用 .zip/.tar/.tgz/.tar.gz）: %s", url)
	}
}

func cacheRootDir() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil || strings.TrimSpace(dir) == "" {
		dir = os.TempDir()
	}
	if strings.TrimSpace(dir) == "" {
		return "", fmt.Errorf("无法确定缓存目录")
	}
	return filepath.Join(dir, "devops-cd", "values-layer-cache"), nil
}

func sha(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func canUseCache(path string, upstreamMod time.Time) (bool, error) {
	st, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	// 过期或上游更新 -> 不复用
	if time.Since(st.ModTime()) > DefaultCacheTTL {
		return false, nil
	}
	if upstreamMod.After(st.ModTime()) {
		return false, nil
	}
	return true, nil
}

func ensureCachedDownload(url string, dest string, applyAuth func(*http.Request)) (time.Time, error) {
	if ok, err := canUseCache(dest, time.Time{}); err != nil {
		return time.Time{}, err
	} else if ok {
		st, err := os.Stat(dest)
		if err != nil {
			return time.Time{}, err
		}
		return st.ModTime(), nil
	}

	tmp := dest + ".tmp"
	_ = os.Remove(tmp)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return time.Time{}, err
	}
	if applyAuth != nil {
		applyAuth(req)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return time.Time{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return time.Time{}, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return time.Time{}, err
	}
	f, err := os.Create(tmp)
	if err != nil {
		return time.Time{}, err
	}
	_, copyErr := io.Copy(f, resp.Body)
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return time.Time{}, copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return time.Time{}, closeErr
	}
	// 原子替换
	if err := os.Rename(tmp, dest); err != nil {
		// 并发场景：如果另一个 goroutine 已经写好了，直接复用现成的
		_ = os.Remove(tmp)
	}

	st, err := os.Stat(dest)
	if err != nil {
		return time.Time{}, err
	}
	return st.ModTime(), nil
}

func sanitizeArchivePath(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", fmt.Errorf("archive inner path 为空")
	}
	// 归一化为 '/' 风格
	p = strings.ReplaceAll(p, "\\", "/")
	p = strings.TrimLeft(p, "/")
	p = filepath.Clean(p)
	// filepath.Clean 在 Windows 会用 '\'，再归一化一次
	p = strings.ReplaceAll(p, "\\", "/")
	if p == "." || p == ".." || strings.HasPrefix(p, "../") {
		return "", fmt.Errorf("archive inner path 非法: %s", p)
	}
	return p, nil
}

func extractSingleFile(archivePath string, kind archiveKind, innerPath string, outPath string) error {
	switch kind {
	case archiveZip:
		return extractZipSingle(archivePath, innerPath, outPath)
	case archiveTar:
		return extractTarSingle(archivePath, innerPath, outPath, false)
	case archiveTgz:
		return extractTarSingle(archivePath, innerPath, outPath, true)
	default:
		return fmt.Errorf("unknown archive kind")
	}
}

func extractZipSingle(archivePath string, innerPath string, outPath string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		name := strings.TrimPrefix(f.Name, "./")
		name = strings.ReplaceAll(name, "\\", "/")
		name = strings.TrimLeft(name, "/")
		name = filepath.Clean(name)
		name = strings.ReplaceAll(name, "\\", "/")
		if name != innerPath {
			continue
		}
		if f.FileInfo().IsDir() {
			return fmt.Errorf("archive entry is a directory: %s", innerPath)
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()
		return writeAtomically(outPath, rc)
	}
	return fmt.Errorf("压缩包内未找到文件: %s", innerPath)
}

func extractTarSingle(archivePath string, innerPath string, outPath string, gz bool) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	var r io.Reader = f
	if gz {
		gr, err := gzip.NewReader(f)
		if err != nil {
			return err
		}
		defer gr.Close()
		r = gr
	}

	tr := tar.NewReader(r)
	for {
		h, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		name := strings.TrimPrefix(h.Name, "./")
		name = strings.ReplaceAll(name, "\\", "/")
		name = strings.TrimLeft(name, "/")
		name = filepath.Clean(name)
		name = strings.ReplaceAll(name, "\\", "/")
		if name != innerPath {
			continue
		}
		switch h.Typeflag {
		case tar.TypeReg, tar.TypeRegA:
			return writeAtomically(outPath, tr)
		default:
			return fmt.Errorf("archive entry is not a regular file: %s", innerPath)
		}
	}
	return fmt.Errorf("压缩包内未找到文件: %s", innerPath)
}

func writeAtomically(dest string, r io.Reader) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	tmp := dest + ".tmp"
	_ = os.Remove(tmp)

	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(f, r)
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
