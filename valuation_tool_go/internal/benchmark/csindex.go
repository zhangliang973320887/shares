package benchmark

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/extrame/xls"
	"valuation/internal/cache"
)

type CsindexSource struct {
	Cache *cache.Cache
}

func (CsindexSource) Key() string   { return "csindex" }
func (CsindexSource) Label() string { return "中证指数(A股,实时)" }
func (CsindexSource) Scope() string { return "A股" }
func (CsindexSource) Note() string  { return "从csindex.com.cn拉行业指数当前PE/股息率,6h缓存" }

var csiIndustryCodes = map[string]string{
	"中证银行":              "399986",
	"中证证券公司":            "399975",
	"中证白酒":              "399997",
	"中证主要消费(800消费)":     "000932",
	"中证医药卫生(800医药)":     "000933",
	"中证金融地产(800金地)":     "000934",
	"中证信息(800信息)":       "000935",
	"中证可选消费(800可选)":     "000931",
	"中证工业(800工业)":       "000930",
	"中证能源(800能源)":       "000928",
	"中证材料(800材料)":       "000929",
	"中证公用(800公用)":       "000936",
	"中证电信业务(800电信)":     "000937",
	"中证环保产业":            "000827",
	"中证新能源汽车":           "399976",
	"中证人工智能主题":          "930713",
	"中证全指":              "000985",
}

func (CsindexSource) Industries() []string {
	keys := make([]string, 0, len(csiIndustryCodes))
	for k := range csiIndustryCodes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

type csiFetched struct {
	PE   float64 `json:"pe"`
	Div  float64 `json:"div"`
	Date string  `json:"date"`
}

func fetchCsi(code string, c *cache.Cache) (*csiFetched, error) {
	key := "csi_" + code
	var cached csiFetched
	if c != nil && c.Get(key, &cached) {
		return &cached, nil
	}
	url := fmt.Sprintf("https://oss-ch.csindex.com.cn/static/html/csindex/public/uploads/file/autofile/indicator/%sindicator.xls", code)
	client := &http.Client{Timeout: 15 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("csindex http %d", resp.StatusCode)
	}
	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	wb, err := xls.OpenReader(bytes.NewReader(buf), "utf-8")
	if err != nil {
		return nil, err
	}
	sheet := wb.GetSheet(0)
	if sheet == nil || sheet.MaxRow == 0 {
		return nil, fmt.Errorf("no sheet")
	}
	// 最后一行 (最新)
	last := sheet.Row(int(sheet.MaxRow))
	if last == nil || last.LastCol() < 8 {
		return nil, fmt.Errorf("col mismatch")
	}
	pe, _ := strconv.ParseFloat(last.Col(6), 64)
	div, _ := strconv.ParseFloat(last.Col(8), 64)
	r := &csiFetched{PE: pe, Div: div, Date: last.Col(0)}
	if c != nil {
		c.Set(key, r)
	}
	return r, nil
}

func (s CsindexSource) Get(industry string) (*Benchmark, error) {
	code, ok := csiIndustryCodes[industry]
	if !ok {
		return nil, fmt.Errorf("行业 %s 未配置", industry)
	}
	d, err := fetchCsi(code, s.Cache)
	if err != nil {
		return nil, err
	}
	peLow := round2(d.PE * 0.75)
	peHigh := round2(d.PE * 1.25)
	pbHigh := round2(d.PE / 8)
	if pbHigh < 1.5 {
		pbHigh = 1.5
	}
	pbLow := round2(pbHigh * 0.5)
	divMin := round2(d.Div * 0.7)
	if divMin < 0.5 {
		divMin = 0.5
	}
	return &Benchmark{
		PELow:  peLow,
		PEHigh: peHigh,
		PBLow:  pbLow,
		PBHigh: pbHigh,
		ROEMin: 10,
		DivMin: divMin,
		PEGMax: 1.3,
		Meta: Meta{
			Source:  "中证指数官网 (csindex.com.cn)",
			Asof:    d.Date,
			LivePE:  d.PE,
			LiveDiv: d.Div,
			Note:    fmt.Sprintf("行业指数当前PE=%.2f, 股息率=%.2f%%, 区间为当前值±25%%", d.PE, d.Div),
		},
	}, nil
}
