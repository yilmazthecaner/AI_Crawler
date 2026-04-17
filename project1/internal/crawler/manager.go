package crawler

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"spidersearch/internal/index"
	"sync"
	"time"
)

type JobManager struct {
	Jobs      map[string]*Job
	FileIndex *index.FileIndex
	mu        sync.RWMutex
}

func NewJobManager(fi *index.FileIndex) *JobManager {
	return &JobManager{
		Jobs:      make(map[string]*Job),
		FileIndex: fi,
	}
}

func (jm *JobManager) CreateJob(origin string, depth, workers, hitRate, queueCap, maxURLs int) *Job {
	job := NewJob(origin, depth, workers, hitRate, queueCap, maxURLs, jm.FileIndex)
	jm.mu.Lock()
	jm.Jobs[job.ID] = job
	jm.mu.Unlock()
	return job
}

func (jm *JobManager) GetJob(id string) *Job {
	jm.mu.RLock()
	defer jm.mu.RUnlock()
	return jm.Jobs[id]
}

func (jm *JobManager) ListJobs() []*Job {
	jm.mu.RLock()
	defer jm.mu.RUnlock()

	list := make([]*Job, 0, len(jm.Jobs))
	for _, job := range jm.Jobs {
		list = append(list, job)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].StartTime.After(list[j].StartTime)
	})
	return list
}

func (jm *JobManager) ClearJobs() {
	jm.mu.Lock()
	jobs := make([]*Job, 0, len(jm.Jobs))
	for _, job := range jm.Jobs {
		jobs = append(jobs, job)
	}
	jm.Jobs = make(map[string]*Job)
	jm.mu.Unlock()

	for _, job := range jobs {
		job.Cancel()
	}
	_ = os.RemoveAll(index.JobsDir)
	_ = os.Remove(index.VisitedURLsFile)
	_ = jm.FileIndex.Reset()
	_ = index.EnsureDataDirs()
}

func (jm *JobManager) LoadPreviousJobs(dir string) {
	if dir == "" {
		dir = index.JobsDir
	}

	_ = index.EnsureDataDirs()

	files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	for _, filePath := range files {
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		var job Job
		if err := json.Unmarshal(data, &job); err != nil {
			continue
		}

		job.FileIndex = jm.FileIndex

		if job.Status == StatusRunning {
			job.Status = StatusCancelled
			if job.EndTime.IsZero() {
				job.EndTime = time.Now()
			}
			job.Duration = int(job.EndTime.Sub(job.StartTime).Seconds())
			job.Logs = append(job.Logs, fmt.Sprintf("[%s] Job recovered after restart and marked as cancelled.", time.Now().Format("15:04:05")))
		}

		jm.mu.Lock()
		jm.Jobs[job.ID] = &job
		jm.mu.Unlock()
	}
}
