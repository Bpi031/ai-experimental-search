package ai

import "io"

// SearchResultIter provides a simple Next/ForEach API similar to go-git iterators.
type SearchResultIter interface {
    Next() (*SearchResult, error)
    ForEach(func(*SearchResult) error) error
    Close()
}

// sliceIter is a simple in-memory iterator over search results.
type sliceIter struct {
    items []SearchResult
    idx   int
    closed bool
}

func NewSliceIter(items []SearchResult) SearchResultIter {
    return &sliceIter{items: items}
}

func (it *sliceIter) Next() (*SearchResult, error) {
    if it.closed { return nil, io.EOF }
    if it.idx >= len(it.items) { return nil, io.EOF }
    v := it.items[it.idx]
    it.idx++
    return &v, nil
}

func (it *sliceIter) ForEach(f func(*SearchResult) error) error {
    for {
        v, err := it.Next()
        if err == io.EOF { return nil }
        if err != nil { return err }
        if err := f(v); err != nil { return err }
    }
}

func (it *sliceIter) Close() { it.closed = true }
