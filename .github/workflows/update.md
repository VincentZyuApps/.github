# 工作流更新方式说明

本项目使用 GitHub Actions 自动更新统计数据，支持以下三种触发方式：

## 🕐 定时任务

所有统计工作流都会定期自动执行，采用错峰策略避免 API rate limit：

| 工作流 | 执行时间 | 说明 |
|--------|---------|------|
| `update-lang-stats.yml` | 每 3 小时（0:00, 3:00, 6:00...） | 组织语言分布统计 |
| `update-git-stats.yml` | 每 3 小时（0:30, 3:30, 6:30...） | GitHub 统计信息 |
| `update-line-stats.yml` | 每 3 小时（1:00, 4:00, 7:00...） | 代码行数统计 |

## 🔄 Push 触发

当代码库中相关文件变更时，对应的工作流会自动执行：

- `update-font.yml`: 当 `sub-font/**` 或 `assets/*.ttf` 变更时触发
- `update-lang-stats.yml`: 当 `lang-stats/**` 变更时触发
- `update-git-stats.yml`: 当 `git-stats/**` 变更时触发
- `update-line-stats.yml`: 当 `line-stats/**` 变更时触发

## 🖱️ 手动触发

可以在 GitHub 网页界面手动执行工作流：

1. 进入仓库的 **Actions** 标签页
2. 在左侧选择对应的工作流
3. 点击右侧的 **Run workflow** 按钮
4. 选择分支并点击 **Run workflow**

支持手动触发的工作流：
- `update-all.yml`: 更新所有统计（包括字体子集化）
- `update-font.yml`: 仅更新字体子集
- `update-lang-stats.yml`: 仅更新语言统计
- `update-git-stats.yml`: 仅更新 GitHub 统计
- `update-line-stats.yml`: 仅更新代码行数统计

## 💡 使用建议

- **日常使用**: 依赖定时任务自动更新即可
- **代码修改后**: Push 会自动触发对应工作流
- **急需更新**: 使用手动触发方式快速更新
- **批量更新**: 使用 `update-all.yml` 一次性更新所有统计

## 📝 本地更新Git手动推送流程

当本地有修改(比如doc修改)需要推送到远程仓库时，推荐使用以下优雅的流程：

### 推荐步骤

```bash
# 1. 查看当前修改状态
git status

# 2. 暂存所有修改
git add .

# 3. 提交修改（使用详细的提交信息）
git commit -m "feat: 功能描述
- 详细说明 1
- 详细说明 2"

# 4. 拉取远程更新（使用 rebase 保持提交历史整洁）
git pull --rebase origin main

# 5. 推送修改到远程仓库
git push origin main
```

### 为什么这样做？

- **`git pull --rebase`**: 将本地修改放在远程更新的后面，保持提交历史线性，避免不必要的合并提交
- **详细的提交信息**: 让团队成员清楚了解每次修改的内容
- **分步操作**: 确保每一步都能检查状态，减少错误

### 处理冲突

如果 rebase 过程中出现冲突：

1. 打开冲突文件，找到 `<<<<<<<` 和 `=======` 之间的部分
2. 保留需要的修改，删除冲突标记
3. 运行 `git add .` 标记冲突已解决
4. 继续 rebase：`git rebase --continue`

这个流程特别适合处理 GitHub Actions 自动更新后的远程仓库同步！
