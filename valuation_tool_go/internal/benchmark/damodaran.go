package benchmark

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

type damodaranRow struct {
	PE  float64 `json:"pe"`
	PB  float64 `json:"pb"`
	ROE float64 `json:"roe"`
	Div float64 `json:"div"`
}

type damodaranFile struct {
	Meta struct {
		Source string `json:"source"`
		Asof   string `json:"asof"`
		Note   string `json:"note"`
	} `json:"_meta"`
	Industries map[string]damodaranRow `json:"industries"`
}

type DamodaranSource struct {
	DataPath string
	once     sync.Once
	data     *damodaranFile
	loadErr  error
}

func (d *DamodaranSource) Key() string   { return "damodaran" }
func (d *DamodaranSource) Label() string { return "Damodaran(美股,年度)" }
func (d *DamodaranSource) Scope() string { return "美股" }
func (d *DamodaranSource) Note() string  { return "NYU Stern 教授年度行业均值快照" }

func (d *DamodaranSource) load() error {
	d.once.Do(func() {
		path := d.DataPath
		if path == "" {
			path = filepath.Join("data", "damodaran_us.json")
		}
		b, err := os.ReadFile(path)
		if err != nil {
			d.loadErr = err
			return
		}
		var f damodaranFile
		if err := json.Unmarshal(b, &f); err != nil {
			d.loadErr = err
			return
		}
		d.data = &f
	})
	return d.loadErr
}

func (d *DamodaranSource) Industries() []string {
	if err := d.load(); err != nil {
		return nil
	}
	keys := make([]string, 0, len(d.data.Industries))
	for k := range d.data.Industries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func (d *DamodaranSource) Get(industry string) (*Benchmark, error) {
	if err := d.load(); err != nil {
		return nil, err
	}
	row, ok := d.data.Industries[industry]
	if !ok {
		return nil, fmt.Errorf("行业 %s 未找到", industry)
	}
	return &Benchmark{
		PELow:  round2(row.PE * 0.7),
		PEHigh: round2(row.PE * 1.3),
		PBLow:  round2(row.PB * 0.7),
		PBHigh: round2(row.PB * 1.3),
		ROEMin: round2(row.ROE * 0.8),
		DivMin: math.Max(round2(row.Div*0.7), 0.0),
		PEGMax: 1.5,
		Meta: Meta{
			Source: d.data.Meta.Source,
			Asof:   d.data.Meta.Asof,
			LivePE: row.PE,
			LivePB: row.PB,
			Note: fmt.Sprintf("Damodaran行业均值 PE=%.1f PB=%.1f ROE=%.1f%% 股息=%.1f%%。区间为均值±30%%",
				row.PE, row.PB, row.ROE, row.Div),
		},
	}, nil
}
