package scoring

import (
	"fmt"

	"valuation/internal/benchmark"
)

type Row struct {
	Name    string  `json:"name"`
	Value   *float64 `json:"value"`
	BMText  string  `json:"bm_text"`
	Verdict string  `json:"verdict"`
	Score   *int    `json:"score"`
}

type Result struct {
	Rows    []Row    `json:"rows"`
	Summary string   `json:"summary"`
	Color   string   `json:"color"`
	Notes   []string `json:"notes"`
}

func judgeBand(val *float64, low, high float64) (string, *int) {
	if val == nil {
		return "未填写", nil
	}
	if *val <= 0 {
		return "数据异常(≤0,可能亏损)", nil
	}
	s := 0
	if *val < low {
		s := -1
		return fmt.Sprintf("低估 (< %g)", low), &s
	}
	if *val > high {
		s := 1
		return fmt.Sprintf("高估 (> %g)", high), &s
	}
	return fmt.Sprintf("合理 (%g~%g)", low, high), &s
}

func judgeMin(val *float64, minimum float64, low, ok string) (string, *int) {
	if val == nil {
		return "未填写", nil
	}
	if *val < minimum {
		s := 1
		return fmt.Sprintf("%s (< %g)", low, minimum), &s
	}
	s := -1
	return fmt.Sprintf("%s (≥ %g)", ok, minimum), &s
}

func judgePEG(val *float64, peg_max float64) (string, *int) {
	if val == nil {
		return "未填写", nil
	}
	if *val <= 0 {
		return "数据异常(≤0,可能负增长)", nil
	}
	if *val < 1.0 {
		s := -1
		return "低估 (PEG < 1)", &s
	}
	if *val > peg_max {
		s := 1
		return fmt.Sprintf("高估 (PEG > %g)", peg_max), &s
	}
	s := 0
	return fmt.Sprintf("合理 (1 ~ %g)", peg_max), &s
}

func composite(sum, n int) (string, string) {
	if n == 0 {
		return "无有效输入", "secondary"
	}
	r := float64(sum) / float64(n)
	if r <= -0.4 {
		return "综合判断: 低估", "success"
	}
	if r >= 0.4 {
		return "综合判断: 高估", "danger"
	}
	return "综合判断: 合理", "warning"
}

func Analyze(bm *benchmark.Benchmark, pe, pb, roe, div, peg *float64) Result {
	rows := []Row{}
	sum, n := 0, 0
	add := func(name string, val *float64, bmText, verdict string, s *int) {
		rows = append(rows, Row{name, val, bmText, verdict, s})
		if s != nil {
			sum += *s
			n++
		}
	}

	v, s := judgeBand(pe, bm.PELow, bm.PEHigh)
	add("PE", pe, fmt.Sprintf("%g ~ %g", bm.PELow, bm.PEHigh), v, s)
	v, s = judgeBand(pb, bm.PBLow, bm.PBHigh)
	add("PB", pb, fmt.Sprintf("%g ~ %g", bm.PBLow, bm.PBHigh), v, s)
	v, s = judgeMin(roe, bm.ROEMin, "偏弱", "达标")
	add("ROE %", roe, fmt.Sprintf("≥ %g%%", bm.ROEMin), v, s)
	v, s = judgeMin(div, bm.DivMin, "偏弱", "达标")
	add("股息率 %", div, fmt.Sprintf("≥ %g%%", bm.DivMin), v, s)
	v, s = judgePEG(peg, bm.PEGMax)
	add("PEG", peg, fmt.Sprintf("1 ~ %g", bm.PEGMax), v, s)

	summary, color := composite(sum, n)

	notes := []string{}
	if pb != nil && roe != nil && *pb > 0 && *roe > 0 {
		if *pb > 3 && *roe < 15 {
			notes = append(notes, "高PB+低ROE: 资本回报无法支撑高PB,警惕高估")
		}
		if *pb < 1 && *roe > 10 {
			notes = append(notes, "低PB+高ROE: 典型价值低估信号")
		}
	}
	if pe != nil && peg != nil && *pe > 30 && *peg > 1.5 {
		notes = append(notes, "高PE+高PEG: 成长性不足以消化估值")
	}

	return Result{Rows: rows, Summary: summary, Color: color, Notes: notes}
}
