"""
本地运行 SVG 出图流程
按顺序执行: 字体子集化 -> byte-stats -> git-stats -> line-stats
"""

import os
import subprocess
import sys
from pathlib import Path

ROOT_DIR = Path(__file__).parent.parent.resolve()

PROXY = os.environ.get("HTTP_PROXY", "http://192.168.31.233:7890")
GH_TOKEN = os.environ.get("GH_TOKEN", "")


def run_cmd(cmd: list, cwd: Path, env: dict = None):
    print(f"\n{'='*50}")
    print(f"📁 {cwd.name}> {' '.join(cmd)}")
    print("=" * 50)

    merged_env = os.environ.copy()
    if env:
        merged_env.update(env)

    result = subprocess.run(cmd, cwd=cwd, env=merged_env)
    if result.returncode != 0:
        print(f"❌ 命令执行失败，退出码: {result.returncode}")
        sys.exit(result.returncode)
    print("✅ 完成")


def main():
    print("🚀 开始本地 SVG 出图流程\n")

    env = {}
    if GH_TOKEN:
        env["GH_TOKEN"] = GH_TOKEN
        print(f"🔑 GH_TOKEN: {'*' * 8}{GH_TOKEN[-4:]}")
    else:
        print("⚠️ GH_TOKEN 未设置，API 请求可能受限")

    print(f"🌐 代理: {PROXY}")

    go_proxy_args = ["-proxy", PROXY] if PROXY else []

    run_cmd(
        [sys.executable, "subset.py"],
        cwd=ROOT_DIR / "scripts",
    )

    run_cmd(
        ["go", "run", "main.go"] + go_proxy_args,
        cwd=ROOT_DIR / "svg" / "byte-stats",
        env=env,
    )

    run_cmd(
        ["go", "run", "main.go"] + go_proxy_args,
        cwd=ROOT_DIR / "svg" / "git-stats",
        env=env,
    )

    run_cmd(
        ["go", "run", "main.go"] + go_proxy_args,
        cwd=ROOT_DIR / "svg" / "line-stats",
        env=env,
    )

    print("\n" + "=" * 50)
    print("🎉 所有任务完成！SVG 文件已生成到 profile/ 目录")
    print("=" * 50)


if __name__ == "__main__":
    main()
