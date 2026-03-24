package searcher

import "spidersearch/internal/index"

type Searcher struct {
	Index *index.FileIndex
}

func NewSearcher(idx *index.FileIndex) *Searcher {
	return &Searcher{
		Index: idx,
	}
}

func (s *Searcher) Search(query string) []index.SearchResult {
	if s == nil || s.Index == nil {
		return nil
	}
	return s.Index.Search(query, "relevance")
}
