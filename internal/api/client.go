package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const Endpoint = "https://api.ytultra.com/ikool/youtube/download"

type response struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data *Data  `json:"data"`
}

type Data struct {
	Title    string  `json:"title"`
	ImageURL string  `json:"imageUrl"`
	Duration string  `json:"duration"`
	Medias   []Media `json:"medias"`
}

type Media struct {
	URL      string `json:"url"`
	Format   string `json:"format"`
	FileSize int64  `json:"fileSize"`
	SizeStr  string `json:"sizeStr"`
}

type VideoInfo struct {
	Title    string
	SafeName string
	Best     Media
	Format   string
	SizeStr  string
	SizeMB   float64
	Ext      string
	URL      string
	Res      int
}

var reRes = regexp.MustCompile(`(\d+)p`)
var reSize = regexp.MustCompile(`\(([\d\.]+)\s*(GB|MB)\)`)
var reExt = regexp.MustCompile(`\[\.(\w+)\]`)

func FetchInfo(youtubeURL string) (*VideoInfo, error) {
	body, _ := json.Marshal(map[string]string{"url": youtubeURL})
	req, err := http.NewRequest("POST", Endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("origin", "https://www.ytultra.com")
	req.Header.Set("referer", "https://www.ytultra.com/")
	req.Header.Set("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(b[:min(500, len(b))]))
	}
	var r response
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}
	if r.Code != "0000" {
		return nil, fmt.Errorf("API error %s: %s", r.Code, r.Msg)
	}
	if r.Data == nil || len(r.Data.Medias) == 0 {
		return nil, fmt.Errorf("nenhum formato encontrado")
	}
	return selectBest(r.Data)
}

func selectBest(data *Data) (*VideoInfo, error) {
	title := strings.TrimSpace(strings.Split(data.Title, "\n")[0])
	if title == "" {
		title = "video"
	}
	title = strings.ReplaceAll(title, "\"", "'")
	if len(title) > 80 {
		title = title[:80]
	}

	// filtra melhor <=1080p, video-only (sem audio)
	type cand struct {
		res int
		m   Media
	}
	var cands []cand
	for _, m := range data.Medias {
		if m.Format == "" || m.URL == "" {
			continue
		}
		// ignora só-audio se tiver video? mas usuário quer só video sem audio, então aceita video/*
		// se quiser só video, filtra mime via url? mas format já indica. Mantém todos com p
		mm := reRes.FindStringSubmatch(m.Format)
		if len(mm) < 2 {
			continue
		}
		var res int
		fmt.Sscanf(mm[1], "%d", &res)
		if res <= 1080 && res > 0 {
			cands = append(cands, cand{res, m})
		}
	}
	// fallback: qualquer p
	if len(cands) == 0 {
		for _, m := range data.Medias {
			mm := reRes.FindStringSubmatch(m.Format)
			if len(mm) >= 2 {
				var res int
				fmt.Sscanf(mm[1], "%d", &res)
				cands = append(cands, cand{res, m})
			}
		}
	}
	if len(cands) == 0 {
		return nil, fmt.Errorf("nenhum formato com resolucao encontrado")
	}
	// ordena maior primeiro
	bestRes := -1
	var best Media
	for _, c := range cands {
		if c.res > bestRes {
			bestRes = c.res
			best = c.m
		}
	}
	fmtStr := best.Format
	if fmtStr == "" {
		fmtStr = fmt.Sprintf("%dp", bestRes)
	}
	// size
	sizeStr := ""
	sizeMB := 0.0
	if m := reSize.FindStringSubmatch(fmtStr); len(m) == 3 {
		var v float64
		fmt.Sscanf(m[1], "%f", &v)
		if m[2] == "GB" {
			sizeMB = v * 1024
		} else {
			sizeMB = v
		}
		sizeStr = fmt.Sprintf("%s %s", m[1], m[2])
	}
	if best.FileSize > 0 {
		sz := best.FileSize
		if sz < 0 {
			sz += 1 << 32
		}
		sizeMB = float64(sz) / 1024 / 1024
		sizeStr = fmt.Sprintf("%.2f MB", sizeMB)
	} else if best.FileSize < 0 {
		// overflow corrigido
		sz := best.FileSize + (1 << 32)
		sizeMB = float64(sz) / 1024 / 1024
		sizeStr = fmt.Sprintf("%.2f MB", sizeMB)
	}
	if sizeStr == "" {
		sizeStr = fmt.Sprintf("%.2f MB", sizeMB)
	}
	ext := ".mp4"
	if m := reExt.FindStringSubmatch(fmtStr); len(m) == 2 {
		ext = "." + m[1]
	} else if strings.Contains(fmtStr, "webm") {
		ext = ".webm"
	}
	safe := sanitize(title)
	return &VideoInfo{
		Title:    title,
		SafeName: safe,
		Best:     best,
		Format:   fmtStr,
		SizeStr:  sizeStr,
		SizeMB:   sizeMB,
		Ext:      ext,
		URL:      best.URL,
		Res:      bestRes,
	}, nil
}

func sanitize(s string) string {
	// remove emojis e caracteres problematicos, mantém alfanum
	re := regexp.MustCompile(`[^\w\-\s\(\)\[\]]`)
	s = re.ReplaceAllString(s, "")
	re2 := regexp.MustCompile(`\s+`)
	s = re2.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	if len(s) > 60 {
		s = s[:60]
	}
	s = strings.Trim(s, "_")
	if s == "" {
		s = "video"
	}
	return s
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
