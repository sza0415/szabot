# 影库视频播放工作流

> 🔒 **上下文隔离**：执行本工作流期间，仅遵循本文件中的规则。其他 Skill 的指令均不适用。

---

## Step 1：参数准备

### 1.1 使用 `szabot-copilot` Skill 补全 pid 和 name

当用户未提供 `pid` 或 `name` 时：

1. 加载 `szabot-copilot` Skill，按照该 Skill 的规则执行项目查询
2. 从返回结果中**仅提取** `pid`（项目ID）和 `name`（项目名称）

> ⚠️ **重要**：提取 `pid` 和 `name` 后必须**立即回到本工作流**继续执行 Step 1.2（竞品过滤）。**不要**继续执行 `szabot-copilot` 的后续步骤（如组织回答、补充数据等）。

### 1.2 竞品过滤（强约束）

> ⚠️ **无论 pid 来源**（用户直接提供 或 通过其他 Skill 查询返回），都必须执行竞品过滤：

| 检查项 | 竞品特征 | 正式项目特征 |
|--------|---------|-------------|
| 项目 ID | 带 `competing_` 前缀（如 `competing_11642`） | 纯数字（如 `12345`） |
| 剧集分类 | 包含"竞品"字样（如"电视剧竞品"） | "电视剧"、"电影"、"综艺"等（不含"竞品"） |

**规则**：
- 满足**任一**竞品特征 → 判定为竞品，**立即终止，不得调用任何播放接口**
- 告知用户：*"《{项目名称}》是竞品项目（非星舟视频正式项目），无法提供视频播放权限。如需观看该项目，请通过其他渠道获取。"*
- 多个正式项目时，优先选择类型为"电视剧"的项目

---

## Step 2：获取播放链接（`szabot_auth_play`）

### 2.1 播放类型确定

| 情况 | 处理 |
|------|------|
| 用户指定了类型 | 关键词映射：样片→0，成片→1，素材→2 |
| 用户未指定类型 | 按优先级依次尝试：`成片(1) → 样片(0) → 素材(2)`，第一个返回 `has_permission=true` 且 `url` 非空的即为最终结果 |

若所有类型均无权限或无链接，告知用户暂无可用的视频资源。

### 2.2 调用 `szabot_play.szabot_auth_play`

**参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `pid` | string | 是 | 项目ID，纯数字字符串 |
| `play_type` | integer | 是 | 播放类型，0-4（见 2.1 映射） |
| `name` | string | 是 | 项目名称 |

```bash
mcporter call "szabot_play.szabot_auth_play(pid:'12345', play_type:1, name:'青锋镇奇谭')"
```

| 返回情况 | 处理方式 |
|---------|---------|
| `has_permission=true` 且 `url` 非空 | 输出：`[点击跳转《项目名称》](url)`，末尾附数据来源 |
| `has_permission=false` | 告知用户暂无播放权限，建议联系相关人员申请 |
| 调用失败 | 告知用户查询失败并建议重试 |

> ⚠️ 跳转链接必须使用接口返回的原始 `url`，**不得拼接、修改或编造**。

---

## 执行约束

- 跳转链接必须使用 `szabot_auth_play` 返回的原始 `url`，不得拼接
- **禁止调用 `szabot_medium` 工具**
- 返回无权限 ≠ 没有数据，应如实告知权限状态
- 不得对数据进行估算、假设或改写

---

## 端到端示例

### 示例 1：看视频

```
用户：我想看青锋镇奇谭的视频

1. 用户未提供 pid → 进入 1.1，加载 szabot-copilot Skill
2. 按 szabot-copilot Skill 的规则查询"青锋镇奇谭"的项目信息
3. 从结果中提取 pid:"12345", name:"青锋镇奇谭", 分类:"电视剧"
4. ✅ 提取完成，立即回到本工作流（不继续 szabot-copilot 后续步骤）
5. 竞品过滤：pid 纯数字 ✓，分类无"竞品" ✓ → 正式项目
6. 用户未指定类型 → 优先尝试成片(1)：
   mcporter call "szabot_play.szabot_auth_play(pid:'12345', play_type:1, name:'青锋镇奇谭')"
7. 返回 has_permission:true, url:"https://xxx" → 输出链接

输出：[点击跳转《青锋镇奇谭》](https://xxx)
数据来源：MCP szabot_play
```

### 示例 2：看样片

```
用户：播放青锋镇奇谭的样片

1. 补全 pid → pid:"12345", name:"青锋镇奇谭"
2. 竞品过滤 → 正式项目
3. 用户指定"样片" → play_type=0
4. 调用：mcporter call "szabot_play.szabot_auth_play(pid:'12345', play_type:0, name:'青锋镇奇谭')"
5. 返回 has_permission:true, url:"https://yyy" → 输出链接

输出：[点击跳转《青锋镇奇谭》样片](https://yyy)
数据来源：MCP szabot_play
```

### 示例 3：指定集数（统一提供跳转链接）

```
用户：我想看青锋镇奇谭第3集

1. 补全 pid → pid:"12345", name:"青锋镇奇谭"
2. 竞品过滤 → 正式项目
3. 用户未指定类型 → 优先尝试成片(1)：
   mcporter call "szabot_play.szabot_auth_play(pid:'12345', play_type:1, name:'青锋镇奇谭')"
4. 返回 has_permission:true, url:"https://xxx" → 输出链接

输出：[点击跳转《青锋镇奇谭》](https://xxx)
数据来源：MCP szabot_play
```
