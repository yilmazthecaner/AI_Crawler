package searcher

import (
	"sort"
	"spidersearch/internal/index"
	"strings"
)

type Searcher struct {
	Index *index.Index
}

func NewSearcher(idx *index.Index) *Searcher {
	return &Searcher{
		Index: idx,
	}
}

// Search returns a list of result triples for a query.
func (s *Searcher) Search(query string) []index.ResultTriple {
	keywords := strings.Fields(strings.ToLower(query))
	if len(keywords) == 0 {
		return nil
	}

	// Find results for each keyword and aggregate
	resultMap := make(map[string]index.ResultTriple)

	for _, k := range keywords {
		results := s.Index.Get(k)
		for _, r := range results {
			if existing, ok := resultMap[r.URL]; ok {
				existing.Relevance += r.Relevance
				resultMap[r.URL] = existing
			} else {
				resultMap[r.URL] = r
			}
		}
	}

	// Convert map to slice and sort by relevance
	var finalResults []index.ResultTriple
	for _, res := range resultMap {
		finalResults = append(finalResults, res)
	}

	sort.Slice(finalResults, func(i, j int) bool {
		return finalResults[i].Relevance > finalResults[j].Relevance
	})

	return finalResults
}
