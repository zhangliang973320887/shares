package benchmark

import "valuation/internal/cache"

type Registry struct {
	sources []Source
}

func NewRegistry(c *cache.Cache, damodaranPath string) *Registry {
	return &Registry{
		sources: []Source{
			BuiltinSource{},
			CsindexSource{Cache: c},
			&DamodaranSource{DataPath: damodaranPath},
		},
	}
}

func (r *Registry) List() []SourceInfo {
	out := make([]SourceInfo, 0, len(r.sources))
	for _, s := range r.sources {
		out = append(out, SourceInfo{
			Key: s.Key(), Label: s.Label(), Scope: s.Scope(), Note: s.Note(),
		})
	}
	return out
}

func (r *Registry) Find(key string) Source {
	for _, s := range r.sources {
		if s.Key() == key {
			return s
		}
	}
	return nil
}

func (r *Registry) Industries(key string) []string {
	if s := r.Find(key); s != nil {
		return s.Industries()
	}
	return nil
}

func (r *Registry) Get(key, industry string) (*Benchmark, error) {
	if s := r.Find(key); s != nil {
		return s.Get(industry)
	}
	return nil, errSource(key)
}

type errSourceType string

func (e errSourceType) Error() string { return string(e) }
func errSource(key string) error      { return errSourceType("未知数据源: " + key) }
