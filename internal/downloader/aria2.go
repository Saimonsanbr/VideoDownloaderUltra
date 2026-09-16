package downloader

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

//go:embed aria2bin/*
var aria2FS embed.FS

// GetAria2Path extrai binário embutido para temp e retorna path
func GetAria2Path() (string, error) {
	var name string
	switch runtime.GOOS {
	case "windows":
		name = "aria2c-windows.exe"
	case "darwin":
		if runtime.GOARCH == "arm64" {
			name = "aria2c-darwin-arm64"
		} else {
			name = "aria2c-darwin-amd64"
		}
	case "linux":
		name = "aria2c-linux-amd64"
	default:
		return "", fmt.Errorf("unsupported OS")
	}
	data, err := aria2FS.ReadFile("aria2bin/" + name)
	if err != nil {
		return "", fmt.Errorf("aria2 bin not embedded for %s: %w (use native downloader)", name, err)
	}
	tmp := filepath.Join(os.TempDir(), name)
	// se já existe e tem mesmo tamanho, reutiliza
	if info, err := os.Stat(tmp); err == nil && info.Size() == int64(len(data)) {
		return tmp, nil
	}
	if err := os.WriteFile(tmp, data, 0755); err != nil {
		return "", err
	}
	return tmp, nil
}

// DownloadWithAria2 tenta usar aria2c embutido, fallback false se não tiver
func DownloadWithAria2(url, dest string) error {
	bin, err := GetAria2Path()
	if err != nil {
		return err
	}
	dir := filepath.Dir(dest)
	base := filepath.Base(dest)
	cmd := exec.Command(bin, "-x16", "-s16", "-k1M", "--summary-interval=1", "--allow-overwrite=true", "--auto-file-renaming=false", "--max-connection-per-server=16", "--min-split-size=1M", "-o", base, url)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// HasEmbedded verifica se tem binário para OS atual
func HasEmbedded() bool {
	_, err := GetAria2Path()
	return err == nil
}
