package stock

import (
	"fmt"
	"strconv"
	"strings"
)

// FetchSinaROE 取最新年报 ROE (单位 %)
// 新浪 openapi 接口结构嵌套较深, 用 generic interface 解析
func FetchSinaROE(code string) (float64, string, error) {
	prefix := "sh"
	if strings.HasPrefix(code, "0") || strings.HasPrefix(code, "3") {
		prefix = "sz"
	}
	u := fmt.Sprintf("https://quotes.sina.cn/cn/api/openapi.php/CompanyFinanceService.getFinanceReport2022?paperCode=%s%s&source=gjzb&type=0&page=1&num=1000",
		prefix, code)

	var raw map[string]interface{}
	if err := httpGetJSON(u, nil, &raw); err != nil {
		return 0, "", err
	}
	result, ok := raw["result"].(map[string]interface{})
	if !ok {
		return 0, "", fmt.Errorf("no result")
	}
	data, ok := result["data"].(map[string]interface{})
	if !ok {
		return 0, "", fmt.Errorf("no data")
	}
	reportList, ok := data["report_list"].(map[string]interface{})
	if !ok {
		return 0, "", fmt.Errorf("no report_list")
	}

	// 优先年报 (yyyy1231)
	keys := make([]string, 0, len(reportList))
	for k := range reportList {
		keys = append(keys, k)
	}
	// 排序: 1231结尾优先, 然后日期倒序
	annual := make([]string, 0)
	others := make([]string, 0)
	for _, k := range keys {
		if strings.HasSuffix(k, "1231") {
			annual = append(annual, k)
		} else {
			others = append(others, k)
		}
	}
	// 倒序
	sortDesc(annual)
	sortDesc(others)
	order := append(annual, others...)

	for _, period := range order {
		rep, ok := reportList[period].(map[string]interface{})
		if !ok {
			continue
		}
		items, ok := rep["data"].([]interface{})
		if !ok {
			continue
		}
		for _, it := range items {
			m, ok := it.(map[string]interface{})
			if !ok {
				continue
			}
			title := getStr(m, "item_title")
			if !strings.Contains(title, "净资产收益率") {
				continue
			}
			val := getStr(m, "item_value")
			if val == "" || val == "--" {
				continue
			}
			f, err := strconv.ParseFloat(val, 64)
			if err != nil {
				continue
			}
			return f, period, nil
		}
	}
	return 0, "", fmt.Errorf("ROE未找到")
}

func sortDesc(s []string) {
	for i := 0; i < len(s); i++ {
		for j := i + 1; j < len(s); j++ {
			if s[j] > s[i] {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}

func getStr(m map[string]interface{}, k string) string {
	if v, ok := m[k]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
