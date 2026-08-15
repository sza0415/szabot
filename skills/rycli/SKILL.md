---
name: rycli
description: "影视综专家模式模拟 CLI（营销/综艺模式/综艺营销/小说/剧本）。当用户表达 @expert:market / variety / variety-marketing / novel / script 等专家意图，或提到营销方案、综艺模式拆解、日周报、剧本/小说分析时使用。"
metadata: {"requires":{"bins":["bash"]}}
---

# 如影 CLI 模拟版 (rycli)

面向 szabot 的影视制片专家模式**离线模拟**技能。原版依赖真实 `rycli` / `ryclaw`
联网服务，在 szabot 环境不可用；本版用 workspace 内自带的纯 bash 脚本
`skills/rycli/bin/sage-sim` 模拟同样的输入输出契约，做到零外部依赖、可离线跑通。

## 命令速查

```bash
# 通过 bash 工具执行模拟器（无需可执行权限位，直接用 bash 解释器调用）
bash skills/rycli/bin/sage-sim ask --agent-id=<agent_id> --text="<text>"
```

- 无需认证：本模拟版**不需要** `auth login`，去掉了 `ryclaw get-token` 等外部依赖。
- 工作目录：用 szabot 的 `bash` 工具执行时，工作目录即工作区根，故路径写 `skills/rycli/bin/sage-sim` 即可。

## 深度指南 (references/)

| 文档 | 内容 |
|------|------|
| [`references/sage.md`](references/sage.md) | sage 问答模块：专家路由表、调用示例、结果处理规则 |

## 重要规则

1. **强制读取参考文档**：激活本 Skill 后，**必须先读取 `references/sage.md`**，
   严格按其中的子 Agent 路由表、参数格式和结果处理规则执行，禁止凭记忆拼接命令参数。
2. **异常处理**：若脚本以非 0 退出（如缺少 `--agent-id` / `--text`），
   按错误信息补全参数后重试；其余异常则终止并给出友好提示。
