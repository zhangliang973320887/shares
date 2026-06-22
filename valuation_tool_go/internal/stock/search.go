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
	q = strings.TrimSpace(q)
	if q == "" {
		return nil
	}

	// 大写字母 = 美股代码直接命中
	if IsUSTicker(q) {
		return []CodeName{{Code: q, Name: q + " (美股)"}}
	}

	s.load()
	s.mu.RLock()
	list := s.list
	s.mu.RUnlock()

	out := make([]CodeName, 0, limit)
	if isAllDigit(q) {
		for _, x := range list {
			if strings.HasPrefix(x.Code, q) {
				out = append(out, x)
				if len(out) >= limit {
					break
				}
			}
		}
		return out
	}

	// 中文名: A股索引模糊匹配
	qNorm := strings.ReplaceAll(q, " ", "")
	hasASCII := isMostlyASCII(qNorm)
	if !hasASCII {
		for _, x := range list {
			if strings.Contains(strings.ReplaceAll(x.Name, " ", ""), qNorm) {
				out = append(out, x)
				if len(out) >= limit {
					break
				}
			}
		}
		return out
	}

	// 英文名: 先A股(沪深B股有英文名罕见), 再Yahoo美股
	for _, x := range list {
		if strings.Contains(strings.ToLower(x.Name), strings.ToLower(q)) {
			out = append(out, x)
			if len(out) >= limit {
				break
			}
		}
	}
	if len(out) < limit {
		usHits := SearchUS(q, limit-len(out))
		out = append(out, usHits...)
	}
	return out
}

func (s *SearchIndex) Resolve(q string) string {
	q = strings.TrimSpace(q)
	if isAllDigit(q) && len(q) == 6 {
		return q
	}
	if IsUSTicker(strings.ToUpper(q)) {
		return strings.ToUpper(q)
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

func isMostlyASCII(s string) bool {
	if s == "" {
		return true
	}
	ascii := 0
	for _, r := range s {
		if r < 128 {
			ascii++
		}
	}
	return ascii*2 > len([]rune(s))
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
