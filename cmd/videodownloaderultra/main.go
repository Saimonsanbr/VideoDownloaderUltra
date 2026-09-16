package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Saimonsanbr/VideoDownloaderUltra/internal/api"
	"github.com/Saimonsanbr/VideoDownloaderUltra/internal/downloader"
	"github.com/Saimonsanbr/VideoDownloaderUltra/internal/ui"

	"github.com/schollz/progressbar/v3"
)

func main() {
	// se sem args ou --gui -> abre GUI
	if len(os.Args) == 1 || (len(os.Args) == 2 && os.Args[1] == "--gui") {
		if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" && os.Getenv("FYNE_FORCE") == "" {
			if isHeadless() {
				printHelp()
				return
			}
		}
		ui.RunGUI()
		return
	}
	// parsing manual para permitir flags antes ou depois da URL (ex: url -o pasta)
	var outDir = "."
	var quality = 1080
	var listOnly, help bool
	var url string
	for i := 1; i < len(os.Args); i++ {
		a := os.Args[i]
		switch a {
		case "-o", "--out":
			if i+1 < len(os.Args) {
				outDir = os.Args[i+1]
				i++
			}
		case "-q", "--quality":
			if i+1 < len(os.Args) {
				fmt.Sscanf(os.Args[i+1], "%d", &quality)
				i++
			}
		case "--list":
			listOnly = true
		case "-h", "--help":
			help = true
		case "--gui":
			ui.RunGUI()
			return
		default:
			if strings.Contains(a, "http") && url == "" {
				url = a
			} else if strings.HasPrefix(a, "-") {
				fmt.Printf("flag desconhecida: %s\n", a)
				printHelp()
				os.Exit(1)
			}
		}
	}
	if help {
		printHelp()
		return
	}
	// fallback para flag package se não achou url (compat)
	if url == "" {
		flag.StringVar(&outDir, "o", ".", "")
		flag.IntVar(&quality, "q", 1080, "")
		flag.BoolVar(&listOnly, "list", false, "")
		flag.BoolVar(&help, "h", false, "")
		flag.BoolVar(&help, "help", false, "")
		flag.Parse()
		if help {
			printHelp()
			return
		}
		if len(flag.Args()) > 0 {
			url = flag.Args()[0]
		}
	}

	if help {
		printHelp()
		return
	}
	if url == "" {
		fmt.Println("Uso: videodownloaderultra <url> [-o pasta] [-q 1080]")
		fmt.Println("Ex: videodownloaderultra https://youtu.be/5q3yoDZljYM -o ./videos")
		fmt.Println("Ou: videodownloaderultra --gui  (abre interface)")
		os.Exit(1)
	}
	if !strings.Contains(url, "http") {
		fmt.Println("URL inválida")
		os.Exit(1)
	}

	fmt.Println("\nConsultando API...")
	info, err := api.FetchInfo(url)
	if err != nil {
		fmt.Printf("Erro API: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("\nTítulo : %s\nFormato: %s\nTamanho: %s (%.2f MB)\nRes    : %dp\n", info.Title, info.Format, info.SizeStr, info.SizeMB, info.Res)

	if listOnly {
		return
	}

	// respeita -q se for menor que best
	if quality < info.Res {
		fmt.Printf("Aviso: melhor disponível é %dp, mas solicitado max %dp - baixando %dp mesmo (use --list)\n", info.Res, quality, info.Res)
	}

	// garante pasta
	if outDir != "" {
		if err := os.MkdirAll(outDir, 0755); err != nil {
			fmt.Printf("Erro criar pasta: %v\n", err)
			os.Exit(1)
		}
	}
	filename := fmt.Sprintf("%s_%dp%s", info.SafeName, info.Res, info.Ext)
	filename = strings.ReplaceAll(filename, "\"", "")
	dest := filepath.Join(outDir, filename)
	fmt.Printf("Arquivo: %s\n\nBaixando \"%s\" [%s] - %s ...\n", dest, info.Title, info.Format, info.SizeStr)

	bar := progressbar.NewOptions64(
		0, // unknown, será atualizado via callback? usa manual
		progressbar.OptionSetDescription("Baixando"),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(40),
		progressbar.OptionClearOnFinish(),
	)

	// usa downloader paralelo com progress que atualiza bar
	// bar precisa de total, então vamos criar wrapper
	err = downloader.DownloadParallel(info.URL, dest, func(d, total int64, speed float64) {
		if total > 0 {
			bar.ChangeMax64(total)
			bar.Set64(d)
			bar.Describe(fmt.Sprintf("%.1f%% %.2f MB/s", float64(d)*100/float64(total), speed/1024/1024))
		}
	})
	if err != nil {
		fmt.Printf("\nFalha downloader nativo: %v\nTentando aria2c embutido...\n", err)
		if err2 := downloader.DownloadWithAria2(info.URL, dest); err2 != nil {
			fmt.Printf("Falha aria2c: %v\n", err2)
			os.Exit(1)
		}
	}
	bar.Finish()
	bar.Clear()
	fmt.Printf("\n✓ Concluído: %s\n", dest)
	if info, err := os.Stat(dest); err == nil {
		fmt.Printf("  Tamanho: %.2f MB\n", float64(info.Size())/1024/1024)
	}
}

func isHeadless() bool {
	// se rodando em CI ou ssh sem display, evita tentar abrir Fyne e travar
	if os.Getenv("CI") != "" {
		return true
	}
	return false
}

func printHelp() {
	fmt.Println(`VideoDownloaderUltra - Downloader de vídeo (teste, API terceiros não oficial)
Uso:
  videodownloaderultra <url> [opções]
  videodownloaderultra --gui          (abre interface gráfica)
  videodownloaderultra                (sem args abre GUI se houver display)

Opções:
  -o, --out <pasta>    pasta de destino (default: pasta atual)
  -q, --quality <n>    qualidade máxima em p (default 1080)
  --list               apenas mostra título/formatos sem baixar
  -h, --help           ajuda

Exemplos:
  videodownloaderultra https://youtu.be/5q3yoDZljYM
  videodownloaderultra https://www.youtube.com/watch?v=... -o ~/Videos -q 720
  videodownloaderultra --gui

Aviso legal: projeto de teste pessoal usando API de terceiros (api.ytultra.com) não oficial e pode parar a qualquer momento. Baixe apenas vídeos com permissão/direitos autorais.`)
}
