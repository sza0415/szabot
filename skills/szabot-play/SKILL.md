---
name: szabot-play
namespace: szbot
trust-level: builtin
category: media
description: "影库影视播放看片。仅限用户需要观看影库项目相关的影视视频内容（样片/成片/素材）时使用，通过项目ID获取带权限的播放链接。触发词：看片、看样片、播放成片、看素材、项目视频、看视频。⛔ 不适用：非影视内容的视频播放、通用视频链接跳转、与影库项目无关的视频请求"
---

# 影库视频播放 Skill

根据影库项目 ID 获取带权限的视频播放跳转链接。

## 执行总则（强约束）

1. **本 Skill 使用 `szabot_play` MCP Server**，包含 `szabot_auth_play` 工具
2. **上下文隔离** — 仅遵循本文件和 `references/` 下的规则
3. **缺少 `pid` / `name` 时** — 加载 `szabot-copilot` Skill 查询，仅提取 `pid` 和 `name`，获取后立即回到本 Skill
4. **跳转链接必须使用 `szabot_auth_play` 返回的原始 `url`，不得自行拼接**
5. **禁止调用 `szabot_medium` 工具**

## 工具清单

| MCP Server | 工具 | 用途 |
|-----------|------|------|
| `szabot_play` | `szabot_auth_play` | 获取视频跳转链接 |

> 详细参数、枚举值、返回结构见 `references/play_workflow.md`，**禁止凭记忆构造参数**。

## 适用 / 不适用

- ✅ 观看影库项目的影视视频（样片/成片/素材）
- ✅ 获取带权限的视频跳转链接
- ❌ 非影视内容的视频播放（教程、会议录屏等）
- ❌ 与影库项目无关的通用视频链接
- ❌ 查询项目信息（走 szabot-copilot）

## 执行流程

```
匹配到"看视频/播放"意图
  ↓
1. 读取 references/play_workflow.md（必须，不可跳过）
  ↓
2. 按 play_workflow.md 中的步骤执行
```

> ⚠️ SKILL.md 仅提供路由和约束，**具体执行步骤、参数定义全部在 `references/play_workflow.md` 中**。
