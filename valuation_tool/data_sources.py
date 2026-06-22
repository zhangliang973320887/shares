"""数据源抽象层。
每个数据源提供:
- industries(): 该源支持的行业列表
- get_benchmark(industry): 返回 {pe:(low,high), pb:(low,high), roe_min, div_min, peg_max, meta}
"""
import json
import time
from pathlib import Path

DATA_DIR = Path(__file__).parent / "data"
CACHE_DIR = Path(__file__).parent / "cache"
CACHE_DIR.mkdir(exist_ok=True)
CACHE_TTL = 3600 * 6  # 默认6小时
CACHE_TTL_STOCK = 300  # 个股实时5分钟


# ============================================================
# 源1: 内置静态A股基准 (经验值,无网络)
# ============================================================
_BUILTIN_BENCHMARKS = {
    "银行": {"pe": (4, 8), "pb": (0.5, 1.0), "roe_min": 10, "div_min": 3.0, "peg_max": 1.0},
    "保险": {"pe": (8, 15), "pb": (1.0, 2.0), "roe_min": 10, "div_min": 2.0, "peg_max": 1.2},
    "证券": {"pe": (15, 25), "pb": (1.0, 2.0), "roe_min": 8, "div_min": 1.5, "peg_max": 1.5},
    "白酒": {"pe": (25, 40), "pb": (5.0, 10.0), "roe_min": 20, "div_min": 1.5, "peg_max": 1.5},
    "食品饮料": {"pe": (20, 35), "pb": (3.0, 6.0), "roe_min": 15, "div_min": 1.5, "peg_max": 1.5},
    "医药生物": {"pe": (25, 40), "pb": (3.0, 6.0), "roe_min": 12, "div_min": 1.0, "peg_max": 1.5},
    "科技互联网": {"pe": (25, 50), "pb": (3.0, 8.0), "roe_min": 10, "div_min": 0.5, "peg_max": 1.5},
    "半导体": {"pe": (30, 60), "pb": (4.0, 8.0), "roe_min": 10, "div_min": 0.5, "peg_max": 1.8},
    "新能源": {"pe": (20, 40), "pb": (3.0, 6.0), "roe_min": 12, "div_min": 0.8, "peg_max": 1.5},
    "汽车": {"pe": (15, 25), "pb": (1.5, 3.0), "roe_min": 10, "div_min": 1.5, "peg_max": 1.3},
    "家电": {"pe": (15, 25), "pb": (2.0, 4.0), "roe_min": 15, "div_min": 2.5, "peg_max": 1.3},
    "地产": {"pe": (5, 10), "pb": (0.6, 1.5), "roe_min": 8, "div_min": 3.0, "peg_max": 1.0},
    "基建建筑": {"pe": (6, 10), "pb": (0.8, 1.2), "roe_min": 8, "div_min": 3.0, "peg_max": 1.0},
    "钢铁有色": {"pe": (5, 12), "pb": (0.6, 1.2), "roe_min": 8, "div_min": 2.5, "peg_max": 1.0},
    "煤炭石油": {"pe": (6, 12), "pb": (1.0, 2.0), "roe_min": 10, "div_min": 4.0, "peg_max": 1.0},
    "公用事业": {"pe": (12, 20), "pb": (1.2, 2.5), "roe_min": 8, "div_min": 3.0, "peg_max": 1.2},
    "制造业(通用)": {"pe": (15, 25), "pb": (1.5, 3.0), "roe_min": 10, "div_min": 1.5, "peg_max": 1.3},
    "消费(通用)": {"pe": (20, 30), "pb": (3.0, 5.0), "roe_min": 15, "div_min": 1.5, "peg_max": 1.3},
    "通用基准": {"pe": (15, 25), "pb": (1.5, 3.0), "roe_min": 10, "div_min": 1.5, "peg_max": 1.3},
}


# ============================================================
# 源2: 中证指数 (实时, A股) - 中证官网行业指数估值
# ============================================================
# 行业 → 中证指数代码
_CSI_INDUSTRY_CODES = {
    "中证银行": "399986",
    "中证证券公司": "399975",
    "中证白酒": "399997",
    "中证主要消费(800消费)": "000932",
    "中证医药卫生(800医药)": "000933",
    "中证金融地产(800金地)": "000934",
    "中证信息(800信息)": "000935",
    "中证可选消费(800可选)": "000931",
    "中证工业(800工业)": "000930",
    "中证能源(800能源)": "000928",
    "中证材料(800材料)": "000929",
    "中证公用(800公用)": "000936",
    "中证电信业务(800电信)": "000937",
    "中证环保产业": "000827",
    "中证新能源汽车": "399976",
    "中证人工智能主题": "930713",
    "中证全指": "000985",
}


def _cache_path(key):
    return CACHE_DIR / f"{key}.json"


def _read_cache(key):
    p = _cache_path(key)
    if not p.exists():
        return None
    try:
        with open(p) as f:
            data = json.load(f)
        ttl = CACHE_TTL_STOCK if key.startswith("stock_") else CACHE_TTL
        if time.time() - data.get("_ts", 0) > ttl:
            return None
        return data.get("value")
    except Exception:
        return None


def _write_cache(key, value):
    try:
        with open(_cache_path(key), "w") as f:
            json.dump({"_ts": time.time(), "value": value}, f, ensure_ascii=False)
    except Exception:
        pass


def _fetch_csi_value(code):
    """从中证指数取最新PE和股息率。返回 (pe, div_yield, date) 或 None。"""
    cached = _read_cache(f"csi_{code}")
    if cached:
        return tuple(cached)
    try:
        import akshare as ak
        df = ak.stock_zh_index_value_csindex(symbol=code)
        if df is None or len(df) == 0:
            return None
        latest = df.iloc[-1]
        pe = float(latest.get("市盈率1") or 0)
        div = float(latest.get("股息率1") or 0)
        date = str(latest.get("日期", ""))
        result = (pe, div, date)
        _write_cache(f"csi_{code}", list(result))
        return result
    except Exception as e:
        print(f"[csi fetch {code}] {e}")
        return None


def _csi_benchmark(industry):
    code = _CSI_INDUSTRY_CODES.get(industry)
    if not code:
        return None
    fetched = _fetch_csi_value(code)
    if not fetched:
        return None
    pe, div, date = fetched
    # PE区间: 当前值 ±25% 作为合理带
    pe_low = round(pe * 0.75, 2)
    pe_high = round(pe * 1.25, 2)
    # PB 中证此接口不直接给, 用经验比例 (PE/8 作为粗略PB上限)
    pb_high = max(round(pe / 8, 2), 1.5)
    pb_low = round(pb_high * 0.5, 2)
    return {
        "pe": (pe_low, pe_high),
        "pb": (pb_low, pb_high),
        "roe_min": 10,
        "div_min": max(round(div * 0.7, 2), 0.5),
        "peg_max": 1.3,
        "meta": {
            "source": "中证指数官网 (csindex.com.cn)",
            "asof": date,
            "live_pe": pe,
            "live_div": div,
            "note": f"行业指数当前PE={pe}, 股息率={div}%, 区间为当前值±25%",
        },
    }


# ============================================================
# 源3: Damodaran 美股行业 (NYU Stern 年度快照)
# ============================================================
_DAMODARAN_CACHE = None


def _load_damodaran():
    global _DAMODARAN_CACHE
    if _DAMODARAN_CACHE is not None:
        return _DAMODARAN_CACHE
    p = DATA_DIR / "damodaran_us.json"
    with open(p, encoding="utf-8") as f:
        _DAMODARAN_CACHE = json.load(f)
    return _DAMODARAN_CACHE


def _damodaran_benchmark(industry):
    data = _load_damodaran()
    inds = data["industries"]
    if industry not in inds:
        return None
    row = inds[industry]
    pe, pb, roe, div = row["pe"], row["pb"], row["roe"], row["div"]
    return {
        "pe": (round(pe * 0.7, 2), round(pe * 1.3, 2)),
        "pb": (round(pb * 0.7, 2), round(pb * 1.3, 2)),
        "roe_min": round(roe * 0.8, 1),
        "div_min": max(round(div * 0.7, 2), 0.0),
        "peg_max": 1.5,
        "meta": {
            "source": data["_meta"]["source"],
            "asof": data["_meta"]["asof"],
            "live_pe": pe,
            "live_pb": pb,
            "note": f"Damodaran行业均值 PE={pe} PB={pb} ROE={roe}% 股息={div}%。区间为均值±30%",
        },
    }


# ============================================================
# 统一接口
# ============================================================
SOURCES = {
    "builtin": {
        "label": "内置静态(A股,经验值)",
        "scope": "A股",
        "note": "完全离线,粗略经验,无市场实时数据",
    },
    "csindex": {
        "label": "中证指数(A股,实时)",
        "scope": "A股",
        "note": "从csindex.com.cn拉行业指数当前PE/股息率,6h缓存",
    },
    "damodaran": {
        "label": "Damodaran(美股,年度)",
        "scope": "美股",
        "note": "NYU Stern教授年度行业均值快照",
    },
}


def industries(source):
    if source == "builtin":
        return list(_BUILTIN_BENCHMARKS.keys())
    if source == "csindex":
        return list(_CSI_INDUSTRY_CODES.keys())
    if source == "damodaran":
        return list(_load_damodaran()["industries"].keys())
    return []


def get_benchmark(source, industry):
    """返回基准dict + meta,失败返回None"""
    if source == "builtin":
        bm = _BUILTIN_BENCHMARKS.get(industry)
        if not bm:
            return None
        result = dict(bm)
        result["meta"] = {"source": "内置经验值", "asof": "静态", "note": "无市场实时数据"}
        return result
    if source == "csindex":
        return _csi_benchmark(industry)
    if source == "damodaran":
        return _damodaran_benchmark(industry)
    return None


# ============================================================
# 个股查询: 自动填充 + 自身PE历史分位
# ============================================================
def _percentile(values, current):
    """计算 current 在 values 中的百分位 (0-100)"""
    if not values:
        return None
    sorted_v = sorted(values)
    rank = sum(1 for v in sorted_v if v <= current)
    return round(rank / len(sorted_v) * 100, 1)


def _em_secid(code):
    """A股代码 → 东财 secid (1.沪市 / 0.深市/北交所)"""
    if code.startswith(("6", "9")):
        return f"1.{code}"
    return f"0.{code}"


def _tencent_prefix(code):
    return "sh" if code.startswith(("6", "9")) else "sz"


def _fetch_tencent_quote(code):
    """腾讯财经qt实时报价 (稳定/低速率限制)."""
    import requests
    try:
        url = f"http://qt.gtimg.cn/q={_tencent_prefix(code)}{code}"
        r = requests.get(url, timeout=8)
        r.encoding = "gbk"
        text = r.text
        if "=" not in text or '"' not in text:
            return {}
        body = text.split('"', 1)[1].rstrip('";\n ')
        p = body.split("~")
        if len(p) < 50:
            return {}

        def f(idx):
            try:
                return float(p[idx]) if p[idx] not in ("", "-") else None
            except Exception:
                return None

        ts_str = p[30] if len(p) > 30 else ""  # YYYYMMDDHHMMSS
        asof = ""
        if len(ts_str) >= 12:
            asof = f"{ts_str[0:4]}-{ts_str[4:6]}-{ts_str[6:8]} {ts_str[8:10]}:{ts_str[10:12]}"

        return {
            "name": p[1] or None,
            "price": f(3),
            "pe_ttm": f(39),       # 腾讯[39] = 市盈率(TTM)
            "circ_cap_yi": f(44),  # 流通市值(亿)
            "mkt_cap_yi": f(45),   # 总市值(亿)
            "pb": f(46),           # 市净率
            "asof_realtime": asof,
            "_source": "tencent",
        }
    except Exception as e:
        print(f"[tencent_quote {code}] {e}")
        return {}


def _fetch_em_industry(code):
    """东财push2只取行业/板块 (不重要时可跳过)."""
    import requests
    headers = {
        "User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 Chrome/120.0",
        "Referer": "https://quote.eastmoney.com/",
    }
    try:
        r = requests.get(
            "https://push2.eastmoney.com/api/qt/stock/get",
            params={"secid": _em_secid(code), "fields": "f127,f128"},
            headers=headers,
            timeout=5,
        )
        data = (r.json().get("data") or {}) if r.ok else {}
        return {"industry_em": data.get("f127"), "area": data.get("f128")}
    except Exception:
        return {}


def _fetch_em_quote(code):
    """组合: 腾讯出价+PE/PB, 东财补行业/板块."""
    q = _fetch_tencent_quote(code)
    ind = _fetch_em_industry(code)
    if ind.get("industry_em"):
        q["industry_em"] = ind["industry_em"]
    if ind.get("area"):
        q["area"] = ind["area"]
    return q


# 东财行业 → 中证指数行业名 (前缀匹配)
_INDUSTRY_TO_CSI = {
    "银行": "中证银行",
    "证券": "中证证券公司",
    "白酒": "中证白酒",
    "食品饮料": "中证主要消费(800消费)",
    "饮料制造": "中证主要消费(800消费)",
    "医药": "中证医药卫生(800医药)",
    "化学制药": "中证医药卫生(800医药)",
    "生物制品": "中证医药卫生(800医药)",
    "医疗": "中证医药卫生(800医药)",
    "中药": "中证医药卫生(800医药)",
    "房地产": "中证金融地产(800金地)",
    "保险": "中证金融地产(800金地)",
    "多元金融": "中证金融地产(800金地)",
    "计算机": "中证信息(800信息)",
    "软件": "中证信息(800信息)",
    "通信": "中证电信业务(800电信)",
    "电子": "中证信息(800信息)",
    "半导体": "中证信息(800信息)",
    "汽车": "中证新能源汽车",
    "乘用车": "中证新能源汽车",
    "商用车": "中证新能源汽车",
    "汽车零部件": "中证新能源汽车",
    "新能源": "中证新能源汽车",
    "电池": "中证新能源汽车",
    "光伏": "中证新能源汽车",
    "煤炭": "中证能源(800能源)",
    "石油": "中证能源(800能源)",
    "钢铁": "中证材料(800材料)",
    "有色": "中证材料(800材料)",
    "化工": "中证材料(800材料)",
    "公用事业": "中证公用(800公用)",
    "电力": "中证公用(800公用)",
    "环保": "中证环保产业",
    "机械": "中证工业(800工业)",
    "建筑": "中证工业(800工业)",
    "国防": "中证工业(800工业)",
    "家电": "中证可选消费(800可选)",
    "纺织": "中证可选消费(800可选)",
    "零售": "中证可选消费(800可选)",
    "教育": "中证可选消费(800可选)",
    "传媒": "中证可选消费(800可选)",
}


def _map_industry(em_name):
    """东财行业名 → 中证行业名"""
    if not em_name:
        return None
    s = em_name.replace("Ⅱ", "").replace("Ⅰ", "").replace("Ⅲ", "").strip()
    for key, csi in _INDUSTRY_TO_CSI.items():
        if key in s:
            return csi
    return None


def fetch_target_price(code, months=12):
    """东财研报API取近N月含目标价的研报+评级分布."""
    import requests, datetime
    end = datetime.datetime.now()
    start = end - datetime.timedelta(days=30 * months)
    try:
        r = requests.get(
            "https://reportapi.eastmoney.com/report/list",
            params={
                "cb": "datatable",
                "industryCode": "*", "pageSize": 100, "pageNo": 1,
                "qType": 0, "code": code, "industry": "*",
                "rating": "*", "ratingChange": "*",
                "beginTime": start.strftime("%Y-%m-%d"),
                "endTime": end.strftime("%Y-%m-%d"),
                "orgCode": "",
            },
            timeout=10,
        )
        text = r.text
        js = json.loads(text[text.index("(") + 1: text.rindex(")")])
    except Exception as e:
        return {"error": f"研报API失败: {str(e)[:80]}"}

    rows = js.get("data") or []
    targets = []
    rating_count = {}
    for row in rows:
        rating = row.get("emRatingName") or row.get("sRatingName") or "其它"
        rating_count[rating] = rating_count.get(rating, 0) + 1
        t = row.get("indvAimPriceT")
        try:
            t_val = float(t) if t else None
        except Exception:
            t_val = None
        if t_val and t_val > 0:
            targets.append({
                "org": row.get("orgSName") or row.get("orgName"),
                "date": (row.get("publishDate") or "")[:10],
                "target": round(t_val, 2),
                "rating": rating,
                "title": row.get("title", "")[:40],
            })

    targets.sort(key=lambda x: x["date"], reverse=True)
    return {
        "total_reports": js.get("hits") or len(rows),
        "rating_count": rating_count,
        "targets": targets[:15],
        "target_avg": round(sum(t["target"] for t in targets) / len(targets), 2) if targets else None,
        "target_min": round(min(t["target"] for t in targets), 2) if targets else None,
        "target_max": round(max(t["target"] for t in targets), 2) if targets else None,
        "target_count": len(targets),
    }


_NAME_INDEX_CACHE = None


def _load_name_index():
    """A股代码↔名称索引,首次加载~10s,本地JSON缓存24h."""
    global _NAME_INDEX_CACHE
    if _NAME_INDEX_CACHE is not None:
        return _NAME_INDEX_CACHE
    idx_path = CACHE_DIR / "name_index.json"
    if idx_path.exists():
        try:
            with open(idx_path) as f:
                data = json.load(f)
            if time.time() - data.get("_ts", 0) < 86400:
                _NAME_INDEX_CACHE = data["list"]
                return _NAME_INDEX_CACHE
        except Exception:
            pass
    try:
        import akshare as ak
        df = ak.stock_info_a_code_name()
        items = [{"code": str(r["code"]).zfill(6), "name": r["name"]} for _, r in df.iterrows()]
        with open(idx_path, "w") as f:
            json.dump({"_ts": time.time(), "list": items}, f, ensure_ascii=False)
        _NAME_INDEX_CACHE = items
        return items
    except Exception as e:
        print(f"[name_index] {e}")
        return []


def search_stock(query, limit=15):
    """根据代码前缀或名称子串搜索,返回 [{code,name}, ...]"""
    q = (query or "").strip()
    if not q:
        return []
    idx = _load_name_index()
    if not idx:
        return []
    q_lower = q.lower()
    # 数字 → 代码前缀; 否则 → 名称子串
    if q.isdigit():
        hits = [x for x in idx if x["code"].startswith(q)]
    else:
        # 名称模糊 (去空格)
        q_norm = q.replace(" ", "")
        hits = [x for x in idx if q_norm in x["name"].replace(" ", "")]
        # 拼音首字母简单匹配 (可选,先跳过)
    return hits[:limit]


def resolve_code(query):
    """输入代码或名称,返回唯一代码。多义返回None。"""
    q = (query or "").strip()
    if not q:
        return None
    if q.isdigit() and len(q) == 6:
        return q
    hits = search_stock(q, limit=2)
    if len(hits) == 1:
        return hits[0]["code"]
    # 完全匹配优先
    exact = [x for x in search_stock(q, limit=50) if x["name"].replace(" ", "") == q.replace(" ", "")]
    if len(exact) == 1:
        return exact[0]["code"]
    return None


def lookup_stock(code):
    """根据股票代码查 PE/PB/PEG + 5年历史分位 + ROE + 公司名 + 行业。
    返回 dict 或 {'error': msg}
    """
    code = (code or "").strip()
    if not code.isdigit() or len(code) != 6:
        return {"error": "请输入6位A股代码,如 600036"}

    cache_key = f"stock_{code}"
    cached = _read_cache(cache_key)
    if cached:
        return cached

    result = {"code": code}

    # 公司名 + 行业板块 + 实时PE/PB (东财push2)
    quote = _fetch_em_quote(code)
    if quote.get("name"):
        result["name"] = quote["name"]
    if quote.get("industry_em"):
        result["industry_em"] = quote["industry_em"]
        mapped = _map_industry(quote["industry_em"])
        if mapped:
            result["industry_csi"] = mapped
    if quote.get("area"):
        result["area"] = quote["area"]
    if quote.get("mkt_cap_yi"):
        result["mkt_cap_yi"] = round(quote["mkt_cap_yi"], 2)
    if quote.get("price"):
        result["price"] = quote["price"]
    if quote.get("asof_realtime"):
        result["asof_realtime"] = quote["asof_realtime"]
    rt_pe = quote.get("pe_ttm")
    rt_pb = quote.get("pb")
    if rt_pe:
        result["pe"] = rt_pe
    if rt_pb:
        result["pb"] = rt_pb
    try:
        import akshare as ak
        df = ak.stock_value_em(symbol=code)
        if df is None or len(df) == 0:
            if not result.get("pe"):
                return {"error": "未查到该股票估值数据"}
        else:
            latest = df.iloc[-1]
            # 仅 PEG 取此源 (实时push2不含PEG); PE/PB 已用实时值
            peg_v = latest.get("PEG值")
            if peg_v:
                try:
                    result["peg"] = float(peg_v)
                except Exception:
                    pass
            result["asof_hist"] = str(latest.get("数据日期", ""))
            if not result.get("pe"):
                try: result["pe"] = float(latest.get("PE(TTM)"))
                except: pass
            if not result.get("pb"):
                try: result["pb"] = float(latest.get("市净率"))
                except: pass

            # 5年历史分位 (1250交易日)
            recent5y = df.tail(1250)
            pe_vals = [float(x) for x in recent5y["PE(TTM)"].dropna().tolist() if x and float(x) > 0]
            pb_vals = [float(x) for x in recent5y["市净率"].dropna().tolist() if x and float(x) > 0]

            if pe_vals and result.get("pe"):
                result["pe_percentile"] = _percentile(pe_vals, result["pe"])
                result["pe_5y_min"] = round(min(pe_vals), 2)
                result["pe_5y_max"] = round(max(pe_vals), 2)
                result["pe_5y_p30"] = round(sorted(pe_vals)[int(len(pe_vals) * 0.3)], 2)
                result["pe_5y_p70"] = round(sorted(pe_vals)[int(len(pe_vals) * 0.7)], 2)
            if pb_vals and result.get("pb"):
                result["pb_percentile"] = _percentile(pb_vals, result["pb"])
                result["pb_5y_min"] = round(min(pb_vals), 2)
                result["pb_5y_max"] = round(max(pb_vals), 2)
    except Exception as e:
        if not result.get("pe"):
            return {"error": f"估值查询失败: {str(e)[:80]}"}
        result["hist_warn"] = f"历史数据失败: {str(e)[:60]}"

    # ROE: 财务报表关键指标 (优先最新年报YYYY1231)
    try:
        import akshare as ak
        fdf = ak.stock_financial_abstract(symbol=code)
        roe_rows = fdf[fdf["指标"].str.contains("净资产收益率", na=False)]
        if len(roe_rows) > 0:
            annual_cols = [c for c in fdf.columns[2:] if str(c).endswith("1231")]
            for col in annual_cols + list(fdf.columns[2:]):
                val = roe_rows.iloc[0][col]
                if val and str(val) != "nan":
                    try:
                        result["roe"] = float(val)
                        result["roe_period"] = col
                        break
                    except Exception:
                        continue
    except Exception as e:
        result["roe_warn"] = f"ROE查询失败: {str(e)[:60]}"

    # 机构目标价
    try:
        tp = fetch_target_price(code, months=12)
        if "error" not in tp:
            result["analyst"] = tp
            if tp.get("target_avg") and result.get("price"):
                result["analyst_upside_pct"] = round(
                    (tp["target_avg"] - result["price"]) / result["price"] * 100, 1
                )
    except Exception as e:
        result["analyst_warn"] = f"研报失败: {str(e)[:60]}"

    _write_cache(cache_key, result)
    return result
