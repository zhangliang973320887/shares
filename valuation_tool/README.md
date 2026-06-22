# 股价估值分析工具

Flask 网页工具,输入 PE/PB/ROE/股息率/PEG,基于多数据源行业基准判断高估/低估。

## 启动

```bash
cd valuation_tool
pip3 install --break-system-packages -r requirements.txt
python3 app.py
```

浏览器: http://127.0.0.1:5000

## 数据源

| 源 | 范围 | 性质 | 来源 |
|----|------|------|------|
| `builtin` 内置静态 | A股 | 离线经验值 | 自编基准表(19行业) |
| `csindex` 中证指数 | A股 | **实时** | csindex.com.cn 行业指数估值, 6h缓存 |
| `damodaran` Damodaran | 美股 | 年度快照 | NYU Stern 教授 Damodaran 数据集(23行业) |

切换数据源 → 行业下拉自动刷新 → 提交后展示该源的基准+说明。

## 估值逻辑

5指标综合打分:

| 指标 | 判断 |
|------|------|
| PE | 低于区间下限 → 低估; 高于上限 → 高估 |
| PB | 同 PE |
| ROE | < 行业最低 → 高估信号(盈利能力不足) |
| 股息率 | < 行业最低 → 高估信号 |
| PEG | <1 低估; 1~上限 合理; >上限 高估 |

区间内 0 分,低估 -1,高估 +1,ROE/股息率反向。综合 ≤ -0.4 低估, ≥ 0.4 高估, 中间合理。

组合提示: 高PB+低ROE / 低PB+高ROE / 高PE+高PEG。

## 文件结构

```
valuation_tool/
├── app.py              # Flask 路由 + 评分
├── data_sources.py     # 数据源抽象 + 3个实现
├── data/
│   └── damodaran_us.json
├── cache/              # 自动生成,中证指数6h缓存
├── templates/index.html
└── requirements.txt
```

## 个股自动填充 + 自身历史分位

输入A股6位代码 → 点 "📥 自动填充" → 自动取:

- PE(TTM) / PB / PEG (东财估值数据, 近10年)
- ROE (最新年报)
- **PE/PB 近5年历史分位** (当前值在历史中处于多少%)

提交分析后,除"行业相对估值"外,额外显示 **"自身历史估值"** 标签:
- PE < 30% 分位 → 自身历史低估
- 30% ~ 70% → 中性
- \> 70% → 自身历史高估

两个视角互相印证: 即便行业基准说合理,自身历史在90%分位也是危险信号。

## API

- `GET /` 主页表单
- `POST /` 提交分析
- `GET /industries?source=builtin|csindex|damodaran` 行业列表 JSON
- `GET /lookup?code=600036` 个股估值数据 JSON

## 后续可扩展

- 加 DCF 内在价值估算
- Damodaran 每年自动同步官方CSV
- 港股/美股代码支持 (akshare 已有 stock_value_em 港美版)
- 个股加股息率自动获取
