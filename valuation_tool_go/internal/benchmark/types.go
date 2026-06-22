package benchmark

// Benchmark 行业基准
type Benchmark struct {
	PELow    float64
	PEHigh   float64
	PBLow    float64
	PBHigh   float64
	ROEMin   float64
	DivMin   float64
	PEGMax   float64
	Meta     Meta
}

type Meta struct {
	Source  string  `json:"source"`
	Asof    string  `json:"asof"`
	Note    string  `json:"note,omitempty"`
	LivePE  float64 `json:"live_pe,omitempty"`
	LivePB  float64 `json:"live_pb,omitempty"`
	LiveDiv float64 `json:"live_div,omitempty"`
}

// Source 数据源接口
type Source interface {
	Key() string
	Label() string
	Scope() string
	Note() string
	Industries() []string
	Get(industry string) (*Benchmark, error)
}

// SourceInfo 给前端用
type SourceInfo struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Scope string `json:"scope"`
	Note  string `json:"note"`
}
