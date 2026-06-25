![.github](https://socialify.git.ci/VincentZyuApps/.github/image?custom_description=%F0%9F%8F%A0+VincentZyuApps+%E7%BB%84%E7%BB%87%E9%97%A8%E9%9D%A2%E4%BB%93%E5%BA%93+-+%F0%9F%A4%96+%E4%BD%BF%E7%94%A8+GitHub+Actions+%E8%87%AA%E5%8A%A8%E7%94%9F%E6%88%90%E7%BB%9F%E8%AE%A1%E6%95%B0%E6%8D%AE+SVG+%E5%9B%BE%E8%A1%A8%EF%BC%88GitHub+Stats%2C+Line+Stats%2C+Byte+Stats%29&description=1&font=Inter&forks=1&issues=1&language=1&logo=https%3A%2F%2Fupload.wikimedia.org%2Fwikipedia%2Fcommons%2Fthumb%2Fc%2Fc3%2FPython-logo-notext.svg%2F120px-Python-logo-notext.svg.png%3F_%3D20250701090410&name=1&owner=1&pulls=1&stargazers=1&theme=Auto)

# VincentZyuApps Organization

![GitHub last commit](https://img.shields.io/github/last-commit/VincentZyuApps/.github?style=for-the-badge)
![GitHub Workflow Status](https://img.shields.io/github/actions/workflow/status/VincentZyuApps/.github/update-all.yml?style=for-the-badge)

## 🎯 仓库简介

本仓库是 **VincentZyuApps** 组织的门面仓库，使用 **GitHub Actions** 自动化生成各种统计数据的 SVG 图表，包括：

- 📊 代码字节分布
- 📈 GitHub 统计信息
- 📏 代码行数统计
- 🔥 提交活跃度

所有统计数据通过 GitHub Actions 自动更新，支持定时任务、Push 触发、手动执行三种方式。

## 📁 主要功能

- **自动生成 SVG 图表**：使用 Go 语言生成美观的统计图表
- **实时数据更新**：每 3 小时自动更新一次统计数据
- **多维度统计**：涵盖代码字节分布、代码行数、GitHub 活动等多个维度
- **自动部署**：生成的图表自动部署到组织主页

## 🔗 组织主页

组织的完整统计信息和门面展示，请访问：

[![VincentZyuApps 组织门面主页](https://img.shields.io/badge/VincentZyuApps-ff69b4?style=for-the-badge&logo=github&logoColor=white&label=🚀%20组织门面主页)](https://github.com/VincentZyuApps)
[![组织主页 README](https://img.shields.io/badge/README-181717?style=for-the-badge&logo=github&logoColor=white&label=📄%20左侧有文件树)](./profile/README.md)

## 📚 相关文档

- [📋 工作流更新方式说明](./.github/workflows/update.md)

## 📁 仓库目录结构

本仓库的目录结构如下：

```
.
├── .github           # GitHub Actions 工作流配置
├── assets             # 字体资源文件
├── svg                # SVG 统计生成器集合
├── profile            # 生成的统计图表和组织主页
├── scripts            # 字体与脚本工具
└── tmp                # 临时文件目录
```

### 主要目录说明

- **.github/**: 存放 GitHub Actions 工作流和相关说明文档
- **assets/**: 存放 LXGWWenKaiMono 字体文件
- **svg/git-stats/**: Go 语言实现的 GitHub 统计生成器
- **svg/byte-stats/**: Go 语言实现的代码字节统计生成器
- **svg/line-stats/**: Go 语言实现的代码行数统计生成器
- **profile/**: 生成的 SVG 图表和组织主页 README.md
- **scripts/**: 字体子集化与本地辅助脚本
- **tmp/**: 临时文件目录，用于测试生成的图表

## 🤖 自动化工作流

本仓库使用以下 GitHub Actions 工作流：

| 工作流 | 功能 | 触发方式 |
|--------|------|----------|
| `update-all.yml` | 更新所有统计 | 手动触发 |
| `update-byte-stats.yml` | 更新代码字节统计 | 定时、Push、手动 |
| `update-git-stats.yml` | 更新 GitHub 统计 | 定时、Push、手动 |
| `update-line-stats.yml` | 更新代码行数统计 | 定时、Push、手动 |
| `update-3d-stats.yml` | 更新 3D GitHub 贡献图 | 定时、Push、手动 |
| `update-font.yml` | 更新字体子集 | Push 触发 |

## 🌟 技术栈

- **后端**：Py(字体子集化) + Go(svg出图)
- **自动化**：GitHub Actions
- **前端**：SVG + CSS
- **字体**：LXGW WenKai Mono
---

<p align="center">
  <img src="https://visitor-badge.laobi.icu/badge?page_id=VincentZyuApps.VincentZyuApps.github" />
</p>
