package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type entry struct {
	Ts    int64           `json:"_ts"`
	Value json.RawMessage `json:"value"`
}

type Cache struct {
	dir       string
	defaultTTL time.Duration
	stockTTL   time.Duration
	mu         sync.Mutex
}

func New(dir string) *Cache {
	_ = os.MkdirAll(dir, 0755)
	return &Cache{
		dir:        dir,
		defaultTTL: 6 * time.Hour,
		stockTTL:   5 * time.Minute,
	}
}

func (c *Cache) ttlFor(key string) time.Duration {
	if strings.HasPrefix(key, "stock_") {
		return c.stockTTL
	}
	return c.defaultTTL
}

func (c *Cache) path(key string) string {
	return filepath.Join(c.dir, key+".json")
}

func (c *Cache) Get(key string, out interface{}) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	b, err := os.ReadFile(c.path(key))
	if err != nil {
		return false
	}
	var e entry
	if err := json.Unmarshal(b, &e); err != nil {
		return false
	}
	if time.Since(time.Unix(e.Ts, 0)) > c.ttlFor(key) {
		return false
	}
	if err := json.Unmarshal(e.Value, out); err != nil {
		return false
	}
	return true
}

func (c *Cache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	raw, err := json.Marshal(value)
	if err != nil {
		return
	}
	e := entry{Ts: time.Now().Unix(), Value: raw}
	b, _ := json.MarshalIndent(e, "", "  ")
	_ = os.WriteFile(c.path(key), b, 0644)
}
