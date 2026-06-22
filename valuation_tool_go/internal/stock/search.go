package stock

import (
	"strings"
	"sync"
	"time"

	"valuation/internal/cache"
)

type SearchIndex struct {
	Cache  *cache.Cache
	mu     sync.RWMutex
	list   []CodeName
	loaded time.Time
}

const indexTTL = 24 * time.Hour

func (s *SearchIndex) load() {
	s.mu.RLock()
	if s.list != nil && time.Since(s.loaded) < indexTTL {
		s.mu.RUnlock()
		return
	}
	s.mu.RUnlock()

	// 尝试缓存
	var cached []CodeName
	if s.Cache != nil && s.Cache.Get("name_index", &cached) && len(cached) > 100 {
		s.mu.Lock()
		s.list = cached
		s.loaded = time.Now()
		s.mu.Unlock()
		return
	}
	// 拉新
	list, err := FetchAllACodeName()
	if err != nil || len(list) == 0 {
		return
	}
	if s.Cache != nil {
		s.Cache.Set("name_index", list)
	}
	s.mu.Lock()
	s.list = list
	s.loaded = time.Now()
	s.mu.Unlock()
}

func (s *SearchIndex) Search(q string, limit int) []CodeName {
	s.load()
	s.mu.RLock()
	defer s.mu.RUnlock()
	if q == "" || s.list == nil {
		return nil
	}
	q = strings.TrimSpace(q)
	out := make([]CodeName, 0, limit)
	if isAllDigit(q) {
		for _, x := range s.list {
			if strings.HasPrefix(x.Code, q) {
				out = append(out, x)
				if len(out) >= limit {
					break
				}
			}
		}
	} else {
		qNorm := strings.ReplaceAll(q, " ", "")
		for _, x := range s.list {
			if strings.Contains(strings.ReplaceAll(x.Name, " ", ""), qNorm) {
				out = append(out, x)
				if len(out) >= limit {
					break
				}
			}
		}
	}
	return out
}

func (s *SearchIndex) Resolve(q string) string {
	q = strings.TrimSpace(q)
	if isAllDigit(q) && len(q) == 6 {
		return q
	}
	hits := s.Search(q, 50)
	if len(hits) == 1 {
		return hits[0].Code
	}
	// 精确名称匹配
	qNorm := strings.ReplaceAll(q, " ", "")
	for _, h := range hits {
		if strings.ReplaceAll(h.Name, " ", "") == qNorm {
			return h.Code
		}
	}
	return ""
}

func isAllDigit(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
