package stock

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"
)

// SearchUS 通过 Yahoo Finance 搜索美股
func SearchUS(query string, limit int) []CodeName {
	if query == "" {
		return nil
	}
	params := url.Values{}
	params.Set("q", query)
	params.Set("quotesCount", "10")
	params.Set("newsCount", "0")
	u := "https://query1.finance.yahoo.com/v1/finance/search?" + params.Encode()

	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	client := &http.Client{Timeout: 6 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	var raw struct {
		Quotes []struct {
			Symbol    string `json:"symbol"`
			ShortName string `json:"shortname"`
			LongName  string `json:"longname"`
			QuoteType string `json:"quoteType"`
			Exchange  string `json:"exchange"`
			ExchDisp  string `json:"exchDisp"`
		} `json:"quotes"`
	}
	if err := decodeJSON(resp, &raw); err != nil {
		return nil
	}

	out := make([]CodeName, 0, limit)
	for _, q := range raw.Quotes {
		if q.QuoteType != "EQUITY" {
			continue
		}
		// 只要美股主要市场 (NMS/NYQ/NGM/ASE/PNK)
		if q.Exchange != "NMS" && q.Exchange != "NYQ" && q.Exchange != "NGM" && q.Exchange != "ASE" {
			continue
		}
		name := q.LongName
		if name == "" {
			name = q.ShortName
		}
		out = append(out, CodeName{Code: q.Symbol, Name: name + " (" + q.ExchDisp + ")"})
		if len(out) >= limit {
			break
		}
	}
	return out
}

func decodeJSON(resp *http.Response, out interface{}) error {
	return json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(out)
}
