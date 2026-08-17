# 工作流详细说明

## 工作流 A：系统消息处理

### 完整流程

1. **拉取未读消息**
   - 调用 `produce_manage_msg_pull_unread`
   - `cates` 默认传 `["2"]`，即默认拉取"电视剧"消息
   - `limit` 为可选参数；不传或超限时由后端使用默认值

2. **发送消息**
   - 读取返回消息后，逐条调用 OpenClaw 平台内置消息能力推送
   - 推送内容应至少包含标题、正文、链接等关键信息（以返回数据为准）

3. **发送成功后 ack**
   - 仅当某条消息推送成功后，才调用 `produce_manage_msg_ack`
   - 参数格式：`{"msg_ids":["消息ID"]}`
   - `msg_ids` 必须与本次成功推送的消息一一对应

4. **结束条件**
   - 无未读消息：正常结束
   - 存在发送失败：记录失败项，不对失败消息 ack
   - ack 失败：记录失败消息 ID，等待后续排查或补偿

---

## 工作流 B：项目搜索与热度查询

### 完整流程

1. **先搜索项目**
   - 根据用户提供的影视名称、别名、关键词调用 `kb_search`
   - `kb_search` 的具体请求参数，必须通过 `mcporter list szbot_message_send` 查 schema 后再构造

2. **确认项目并提取 `cid`**
   - 从返回结果中找到目标项目
   - 结果字段中必须包含【专辑id】；将该字段作为 `cid`
   - 如果出现多个同名/相近项目且无法唯一确认，应先让用户确认目标项目，再继续

3. **查询热度**
   - 从环境变量读取 `SZBOT_HOT_VALUE_KEY`，作为 `app_key`
   - 调用 `get_hot_value_info`
   - 入参结构：`{"app_key":"<env:SZBOT_HOT_VALUE_KEY>","cid":"<专辑id>"}`

4. **输出结果**
   - 汇报时至少包含：项目名、`cid`、热度结果、查询时间
   - 若后端返回了更多关键字段，可一并摘要，但不要虚构不存在的字段

---

## 工作流 C：创建影视相关 cron 任务

### 前置要求

当用户要求"每隔多久查一次热度""每天/每周定时同步影视信息"时执行：

1. **先完成一次前置解析，不要直接盲建任务**
   - 先调用 `kb_search` 确认目标项目
   - 先拿到唯一、可信的 `cid`
   - 若 `SZBOT_HOT_VALUE_KEY` 缺失，直接报错并停止，不要创建失败任务

2. **创建 cron 任务**
   - 阅读 OpenClaw 创建 cron 定时任务能力的手册，然后用 `openclaw cron add` 生成任务
   - 消息需要投递到当前的对话 id，带上请求所需的相关参数
   - schedule 必须为 cron 模式
   - sessionTarget 为 isolated 模式，sessionKey 为当前对话 ID（`agent:main:` 开头的 ID）
   - 定时任务需要说明用户的需求如【每天查一次热度】
   - 检查任务必须为运行状态（enabled 为 true）
   - 检查 `~/.openclaw/cron/jobs.json`，保证 cron 任务正常运行
   - 任务需要优先使用 szabot-message-send skill 能力

---

## 返回数据使用约定

### `produce_manage_msg_pull_unread`

若返回中包含以下常见字段，可按其语义使用：
- `messages`：消息对象列表
- `has_more`：是否还有更多消息

### `kb_search`

- 必须能拿到用于后续热度查询的【专辑id】
- 若字段命名与预期不同，以 MCP 实际返回 schema 和数据为准

### `get_hot_value_info`

- 至少依赖 `app_key` 与 `cid`
- 若返回中包含热度数值、更新时间、状态字段等，可按实际字段展示
- 以 MCP 实际返回 schema 和数据为准，不要在 Skill 中写死不存在的字段

---

## 调用示例

### 拉取未读消息

```bash
mcporter call "szbot_message_send" "produce_manage_msg_pull_unread" --args '{"cates": ["2"], "limit": 10}'
```

### 查询热度前先检查 schema

```bash
mcporter list szbot_message_send --schema
```

### 热度查询

```bash
mcporter call "szbot_message_send" "get_hot_value_info" --args '{"app_key": "<SZBOT_HOT_VALUE_KEY>", "cid": "<专辑id>"}'
```

### ack 示例

```bash
mcporter call "szbot_message_send" "produce_manage_msg_ack" --args '{"msg_ids": ["21"]}'
```
