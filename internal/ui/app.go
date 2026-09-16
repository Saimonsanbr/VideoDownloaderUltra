package ui

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/Saimonsanbr/VideoDownloaderUltra/internal/api"
	"github.com/Saimonsanbr/VideoDownloaderUltra/internal/downloader"
)

func RunGUI() {
	// encontra porta livre
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("falha porta: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/", serveHTML)
	mux.HandleFunc("/api/info", handleInfo)
	mux.HandleFunc("/api/download", handleDownload)
	mux.HandleFunc("/api/pick-folder", handlePickFolder)

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	url := "http://" + addr + "/"
	fmt.Printf("\nVideoDownloaderUltra GUI em %s\nAbrindo navegador...\n", url)
	openBrowser(url)

	fmt.Printf("Se o navegador não abrir, acesse manualmente: %s\n", url)
	fmt.Println("Pressione Ctrl+C para sair")

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

var htmlPage = `<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><title>VideoDownloaderUltra</title>
<style>
body{font-family:system-ui,Arial,sans-serif;max-width:700px;margin:20px auto;padding:20px;background:#1a1a1a;color:#eee}
h1{color:#ff3b30;text-align:center}
input,button{width:100%;padding:12px;margin:8px 0;box-sizing:border-box;font-size:14px;border-radius:8px;border:1px solid #333}
input{background:#2a2a2a;color:#fff}
button{background:#007aff;color:#fff;border:none;cursor:pointer;font-weight:bold}
button:disabled{background:#555;cursor:not-allowed}
#logs{background:#000;color:#0f0;padding:10px;height:250px;overflow-y:auto;white-space:pre-wrap;font-family:monospace;font-size:12px;border:1px solid #333;border-radius:8px}
#progress{width:100%;height:20px;background:#333;border-radius:10px;overflow:hidden;margin:10px 0}
#progress div{height:100%;background:#007aff;width:0%;transition:width 0.3s}
#status{text-align:center;color:#aaa;font-size:13px}
.row{display:flex;gap:8px}
.row input{flex:1}
.row button{width:120px}
</style>
</head>
<body>
<h1>VideoDownloaderUltra</h1>
<p style="text-align:center;color:#888;font-size:12px">Teste pessoal - API terceiros não oficial, pode parar a qualquer momento. Baixe apenas com permissão.</p>
<label>Link do vídeo:</label>
<input id="url" placeholder="https://www.youtube.com/watch?v=... ou https://youtu.be/...">
<label>Pasta de downloads (clique em Escolher para abrir Finder/Explorer):</label>
<div class="row">
<input id="folder" placeholder="vazio = pasta onde o programa foi aberto" value="." readonly style="cursor:pointer" onclick="pickFolder()" title="Clique em Escolher para abrir o seletor nativo">
<button onclick="pickFolder()">Escolher pasta</button>
</div>
<button id="btn" onclick="baixar()">Baixar</button>
<div id="progress"><div id="bar"></div></div>
<div id="status">Aguardando...</div>
<div id="logs"></div>
<script>
function log(m){const e=document.getElementById('logs');e.textContent+=m+"\\n";e.scrollTop=e.scrollHeight}
function setStatus(s){document.getElementById('status').textContent=s}
function setProgress(p){document.getElementById('bar').style.width=p+"%"}
async function pickFolder(){
  try{
    setStatus("Abrindo Finder/Explorer...");
    const r=await fetch("/api/pick-folder",{method:"POST"});
    const j=await r.json();
    if(j.path){
      document.getElementById('folder').value=j.path;
      log("Pasta escolhida: "+j.path);
      setStatus("Pasta: "+j.path);
      return;
    }
    if(j.error) throw new Error(j.error);
  }catch(e){
    log("Dialog nativo falhou: "+e.message+" - usando prompt manual");
  }
  const f=prompt("Digite o caminho da pasta (ou deixe . para pasta atual):", document.getElementById('folder').value);
  if(f!==null) document.getElementById('folder').value=f;
  setStatus("Aguardando...");
}
async function baixar(){
  const url=document.getElementById('url').value.trim();
  const folder=document.getElementById('folder').value.trim()||".";
  if(!url){alert("Cole o link");return}
  const btn=document.getElementById('btn');
  btn.disabled=true; setStatus("Consultando API..."); log("Consultando API para "+url+"...");
  try{
    const r=await fetch("/api/info",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({url})});
    const j=await r.json();
    if(!r.ok){throw new Error(j.error||"erro API")}
    log("Título: "+j.title);
    log("Formato: "+j.format+" | Tamanho: "+j.sizeStr+" | Res: "+j.res+"p");
    log("Arquivo: "+j.filename);
    setStatus("Baixando "+j.format+"...");
    // inicia download com SSE
    const es=new EventSource("/api/download?url="+encodeURIComponent(url)+"&folder="+encodeURIComponent(folder));
    es.onmessage=function(e){
      try{
        const d=JSON.parse(e.data);
        if(d.type=="progress"){
          setProgress(d.percent);
          setStatus(d.percent.toFixed(1)+"% - "+d.speed);
        } else if(d.type=="log"){
          log(d.msg);
        } else if(d.type=="done"){
          setProgress(100); setStatus("Concluído: "+d.file); log("✓ Concluído: "+d.file);
          es.close(); btn.disabled=false;
        } else if(d.type=="error"){
          log("✗ Erro: "+d.msg); setStatus("Erro"); es.close(); btn.disabled=false;
        }
      }catch(e){log(e.data)}
    };
    es.onerror=function(){log("Conexão SSE falhou"); es.close(); btn.disabled=false;}
  }catch(e){log("Erro: "+e.message); setStatus("Erro"); btn.disabled=false;}
}
</script>
</body>
</html>`

func serveHTML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(htmlPage))
}

func handleInfo(w http.ResponseWriter, r *http.Request) {
	var req struct{ URL string `json:"url"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"json invalido"}`, 400)
		return
	}
	info, err := api.FetchInfo(req.URL)
	if err != nil {
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	filename := fmt.Sprintf("%s_%dp%s", info.SafeName, info.Res, info.Ext)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"title":    info.Title,
		"format":   info.Format,
		"sizeStr":  info.SizeStr,
		"res":      info.Res,
		"ext":      info.Ext,
		"filename": filename,
	})
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	urlStr := r.URL.Query().Get("url")
	folder := r.URL.Query().Get("folder")
	if folder == "" {
		folder = "."
	}
	// SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE não suportado", 500)
		return
	}

	send := func(data interface{}) {
		b, _ := json.Marshal(data)
		fmt.Fprintf(w, "data: %s\n\n", b)
		flusher.Flush()
	}

	info, err := api.FetchInfo(urlStr)
	if err != nil {
		send(map[string]string{"type": "error", "msg": err.Error()})
		return
	}
	// garante pasta
	if folder == "." {
		// pasta onde o binário foi chamado, não onde está o binário
		if wd, err := os.Getwd(); err == nil {
			folder = wd
		}
	}
	if err := os.MkdirAll(folder, 0755); err != nil {
		send(map[string]string{"type": "error", "msg": "pasta: " + err.Error()})
		return
	}
	filename := fmt.Sprintf("%s_%dp%s", info.SafeName, info.Res, info.Ext)
	dest := filepath.Join(folder, filename)
	send(map[string]string{"type": "log", "msg": "Baixando para " + dest + " ..."})

	var mu sync.Mutex
	lastPercent := -1.0
	err = downloader.DownloadParallel(info.URL, dest, func(d, total int64, speed float64) {
		if total <= 0 {
			return
		}
		percent := float64(d) * 100 / float64(total)
		mu.Lock()
		if percent-lastPercent >= 0.5 || percent >= 99.9 {
			lastPercent = percent
			send(map[string]interface{}{
				"type":    "progress",
				"percent": percent,
				"speed":   fmt.Sprintf("%.2f MB/s", speed/1024/1024),
			})
		}
		mu.Unlock()
	})
	if err != nil {
		// tenta aria2 fallback
		send(map[string]string{"type": "log", "msg": "Falha paralelo, tentando aria2c: " + err.Error()})
		if err2 := downloader.DownloadWithAria2(info.URL, dest); err2 != nil {
			send(map[string]string{"type": "error", "msg": err2.Error()})
			return
		}
	}
	send(map[string]string{"type": "done", "file": dest})
}

func handlePickFolder(w http.ResponseWriter, r *http.Request) {
	path, err := pickFolderNative()
	if err != nil {
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"path": path})
}

func pickFolderNative() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		// Finder nativo via AppleScript
		out, err := exec.Command("osascript", "-e", `POSIX path of (choose folder with prompt "Escolha a pasta de downloads")`).Output()
		if err != nil {
			return "", fmt.Errorf("Finder cancelado ou erro: %w", err)
		}
		s := strings.TrimSpace(string(out))
		// osascript retorna com / no final, mantém
		return s, nil
	case "windows":
		// Explorer nativo via PowerShell FolderBrowserDialog
		ps := `Add-Type -AssemblyName System.Windows.Forms; $f=New-Object System.Windows.Forms.FolderBrowserDialog; $f.Description="Escolha a pasta de downloads"; $f.ShowNewFolderButton=$true; if($f.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK){Write-Output $f.SelectedPath} else { exit 1 }`
		out, err := exec.Command("powershell", "-NoProfile", "-Command", ps).Output()
		if err != nil {
			return "", fmt.Errorf("Explorer cancelado ou erro: %w", err)
		}
		s := strings.TrimSpace(string(out))
		if s == "" {
			return "", fmt.Errorf("nenhuma pasta escolhida")
		}
		return s, nil
	default:
		// Linux Debian 13: tenta zenity, kdialog, yad nessa ordem
		if _, err := exec.LookPath("zenity"); err == nil {
			out, err := exec.Command("zenity", "--file-selection", "--directory", "--title=Escolha a pasta de downloads").Output()
			if err != nil {
				return "", fmt.Errorf("zenity cancelado")
			}
			return strings.TrimSpace(string(out)), nil
		}
		if _, err := exec.LookPath("kdialog"); err == nil {
			out, err := exec.Command("kdialog", "--getexistingdirectory", ".", "--title", "Escolha a pasta").Output()
			if err != nil {
				return "", fmt.Errorf("kdialog cancelado")
			}
			return strings.TrimSpace(string(out)), nil
		}
		if _, err := exec.LookPath("yad"); err == nil {
			out, err := exec.Command("yad", "--file", "--directory", "--title=Escolha a pasta").Output()
			if err != nil {
				return "", fmt.Errorf("yad cancelado")
			}
			return strings.TrimSpace(string(out)), nil
		}
		return "", fmt.Errorf("nenhum dialog nativo encontrado (instale zenity: sudo apt install zenity)")
	}
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
