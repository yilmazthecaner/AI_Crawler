package crawler

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"spidersearch/internal/index"
	"strings"
	"sync"
	"time"
)

type JobStatus string

const (
	StatusRunning   JobStatus = "running"
	StatusFinished  JobStatus = "finished"
	StatusError     JobStatus = "error"
	StatusCancelled JobStatus = "cancelled"
)

var (
	linkPattern       = regexp.MustCompile(`(?i)href=["']([^"'#]+)["']`)
	scriptPattern     = regexp.MustCompile(`(?is)<script.*?</script>`)
	stylePattern      = regexp.MustCompile(`(?is)<style.*?</style>`)
	htmlTagPattern    = regexp.MustCompile(`(?s)<[^>]+>`)
	bodyTokenPattern  = regexp.MustCompile(`[a-z0-9]+`)
	defaultUserAgent  = "SpiderSearch/1.0 (+https://localhost)"
	maxResponseBytes  = int64(5 << 20)
	defaultHTTPClient = &http.Client{Timeout: 15 * time.Second}
)

type Job struct {
	ID                 string           `json:"id"`
	OriginURL          string           `json:"origin_url"`
	Depth              int              `json:"depth"`
	Status             JobStatus        `json:"status"`
	Logs               []string         `json:"logs"`
	Queue              []string         `json:"queue"`
	StartTime          time.Time        `json:"start_time"`
	EndTime            time.Time        `json:"end_time"`
	Duration           int              `json:"duration"`
	MaxWorkers         int              `json:"max_workers"`
	HitRate            int              `json:"hit_rate"`
	QueueCapacity      int              `json:"queue_capacity"`
	MaxURLs            int              `json:"max_urls"`
	CrawledCount       int              `json:"crawled_count"`
	ErrorCount         int              `json:"error_count"`
	TotalFound         int              `json:"total_found"`
	ActiveWorkers      int              `json:"active_workers"`
	BackPressureActive bool             `json:"back_pressure_active"`
	BackPressureReason string           `json:"back_pressure_reason"`
	FileIndex          *index.FileIndex `json:"-"`
	mu                 sync.Mutex
	wg                 sync.WaitGroup
	visitedFile        string
	visited            map[string]bool
	semaphore          chan struct{}
	rateLimiter        *time.Ticker
	ctx                context.Context
	cancel             context.CancelFunc
	httpClient         *http.Client
}

func NewJob(origin string, depth, workers, hitRate, queueCap, maxURLs int, fi *index.FileIndex) *Job {
	if workers <= 0 {
		workers = 10
	}
	if hitRate <= 0 {
		hitRate = 10
	}

	ctx, cancel := context.WithCancel(context.Background())
	ticker := time.NewTicker(time.Second / time.Duration(hitRate))

	return &Job{
		ID:            fmt.Sprintf("%d_%d", time.Now().Unix(), time.Now().UnixNano()%1000),
		OriginURL:     origin,
		Depth:         depth,
		Status:        StatusRunning,
		StartTime:     time.Now(),
		MaxWorkers:    workers,
		HitRate:       hitRate,
		QueueCapacity: queueCap,
		MaxURLs:       maxURLs,
		FileIndex:     fi,
		visitedFile:   index.VisitedURLsFile,
		visited:       loadVisitedSet(index.VisitedURLsFile),
		semaphore:     make(chan struct{}, workers),
		rateLimiter:   ticker,
		ctx:           ctx,
		cancel:        cancel,
		httpClient:    defaultHTTPClient,
	}
}

func loadVisitedSet(path string) map[string]bool {
	visited := make(map[string]bool)

	file, err := os.Open(path)
	if err != nil {
		return visited
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		visited[line] = true
	}

	return visited
}

func (j *Job) Cancel() {
	j.mu.Lock()
	if j.Status != StatusRunning {
		j.mu.Unlock()
		return
	}
	j.cancel()
	if j.rateLimiter != nil {
		j.rateLimiter.Stop()
	}
	j.EndTime = time.Now()
	j.Duration = int(j.EndTime.Sub(j.StartTime).Seconds())
	j.Status = StatusCancelled
	j.mu.Unlock()
	j.Log("Job cancelled by user.")
}

func (j *Job) Log(msg string) {
	j.mu.Lock()
	defer j.mu.Unlock()

	timestamp := time.Now().Format("15:04:05")
	formatted := fmt.Sprintf("[%s] %s", timestamp, msg)
	j.Logs = append(j.Logs, formatted)
	fmt.Println(formatted)
	j.saveLocked()
}

func (j *Job) Save() {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.saveLocked()
}

func (j *Job) saveLocked() {
	_ = index.EnsureDataDirs()
	filename := filepath.Join(index.JobsDir, fmt.Sprintf("%s.json", j.ID))
	data, _ := json.MarshalIndent(j, "", "  ")
	_ = os.WriteFile(filename, data, 0644)
}

func (j *Job) Start() {
	j.Log(fmt.Sprintf("Starting job %s for %s (depth=%d workers=%d hitRate=%d/sec queueCap=%d maxURLs=%d)", j.ID, j.OriginURL, j.Depth, j.MaxWorkers, j.HitRate, j.QueueCapacity, j.MaxURLs))

	if !j.scheduleURL(j.OriginURL) {
		j.mu.Lock()
		j.Status = StatusFinished
		j.EndTime = time.Now()
		j.Duration = int(j.EndTime.Sub(j.StartTime).Seconds())
		j.mu.Unlock()
		if j.rateLimiter != nil {
			j.rateLimiter.Stop()
		}
		j.Log("Origin URL already indexed previously. No new pages scheduled.")
		j.Save()
		return
	}

	j.Save()
	j.wg.Add(1)

	go func() {
		j.crawl(j.OriginURL, 0)
		j.wg.Wait()
		j.finish()
	}()
}

func (j *Job) finish() {
	if j.rateLimiter != nil {
		j.rateLimiter.Stop()
	}

	j.mu.Lock()
	finalStatus := j.Status
	if finalStatus == StatusRunning {
		finalStatus = StatusFinished
		j.Status = StatusFinished
	}
	if j.EndTime.IsZero() {
		j.EndTime = time.Now()
		j.Duration = int(j.EndTime.Sub(j.StartTime).Seconds())
	}
	j.mu.Unlock()

	if finalStatus == StatusFinished {
		j.Log("Job completed successfully.")
	} else {
		j.Save()
	}
}

func (j *Job) crawl(targetURL string, currentDepth int) {
	defer j.wg.Done()
	j.dequeue(targetURL)

	select {
	case <-j.ctx.Done():
		return
	default:
	}

	if currentDepth > j.Depth {
		return
	}

	if j.MaxURLs > 0 {
		j.mu.Lock()
		limitReached := j.CrawledCount >= j.MaxURLs
		j.mu.Unlock()
		if limitReached {
			return
		}
	}

	select {
	case j.semaphore <- struct{}{}:
		j.setWorkerDelta(1)
		defer func() {
			<-j.semaphore
			j.setWorkerDelta(-1)
		}()
	case <-j.ctx.Done():
		return
	}

	if j.rateLimiter != nil {
		select {
		case <-j.rateLimiter.C:
		case <-j.ctx.Done():
			return
		}
	}

	j.Log(fmt.Sprintf("Crawling %s at depth %d", targetURL, currentDepth))

	body, err := j.fetch(targetURL)

	j.mu.Lock()
	j.CrawledCount++
	if err != nil {
		j.ErrorCount++
		j.mu.Unlock()
		j.Log(fmt.Sprintf("Error fetching %s: %v", targetURL, err))
		return
	}
	j.mu.Unlock()

	j.indexContent(targetURL, body, currentDepth)

	if currentDepth >= j.Depth {
		return
	}

	for _, nextURL := range j.extractLinks(targetURL, body) {
		if j.ctx.Err() != nil {
			return
		}

		if !j.scheduleURL(nextURL) {
			continue
		}

		j.wg.Add(1)
		go j.crawl(nextURL, currentDepth+1)
	}
}

func (j *Job) fetch(targetURL string) (string, error) {
	req, err := http.NewRequestWithContext(j.ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", defaultUserAgent)

	resp, err := j.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return "", err
	}

	return string(bodyBytes), nil
}

func (j *Job) scheduleURL(targetURL string) bool {
	j.mu.Lock()
	defer j.mu.Unlock()

	if j.visited[targetURL] {
		return false
	}

	if j.MaxURLs > 0 && j.TotalFound >= j.MaxURLs {
		j.BackPressureActive = true
		j.BackPressureReason = fmt.Sprintf("max URL limit reached (%d)", j.MaxURLs)
		return false
	}

	if j.QueueCapacity > 0 && len(j.Queue) >= j.QueueCapacity {
		j.BackPressureActive = true
		j.BackPressureReason = fmt.Sprintf("queue at capacity (%d)", j.QueueCapacity)
		return false
	}

	j.visited[targetURL] = true
	j.Queue = append(j.Queue, targetURL)
	j.TotalFound++
	j.releaseQueuePressureLocked()
	j.appendVisitedLocked(targetURL)
	j.saveLocked()
	return true
}

func (j *Job) appendVisitedLocked(targetURL string) {
	_ = index.EnsureDataDirs()

	file, err := os.OpenFile(j.visitedFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer file.Close()

	_, _ = file.WriteString(targetURL + "\n")
}

func (j *Job) dequeue(targetURL string) {
	j.mu.Lock()
	defer j.mu.Unlock()

	for i, queuedURL := range j.Queue {
		if queuedURL != targetURL {
			continue
		}
		j.Queue = append(j.Queue[:i], j.Queue[i+1:]...)
		break
	}

	j.releaseQueuePressureLocked()
	j.saveLocked()
}

func (j *Job) releaseQueuePressureLocked() {
	if j.QueueCapacity == 0 {
		j.BackPressureActive = false
		if strings.HasPrefix(j.BackPressureReason, "queue at capacity") {
			j.BackPressureReason = ""
		}
		return
	}

	if len(j.Queue) < j.QueueCapacity && strings.HasPrefix(j.BackPressureReason, "queue at capacity") {
		j.BackPressureActive = false
		j.BackPressureReason = ""
	}
}

func (j *Job) setWorkerDelta(delta int) {
	j.mu.Lock()
	j.ActiveWorkers += delta
	j.saveLocked()
	j.mu.Unlock()
}

func (j *Job) extractLinks(baseURL, body string) []string {
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil
	}

	matches := linkPattern.FindAllStringSubmatch(body, -1)
	seen := make(map[string]bool, len(matches))
	links := make([]string, 0, len(matches))

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		rawLink := strings.TrimSpace(match[1])
		if rawLink == "" {
			continue
		}

		ref, err := url.Parse(rawLink)
		if err != nil {
			continue
		}

		resolved := base.ResolveReference(ref)
		resolved.Fragment = ""
		if resolved.Scheme != "http" && resolved.Scheme != "https" {
			continue
		}

		normalized := resolved.String()
		if seen[normalized] {
			continue
		}
		seen[normalized] = true
		links = append(links, normalized)
	}

	return links
}

func (j *Job) indexContent(targetURL, body string, depth int) {
	cleaned := scriptPattern.ReplaceAllString(body, " ")
	cleaned = stylePattern.ReplaceAllString(cleaned, " ")
	cleaned = htmlTagPattern.ReplaceAllString(cleaned, " ")
	cleaned = html.UnescapeString(cleaned)

	words := bodyTokenPattern.FindAllString(strings.ToLower(cleaned), -1)
	counts := make(map[string]int)
	for _, word := range words {
		if len(word) < 2 {
			continue
		}
		counts[word]++
	}

	batch := make(map[string]index.WordResult, len(counts))
	for word, count := range counts {
		batch[word] = index.WordResult{
			URL:       targetURL,
			OriginURL: j.OriginURL,
			Depth:     depth,
			Count:     count,
		}
	}

	if err := j.FileIndex.AddBatch(batch); err != nil {
		j.Log(fmt.Sprintf("Error persisting index batch for %s: %v", targetURL, err))
	}
}
