# VideoDownloaderUltra

> **⚠️ Aviso Legal / Disclaimer - Projeto de teste pessoal**
>
> Este projeto é **apenas para fins de teste e estudo**, usando uma **API de terceiros não oficial** (`https://api.ytultra.com/ikool/youtube/download`) que **pode parar de funcionar a qualquer momento** sem aviso. **Não é um serviço oficial do YouTube.**
>
> **Não baixe vídeos sem permissão.** Respeite os Termos de Serviço do YouTube, direitos autorais e leis locais. Baixe apenas conteúdo que você criou ou tem autorização expressa do detentor dos direitos. O autor não se responsabiliza por uso indevido.

Downloader rápido de vídeo **apenas video sem áudio** (video-only) até `1080p` (fallback `720p>480p>...`). Binário único `videodownloaderultra` funciona como **CLI** e como **GUI** (pesquisável no menu de apps).

- **CLI:** `videodownloaderultra <url> -o pasta -q 1080`
- **GUI:** `videodownloaderultra` sem args ou `videodownloaderultra --gui` -> janela com URL, pasta, botão Baixar, barra + logs
- Pasta padrão = **onde foi chamado** (`pwd`)
- Downloader nativo paralelo **16 conexões** (igual/mais rápido que navegador, bypass throttle `googlevideo`), com fallback `aria2c` embutido (macOS `arm64/amd64` já embutido, Linux/Win via downloader nativo)

## Instalação rápida

### macOS / Linux (Debian 13 testado)
```bash
curl -fsSL https://raw.githubusercontent.com/Saimonsanbr/VideoDownloaderUltra/main/scripts/install.sh | sh
# ou com wget
# wget -qO- https://raw.githubusercontent.com/Saimonsanbr/VideoDownloaderUltra/main/scripts/install.sh | sh
```
Instala em `/usr/local/bin` (ou `~/.local/bin` se sem sudo) e adiciona ao `PATH`.

**macOS Gatekeeper (xattr):**
```bash
xattr -c /usr/local/bin/videodownloaderultra
# ou se baixou via release zip:
# xattr -c videodownloaderultra
```

### Windows (PowerShell como Admin opcional)
```powershell
irm https://raw.githubusercontent.com/Saimonsanbr/VideoDownloaderUltra/main/scripts/install.ps1 | iex
# adiciona %USERPROFILE%\bin ao PATH
```

### Manual (GitHub Releases)
Baixe o binário para seu SO em [Releases](https://github.com/Saimonsanbr/VideoDownloaderUltra/releases) (`videodownloaderultra-darwin-arm64`, `...-linux-amd64`, `...-windows-amd64.exe`), `chmod +x` e coloque no `PATH`.

## Uso

```bash
# CLI - baixa na pasta atual
videodownloaderultra https://youtu.be/5q3yoDZljYM

# CLI com pasta e qualidade
videodownloaderultra "https://www.youtube.com/watch?v=dQw4w9WgXcQ" -o ~/Videos -q 720

# GUI
videodownloaderultra --gui
# ou apenas
videodownloaderultra
# pesquisável no menu: "VideoDownloaderUltra"
```

GUI: 2 campos (Link + Pasta) + Botão Baixar + quadrado de logs + barra de progresso.

## Build local

```bash
go mod tidy
go run ./cmd/videodownloaderultra --help
go run ./cmd/videodownloaderultra https://youtu.be/5q3yoDZljYM

# GUI
go run ./cmd/videodownloaderultra --gui

# cross
GOOS=windows GOARCH=amd64 go build -o videodownloaderultra.exe ./cmd/videodownloaderultra
GOOS=linux GOARCH=amd64 go build -o videodownloaderultra-linux ./cmd/videodownloaderultra
GOOS=darwin GOARCH=arm64 go build -o videodownloaderultra-darwin-arm64 ./cmd/videodownloaderultra
```

## Como funciona

1. `POST https://api.ytultra.com/ikool/youtube/download` com `{"url": "..."}` (headers `origin/referer`)
2. Filtra `medias[]` com regex `(\d+)p` <=1080, pega maior (ex `1080p (77MB) [.mp4]`)
3. Baixa `redirector.googlevideo.com` (2 redirects `302` até `r*---sn-*.googlevideo.com`) com downloader paralelo nativo 16 `Range` + fallback `aria2c` embutido.

## Licença

MIT - veja `LICENSE`.
