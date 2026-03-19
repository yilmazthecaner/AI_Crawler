package crawler

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"spidersearch/internal/index"
	"sync"
	"os"
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

func (jm *JobManager) CreateJob(origin string, depth, workers int) *Job {
	job := NewJob(origin, depth, workers, jm.FileIndex)
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

	var list []*Job
	for _, j := range jm.Jobs {
		list = append(list, j)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].StartTime.After(list[j].StartTime)
	})
	return list
}

func (jm *JobManager) ClearJobs() {
	jm.mu.Lock()
	defer jm.mu.Unlock()

	for id := range jm.Jobs {
		os.Remove(fmt.Sprintf("%s.data", id))
	}
	jm.Jobs = make(map[string]*Job)
	os.Remove("visited_urls.data")
}

func (jm *JobManager) LoadPreviousJobs(dir string) {
	files, _ := filepath.Glob("*.data")
	for _, f := range files {
		if f == "visited_urls.data" {
			continue
		}
		
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}

		var job Job
		if err := json.Unmarshal(data, &job); err == nil {
			job.FileIndex = jm.FileIndex
			jm.mu.Lock()
			jm.Jobs[job.ID] = &job
			jm.mu.Unlock()
		}
	}
}
