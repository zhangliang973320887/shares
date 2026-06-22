package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"

	"valuation/internal/benchmark"
	"valuation/internal/cache"
	"valuation/internal/scoring"
	"valuation/internal/stock"
)

func parsef(s string) *float64 {
	if s == "" {
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &v
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}
	host := os.Getenv("HOST")
	if host == "" {
		host = "127.0.0.1"
	}

	c := cache.New("cache")
	reg := benchmark.NewRegistry(c, "data/damodaran_us.json")
	idx := &stock.SearchIndex{Cache: c}

	r := gin.Default()
	r.SetFuncMap(template.FuncMap{
		"fmt2": func(v float64) string {
			return strconv.FormatFloat(v, 'f', 2, 64)
		},
		"deref": func(p *float64) float64 {
			if p == nil {
				return 0
			}
			return *p
		},
		"notnil": func(p *float64) bool { return p != nil },
		"derefi": func(p *int) int {
			if p == nil {
				return 0
			}
			return *p
		},
		"notnili": func(p *int) bool { return p != nil },
	})
	r.LoadHTMLGlob("web/templates/*")
	r.Static("/static", "./web/static")

	// PWA: SW + manifest 走根路径
	r.GET("/sw.js", func(ctx *gin.Context) {
		ctx.Header("Service-Worker-Allowed", "/")
		ctx.Header("Content-Type", "application/javascript")
		ctx.File("./web/static/sw.js")
	})
	r.GET("/manifest.json", func(ctx *gin.Context) {
		ctx.Header("Content-Type", "application/manifest+json")
		ctx.File("./web/static/manifest.json")
	})

	indexHandler := func(ctx *gin.Context) {
		_ = ctx.Request.ParseForm()
		var form map[string][]string
		if ctx.Request.Method == "POST" {
			form = ctx.Request.PostForm
		}
		renderIndex(ctx, reg, c, idx, form, "")
	}
	r.GET("/", indexHandler)
	r.POST("/", indexHandler)

	r.GET("/industries", func(ctx *gin.Context) {
		source := ctx.DefaultQuery("source", "builtin")
		ctx.JSON(http.StatusOK, gin.H{"industries": reg.Industries(source)})
	})

	r.GET("/search", func(ctx *gin.Context) {
		q := ctx.Query("q")
		ctx.JSON(http.StatusOK, gin.H{"results": idx.Search(q, 15)})
	})

	r.GET("/lookup", func(ctx *gin.Context) {
		q := ctx.Query("code")
		code := idx.Resolve(q)
		if code == "" {
			hits := idx.Search(q, 10)
			if len(hits) == 0 {
				ctx.JSON(http.StatusOK, gin.H{"error": "未找到匹配 '" + q + "'"})
				return
			}
			ctx.JSON(http.StatusOK, gin.H{
				"error":       "'" + q + "' 匹配多个,请选择",
				"suggestions": hits,
			})
			return
		}
		info := stock.Lookup(code, c)
		ctx.JSON(http.StatusOK, info)
	})

	addr := host + ":" + port
	log.Println("Listening on http://" + addr)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}

type viewData struct {
	Sources           []benchmark.SourceInfo
	SourcesMap        map[string]benchmark.SourceInfo
	InitialIndustries []string
	Form              formData
	Result            *resultView
	Error             string
}

type formData struct {
	Source   string
	Industry string
	Code     string
	Name     string
	PE       string
	PB       string
	ROE      string
	Div      string
	PEG      string
}

type resultView struct {
	Rows        []scoring.Row
	Summary     string
	Color       string
	Source      string
	Industry    string
	Name        string
	Notes       []string
	Meta        benchmark.Meta
	SelfHistory *selfHistoryView
	Analyst     *analystView
	AutoLocked  bool
}

type selfHistoryView struct {
	Label         string
	Color         string
	PERange       string
	PEBand        string
	PBPercentile  *float64
	Asof          string
}

type analystView struct {
	TotalReports int
	RatingCount  map[string]int
	Targets      []stock.AnalystReport
	TargetAvg    float64
	TargetMin    float64
	TargetMax    float64
	TargetCount  int
	Price        float64
	UpsidePct    *float64
}

func renderIndex(ctx *gin.Context, reg *benchmark.Registry, c *cache.Cache, idx *stock.SearchIndex, form map[string][]string, errMsg string) {
	get := func(k string) string {
		if form == nil {
			return ""
		}
		if vs, ok := form[k]; ok && len(vs) > 0 {
			return vs[0]
		}
		return ""
	}
	srcKey := get("source")
	if srcKey == "" {
		srcKey = "builtin"
	}

	fd := formData{
		Source:   srcKey,
		Industry: get("industry"),
		Code:     get("code"),
		Name:     get("name"),
		PE:       get("pe"),
		PB:       get("pb"),
		ROE:      get("roe"),
		Div:      get("div"),
		PEG:      get("peg"),
	}

	sources := reg.List()
	srcMap := map[string]benchmark.SourceInfo{}
	for _, s := range sources {
		srcMap[s.Key] = s
	}

	data := viewData{
		Sources:           sources,
		SourcesMap:        srcMap,
		InitialIndustries: reg.Industries(srcKey),
		Form:              fd,
		Error:             errMsg,
	}

	if ctx.Request.Method == "POST" {
		var stockInfo *stock.StockInfo
		var analyst *analystView
		var selfHist *selfHistoryView
		autoLocked := false

		// 先解析股票, 拿到行业自动锁定 source/industry
		if fd.Code != "" {
			resolved := idx.Resolve(fd.Code)
			if resolved != "" {
				fd.Code = resolved
				if get("use_self_history") == "1" || fd.Industry == "" {
					stockInfo = stock.Lookup(resolved, c)
					if stockInfo != nil && stockInfo.IndustryCSI != "" {
						fd.Source = "csindex"
						fd.Industry = stockInfo.IndustryCSI
						autoLocked = true
					}
				}
				if stockInfo != nil && stockInfo.Analyst != nil {
					analyst = &analystView{
						TotalReports: stockInfo.Analyst.TotalReports,
						RatingCount:  stockInfo.Analyst.RatingCount,
						Targets:      stockInfo.Analyst.Targets,
						TargetAvg:    stockInfo.Analyst.TargetAvg,
						TargetMin:    stockInfo.Analyst.TargetMin,
						TargetMax:    stockInfo.Analyst.TargetMax,
						TargetCount:  stockInfo.Analyst.TargetCount,
						Price:        stockInfo.Price,
						UpsidePct:    stockInfo.AnalystUpsidePct,
					}
				}
			}
		}

		bm, err := reg.Get(fd.Source, fd.Industry)
		if err != nil {
			data.Error = "无法获取基准: " + err.Error()
			ctx.HTML(http.StatusOK, "index.html", data)
			return
		}

		pe := parsef(fd.PE)
		pb := parsef(fd.PB)
		roe := parsef(fd.ROE)
		div := parsef(fd.Div)
		peg := parsef(fd.PEG)
		result := scoring.Analyze(bm, pe, pb, roe, div, peg)

		// 自身历史
		if stockInfo != nil && stockInfo.PEPercentile != nil && pe != nil {
			p := *stockInfo.PEPercentile
			lbl, col := "PE处于近5年 中性", "warning"
			switch {
			case p < 30:
				lbl, col = "PE处于近5年低估", "success"
			case p > 70:
				lbl, col = "PE处于近5年高估", "danger"
			}
			selfHist = &selfHistoryView{
				Label:        lbl + " (" + strconv.FormatFloat(p, 'f', 1, 64) + "%分位)",
				Color:        col,
				PERange:      strconv.FormatFloat(stockInfo.PEMin, 'f', 2, 64) + " ~ " + strconv.FormatFloat(stockInfo.PEMax, 'f', 2, 64),
				PEBand:       "P30=" + strconv.FormatFloat(stockInfo.PEP30, 'f', 2, 64) + ", P70=" + strconv.FormatFloat(stockInfo.PEP70, 'f', 2, 64),
				PBPercentile: stockInfo.PBPercentile,
				Asof:         stockInfo.AsofHist,
			}
		}

		name := fd.Name
		if name == "" {
			name = "目标公司"
		}
		data.Result = &resultView{
			Rows:        result.Rows,
			Summary:     result.Summary,
			Color:       result.Color,
			Source:      fd.Source,
			Industry:    fd.Industry,
			Name:        name,
			Notes:       result.Notes,
			Meta:        bm.Meta,
			SelfHistory: selfHist,
			Analyst:     analyst,
			AutoLocked:  autoLocked,
		}
		// Form 同步, 让下次表单显示锁定值
		data.Form = fd
		data.InitialIndustries = reg.Industries(fd.Source)
	}
	ctx.HTML(http.StatusOK, "index.html", data)
}
