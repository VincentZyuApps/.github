"""
字体子集化脚本
从 LXGW WenKai Mono 字体中提取 SVG 用到的字符子集，
生成轻量级的 .woff2 字体文件和 font_face.css（含 Base64 内嵌）。

Go 程序直接 os.ReadFile 读取 font_face.css 嵌入 SVG <style> 即可。

依赖: pip install fonttools brotli
"""

import os
import base64
from fontTools.subset import Subsetter, Options
from fontTools.ttLib import TTFont

# ========== 配置 ==========

ASSETS_DIR = os.path.join(os.path.dirname(__file__), "..", "assets")
FONTS = {
    "regular": os.path.join(ASSETS_DIR, "LXGWWenKaiMono-Regular.ttf"),
    "medium": os.path.join(ASSETS_DIR, "LXGWWenKaiMono-Medium.ttf"),
}

OUTPUT_DIR = os.path.join(os.path.dirname(__file__), "output")

# SVG 中用到的所有字符
BASE_CHARS = (
    "0123456789"
    "abcdefghijklmnopqrstuvwxyz"
    "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
    "+-.,;:!?@#$%^&*()[]{}|/\\<>=_~`'\""
    " "
)

# SVG 文本中用到的中文和特殊字符
CJK_CHARS = "代码行数统计语言分布共行不含空和注释自动更新于组织·"

ALL_CHARS = "".join(sorted(set(BASE_CHARS + CJK_CHARS)))


def subset_font(input_path: str, output_path: str, chars: str, flavor: str = "woff2"):
    """对字体做子集化"""
    font = TTFont(input_path)

    options = Options()
    options.flavor = flavor
    options.desubroutinize = True

    subsetter = Subsetter(options=options)
    subsetter.populate(text=chars)
    subsetter.subset(font)

    font.save(output_path)
    size = os.path.getsize(output_path)
    print(f"  ✅ {os.path.basename(output_path)}: {size:,} bytes ({size/1024:.1f} KB)")
    return output_path


def font_to_base64(font_path: str) -> str:
    with open(font_path, "rb") as f:
        return base64.b64encode(f.read()).decode("ascii")


def generate_css_file(regular_b64: str, medium_b64: str, output_path: str):
    """生成包含 @font-face + Base64 的 CSS 文件，Go 直接读取嵌入 SVG"""
    css = f"""    @font-face {{
      font-family: 'LXGW WenKai Mono';
      font-style: normal;
      font-weight: 400;
      src: url(data:font/woff2;base64,{regular_b64}) format('woff2');
    }}
    @font-face {{
      font-family: 'LXGW WenKai Mono';
      font-style: normal;
      font-weight: 600;
      src: url(data:font/woff2;base64,{medium_b64}) format('woff2');
    }}
    @font-face {{
      font-family: 'LXGW WenKai Mono';
      font-style: normal;
      font-weight: 700;
      src: url(data:font/woff2;base64,{medium_b64}) format('woff2');
    }}"""
    with open(output_path, "w", encoding="utf-8") as f:
        f.write(css)
    size = os.path.getsize(output_path)
    print(f"  ✅ CSS file: {output_path} ({size:,} bytes, {size/1024:.1f} KB)")


def main():
    os.makedirs(OUTPUT_DIR, exist_ok=True)

    print(f"📝 字符集: {len(ALL_CHARS)} 个字符")
    print(f"   其中中文: {CJK_CHARS}")
    print()

    # 1. 子集化
    print("🔪 正在子集化字体...")
    regular_woff2 = subset_font(
        FONTS["regular"],
        os.path.join(OUTPUT_DIR, "LXGWWenKaiMono-Regular.subset.woff2"),
        ALL_CHARS,
    )
    medium_woff2 = subset_font(
        FONTS["medium"],
        os.path.join(OUTPUT_DIR, "LXGWWenKaiMono-Medium.subset.woff2"),
        ALL_CHARS,
    )

    # 2. Base64 编码
    print("\n📦 正在编码 Base64...")
    regular_b64 = font_to_base64(regular_woff2)
    medium_b64 = font_to_base64(medium_woff2)
    print(f"  Regular Base64: {len(regular_b64):,} chars ({len(regular_b64)/1024:.1f} KB)")
    print(f"  Medium Base64:  {len(medium_b64):,} chars ({len(medium_b64)/1024:.1f} KB)")

    # 3. 生成 CSS
    print("\n🔧 正在生成 font_face.css...")
    generate_css_file(
        regular_b64,
        medium_b64,
        os.path.join(OUTPUT_DIR, "font_face.css"),
    )

    print("\n✅ 完成！")
    print(f"   输出目录: {os.path.abspath(OUTPUT_DIR)}")
    print("   Go 程序通过 os.ReadFile(\"../sub-font/output/font_face.css\") 即可使用。")


if __name__ == "__main__":
    main()
