#!/usr/bin/env python3
"""
《97.4》剧本分析助手 (ScreenplayAnalyzer)
========================================
针对工作区内剧本 `97.4.md` 做结构化统计分析。

能输出：
  1. 剧本基本信息（总字数、场景数、角色数）
  2. 每场景台词密度分布（可视化图表）
  3. 每个角色的台词统计（条数、字数、占比）
  4. 台词量 TOP 榜
  5. 关键角色出场场景追踪

用法：
  python screenplay_analyzer.py [剧本路径]

依赖：仅标准库，无需安装任何第三方包。
"""

import re
import sys
from collections import defaultdict

SCREENPLAY = "97.4.md"

# 台词行格式： 角色名（可选动作）> 台词内容
# 例： 陆一舟（没有抬头）
#      > 深夜没人机...
# 这里实际是：角色行 之后紧跟着以 > 开头的台词行
LINE_RE = re.compile(r"^> (.+)$")

# 场景标题，例如： "### 第3场 · 录音室"
SCENE_RE = re.compile(r"^#{2,3}\s+.*(?:第\d+场|第[一二三四五六七八九十]+场).*$")


def extract_scenes(text):
    """按场景标题切分文本，返回 [(场景名, 内容段)]"""
    lines = text.splitlines()
    scenes = []
    current_name = "前置部分"
    current_body = []
    for line in lines:
        if SCENE_RE.search(line):
            if current_body or current_name != "前置部分":
                scenes.append((current_name, "\n".join(current_body)))
            current_name = line.strip().lstrip("# ").strip()
            current_body = []
        else:
            current_body.append(line)
    if current_body:
        scenes.append((current_name, "\n".join(current_body)))
    return scenes


def extract_dialogue(text):
    """
    解析台词，返回 [(角色名(含动作), 台词内容)]。
    处理规律：角色行（可带括号动作）下一行若是 '> xxx' 即为该角色的台词。
    角色行背景括号动作被剥离。
    """
    lines = text.splitlines()
    dialogue = []
    current_speaker = None

    for i, line in enumerate(lines):
        # 台词行：以 > 开头
        if line.startswith(">"):
            content = line[1:].strip()
            speaker = current_speaker if current_speaker else "未知"
            dialogue.append((speaker, content))
        else:
            stripped = line.strip()
            # 判断是否是"角色名（动作）"这一行 —— 特征：后一行是 '>'
            nxt = lines[i + 1] if i + 1 < len(lines) else ""
            if nxt.startswith(">") and stripped and not stripped.startswith(">"):
                current_speaker = stripped
            # 若不是带动作的角色行格式，更新为当前推断说话人
            # 简单起见：若该行为纯角色名（无括号/无标点符号），认为是换人
            elif stripped and not stripped.startswith(("（", "(", "#", "-", "*")):
                # 排除场景/内景/外景/舞台指示开头
                if stripped.startswith(("内景", "外景", "场景", "转场", "空镜", "同期声", "画外音")):
                    continue
                # 纯角色名形式：如 "顾念", "林副台长"
                if not any(ch in stripped for ch in "，。：；《》？'\"（）()、"):
                    current_speaker = stripped
    return dialogue


def analyze(path):
    with open(path, "r", encoding="utf-8") as f:
        text = f.read()

    print("=" * 56)
    print(f"🎬  《97.4》剧本分析报告")
    print("=" * 56)

    # ---- 基本信息 ----
    total_chars = len(re.sub(r"\s", "", text))  # 去掉所有空白后的总字数
    print("\n📄 基本信息")
    print(f"  · 全文含空白总字数  : {len(text):,}")
    print(f"  · 去空白纯文字字数  : {total_chars:,}")
    print(f"  · 总行数            : {len(text.splitlines()):,}")

    # ---- 场景 ----
    scenes = extract_scenes(text)
    print(f"\n🏞  场景统计（共 {len(scenes)} 个场景段）")
    scene_chars = []
    for name, body in scenes:
        c = len(re.sub(r"\s", "", body))
        scene_chars.append((name, c))

    # 生成条形图
    max_c = max(c for _, c in scene_chars) if scene_chars else 1
    print("\n  每场景文字量分布（单位：千字，每格=200字）")
    for name, c in scene_chars:
        bar = "█" * (c // 200)
        print(f"  {name[:20]:<22}{bar:<30} {c:,}字")

    # ---- 角色台词统计 ----
    dialogue = extract_dialogue(text)
    total_lines = len(dialogue)
    print(f"\n💬 台词统计（共 {total_lines} 条台词）")

    # 角色别名映射：把同一个人物的不同称呼合并（如 女声(顾念) -> 顾念）
    ALIAS = {
        "女声（顾念）": "顾念",
        "女声(顾念)": "顾念",
        "女声": "顾念",
        "陆一舟（低声，带着专业口吻）": "陆一舟",
    }
    # 非角色词（舞台指示/场景描述等误识别的要剔除）
    NOT_A_SPEAKER = {"未知", "前置部分", "旁白"}

    def norm_speaker(s):
        s = s.strip()
        # 先查别名映射
        if s in ALIAS:
            return ALIAS[s]
        # 以"女声"开头的都视为顾念
        if s.startswith("女声"):
            return "顾念"
        # 去括号动作： 名字（动作）
        m = re.match(r"^(.+?)[（(]", s)
        if m:
            s = m.group(1).strip()
        return s

    per_role = defaultdict(lambda: {"条数": 0, "字数": 0})
    for speaker, content in dialogue:
        name = norm_speaker(speaker)
        if name in NOT_A_SPEAKER:
            continue
        per_role[name]["条数"] += 1
        per_role[name]["字数"] += len(re.sub(r"\s", "", content))

    # 排序并输出
    ranked = sorted(per_role.items(),
                    key=lambda kv: kv[1]["字数"], reverse=True)
    print(f"\n  {'角色':<14}{'台词条数':<10}{'字数':<10}{'字数占比'}")
    for name, stat in ranked:
        pct = stat["字数"] / total_chars * 100
        print(f"  {name:<14}{stat['条数']:<10}{stat['字数']:<10}{pct:>6.1f}%")

    # TOP3
    print("\n🏆 台词量 TOP 角色")
    for i, (name, stat) in enumerate(ranked[:3], 1):
        print(f"  #{i}  {name}  —— {stat['字数']:,}字 / {stat['条数']}条")

    return ranked


if __name__ == "__main__":
    path = sys.argv[1] if len(sys.argv) > 1 else SCREENPLAY
    analyze(path)
    print("\n" + "=" * 56)
    print("✅ 分析完成")
