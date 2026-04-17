package index

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const (
	DataDir         = "data"
	StorageDir      = "data/storage"
	JobsDir         = "data/jobs"
	VisitedURLsFile = "data/visited_urls.data"
)

var tokenPattern = regexp.MustCompile(`[a-z0-9]+`)

type WordResult struct {
	URL       string
	OriginURL string
	Depth     int
	Count     int
}

type StorageEntry struct {
	Word      string `json:"word"`
	URL       string `json:"url"`
	OriginURL string `json:"origin_url"`
	Depth     int    `json:"depth"`
	Frequency int    `json:"frequency"`
}

type SearchResult struct {
	URL            string   `json:"url"`
	RelevantURL    string   `json:"relevant_url"`
	OriginURL      string   `json:"origin_url"`
	Depth          int      `json:"depth"`
	Frequency      int      `json:"frequency"`
	RelevanceScore int      `json:"relevance_score"`
	MatchedTerms   []string `json:"matched_terms,omitempty"`
}

type FileIndex struct {
	Data map[string]map[string][]WordResult
	mu   sync.RWMutex
}

func NewFileIndex() *FileIndex {
	_ = EnsureDataDirs()

	fi := &FileIndex{
		Data: make(map[string]map[string][]WordResult),
	}
	fi.loadAll()
	return fi
}

func EnsureDataDirs() error {
	for _, dir := range []string{DataDir, StorageDir, JobsDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}

func StoragePathForLetter(letter string) string {
	return filepath.Join(StorageDir, fmt.Sprintf("%s.data", strings.ToLower(letter)))
}

func QueryTerms(text string) []string {
	matches := tokenPattern.FindAllString(strings.ToLower(text), -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]bool, len(matches))
	terms := make([]string, 0, len(matches))
	for _, match := range matches {
		if seen[match] {
			continue
		}
		seen[match] = true
		terms = append(terms, match)
	}
	return terms
}

func Score(frequency, depth int) int {
	return (frequency * 10) + 1000 - (depth * 5)
}

func (fi *FileIndex) Reset() error {
	fi.mu.Lock()
	fi.Data = make(map[string]map[string][]WordResult)
	fi.mu.Unlock()

	if err := os.RemoveAll(StorageDir); err != nil {
		return err
	}
	return EnsureDataDirs()
}

func (fi *FileIndex) loadAll() {
	files, _ := filepath.Glob(filepath.Join(StorageDir, "*.data"))

	fi.mu.Lock()
	defer fi.mu.Unlock()

	for _, filePath := range files {
		file, err := os.Open(filePath)
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			entry, err := parseStorageLine(scanner.Text())
			if err != nil {
				continue
			}
			fi.insertLocked(entry.Word, WordResult{
				URL:       entry.URL,
				OriginURL: entry.OriginURL,
				Depth:     entry.Depth,
				Count:     entry.Frequency,
			})
		}
		file.Close()
	}
}

func parseStorageLine(line string) (StorageEntry, error) {
	fields := strings.Fields(line)
	if len(fields) != 5 {
		return StorageEntry{}, fmt.Errorf("invalid storage line: %q", line)
	}

	depth, err := strconv.Atoi(fields[3])
	if err != nil {
		return StorageEntry{}, err
	}

	frequency, err := strconv.Atoi(fields[4])
	if err != nil {
		return StorageEntry{}, err
	}

	return StorageEntry{
		Word:      fields[0],
		URL:       fields[1],
		OriginURL: fields[2],
		Depth:     depth,
		Frequency: frequency,
	}, nil
}

func storageLetter(word string) string {
	if word == "" {
		return "other"
	}
	letter := string([]rune(strings.ToLower(word))[0])
	if !isStandardLetter(letter) {
		return "other"
	}
	return letter
}

func isStandardLetter(s string) bool {
	if len(s) == 0 {
		return false
	}
	r := []rune(s)[0]
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
}

func (fi *FileIndex) insertLocked(word string, result WordResult) {
	letter := storageLetter(word)
	if fi.Data[letter] == nil {
		fi.Data[letter] = make(map[string][]WordResult)
	}

	existing := fi.Data[letter][word]
	for i, current := range existing {
		if current.URL == result.URL && current.OriginURL == result.OriginURL && current.Depth == result.Depth {
			existing[i].Count += result.Count
			fi.Data[letter][word] = existing
			return
		}
	}

	fi.Data[letter][word] = append(fi.Data[letter][word], result)
}

func (fi *FileIndex) AddBatch(results map[string]WordResult) error {
	if err := EnsureDataDirs(); err != nil {
		return err
	}

	fi.mu.Lock()
	defer fi.mu.Unlock()

	byLetter := make(map[string][]StorageEntry)

	for word, result := range results {
		normalizedTerms := QueryTerms(word)
		if len(normalizedTerms) == 0 {
			continue
		}

		normalized := normalizedTerms[0]
		fi.insertLocked(normalized, result)

		letter := storageLetter(normalized)
		byLetter[letter] = append(byLetter[letter], StorageEntry{
			Word:      normalized,
			URL:       result.URL,
			OriginURL: result.OriginURL,
			Depth:     result.Depth,
			Frequency: result.Count,
		})
	}

	for letter, entries := range byLetter {
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].Word != entries[j].Word {
				return entries[i].Word < entries[j].Word
			}
			if entries[i].URL != entries[j].URL {
				return entries[i].URL < entries[j].URL
			}
			if entries[i].OriginURL != entries[j].OriginURL {
				return entries[i].OriginURL < entries[j].OriginURL
			}
			return entries[i].Depth < entries[j].Depth
		})

		file, err := os.OpenFile(StoragePathForLetter(letter), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			if _, err := fmt.Fprintf(file, "%s %s %s %d %d\n", entry.Word, entry.URL, entry.OriginURL, entry.Depth, entry.Frequency); err != nil {
				file.Close()
				return err
			}
		}

		if err := file.Close(); err != nil {
			return err
		}
	}

	return nil
}

func (fi *FileIndex) EntriesForWord(word string) []StorageEntry {
	terms := QueryTerms(word)
	if len(terms) == 0 {
		return nil
	}

	term := terms[0]
	letter := storageLetter(term)

	fi.mu.RLock()
	results := append([]WordResult(nil), fi.Data[letter][term]...)
	fi.mu.RUnlock()

	entries := make([]StorageEntry, 0, len(results))
	for _, result := range results {
		entries = append(entries, StorageEntry{
			Word:      term,
			URL:       result.URL,
			OriginURL: result.OriginURL,
			Depth:     result.Depth,
			Frequency: result.Count,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		leftScore := Score(entries[i].Frequency, entries[i].Depth)
		rightScore := Score(entries[j].Frequency, entries[j].Depth)
		if leftScore != rightScore {
			return leftScore > rightScore
		}
		return entries[i].URL < entries[j].URL
	})

	return entries
}

func (fi *FileIndex) StorageEntriesForLetter(letter string) ([]StorageEntry, error) {
	file, err := os.Open(StoragePathForLetter(letter))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []StorageEntry
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		entry, err := parseStorageLine(scanner.Text())
		if err != nil {
			continue
		}
		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Word != entries[j].Word {
			return entries[i].Word < entries[j].Word
		}
		if entries[i].URL != entries[j].URL {
			return entries[i].URL < entries[j].URL
		}
		return entries[i].Depth < entries[j].Depth
	})

	return entries, nil
}

func (fi *FileIndex) Search(query, sortBy string) []SearchResult {
	terms := QueryTerms(query)
	if len(terms) == 0 {
		return nil
	}

	fi.mu.RLock()
	defer fi.mu.RUnlock()

	resultsByKey := make(map[string]SearchResult)

	for _, term := range terms {
		letter := storageLetter(term)
		letterData, ok := fi.Data[letter]
		if !ok {
			continue
		}

		for _, result := range letterData[term] {
			key := fmt.Sprintf("%s|%s|%d", result.URL, result.OriginURL, result.Depth)
			score := Score(result.Count, result.Depth)

			if existing, ok := resultsByKey[key]; ok {
				existing.Frequency += result.Count
				existing.RelevanceScore += score
				existing.MatchedTerms = appendUnique(existing.MatchedTerms, term)
				resultsByKey[key] = existing
				continue
			}

			resultsByKey[key] = SearchResult{
				URL:            result.URL,
				RelevantURL:    result.URL,
				OriginURL:      result.OriginURL,
				Depth:          result.Depth,
				Frequency:      result.Count,
				RelevanceScore: score,
				MatchedTerms:   []string{term},
			}
		}
	}

	finalResults := make([]SearchResult, 0, len(resultsByKey))
	for _, result := range resultsByKey {
		finalResults = append(finalResults, result)
	}

	sortSearchResults(finalResults, sortBy)
	return finalResults
}

func sortSearchResults(results []SearchResult, sortBy string) {
	sort.Slice(results, func(i, j int) bool {
		switch strings.ToLower(sortBy) {
		case "depth":
			if results[i].Depth != results[j].Depth {
				return results[i].Depth < results[j].Depth
			}
		case "frequency":
			if results[i].Frequency != results[j].Frequency {
				return results[i].Frequency > results[j].Frequency
			}
		default:
			if results[i].RelevanceScore != results[j].RelevanceScore {
				return results[i].RelevanceScore > results[j].RelevanceScore
			}
		}

		if results[i].Frequency != results[j].Frequency {
			return results[i].Frequency > results[j].Frequency
		}
		if results[i].Depth != results[j].Depth {
			return results[i].Depth < results[j].Depth
		}
		return results[i].URL < results[j].URL
	})
}

func appendUnique(values []string, candidate string) []string {
	for _, value := range values {
		if value == candidate {
			return values
		}
	}
	return append(values, candidate)
}
