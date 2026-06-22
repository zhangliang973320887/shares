# 股价估值分析工具 (Go 版)

Python版的Go重写: 单二进制, 零Python依赖, 适合服务器部署。

## 启动

```bash
cd valuation_tool_go
./start.sh
```

或手动:
```bash
go mod tidy
go build -o valuation .
PORT=5000 HOST=0.0.0.0 ./valuation
```

## 功能

与Python版一致:
- 3 数据源行业基准 (内置/中证指数实时/Damodaran美股)
- 个股查询: 代码或公司名搜索 (~5800股A股索引)
- 自动填充: 实时PE/PB/PEG + ROE + 行业/板块/市值
- 5年PE/PB历史分位
- 机构研报 + 目标价

## 数据源映射 (无 akshare)

| 数据 | 端点 |
|------|------|
| 实时报价 | 腾讯 `qt.gtimg.cn` (GBK CSV) |
| 个股行业 | 东财 push2 `f127/f128` |
| 历史估值PE/PB/PEG | 东财 datacenter `RPT_VALUEANALYSIS_DET` |
| 研报+目标价 | 东财 reportapi `indvAimPriceT` |
| ROE 年报 | 新浪 openapi `CompanyFinanceService` |
| 全A代码名索引 | 东财 clist 分页 (每页100,~60页) |
| 中证行业指数估值 | csindex `oss-ch` .xls 文件 |

## 文件结构

```
valuation_tool_go/
├── main.go                    # gin 路由 + 视图组装
├── go.mod / go.sum
├── internal/
│   ├── benchmark/             # 数据源抽象
│   │   ├── types.go
│   │   ├── builtin.go
│   │   ├── damodaran.go
│   │   ├── csindex.go         # xls 解析 (github.com/extrame/xls)
│   │   └── registry.go
│   ├── stock/                 # 个股查询
│   │   ├── tencent.go         # GBK 解码 (x/text)
│   │   ├── em.go              # 东财所有 API
│   │   ├── sina.go            # 新浪 ROE
│   │   ├── search.go          # 名称索引+模糊匹配
│   │   └── lookup.go          # 综合查询入口
│   ├── scoring/scoring.go     # 5指标打分
│   └── cache/cache.go         # 文件缓存 (TTL: stock 5min, 其他 6h)
├── data/damodaran_us.json
├── web/templates/index.html   # Go html/template
├── cache/                     # 运行时生成,自动忽略
├── start.sh
└── README.md
```

## 部署

CentOS / Linux:
```bash
# 本机编译,丢服务器即可
GOOS=linux GOARCH=amd64 go build -o valuation .
scp valuation root@server:/opt/valuation/
scp -r web data start.sh root@server:/opt/valuation/

# 服务器
cd /opt/valuation
PORT=5000 HOST=0.0.0.0 ./valuation
```

无需 Python, 无需 pip, 单二进制 ~20MB。

## API

- `GET  /` 主页
- `POST /` 表单分析
- `GET  /industries?source=builtin|csindex|damodaran` 行业列表 JSON
- `GET  /search?q=xxx` 代码/名称搜索 JSON
- `GET  /lookup?code=xxx` 个股完整数据 JSON

## 环境变量

- `PORT` 监听端口 (默认 5000)
- `HOST` 监听地址 (默认 127.0.0.1, 公网部署设 0.0.0.0)
