# 环境依赖安装（完整指南）

本文档是 `szabot-file-editor` 环境准备的**详细兜底指南**。

> 💡 **优先使用 SKILL.md 中的方式 A（离线 wheel 包安装）**，只有在方式 A 失败后才需要阅读本文档尝试其他方式。

## 沙箱/容器环境特别说明

- `pip`、`pip3` 命令通常**不存在**（报 `command not found`）
- `ensurepip` 模块可能**被精简移除**（报 `No module named ensurepip`）
- Debian 12+ 系统启用了 **PEP 668** 保护，直接 `pip install` 会报 `externally-managed-environment` 错误，**必须加 `--break-system-packages` 参数**
- 内网或国内服务器从 PyPI 官方源下载依赖时**经常超时**，**必须指定国内镜像源**
- 部分容器环境 `curl` **外网 DNS 不通**（报 `Could not resolve host`），但 **Python `urllib` 通常可以正常访问外网**，此时应优先使用方式 B 的 Python urllib 下载方案，如果仍不通再使用 apt 换源方案（方式 E）

---

## Step 1：检测 Python 环境

```bash
python3 --version
```

如果 `python3` 不可用，尝试 `python --version`。如果都不可用，需要先安装 Python。

---

## Step 2：引导安装 pip（方式 A 失败后的兜底方案）

按以下优先级依次尝试，**成功一种即可进入 Step 3**：

### 方式 B：get-pip.py（⭐ 网络可用时的首选兜底）

先下载 `get-pip.py`，再执行安装。下载方式按优先级尝试：

```bash
# 优先用 curl 下载
curl -sS https://bootstrap.pypa.io/get-pip.py -o /tmp/get-pip.py
```

如果 `curl` 报 `Could not resolve host`（容器中常见的 DNS 问题），**改用 Python urllib 下载**：

```bash
# curl 不可用时，用 Python 自带的 urllib 下载（⭐ 容器环境推荐）
python3 -c "import urllib.request; urllib.request.urlretrieve('https://bootstrap.pypa.io/get-pip.py', '/tmp/get-pip.py'); print('get-pip.py downloaded')"
```

下载成功后，执行安装：

```bash
python3 /tmp/get-pip.py --break-system-packages -i https://mirrors.cloud.tencent.com/pypi/simple --trusted-host mirrors.cloud.tencent.com
```

> ⚠️ **必须先下载到文件再执行**（`-o /tmp/get-pip.py`），不要用管道 `| python3 -`，否则如果 URL 返回 HTML 错误页面会导致 `SyntaxError`。
> ⚠️ **必须加 `--break-system-packages`**，否则 Debian 12+ 系统会拒绝安装。
> 💡 **容器环境中 `curl` 经常因 DNS 不通而失败，但 Python `urllib` 通常可以正常访问外网**，因此 Python 下载方式是更可靠的备选。

### 方式 C：ensurepip（简单但容器中通常不可用）

```bash
python3 -m ensurepip --default-pip
```

> 如果报 `No module named ensurepip`，说明系统精简了该模块，跳回方式 B。
> 如果报 `externally-managed-environment`，加参数：`python3 -m ensurepip --default-pip --break-system-packages`

### 方式 D：apt 安装（如果 apt 可用且网络较快）

```bash
apt-get update -qq && apt-get install -y -qq python3-pip
```

### 方式 E：换腾讯云 apt 源后安装（⭐ 外网 DNS 不通或 apt 太慢时使用）

当容器环境外网 DNS 不通（`curl` 报 `Could not resolve host`）或 `apt-get update` 太慢时，先将 apt 源换成腾讯云内网镜像，再通过 apt 安装 pip 和依赖：

```bash
# 第一步：换腾讯云内网 apt 源（二选一，取决于系统使用哪种配置文件）
sed -i 's|deb.debian.org|mirrors.tencentyun.com|g' /etc/apt/sources.list.d/debian.sources 2>/dev/null; sed -i 's|deb.debian.org|mirrors.tencentyun.com|g' /etc/apt/sources.list 2>/dev/null

# 第二步：安装 pip 和基础 Python 依赖
apt-get update -qq && apt-get install -y -qq python3-pip python3-openpyxl python3-docx python3-lxml python3-pil python3-reportlab python3-pptx

# 第三步：用 pip 补装 apt 中没有的包（pdfplumber、pypdf）
cd {skill_base_dir} && python3 -m pip install --break-system-packages -r requirements.txt -i https://mirrors.cloud.tencent.com/pypi/simple --trusted-host mirrors.cloud.tencent.com --timeout 120
```

> ⚠️ 换源后 `apt-get update` 速度会快很多（走腾讯云内网）。第三步用 pip 安装 `requirements.txt` 可以补齐 apt 中缺少的包（如 `pdfplumber`、`pypdf`），已通过 apt 安装的包会自动跳过。

---

## Step 3：安装 Python 依赖（如果方式 A 离线安装已成功，可跳过此步骤）

> ⚠️ **必须使用 `python3 -m pip`**，不要使用 `pip` 或 `pip3` 命令（沙箱中通常不存在）。
> ⚠️ **必须加 `--break-system-packages`**（Debian 12+ / PEP 668 环境）。
> ⚠️ **必须指定国内镜像源**，否则大概率因网络超时导致安装失败。

**推荐命令（使用腾讯云镜像源）**：

```bash
cd {skill_base_dir} && python3 -m pip install --break-system-packages -r requirements.txt -i https://mirrors.cloud.tencent.com/pypi/simple --trusted-host mirrors.cloud.tencent.com --timeout 120
```

如果腾讯云镜像不可用，依次尝试以下备选镜像源：

```bash
# 备选 1：清华镜像
cd {skill_base_dir} && python3 -m pip install --break-system-packages -r requirements.txt -i https://pypi.tuna.tsinghua.edu.cn/simple --trusted-host pypi.tuna.tsinghua.edu.cn --timeout 120

# 备选 2：阿里云镜像
cd {skill_base_dir} && python3 -m pip install --break-system-packages -r requirements.txt -i https://mirrors.aliyun.com/pypi/simple --trusted-host mirrors.aliyun.com --timeout 120
```

---

## Step 4：验证安装

```bash
python3 -c "import openpyxl, docx, pypdf, pdfplumber, reportlab, pptx; print('All dependencies OK')"
```

如果输出 `All dependencies OK` 则安装成功，可以继续执行工作流。

---

## Step 5：安装中文字体（PDF 填写必需）

PDF 的 `text_overlay` 模式需要系统中有中文字体文件，否则中文将显示为乱码。

**检测是否有中文字体**：

```bash
fc-list :lang=zh 2>/dev/null | head -5 || find /usr/share/fonts -name "*wqy*" -o -name "*noto*cjk*" -o -name "*fang*" 2>/dev/null | head -5
```

如果没有输出，需要安装中文字体：

```bash
# Debian/Ubuntu
apt-get install -y fonts-wqy-zenhei

# CentOS/RHEL
yum install -y wqy-zenhei-fonts

# 如果 apt-get/yum 不可用，尝试安装 Noto CJK
apt-get install -y fonts-noto-cjk || yum install -y google-noto-sans-cjk-fonts
```

> ⚠️ 如果系统包管理器不可用（沙箱环境），`pdf_filler.py` 会尝试使用 reportlab 内置的 CIDFont（STSong-Light）作为兜底。如果仍然失败，stderr 中会输出 `⚠️ 未找到中文字体` 警告。

---

## 当前依赖清单

| 库 | 版本要求 | 用途 | 使用的脚本 |
|----|---------|------|-----------|
| `openpyxl` | ≥3.1.0 | xlsx 文件读写 | `xlsx_parser.py`, `xlsx_filler.py`, `content_extractor.py` |
| `python-docx` | ≥1.0.0 | docx 文件读写 | `docx_parser.py`, `docx_filler.py`, `content_extractor.py` |
| `pypdf` | ≥3.0.0 | pdf 表单域读写 | `pdf_parser.py`, `pdf_filler.py` |
| `pdfplumber` | ≥0.10.0 | pdf 文本/表格提取 | `pdf_parser.py`, `content_extractor.py` |
| `reportlab` | ≥4.0.0 | pdf 文本叠加 | `pdf_filler.py` |
| `python-pptx` | ≥0.6.21 | pptx 文件读写 | `pptx_parser.py`, `pptx_filler.py`, `content_extractor.py` |

---

## 更新 wheel 包（仅维护者）

如果需要更新 wheel 包（如依赖版本变更），在本机执行以下命令重新下载并打包，然后提交到代码仓库通过流水线发布：

```bash
cd {skill_base_dir}/scripts && bash download_wheels.sh --python-version 3.11 --platform manylinux2014_x86_64
```

可选参数：`--python-version` 指定目标 Python 版本，`--platform` 指定目标平台，`--mirror <URL>` 指定 PyPI 镜像源。
