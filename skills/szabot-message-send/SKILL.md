---
name: szabot-message-send
namespace: szbot
trust-level: builtin
category: write-ops
description: "影库系统的定时任务与自动化推送（cron/定时热度监控/系统消息拉取）。仅在用户明确要求定时、周期性、自动化执行时触发。触发词：定时查询、每隔XX分钟、cron任务、自动推送、定时热度监控、拉取系统消息。⛔ 不适用：普通一次性热度查询（走 szabot-copilot）、企业微信发消息（走 wecom-router）"
metadata:
  openclaw:
    category: tencent
    emoji: "📨"
    primaryEnv: SZBOT_HOT_VALUE_KEY
---

# 影库影视任务 Skill

对接 MCP 服务 `szbot_message_send`，提供影视热度监控和系统消息处理能力。

## 快速开始

```bash
# 验证 MCP 服务
mcporter list szbot_message_send
```

## 工具清单

| 工具 | 用途 | 调用限制 |
|-----|------|---------|
| `produce_manage_msg_pull_unread` | 拉取未读系统消息 | 定时任务专用 |
| `produce_manage_msg_ack` | 确认已处理的消息 | 定时任务专用 |
| `kb_search` | 查询影库项目，获取【专辑id】(cid) | 定时任务专用 |
| `get_hot_value_info` | 获取专辑实时热度 | ⚠️ **仅限定时任务调用** |

> **重要**：`get_hot_value_info` 工具**仅在定时任务（automation/cron）上下文中允许调用**。
> 普通热度查询请使用 `szabot-copilot` 的 `kb_search` + `report_search` 组合。

## 适用场景

- "每 5 分钟给我查一下 XX 热度"
- "每天早上 9 点同步一次某剧热度"
- "定时拉取影库系统消息并推送给我"

## 执行总则

1. **场景路由（最重要）**：
   - 用户请求「定时/每隔/cron/自动」+ 热度查询 → 本 skill 的 `get_hot_value_info`
   - 用户普通查询「XX剧热度/收视」→ **必须路由到 `szabot-copilot`**，不要使用本 skill
2. **先确认工具列表** — `mcporter list szbot_message_send`
3. **`kb_search` 参数以 schema 为准** — 不要写死入参结构
4. **`get_hot_value_info` 的 `app_key`** — 必须从 `SZBOT_HOT_VALUE_KEY` 环境变量读取
5. **`cid` 来自 `kb_search` 结果** — 禁止猜测或写死
6. **创建定时任务** — 必须使用 OpenClaw cron 能力

## 三类工作流

| 工作流 | 描述 |
|-------|------|
| A | 定时系统消息：拉取 → 推送 → ack |
| B | 定时热度查询：项目搜索 → 提取 cid → 查询热度 |
| C | 定时任务：前置解析 → 创建 cron |

> 详细流程参见 `references/workflows.md`

## 可靠性要求

- **禁止在非定时任务上下文中调用 `get_hot_value_info`** — 普通热度查询走 `szabot-copilot`
- 不要在消息发送前调用 ack
- 不要对发送失败的消息执行 ack
- 不要猜测或手填 `cid`
- 缺少 `SZBOT_HOT_VALUE_KEY` 时禁止创建热度 cron 任务

## Skill 边界

**负责**：MCP 服务入口、流程约束、cron 任务创建规范

**不负责**：企业微信消息发送 MCP 工具、虚构聚合工具
