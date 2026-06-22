package stock

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"valuation/internal/cache"
)

type StockInfo struct {
	Code             string         `json:"code"`
	Name             string         `json:"name,omitempty"`
	Price            float64        `json:"price,omitempty"`
	PE               float64        `json:"pe,omitempty"`
	PB               float64        `json:"pb,omitempty"`
	PEG              float64        `json:"peg,omitempty"`
	ROE              float64        `json:"roe,omitempty"`
	ROEPeriod        string         `json:"roe_period,omitempty"`
	MktCapYi         float64        `json:"mkt_cap_yi,omitempty"`
	AsofRealtime     string         `json:"asof_realtime,omitempty"`
	AsofHist         string         `json:"asof_hist,omitempty"`
	IndustryEM       string         `json:"industry_em,omitempty"`
	IndustryCSI      string         `json:"industry_csi,omitempty"`
	Area             string         `json:"area,omitempty"`
	PEPercentile     *float64       `json:"pe_percentile,omitempty"`
	PEMin            float64        `json:"pe_5y_min,omitempty"`
	PEMax            float64        `json:"pe_5y_max,omitempty"`
	PEP30            float64        `json:"pe_5y_p30,omitempty"`
	PEP70            float64        `json:"pe_5y_p70,omitempty"`
	PBPercentile     *float64       `json:"pb_percentile,omitempty"`
	PBMin            float64        `json:"pb_5y_min,omitempty"`
	PBMax            float64        `json:"pb_5y_max,omitempty"`
	Analyst          *AnalystInfo   `json:"analyst,omitempty"`
	AnalystUpsidePct *float64       `json:"analyst_upside_pct,omitempty"`
	Warn             []string       `json:"warn,omitempty"`
}

// IndustryToCSI 东财行业 → 中证指数行业 (前缀匹配)
var industryToCSI = []struct{ Key, CSI string }{
	{"银行", "中证银行"},
	{"城商行", "中证银行"},
	{"农商行", "中证银行"},
	{"股份制", "中证银行"},
	{"国有大行", "中证银行"},
	{"证券", "中证证券公司"},
	{"白酒", "中证白酒"},
	{"食品", "中证主要消费(800消费)"},
	{"饮料", "中证主要消费(800消费)"},
	{"乳品", "中证主要消费(800消费)"},
	{"肉制品", "中证主要消费(800消费)"},
	{"医药", "中证医药卫生(800医药)"},
	{"化学制药", "中证医药卫生(800医药)"},
	{"生物制品", "中证医药卫生(800医药)"},
	{"医疗", "中证医药卫生(800医药)"},
	{"中药", "中证医药卫生(800医药)"},
	{"医美", "中证医药卫生(800医药)"},
	{"房地产", "中证金融地产(800金地)"},
	{"地产", "中证金融地产(800金地)"},
	{"保险", "中证金融地产(800金地)"},
	{"多元金融", "中证金融地产(800金地)"},
	{"信托", "中证金融地产(800金地)"},
	{"计算机", "中证信息(800信息)"},
	{"软件", "中证信息(800信息)"},
	{"IT", "中证信息(800信息)"},
	{"互联网", "中证信息(800信息)"},
	{"通信", "中证电信业务(800电信)"},
	{"电子", "中证信息(800信息)"},
	{"半导体", "中证信息(800信息)"},
	{"芯片", "中证信息(800信息)"},
	{"消费电子", "中证信息(800信息)"},
	{"光学光电", "中证信息(800信息)"},
	{"元件", "中证信息(800信息)"},
	{"PCB", "中证信息(800信息)"},
	{"乘用车", "中证新能源汽车"},
	{"商用车", "中证新能源汽车"},
	{"汽车零部件", "中证新能源汽车"},
	{"汽车", "中证新能源汽车"},
	{"新能源", "中证新能源汽车"},
	{"电池", "中证新能源汽车"},
	{"锂电", "中证新能源汽车"},
	{"光伏", "中证新能源汽车"},
	{"太阳能", "中证新能源汽车"},
	{"风电", "中证新能源汽车"},
	{"储能", "中证新能源汽车"},
	{"充电桩", "中证新能源汽车"},
	{"煤炭", "中证能源(800能源)"},
	{"石油", "中证能源(800能源)"},
	{"油气", "中证能源(800能源)"},
	{"采掘", "中证能源(800能源)"},
	{"钢铁", "中证材料(800材料)"},
	{"有色", "中证材料(800材料)"},
	{"化工", "中证材料(800材料)"},
	{"塑料", "中证材料(800材料)"},
	{"橡胶", "中证材料(800材料)"},
	{"金属", "中证材料(800材料)"},
	{"建材", "中证材料(800材料)"},
	{"水泥", "中证材料(800材料)"},
	{"造纸", "中证材料(800材料)"},
	{"公用事业", "中证公用(800公用)"},
	{"电力", "中证公用(800公用)"},
	{"燃气", "中证公用(800公用)"},
	{"水务", "中证公用(800公用)"},
	{"环保", "中证环保产业"},
	{"机械", "中证工业(800工业)"},
	{"建筑", "中证工业(800工业)"},
	{"工程", "中证工业(800工业)"},
	{"国防", "中证工业(800工业)"},
	{"军工", "中证工业(800工业)"},
	{"航空", "中证工业(800工业)"},
	{"航天", "中证工业(800工业)"},
	{"船舶", "中证工业(800工业)"},
	{"铁路", "中证工业(800工业)"},
	{"交运", "中证工业(800工业)"},
	{"物流", "中证工业(800工业)"},
	{"港口", "中证工业(800工业)"},
	{"机场", "中证工业(800工业)"},
	{"家电", "中证可选消费(800可选)"},
	{"白色家电", "中证可选消费(800可选)"},
	{"黑色家电", "中证可选消费(800可选)"},
	{"纺织", "中证可选消费(800可选)"},
	{"服装", "中证可选消费(800可选)"},
	{"零售", "中证可选消费(800可选)"},
	{"商业", "中证可选消费(800可选)"},
	{"贸易", "中证可选消费(800可选)"},
	{"教育", "中证可选消费(800可选)"},
	{"传媒", "中证可选消费(800可选)"},
	{"广告", "中证可选消费(800可选)"},
	{"游戏", "中证可选消费(800可选)"},
	{"影视", "中证可选消费(800可选)"},
	{"酒店", "中证可选消费(800可选)"},
	{"旅游", "中证可选消费(800可选)"},
	{"餐饮", "中证可选消费(800可选)"},
	{"农林", "中证主要消费(800消费)"},
	{"种植", "中证主要消费(800消费)"},
	{"养殖", "中证主要消费(800消费)"},
	{"渔业", "中证主要消费(800消费)"},
	{"农产品", "中证主要消费(800消费)"},
}

func mapIndustryToCSI(emName string) string {
	clean := strings.ReplaceAll(emName, "Ⅱ", "")
	clean = strings.ReplaceAll(clean, "Ⅰ", "")
	clean = strings.ReplaceAll(clean, "Ⅲ", "")
	clean = strings.TrimSpace(clean)
	for _, m := range industryToCSI {
		if strings.Contains(clean, m.Key) {
			return m.CSI
		}
	}
	return ""
}

func percentile(values []float64, current float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	count := 0
	for _, v := range sorted {
		if v <= current {
			count++
		}
	}
	return float64(int(float64(count)/float64(len(sorted))*1000+0.5)) / 10
}

func pAt(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted)) * p)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func lookupUS(code string, c *cache.Cache) *StockInfo {
	key := "stock_" + code
	if c != nil {
		var cached StockInfo
		if c.Get(key, &cached) {
			return &cached
		}
	}
	info := &StockInfo{Code: code}
	q, err := FetchTencent(code)
	if err != nil {
		return &StockInfo{Code: code, Warn: []string{fmt.Sprintf("美股查询失败: %v", err)}}
	}
	info.Name = q.Name
	info.Price = q.Price
	info.PE = q.PETTM
	info.PB = q.PB
	info.MktCapYi = q.MktCapYi
	info.AsofRealtime = q.AsofRealtime
	info.IndustryEM = "美股"
	info.Warn = append(info.Warn, "美股: 暂无历史分位/ROE/机构目标价")
	if c != nil {
		c.Set(key, info)
	}
	return info
}

func Lookup(code string, c *cache.Cache) *StockInfo {
	if IsUSTicker(strings.ToUpper(code)) {
		return lookupUS(strings.ToUpper(code), c)
	}
	if len(code) != 6 || !isAllDigit(code) {
		return &StockInfo{Code: code, Warn: []string{"无效代码 (A股6位数字 / 美股大写字母)"}}
	}
	key := "stock_" + code
	if c != nil {
		var cached StockInfo
		if c.Get(key, &cached) {
			return &cached
		}
	}

	info := &StockInfo{Code: code}

	// 腾讯实时
	if q, err := FetchTencent(code); err == nil {
		info.Name = q.Name
		info.Price = q.Price
		info.PE = q.PETTM
		info.PB = q.PB
		info.MktCapYi = q.MktCapYi
		info.AsofRealtime = q.AsofRealtime
	} else {
		info.Warn = append(info.Warn, fmt.Sprintf("tencent: %v", err))
	}

	// 东财行业
	if ind, err := FetchEMIndustry(code); err == nil {
		info.IndustryEM = ind.Industry
		info.Area = ind.Area
		if mapped := mapIndustryToCSI(ind.Industry); mapped != "" {
			info.IndustryCSI = mapped
		}
	}

	// 历史估值 + 分位
	if hist, err := FetchEMValueHistory(code); err == nil && len(hist) > 0 {
		latest := hist[len(hist)-1]
		info.AsofHist = latest.Date
		if info.PEG == 0 {
			info.PEG = latest.PEG
		}
		if info.PE == 0 && latest.PE > 0 {
			info.PE = latest.PE
		}
		if info.PB == 0 && latest.PB > 0 {
			info.PB = latest.PB
		}
		from := 0
		if len(hist) > 1250 {
			from = len(hist) - 1250
		}
		pe5y := make([]float64, 0)
		pb5y := make([]float64, 0)
		for _, r := range hist[from:] {
			if r.PE > 0 {
				pe5y = append(pe5y, r.PE)
			}
			if r.PB > 0 {
				pb5y = append(pb5y, r.PB)
			}
		}
		if len(pe5y) > 0 && info.PE > 0 {
			p := percentile(pe5y, info.PE)
			info.PEPercentile = &p
			sorted := append([]float64(nil), pe5y...)
			sort.Float64s(sorted)
			info.PEMin = round2(sorted[0])
			info.PEMax = round2(sorted[len(sorted)-1])
			info.PEP30 = round2(pAt(sorted, 0.3))
			info.PEP70 = round2(pAt(sorted, 0.7))
		}
		if len(pb5y) > 0 && info.PB > 0 {
			p := percentile(pb5y, info.PB)
			info.PBPercentile = &p
			sorted := append([]float64(nil), pb5y...)
			sort.Float64s(sorted)
			info.PBMin = round2(sorted[0])
			info.PBMax = round2(sorted[len(sorted)-1])
		}
	} else if err != nil {
		info.Warn = append(info.Warn, fmt.Sprintf("history: %v", err))
	}

	// ROE
	if roe, period, err := FetchSinaROE(code); err == nil {
		info.ROE = roe
		info.ROEPeriod = period
	}

	// 研报+目标价
	if a, err := FetchAnalystReports(code, 12); err == nil {
		info.Analyst = a
		if a.TargetAvg > 0 && info.Price > 0 {
			up := math.Round((a.TargetAvg-info.Price)/info.Price*1000) / 10
			info.AnalystUpsidePct = &up
		}
	}

	if c != nil {
		c.Set(key, info)
	}
	return info
}
