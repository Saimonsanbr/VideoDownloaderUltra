package downloader

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

// Progress callback: downloaded, total, speed bytes/sec
type ProgressFunc func(downloaded, total int64, speed float64)

func DownloadParallel(url, dest string, progress ProgressFunc) error {
	// HEAD para pegar tamanho e verificar Range
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	// HEAD com follow redirect
	req, _ := http.NewRequest("HEAD", url, nil)
	resp, err := client.Do(req)
	if err != nil {
		// fallback single se HEAD falhar
		return downloadSingle(url, dest, progress)
	}
	defer resp.Body.Close()

	total := resp.ContentLength
	acceptRanges := resp.Header.Get("Accept-Ranges")
	if total <= 0 || acceptRanges == "" || acceptRanges == "none" {
		return downloadSingle(url, dest, progress)
	}
	// tenta também via Content-Range fallback
	if total <= 0 {
		if cl := resp.Header.Get("Content-Length"); cl != "" {
			if v, err := strconv.ParseInt(cl, 10, 64); err == nil {
				total = v
			}
		}
	}
	if total <= 0 {
		return downloadSingle(url, dest, progress)
	}

	// cria arquivo
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := f.Truncate(total); err != nil {
		return err
	}

	numWorkers := 16
	if total < 16*1024*1024 {
		numWorkers = 8
	}
	if total < 4*1024*1024 {
		numWorkers = 4
	}
	chunkSize := total / int64(numWorkers)

	var wg sync.WaitGroup
	errCh := make(chan error, numWorkers)
	var downloaded int64
	var mu sync.Mutex
	start := time.Now()
	ticker := time.NewTicker(500 * time.Millisecond)
	done := make(chan struct{})
	// progress ticker
	go func() {
		for {
			select {
			case <-ticker.C:
				mu.Lock()
				d := downloaded
				mu.Unlock()
				elapsed := time.Since(start).Seconds()
				speed := float64(d) / elapsed
				if progress != nil {
					progress(d, total, speed)
				}
			case <-done:
				ticker.Stop()
				return
			}
		}
	}()

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		startByte := int64(i) * chunkSize
		endByte := startByte + chunkSize - 1
		if i == numWorkers-1 {
			endByte = total - 1
		}
		go func(s, e int64) {
			defer wg.Done()
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", s, e))
			// http client sem timeout longo por chunk
			c := &http.Client{Timeout: 0}
			resp, err := c.Do(req)
			if err != nil {
				errCh <- err
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != 206 && resp.StatusCode != 200 {
				errCh <- fmt.Errorf("range %d-%d status %d", s, e, resp.StatusCode)
				return
			}
			buf := make([]byte, 32*1024)
			offset := s
			for {
				n, err := resp.Body.Read(buf)
				if n > 0 {
					if _, werr := f.WriteAt(buf[:n], offset); werr != nil {
						errCh <- werr
						return
					}
					offset += int64(n)
					mu.Lock()
					downloaded += int64(n)
					mu.Unlock()
				}
				if err != nil {
					if err == io.EOF {
						break
					}
					errCh <- err
					return
				}
			}
		}(startByte, endByte)
	}
	wg.Wait()
	close(done)
	// final progress
	if progress != nil {
		elapsed := time.Since(start).Seconds()
		progress(total, total, float64(total)/elapsed)
	}
	select {
	case e := <-errCh:
		return e
	default:
		return nil
	}
}

func downloadSingle(url, dest string, progress ProgressFunc) error {
	client := &http.Client{Timeout: 0}
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 && resp.StatusCode != 206 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	total := resp.ContentLength
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	var downloaded int64
	buf := make([]byte, 32*1024)
	start := time.Now()
	lastReport := time.Now()
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				return werr
			}
			downloaded += int64(n)
			if progress != nil && time.Since(lastReport) > 300*time.Millisecond {
				elapsed := time.Since(start).Seconds()
				progress(downloaded, total, float64(downloaded)/elapsed)
				lastReport = time.Now()
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}
	if progress != nil {
		elapsed := time.Since(start).Seconds()
		if elapsed == 0 {
			elapsed = 1
		}
		progress(downloaded, total, float64(downloaded)/elapsed)
	}
	return nil
}
