from flask import Flask, render_template, request, jsonify
import data_sources as ds

app = Flask(__name__)


def judge_band(val, low, high):
    if val is None:
        return ("未填写", None)
    if val <= 0:
        return ("数据异常(≤0,可能亏损)", None)
    if val < low:
        return (f"低估 (< {low})", -1)
    if val > high:
        return (f"高估 (> {high})", 1)
    return (f"合理 ({low}~{high})", 0)


def judge_min(val, minimum, label_low="偏弱", label_ok="达标"):
    if val is None:
        return ("未填写", None)
    if val < minimum:
        return (f"{label_low} (< {minimum})", 1)
    return (f"{label_ok} (≥ {minimum})", -1)


def judge_peg(val, peg_max):
    if val is None:
        return ("未填写", None)
    if val <= 0:
        return ("数据异常(≤0,可能负增长)", None)
    if val < 1.0:
        return ("低估 (PEG < 1)", -1)
    if val > peg_max:
        return (f"高估 (PEG > {peg_max})", 1)
    return (f"合理 (1 ~ {peg_max})", 0)


def composite(score_sum, n_signals):
    if n_signals == 0:
        return ("无有效输入", "secondary")
    ratio = score_sum / n_signals
    if ratio <= -0.4:
        return ("综合判断: 低估", "success")
    if ratio >= 0.4:
        return ("综合判断: 高估", "danger")
    return ("综合判断: 合理", "warning")


@app.route("/industries")
def industries_api():
    source = request.args.get("source", "builtin")
    return jsonify({"industries": ds.industries(source)})


@app.route("/lookup")
def lookup_api():
    q = request.args.get("code", "")
    code = ds.resolve_code(q)
    if not code:
        hits = ds.search_stock(q, limit=10)
        if not hits:
            return jsonify({"error": f"未找到匹配 '{q}',请输入完整代码或精确名称"})
        return jsonify({"error": f"'{q}' 匹配多个,请选择", "suggestions": hits})
    return jsonify(ds.lookup_stock(code))


@app.route("/search")
def search_api():
    q = request.args.get("q", "")
    return jsonify({"results": ds.search_stock(q, limit=15)})


@app.route("/", methods=["GET", "POST"])
def index():
    result = None
    error = None
    form_data = {
        "source": request.values.get("source", "builtin"),
        "industry": request.values.get("industry", ""),
        "code": "",
        "pe": "", "pb": "", "roe": "", "div": "", "peg": "", "name": "",
    }
    percentile_info = None

    if request.method == "POST":
        form_data["name"] = request.form.get("name", "").strip()
        form_data["code"] = request.form.get("code", "").strip()
        for k in ("pe", "pb", "roe", "div", "peg"):
            form_data[k] = request.form.get(k, "").strip()

        # 若有股票代码/名称且勾选,解析后取分位+研报
        analyst_info = None
        if form_data["code"] and request.form.get("use_self_history") == "1":
            resolved = ds.resolve_code(form_data["code"])
            if resolved:
                form_data["code"] = resolved
                stock_info = ds.lookup_stock(resolved)
                if "error" not in stock_info:
                    percentile_info = stock_info
                    if stock_info.get("analyst"):
                        analyst_info = {
                            **stock_info["analyst"],
                            "price": stock_info.get("price"),
                            "upside_pct": stock_info.get("analyst_upside_pct"),
                        }

        bm = ds.get_benchmark(form_data["source"], form_data["industry"])
        if not bm:
            error = f"无法获取数据源 [{form_data['source']}] 行业 [{form_data['industry']}] 的基准。可能是网络问题或行业未配置。"
        else:
            def parse(x):
                try:
                    return float(x) if x != "" else None
                except ValueError:
                    return None

            pe = parse(form_data["pe"])
            pb = parse(form_data["pb"])
            roe = parse(form_data["roe"])
            div = parse(form_data["div"])
            peg = parse(form_data["peg"])

            rows = []
            score_sum = 0
            n = 0

            verdict, s = judge_band(pe, *bm["pe"])
            rows.append(("PE", pe, f"{bm['pe'][0]} ~ {bm['pe'][1]}", verdict, s))
            if s is not None: score_sum += s; n += 1

            verdict, s = judge_band(pb, *bm["pb"])
            rows.append(("PB", pb, f"{bm['pb'][0]} ~ {bm['pb'][1]}", verdict, s))
            if s is not None: score_sum += s; n += 1

            verdict, s = judge_min(roe, bm["roe_min"])
            rows.append(("ROE %", roe, f"≥ {bm['roe_min']}%", verdict, s))
            if s is not None: score_sum += s; n += 1

            verdict, s = judge_min(div, bm["div_min"])
            rows.append(("股息率 %", div, f"≥ {bm['div_min']}%", verdict, s))
            if s is not None: score_sum += s; n += 1

            verdict, s = judge_peg(peg, bm["peg_max"])
            rows.append(("PEG", peg, f"1 ~ {bm['peg_max']}", verdict, s))
            if s is not None: score_sum += s; n += 1

            summary, color = composite(score_sum, n)

            notes = []
            if pb is not None and roe is not None and pb > 0 and roe > 0:
                if pb > 3 and roe < 15:
                    notes.append("高PB+低ROE: 资本回报无法支撑高PB,警惕高估")
                if pb < 1 and roe > 10:
                    notes.append("低PB+高ROE: 典型价值低估信号")
            if pe is not None and peg is not None and pe > 30 and peg > 1.5:
                notes.append("高PE+高PEG: 成长性不足以消化估值")

            # 自身历史分位评分(若有)
            self_history = None
            if percentile_info and pe and "pe_percentile" in percentile_info:
                p = percentile_info["pe_percentile"]
                if p < 30:
                    label, sh_color = f"PE处于近5年 {p}% 分位 → 自身历史低估", "success"
                elif p > 70:
                    label, sh_color = f"PE处于近5年 {p}% 分位 → 自身历史高估", "danger"
                else:
                    label, sh_color = f"PE处于近5年 {p}% 分位 → 自身历史中性", "warning"
                self_history = {
                    "label": label,
                    "color": sh_color,
                    "pe_range": f"{percentile_info.get('pe_5y_min')} ~ {percentile_info.get('pe_5y_max')}",
                    "pe_band": f"P30={percentile_info.get('pe_5y_p30')}, P70={percentile_info.get('pe_5y_p70')}",
                    "pb_percentile": percentile_info.get("pb_percentile"),
                    "asof": percentile_info.get("asof"),
                }

            result = {
                "rows": rows,
                "summary": summary,
                "color": color,
                "source": form_data["source"],
                "industry": form_data["industry"],
                "name": form_data["name"] or "目标公司",
                "notes": notes,
                "meta": bm.get("meta", {}),
                "self_history": self_history,
                "analyst": analyst_info,
            }

    return render_template(
        "index.html",
        sources=ds.SOURCES,
        initial_industries=ds.industries(form_data["source"]),
        form=form_data,
        result=result,
        error=error,
    )


if __name__ == "__main__":
    app.run(host="127.0.0.1", port=5000, debug=True)
