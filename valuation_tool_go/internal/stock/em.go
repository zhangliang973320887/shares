package stock

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// EMIndustry 东财行业/板块
type EMIndustry struct {
	Industry string `json:"industry_em"`
	Area     string `json:"area"`
}

func emSecid(code string) string {
	if strings.HasPrefix(code, "6") || strings.HasPrefix(code, "9") {
		return "1." + code
	}
	return "0." + code
}

func httpGetJSON(u string, headers map[string]string, out interface{}) error {
	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X) Chrome/120.0")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(out)
}

// FetchEMIndustryHSF10 从东财个股F10公司概况页面取详细行业 (主接口, 稳定)
// EM2016 字段例: "食品饮料-饮料-白酒"
func FetchEMIndustryHSF10(code string) (*EMIndustry, error) {
	prefix := "SH"
	if !strings.HasPrefix(code, "6") && !strings.HasPrefix(code, "9") {
		prefix = "SZ"
	}
	u := fmt.Sprintf("https://emweb.securities.eastmoney.com/PC_HSF10/CompanySurvey/PageAjax?code=%s%s", prefix, code)
	var raw struct {
		Jbzl []struct {
			EM2016 string `json:"EM2016"`
			Trade  string `json:"TRADE_MARKET"`
		} `json:"jbzl"`
	}
	if err := httpGetJSON(u, nil, &raw); err != nil {
		return nil, err
	}
	if len(raw.Jbzl) == 0 || raw.Jbzl[0].EM2016 == "" {
		return nil, fmt.Errorf("no industry")
	}
	parts := strings.Split(raw.Jbzl[0].EM2016, "-")
	leaf := parts[len(parts)-1]
	return &EMIndustry{Industry: leaf, Area: ""}, nil
}

func FetchEMIndustry(code string) (*EMIndustry, error) {
	// 优先用 F10 接口 (稳定), 失败回退 push2
	if ind, err := FetchEMIndustryHSF10(code); err == nil && ind.Industry != "" {
		return ind, nil
	}
	u := fmt.Sprintf("https://push2.eastmoney.com/api/qt/stock/get?secid=%s&fields=f127,f128", emSecid(code))
	headers := map[string]string{"Referer": "https://quote.eastmoney.com/"}
	var resp struct {
		Data struct {
			F127 string `json:"f127"`
			F128 string `json:"f128"`
		} `json:"data"`
	}
	// 3次重试 (push2 偶发掉线)
	var lastErr error
	for i := 0; i < 3; i++ {
		if err := httpGetJSON(u, headers, &resp); err == nil && resp.Data.F127 != "" {
			return &EMIndustry{Industry: resp.Data.F127, Area: resp.Data.F128}, nil
		} else if err != nil {
			lastErr = err
		}
		time.Sleep(300 * time.Millisecond)
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return &EMIndustry{Industry: resp.Data.F127, Area: resp.Data.F128}, nil
}

// EMValueHistory 历史估值 (PE/PB/PEG)
type EMValueRow struct {
	Date string  `json:"date"`
	PE   float64 `json:"pe"`
	PB   float64 `json:"pb"`
	PEG  float64 `json:"peg"`
}

func FetchEMValueHistory(code string) ([]EMValueRow, error) {
	params := url.Values{}
	params.Set("sortColumns", "TRADE_DATE")
	params.Set("sortTypes", "-1")
	params.Set("pageSize", "5000")
	params.Set("pageNumber", "1")
	params.Set("reportName", "RPT_VALUEANALYSIS_DET")
	params.Set("columns", "ALL")
	params.Set("source", "WEB")
	params.Set("client", "WEB")
	params.Set("filter", fmt.Sprintf(`(SECURITY_CODE="%s")`, code))
	u := "https://datacenter-web.eastmoney.com/api/data/v1/get?" + params.Encode()

	var resp struct {
		Result struct {
			Data []map[string]interface{} `json:"data"`
		} `json:"result"`
	}
	if err := httpGetJSON(u, nil, &resp); err != nil {
		return nil, err
	}
	rows := make([]EMValueRow, 0, len(resp.Result.Data))
	for _, r := range resp.Result.Data {
		row := EMValueRow{
			Date: getString(r, "TRADE_DATE"),
			PE:   getFloat(r, "PE_TTM"),
			PB:   getFloat(r, "PB_MRQ"),
			PEG:  getFloat(r, "PEG_CAR"),
		}
		if row.Date != "" && len(row.Date) > 10 {
			row.Date = row.Date[:10]
		}
		rows = append(rows, row)
	}
	// sort ascending by date
	sort.Slice(rows, func(i, j int) bool { return rows[i].Date < rows[j].Date })
	return rows, nil
}

func getString(m map[string]interface{}, k string) string {
	if v, ok := m[k]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getFloat(m map[string]interface{}, k string) float64 {
	if v, ok := m[k]; ok {
		switch x := v.(type) {
		case float64:
			return x
		case string:
			f, _ := strconv.ParseFloat(x, 64)
			return f
		}
	}
	return 0
}

// AnalystReport 一份研报含目标价
type AnalystReport struct {
	Org    string  `json:"org"`
	Date   string  `json:"date"`
	Target float64 `json:"target"`
	Rating string  `json:"rating"`
	Title  string  `json:"title"`
}

type AnalystInfo struct {
	TotalReports int               `json:"total_reports"`
	RatingCount  map[string]int    `json:"rating_count"`
	Targets      []AnalystReport   `json:"targets"`
	TargetAvg    float64           `json:"target_avg,omitempty"`
	TargetMin    float64           `json:"target_min,omitempty"`
	TargetMax    float64           `json:"target_max,omitempty"`
	TargetCount  int               `json:"target_count"`
}

func FetchAnalystReports(code string, months int) (*AnalystInfo, error) {
	end := time.Now()
	start := end.AddDate(0, -months, 0)
	params := url.Values{}
	params.Set("cb", "datatable")
	params.Set("industryCode", "*")
	params.Set("pageSize", "100")
	params.Set("pageNo", "1")
	params.Set("qType", "0")
	params.Set("code", code)
	params.Set("industry", "*")
	params.Set("rating", "*")
	params.Set("ratingChange", "*")
	params.Set("beginTime", start.Format("2006-01-02"))
	params.Set("endTime", end.Format("2006-01-02"))
	params.Set("orgCode", "")
	u := "https://reportapi.eastmoney.com/report/list?" + params.Encode()

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	text := string(b)
	// strip JSONP wrap: datatable( ... )
	li := strings.Index(text, "(")
	ri := strings.LastIndex(text, ")")
	if li < 0 || ri <= li {
		return nil, fmt.Errorf("bad jsonp")
	}
	inner := text[li+1 : ri]

	var raw struct {
		Hits int                      `json:"hits"`
		Data []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal([]byte(inner), &raw); err != nil {
		return nil, err
	}

	info := &AnalystInfo{TotalReports: raw.Hits, RatingCount: map[string]int{}}
	if info.TotalReports == 0 {
		info.TotalReports = len(raw.Data)
	}
	for _, r := range raw.Data {
		rating := getString(r, "emRatingName")
		if rating == "" {
			rating = getString(r, "sRatingName")
		}
		if rating == "" {
			rating = "其它"
		}
		info.RatingCount[rating]++

		tStr := getString(r, "indvAimPriceT")
		if tStr == "" {
			continue
		}
		t, err := strconv.ParseFloat(tStr, 64)
		if err != nil || t <= 0 {
			continue
		}
		date := getString(r, "publishDate")
		if len(date) > 10 {
			date = date[:10]
		}
		title := getString(r, "title")
		runes := []rune(title)
		if len(runes) > 40 {
			title = string(runes[:40])
		}
		org := getString(r, "orgSName")
		if org == "" {
			org = getString(r, "orgName")
		}
		info.Targets = append(info.Targets, AnalystReport{
			Org: org, Date: date, Target: t, Rating: rating, Title: title,
		})
	}
	sort.Slice(info.Targets, func(i, j int) bool { return info.Targets[i].Date > info.Targets[j].Date })
	info.TargetCount = len(info.Targets)
	if info.TargetCount > 0 {
		sum, mn, mx := 0.0, info.Targets[0].Target, info.Targets[0].Target
		for _, t := range info.Targets {
			sum += t.Target
			if t.Target < mn {
				mn = t.Target
			}
			if t.Target > mx {
				mx = t.Target
			}
		}
		info.TargetAvg = round2(sum / float64(info.TargetCount))
		info.TargetMin = round2(mn)
		info.TargetMax = round2(mx)
	}
	if len(info.Targets) > 15 {
		info.Targets = info.Targets[:15]
	}
	return info, nil
}

func round2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}

// CodeName 全A代码-名称索引
type CodeName struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

func FetchAllACodeName() ([]CodeName, error) {
	fs := "m:0+t:6,m:0+t:13,m:0+t:80,m:1+t:2,m:1+t:23,m:0+t:81+s:2048"
	const pageSize = 500
	out := make([]CodeName, 0, 6000)
	for page := 1; page < 100; page++ {
		u := fmt.Sprintf(
			"http://82.push2.eastmoney.com/api/qt/clist/get?pn=%d&pz=%d&po=1&np=1&fields=f12,f14&fs=%s",
			page, pageSize, fs,
		)
		var resp struct {
			Data struct {
				Total int                 `json:"total"`
				Diff  []map[string]string `json:"diff"`
			} `json:"data"`
		}
		if err := httpGetJSON(u, nil, &resp); err != nil {
			return out, err
		}
		if len(resp.Data.Diff) == 0 {
			break
		}
		for _, r := range resp.Data.Diff {
			out = append(out, CodeName{Code: r["f12"], Name: r["f14"]})
		}
		if resp.Data.Total > 0 && len(out) >= resp.Data.Total {
			break
		}
	}
	return out, nil
}
