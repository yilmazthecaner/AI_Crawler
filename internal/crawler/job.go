package crawler

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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

type Job struct {
	ID           string               `json:"id"`
	OriginURL    string               `json:"origin_url"`
	Depth        int                  `json:"depth"`
	Status       JobStatus            `json:"status"`
	Logs         []string             `json:"logs"`
	Queue        []string             `json:"queue"`
	StartTime    time.Time            `json:"start_time"`
	EndTime      time.Time            `json:"end_time"`
	Duration     int                  `json:"duration"`
	MaxWorkers    int                  `json:"max_workers"`
	HitRate       int                  `json:"hit_rate"`       // Hits per second
	QueueCapacity int                  `json:"queue_capacity"` // Max length of internal slice queue
	MaxURLs       int                  `json:"max_urls"`       // Max URLs to visit total
	CrawledCount  int                  `json:"crawled_count"`
	ErrorCount    int                  `json:"error_count"`
	TotalFound    int                  `json:"total_found"`
	FileIndex     *index.FileIndex     `json:"-"`
	mu            sync.Mutex
	wg            sync.WaitGroup
	visitedFile   string
	visited       map[string]bool
	semaphore     chan struct{}
	rateLimiter   *time.Ticker
	ctx           context.Context
	cancel        context.CancelFunc
}

func NewJob(origin string, depth, workers, hitRate, queueCap, maxURLs int, fi *index.FileIndex) *Job {
	// ID Format: [Epoch]_[ID]
	id := fmt.Sprintf("%d_%d", time.Now().Unix(), time.Now().UnixNano()%1000)
	if workers <= 0 {
		workers = 10
	}
	if hitRate <= 0 {
		hitRate = 10 // Default 10 req/s
	}
	ctx, cancel := context.WithCancel(context.Background())
	
	var ticker *time.Ticker
	if hitRate > 0 {
		ticker = time.NewTicker(time.Second / time.Duration(hitRate))
	}

	return &Job{
		ID:            id,
		OriginURL:     origin,
		Depth:         depth,
		MaxWorkers:    workers,
		HitRate:       hitRate,
		QueueCapacity: queueCap,
		MaxURLs:       maxURLs,
		Status:        StatusRunning,
		StartTime:     time.Now(),
		FileIndex:     fi,
		CrawledCount:  0,
		ErrorCount:    0,
		TotalFound:    1, // The origin URL
		visitedFile:   "visited_urls.data",
		visited:       make(map[string]bool),
		semaphore:     make(chan struct{}, workers),
		rateLimiter:   ticker,
		ctx:           ctx,
		cancel:        cancel,
	}
}

func (j *Job) Cancel() {
	j.mu.Lock()
	if j.Status != StatusRunning {
		j.mu.Unlock()
		return
	}
	j.cancel()
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
	filename := fmt.Sprintf("%s.data", j.ID)
	data, _ := json.MarshalIndent(j, "", "  ")
	os.WriteFile(filename, data, 0644)
}

func (j *Job) Start() {
	j.Log(fmt.Sprintf("Starting job %s for %s (Depth: %d, Workers: %d)", j.ID, j.OriginURL, j.Depth, j.MaxWorkers))
	j.Save()
	j.wg.Add(1)
	go func() {
		j.crawl(j.OriginURL, 0)
		j.wg.Wait()
		j.mu.Lock()
		j.EndTime = time.Now()
		j.Duration = int(j.EndTime.Sub(j.StartTime).Seconds())
		j.Status = StatusFinished
		j.mu.Unlock()
		j.Log("Job completed successfully.")
		j.Save()
	}()
}

func (j *Job) crawl(u string, currentDepth int) {
	defer j.wg.Done()

	select {
	case <-j.ctx.Done():
		return
	default:
	}

	if currentDepth > j.Depth {
		return
	}

	// Max URLs check
	j.mu.Lock()
	if j.MaxURLs > 0 && j.CrawledCount >= j.MaxURLs {
		j.mu.Unlock()
		return
	}

	// Thread-safe visited check
	if j.visited[u] {
		j.mu.Unlock()
		return
	}
	j.visited[u] = true
	j.mu.Unlock()

	// Persistence of visited URLs
	j.markVisited(u)

	// Concurrency control: Acquire semaphore (cancellable)
	select {
	case j.semaphore <- struct{}{}:
		defer func() { <-j.semaphore }()
	case <-j.ctx.Done():
		return
	}

	// Rate limiting
	if j.rateLimiter != nil {
		select {
		case <-j.rateLimiter.C:
		case <-j.ctx.Done():
			return
		}
	}

	j.Log(fmt.Sprintf("Crawling %s at depth %d", u, currentDepth))

	// Requirement: Use curl command (with User-Agent) - Use CommandContext for cancellation
	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"
	cmd := exec.CommandContext(j.ctx, "curl", "-s", "-L", "-A", userAgent, u)
	output, err := cmd.CombinedOutput()
	
	// Double check cancellation right after the command
	if j.ctx.Err() != nil {
		return
	}

	j.mu.Lock()
	j.CrawledCount++
	if err != nil {
		j.ErrorCount++
		j.mu.Unlock()
		j.Log(fmt.Sprintf("Error fetching %s: %v", u, err))
		return
	}
	j.mu.Unlock()

	body := string(output)

	// Extract Title and Words
	title := j.extractTitle(body)
	j.indexContent(u, body, title, currentDepth)

	// Extract Links for recursion
	if currentDepth < j.Depth {
		links := j.extractLinks(body)
		for _, link := range links {
			// Check cancellation before starting new goroutines in loop
			select {
			case <-j.ctx.Done():
				return
			default:
			}

			j.mu.Lock()
			// Queue Capacity Check
			if j.QueueCapacity > 0 && (j.TotalFound-j.CrawledCount) >= j.QueueCapacity {
				j.mu.Unlock()
				// j.Log(fmt.Sprintf("Queue full (%d/%d), skipping link: %s", j.TotalFound-j.CrawledCount, j.QueueCapacity, link))
				continue
			}

			if !j.visited[link] {
				j.TotalFound++
				j.wg.Add(1)
				go j.crawl(link, currentDepth+1)
			}
			j.mu.Unlock()
		}
	}
}

func (j *Job) markVisited(u string) {
	// Simple append is generally safe but let's wrap it in a lock since we already have j.mu
	j.mu.Lock()
	defer j.mu.Unlock()
	
	f, err := os.OpenFile(j.visitedFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(u + "\n")
}

func (j *Job) extractTitle(body string) string {
	re := regexp.MustCompile(`(?i)<title>(.*?)</title>`)
	match := re.FindStringSubmatch(body)
	if len(match) > 1 {
		return match[1]
	}
	return ""
}

func (j *Job) extractLinks(body string) []string {
	re := regexp.MustCompile(`(?i)href=["'](http[s]?://[^"']+)["']`)
	matches := re.FindAllStringSubmatch(body, -1)
	var links []string
	for _, m := range matches {
		if len(m) > 1 {
			links = append(links, m[1])
		}
	}
	return links
}

func (j *Job) indexContent(u, body, title string, depth int) {
	words := strings.Fields(strings.ToLower(body))
	counts := make(map[string]int)
	for _, w := range words {
		if len(w) >= 2 {
			counts[w]++
		}
	}

	batch := make(map[string]index.WordResult)
	for word, count := range counts {
		relevance := float64(count)
		if strings.Contains(strings.ToLower(title), word) {
			relevance += 10.0
		}
		batch[word] = index.WordResult{
			URL:       u,
			OriginURL: j.OriginURL,
			Depth:     depth,
			Count:     count,
			Relevance: relevance,
		}
	}
	j.FileIndex.AddBatch(batch)
}
