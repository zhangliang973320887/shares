package benchmark

import "fmt"

type BuiltinSource struct{}

func (BuiltinSource) Key() string   { return "builtin" }
func (BuiltinSource) Label() string { return "内置静态(A股,经验值)" }
func (BuiltinSource) Scope() string { return "A股" }
func (BuiltinSource) Note() string  { return "完全离线,粗略经验,无市场实时数据" }

var builtinData = map[string]Benchmark{
	"银行":       {PELow: 4, PEHigh: 8, PBLow: 0.5, PBHigh: 1.0, ROEMin: 10, DivMin: 3.0, PEGMax: 1.0},
	"保险":       {PELow: 8, PEHigh: 15, PBLow: 1.0, PBHigh: 2.0, ROEMin: 10, DivMin: 2.0, PEGMax: 1.2},
	"证券":       {PELow: 15, PEHigh: 25, PBLow: 1.0, PBHigh: 2.0, ROEMin: 8, DivMin: 1.5, PEGMax: 1.5},
	"白酒":       {PELow: 25, PEHigh: 40, PBLow: 5.0, PBHigh: 10.0, ROEMin: 20, DivMin: 1.5, PEGMax: 1.5},
	"食品饮料":     {PELow: 20, PEHigh: 35, PBLow: 3.0, PBHigh: 6.0, ROEMin: 15, DivMin: 1.5, PEGMax: 1.5},
	"医药生物":     {PELow: 25, PEHigh: 40, PBLow: 3.0, PBHigh: 6.0, ROEMin: 12, DivMin: 1.0, PEGMax: 1.5},
	"科技互联网":    {PELow: 25, PEHigh: 50, PBLow: 3.0, PBHigh: 8.0, ROEMin: 10, DivMin: 0.5, PEGMax: 1.5},
	"半导体":      {PELow: 30, PEHigh: 60, PBLow: 4.0, PBHigh: 8.0, ROEMin: 10, DivMin: 0.5, PEGMax: 1.8},
	"新能源":      {PELow: 20, PEHigh: 40, PBLow: 3.0, PBHigh: 6.0, ROEMin: 12, DivMin: 0.8, PEGMax: 1.5},
	"汽车":       {PELow: 15, PEHigh: 25, PBLow: 1.5, PBHigh: 3.0, ROEMin: 10, DivMin: 1.5, PEGMax: 1.3},
	"家电":       {PELow: 15, PEHigh: 25, PBLow: 2.0, PBHigh: 4.0, ROEMin: 15, DivMin: 2.5, PEGMax: 1.3},
	"地产":       {PELow: 5, PEHigh: 10, PBLow: 0.6, PBHigh: 1.5, ROEMin: 8, DivMin: 3.0, PEGMax: 1.0},
	"基建建筑":     {PELow: 6, PEHigh: 10, PBLow: 0.8, PBHigh: 1.2, ROEMin: 8, DivMin: 3.0, PEGMax: 1.0},
	"钢铁有色":     {PELow: 5, PEHigh: 12, PBLow: 0.6, PBHigh: 1.2, ROEMin: 8, DivMin: 2.5, PEGMax: 1.0},
	"煤炭石油":     {PELow: 6, PEHigh: 12, PBLow: 1.0, PBHigh: 2.0, ROEMin: 10, DivMin: 4.0, PEGMax: 1.0},
	"公用事业":     {PELow: 12, PEHigh: 20, PBLow: 1.2, PBHigh: 2.5, ROEMin: 8, DivMin: 3.0, PEGMax: 1.2},
	"制造业(通用)": {PELow: 15, PEHigh: 25, PBLow: 1.5, PBHigh: 3.0, ROEMin: 10, DivMin: 1.5, PEGMax: 1.3},
	"消费(通用)":  {PELow: 20, PEHigh: 30, PBLow: 3.0, PBHigh: 5.0, ROEMin: 15, DivMin: 1.5, PEGMax: 1.3},
	"通用基准":     {PELow: 15, PEHigh: 25, PBLow: 1.5, PBHigh: 3.0, ROEMin: 10, DivMin: 1.5, PEGMax: 1.3},
}

func (BuiltinSource) Industries() []string {
	return []string{"银行", "保险", "证券", "白酒", "食品饮料", "医药生物", "科技互联网",
		"半导体", "新能源", "汽车", "家电", "地产", "基建建筑", "钢铁有色", "煤炭石油",
		"公用事业", "制造业(通用)", "消费(通用)", "通用基准"}
}

func (BuiltinSource) Get(industry string) (*Benchmark, error) {
	bm, ok := builtinData[industry]
	if !ok {
		return nil, fmt.Errorf("行业 %s 未配置", industry)
	}
	bm.Meta = Meta{Source: "内置经验值", Asof: "静态", Note: "无市场实时数据"}
	return &bm, nil
}
