# 综艺大盘数据查询 SQL 参考

## 表选择决策树

```
用户查询需求
├── 综艺【项目粒度】——用户询问某个具体综艺节目的播放VV/UV、播放时长、热度、首播UV等日增量指标
│   └── → dws_content_zongyi_content_metrics_di（按 pid 或 content_title 过滤）
│       🚨 仅支持「单日 / 按天分布」查询，禁止跨天 SUM；若用户问累计/总计/上线以来等口径，本技能不支持，需直接告知
├── 综艺【大盘粒度】——无具体项目，查询整体播放VV/UV、播放时长、有效UV、曝光、点击、收入、正价开通等
│   └── → ads_zongyi_dapan_metrics_di
└── 表最新更新的分区进度
    └── → 查询 MAX(imp_date) FROM 对应表
```

> 🚨 **指标名强约束（最高优先级）**：输出时展示的每一个指标名，**必须**与 `references/zongyi/schema/*.md` 中对应字段的 `COMMENT` **逐字一致**，禁止改写、美化、翻译、缩写、扩写、同义替换、合并。
>
> **唯一例外——后缀省略**：展示时**必须**把 COMMENT 末尾的 `-app端`/`-pc端`/`-ott端` 去掉，其余字符原样保留。三端合计时直接用不带端后缀的名称。
>
> - ✅ `播放VV-app端` → 展示为「播放VV」（三端合计时）
> - ✅ `精选页-曝光次数-app端` → 展示为「精选页-曝光次数」
> - ❌ 不得编造 schema 中不存在的指标名
>
> 输出总结前，**必须**对每个展示的指标名回查 `references/zongyi/schema/*.md`，不一致立即改回原文。

> 🚫 **字段来源强约束**：SQL 中所有字段必须来自 `references/zongyi/schema/*.md`；**严禁**执行 `DESCRIBE` / `DESC` / `SHOW COLUMNS` / `SHOW CREATE TABLE` / 查询 `INFORMATION_SCHEMA` / `SELECT *` 等方式到 DB 反查表结构。若 schema 文件中不存在所需字段，必须直接告知用户"当前 schema 文件中不存在该字段，无法查询"。

---

## MCP 工具调用方式

### 工具信息

- **MCP Server**: `szabot_data_query_svr`
- **工具名**: `mcp_exec_sql`
- **参数**:
  - `db_name`: 数据库名称（必填）
  - `sql`: SQL 查询语句（必填）

### 调用示例

```bash
# 使用 --args 参数传递 JSON
mcporter call 'szabot_data_query_svr.mcp_exec_sql' --args '{
  "db_name": "zongyi",
  "sql": "SELECT imp_date, in_vstart_vv_app + in_vstart_vv_pc + in_vstart_vv_ott AS vv_total FROM ads_zongyi_dapan_metrics_di WHERE imp_date = (SELECT MAX(imp_date) FROM ads_zongyi_dapan_metrics_di)"
}'
```

### ⚠️ 关键要点

1. **必须使用 `--args` 参数**传递 JSON，不能用函数式调用
2. 查询综艺业务表时 `db_name` 用 `"zongyi"`

---

## 常用 SQL 模板

### 1. 查询最新单日大盘指标

```sql
SELECT
    imp_date,
    in_vstart_vv_app + in_vstart_vv_pc + in_vstart_vv_ott AS vv_total,
    in_vstart_uv_app + in_vstart_uv_pc + in_vstart_uv_ott AS uv_total,
    ROUND((in_pdtm_s_app + in_pdtm_s_pc + in_pdtm_s_ott) / 3600 / 10000, 2) AS pdtm_wanhour_total,
    ROUND((in_pdtm_s_app + in_pdtm_s_pc + in_pdtm_s_ott) / (in_vstart_uv_app + in_vstart_uv_pc + in_vstart_uv_ott) / 60, 2) AS avg_pdtm_min_per_uv
FROM ads_zongyi_dapan_metrics_di
WHERE imp_date = (SELECT MAX(imp_date) FROM ads_zongyi_dapan_metrics_di)
```

### 2. 查询指定日期区间大盘指标（趋势）

```sql
SELECT
    imp_date,
    in_vstart_vv_app + in_vstart_vv_pc + in_vstart_vv_ott AS vv_total,
    in_vstart_uv_app + in_vstart_uv_pc + in_vstart_uv_ott AS uv_total
FROM ads_zongyi_dapan_metrics_di
WHERE imp_date BETWEEN :start_date AND :end_date
ORDER BY imp_date ASC
```

### 3. 查询精选页指标

```sql
SELECT
    imp_date,
    jingxuanye_in_imp_pv_app,
    jingxuanye_in_click_pv_app,
    jingxuanye_in_click_ctr_app,
    jingxuanye_in_vstart_vv_app
FROM ads_zongyi_dapan_metrics_di
WHERE imp_date = (SELECT MAX(imp_date) FROM ads_zongyi_dapan_metrics_di)
```

### 4. 查询新热剧指标

```sql
SELECT
    imp_date,
    new_hot_in_vstart_vv_app + new_hot_in_vstart_vv_pc + new_hot_in_vstart_vv_ott AS new_hot_vv_total,
    new_hot_in_vstart_uv_app + new_hot_in_vstart_uv_pc + new_hot_in_vstart_uv_ott AS new_hot_uv_total
FROM ads_zongyi_dapan_metrics_di
WHERE imp_date = (SELECT MAX(imp_date) FROM ads_zongyi_dapan_metrics_di)
```

---

## 强约束规则

### 项目粒度查询规则（dws_content_zongyi_content_metrics_di）

- 查询综艺项目粒度指标时，**默认使用直播+点播字段**（含 `_all_` 的字段，如 `in_vstart_vv_all_*`、`in_vstart_uv_all_*`、`in_pdtm_ms_all_*`）
- 🚨 **禁止默认使用纯点播字段**（`in_vstart_vv_*`、`in_vstart_uv_*`、`in_pdtm_ms_*`），除非用户明确指定「仅点播」
- 🚨 **时长单位为毫秒（ms），不是秒**：`in_pdtm_ms_*` 字段单位是毫秒，换算万小时需 ÷3600000÷10000，换算分钟需 ÷60000，**严禁将毫秒值当秒处理**
- 最新日期子查询：`(SELECT MAX(imp_date) FROM dws_content_zongyi_content_metrics_di)`

### 三端汇总规则

- 查询三端合计时，将 `_app`、`_pc`、`_ott` 三个字段相加
- 展示时去掉端后缀，如「播放VV」= `in_vstart_vv_app + in_vstart_vv_pc + in_vstart_vv_ott`
- 若用户只问某一端，则只取对应端字段，展示时保留端后缀（去掉 `-app端` 改为「(app)」或直接说明）

### 两端汇总规则（app + pc）

- 用户明确要求「app+pc」或「移动端+pc」合计时，将 `_app`、`_pc` 两个字段相加，**不包含** `_ott`
- 展示时在指标名后注明「(app+pc)」，如「播放VV(app+pc)」= `in_vstart_vv_app + in_vstart_vv_pc`
- 时长两端合计同样遵循默认换算规则：总时长÷3600 转为**小时**，人均时长÷60 转为**分钟/人**

### 时间处理规则

- **禁止写死日期**，最新日期必须用子查询 `(SELECT MAX(imp_date) FROM ads_zongyi_dapan_metrics_di)` 动态获取
- 🚨 **年份必须以当前系统时间为准**：用户说「4月25日」「上周」等不含年份的日期时，年份取当前系统年份（如当前是2026年，则「4月25日」= `20260425`），**严禁默认使用训练数据截止年份（如2024/2025年）**
- 🚨 **输出文字年份必须与 SQL 一致**：在回答中描述日期时（如「2026年5月6日（昨天）」），必须与 SQL 实际查询的 `imp_date` 年份完全一致，**严禁 SQL 查的是2026年但输出文字写成2025年**
- 区间查询：`imp_date BETWEEN :start_date AND :end_date`
- 多日趋势：`ORDER BY imp_date ASC`

### 时长单位换算

- `in_pdtm_s_*` 字段单位为**秒**
- **默认换算规则**（用户未指定单位时强制执行）：
  - **总播放时长**（三端合计或单端汇总）：÷3600÷10000 转换为**万小时**，结果保留两位小数，展示时注明「万小时」
  - **人均播放时长**（总时长÷UV）：÷60 转换为**分钟**，结果保留两位小数，展示时注明「分钟/人」
- 用户明确指定单位时，以用户要求为准
- 🚨 **换算必须在 SQL 中完成**：所有单位换算（÷3600、÷10000、÷60 等）必须写入 SQL 的 `SELECT` 子句中（使用 `ROUND(..., 2)`），**严禁**大模型对查询结果自行进行二次计算或换算，直接展示 SQL 返回值即可

### 无项目维度

- 本表为**大盘表**，无 `pid`/`cid` 字段，**无需前置查询项目ID**，直接按日期查询

---

## 输出前自检清单

在生成最终回答前，逐条核对：

- [ ] 每个展示的指标名是否与 `references/zongyi/schema/ads_zongyi_dapan_metrics_di.md` 中的 `COMMENT` 逐字一致（允许去掉 `-app端`/`-pc端`/`-ott端` 后缀）？
- [ ] SQL 中所有字段是否均来自 schema 文件，无编造字段？
- [ ] 是否未使用 `DESCRIBE`/`SHOW COLUMNS`/`INFORMATION_SCHEMA` 等反查表结构的语句？
- [ ] 时间条件是否使用动态子查询，未写死日期？
- [ ] 三端合计是否正确相加了 `_app`、`_pc`、`_ott` 三个字段？
- [ ] 两端合计（app+pc）是否只相加了 `_app`、`_pc`，未混入 `_ott`，且展示名已注明「(app+pc)」？
- [ ] 时长换算是否已在 SQL 的 SELECT 子句中完成（使用 ROUND），未对查询结果进行二次计算？
- [ ] 时长字段是否已注明单位（万小时/分钟）？
- [ ] 输出结尾是否附带执行时间及数据来源（`ads_zongyi_dapan_metrics_di` 或 `dws_content_zongyi_content_metrics_di`）？
- [ ] 🚨 输出文字中提到的日期（如「2026年5月6日」）是否与 SQL 实际查询的 `imp_date` 完全一致？**严禁将 SQL 查询的年份（如2026）在输出文字中写成其他年份（如2025）**
- [ ] 项目粒度查询是否已通过 `pid`（项目ID）或 `content_title`（项目名称）过滤，未遗漏项目条件？
- [ ] 项目粒度查询是否使用了直播+点播字段（`_all_` 字段），而非纯点播字段？
