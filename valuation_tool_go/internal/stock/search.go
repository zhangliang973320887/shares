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

	// 大写字母全 ASCII = 美股代码直接命中
	if IsUSTicker(q) {
		return []CodeName{{Code: q, Name: q + " (美股)"}}
	}

	// 优先用东财统一搜索 (支持 中/英 + A股/美股/港股)
	hits := SearchAllMarkets(q, limit)
	if len(hits) > 0 {
		out := make([]CodeName, 0, len(hits))
		for _, h := range hits {
			out = append(out, CodeName{
				Code: h.Code,
				Name: h.Name + " (" + h.Market + ")",
			})
		}
		return out
	}

	// 兜底: 本地 A股 索引模糊匹配
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
	qNorm := strings.ReplaceAll(q, " ", "")
	for _, x := range list {
		nameNorm := strings.ReplaceAll(x.Name, " ", "")
		if strings.Contains(nameNorm, qNorm) ||
			strings.Contains(strings.ToLower(nameNorm), strings.ToLower(qNorm)) {
			out = append(out, x)
			if len(out) >= limit {
				break
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
	if IsUSTicker(strings.ToUpper(q)) {
		return strings.ToUpper(q)
	}
	hits := s.Search(q, 50)
	if len(hits) == 1 {
		return hits[0].Code
	}
	if len(hits) == 0 {
		return ""
	}
	// 精确名称匹配 (去括号注释/空格)
	qNorm := strings.ReplaceAll(q, " ", "")
	for _, h := range hits {
		clean := cleanName(h.Name)
		if clean == qNorm {
			return h.Code
		}
	}
	// 首个结果若主体名匹配 (即 cleanName 完全等于 q), 直接用
	if cleanName(hits[0].Name) == qNorm {
		return hits[0].Code
	}
	return ""
}

// cleanName 去掉名称里的 "(A股)" "(美股)" 等标注 + 空格
func cleanName(s string) string {
	if i := strings.Index(s, " ("); i > 0 {
		s = s[:i]
	}
	return strings.ReplaceAll(s, " ", "")
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
