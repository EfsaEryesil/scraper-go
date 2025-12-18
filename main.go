package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
)

const (
	maxHTMLBytes = 10 << 20
	userAgent    = "GoScraper/1.0 (CTI assignment)"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Kullanım: scraper <url>")
		os.Exit(1)
	}

	raw := strings.TrimSpace(os.Args[1])
	targetURL, err := normalizeURL(raw)
	if err != nil {
		fmt.Printf("URL hatalı: %v\n", err)
		os.Exit(1)
	}

	htmlBytes, status, err := fetchHTML(targetURL)
	if err != nil {
		fmt.Printf("HTTP hata: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("[INFO] HTTP %d - HTML alındı (%d byte)\n", status, len(htmlBytes))
	if status >= 400 {
		fmt.Printf("[WARN] Sunucu hata döndürdü: HTTP %d\n", status)
	}

	htmlPath := "site_data.html"
	if err := os.WriteFile(htmlPath, htmlBytes, 0644); err != nil {
		fmt.Printf("Dosyaya yazma hatası (%s): %v\n", htmlPath, err)
		os.Exit(1)
	}
	fmt.Printf("[OK] HTML kaydedildi: %s\n", htmlPath)

	if err := takeScreenshot(targetURL, "screenshot.png"); err != nil {
		fmt.Printf("[WARN] Screenshot alınamadı: %v\n", err)
	} else {
		fmt.Println("[OK] Screenshot kaydedildi: screenshot.png")
	}

	if err := extractLinks(targetURL, htmlBytes, "links.txt"); err != nil {
		fmt.Printf("[WARN] Link çıkarma yapılamadı: %v\n", err)
	} else {
		fmt.Println("[OK] Link listesi kaydedildi: links.txt")
	}
}

func normalizeURL(raw string) (string, error) {

	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.ParseRequestURI(raw)
	if err != nil {
		return "", err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("desteklenmeyen scheme: %s", u.Scheme)
	}
	return u.String(), nil
}

func fetchHTML(target string) ([]byte, int, error) {
	client := &http.Client{Timeout: 20 * time.Second}

	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", userAgent)

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxHTMLBytes))
	if err != nil {
		return nil, resp.StatusCode, err
	}

	fmt.Printf("[INFO] Süre: %s\n", time.Since(start).Round(time.Millisecond))
	return body, resp.StatusCode, nil
}

func takeScreenshot(target, outFile string) error {
	allocOpts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), allocOpts...)
	defer cancel()

	ctx, cancel2 := chromedp.NewContext(allocCtx)
	defer cancel2()

	ctx, cancel3 := context.WithTimeout(ctx, 30*time.Second)
	defer cancel3()

	var buf []byte
	err := chromedp.Run(ctx, chromedp.Tasks{
		chromedp.Navigate(target),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.FullScreenshot(&buf, 90),
	})
	if err == nil {
		return os.WriteFile(outFile, buf, 0644)
	}

	var buf2 []byte
	err2 := chromedp.Run(ctx, chromedp.Tasks{
		chromedp.Navigate(target),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.EmulateViewport(1366, 768),
		chromedp.CaptureScreenshot(&buf2),
	})
	if err2 != nil {
		return err
	}

	return os.WriteFile(outFile, buf2, 0644)
}

func extractLinks(baseURL string, html []byte, outFile string) error {
	base, err := url.Parse(baseURL)
	if err != nil {
		return err
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(html)))
	if err != nil {
		return err
	}

	seen := make(map[string]struct{})
	var lines []string

	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, ok := s.Attr("href")
		if !ok {
			return
		}
		href = strings.TrimSpace(href)
		if href == "" {
			return
		}

		u, err := url.Parse(href)
		if err != nil {
			return
		}

		abs := base.ResolveReference(u).String()

		if strings.HasPrefix(abs, "mailto:") || strings.HasPrefix(abs, "javascript:") {
			return
		}

		if _, exists := seen[abs]; !exists {
			seen[abs] = struct{}{}
			lines = append(lines, abs)
		}
	})

	if len(lines) == 0 {
		return os.WriteFile(outFile, []byte(""), 0644)
	}

	return os.WriteFile(outFile, []byte(strings.Join(lines, "\n")+"\n"), 0644)
}
