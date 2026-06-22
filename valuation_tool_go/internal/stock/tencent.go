package stock

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// TencentQuote 腾讯实时报价
type TencentQuote struct {
	Name          string  `json:"name"`
	Price         float64 `json:"price"`
	PETTM         float64 `json:"pe_ttm"`
	PB            float64 `json:"pb"`
	MktCapYi      float64 `json:"mkt_cap_yi"`
	CircCapYi     float64 `json:"circ_cap_yi"`
	AsofRealtime  string  `json:"asof_realtime"`
}

func tencentPrefix(code string) string {
	if strings.HasPrefix(code, "6") || strings.HasPrefix(code, "9") {
		return "sh"
	}
	return "sz"
}

func FetchTencent(code string) (*TencentQuote, error) {
	url := fmt.Sprintf("http://qt.gtimg.cn/q=%s%s", tencentPrefix(code), code)
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	rd := transform.NewReader(resp.Body, simplifiedchinese.GBK.NewDecoder())
	b, err := io.ReadAll(rd)
	if err != nil {
		return nil, err
	}
	text := string(b)
	i := strings.Index(text, "\"")
	if i < 0 {
		return nil, fmt.Errorf("bad resp")
	}
	body := strings.TrimRight(text[i+1:], "\";\n ")
	parts := strings.Split(body, "~")
	if len(parts) < 50 {
		return nil, fmt.Errorf("parts %d", len(parts))
	}

	f := func(idx int) float64 {
		if idx >= len(parts) {
			return 0
		}
		s := strings.TrimSpace(parts[idx])
		if s == "" || s == "-" {
			return 0
		}
		v, _ := strconv.ParseFloat(s, 64)
		return v
	}

	asof := ""
	if len(parts) > 30 && len(parts[30]) >= 12 {
		t := parts[30]
		asof = fmt.Sprintf("%s-%s-%s %s:%s", t[0:4], t[4:6], t[6:8], t[8:10], t[10:12])
	}

	q := &TencentQuote{
		Name:         strings.TrimSpace(parts[1]),
		Price:        f(3),
		PETTM:        f(39),
		CircCapYi:    f(44),
		MktCapYi:     f(45),
		PB:           f(46),
		AsofRealtime: asof,
	}
	return q, nil
}
