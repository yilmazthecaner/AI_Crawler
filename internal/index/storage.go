package index

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type WordResult struct {
	URL       string  `json:"url"`
	OriginURL string  `json:"origin_url"`
	Depth     int     `json:"depth"`
	Count     int     `json:"count"`
	Relevance float64 `json:"relevance"`
}

type FileIndex struct {
	Data map[string]map[string][]WordResult // letter -> word -> results
	mu   sync.RWMutex
}

func NewFileIndex() *FileIndex {
	fi := &FileIndex{
		Data: make(map[string]map[string][]WordResult),
	}
	fi.loadAll()
	return fi
}

func (fi *FileIndex) loadAll() {
	os.MkdirAll("storage", 0755)
	files, _ := filepath.Glob(filepath.Join("storage", "*.data"))
	for _, f := range files {
		letter := strings.TrimSuffix(filepath.Base(f), ".data")
		fileBytes, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var letterData map[string][]WordResult
		if err := json.Unmarshal(fileBytes, &letterData); err == nil {
			fi.mu.Lock()
			fi.Data[letter] = letterData
			fi.mu.Unlock()
		}
	}
}

func isStandardLetter(s string) bool {
    if len(s) == 0 { return false }
    r := []rune(s)[0]
    return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
}

// AddBatch stores multiple word results for a page efficiently
func (fi *FileIndex) AddBatch(results map[string]WordResult) error {
	fi.mu.Lock()
	defer fi.mu.Unlock()

	byLetter := make(map[string]map[string]WordResult)
	for word, res := range results {
		runes := []rune(strings.ToLower(word))
		if len(runes) == 0 {
			continue
		}
		letter := string(runes[0])
		if !isStandardLetter(letter) {
			letter = "other"
		}
		if _, ok := byLetter[letter]; !ok {
			byLetter[letter] = make(map[string]WordResult)
		}
		byLetter[letter][word] = res
	}

	for letter, words := range byLetter {
		if fi.Data[letter] == nil {
			fi.Data[letter] = make(map[string][]WordResult)
		}
		
		letterData := fi.Data[letter]
		for word, result := range words {
			found := false
			for i, existing := range letterData[word] {
				if existing.URL == result.URL {
					letterData[word][i].Count += result.Count
					letterData[word][i].Relevance += result.Relevance
					found = true
					break
				}
			}
			if !found {
				letterData[word] = append(letterData[word], result)
			}
		}

		// Persistent save
		filename := filepath.Join("storage", fmt.Sprintf("%s.data", letter))
		newData, _ := json.MarshalIndent(letterData, "", "  ")
		os.WriteFile(filename, newData, 0644)
	}
	return nil
}

// Search queries the in-memory index for keywords
func (fi *FileIndex) Search(query string) []WordResult {
	words := strings.Fields(strings.ToLower(query))
	if len(words) == 0 {
		return nil
	}

	fi.mu.RLock()
	defer fi.mu.RUnlock()

	resultMap := make(map[string]WordResult)

	for _, word := range words {
		runes := []rune(word)
		if len(runes) == 0 { continue }
		letter := string(runes[0])
		if !isStandardLetter(letter) {
			letter = "other"
		}
		
		letterData, ok := fi.Data[letter]
		if !ok {
			continue
		}

		for _, r := range letterData[word] {
			if existing, ok := resultMap[r.URL]; ok {
				existing.Relevance += r.Relevance
				resultMap[r.URL] = existing
			} else {
				resultMap[r.URL] = r
			}
		}
	}

	var results []WordResult
	for _, res := range resultMap {
		results = append(results, res)
	}
	return results
}
