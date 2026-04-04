package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ========== 配置 ==========

const (
	org         = "VincentZyuApps"
	maxLangs    = 114514       // 2026年3月21日20:36:48 改成 无限制~
	svgWidth    = 480          // SVG 宽度
	barHeight   = 28           // 每行高度
	barPadding  = 6            // 行间距
	outputDir   = "../profile" // 输出目录
	svgFileName = "org-languages.svg"
)

// 排除的语言（占比过大导致其他语言看不清）
var excludeLangs = map[string]bool{
	"Vue": true,
}

// GitHub 语言颜色 (常见语言)
var langColors = map[string]string{
	"Python":     "#3572A5",
	"JavaScript": "#f1e05a",
	"TypeScript": "#3178c6",
	"TSX":        "#3178c6",
	"Go":         "#00ADD8",
	"C++":        "#f34b7d",
	"C":          "#555555",
	"C#":         "#178600",
	"Java":       "#b07219",
	"HTML":       "#e34c26",
	"CSS":        "#563d7c",
	"Shell":      "#89e051",
	"Rust":       "#dea584",
	"Kotlin":     "#A97BFF",
	"Dart":       "#00B4AB",
	"Swift":      "#F05138",
	"Ruby":       "#701516",
	"PHP":        "#4F5D95",
	"Lua":        "#000080",
	"Vue":        "#41b883",
	"SCSS":       "#c6538c",
	"Makefile":   "#427819",
	"Dockerfile": "#384d54",
	"Batchfile":  "#C1F12E",
	"PowerShell": "#012456",
	"CMake":      "#DA3434",
	"GLSL":       "#5686a5",
	"HLSL":       "#aace60",
	"Svelte":     "#ff3e00",
	"Astro":      "#ff5a03",
}

// ========== GitHub API ==========

type Repo struct {
	Name         string `json:"name"`
	Fork         bool   `json:"fork"`
	LanguagesURL string `json:"languages_url"`
}

func getRepos(token string) ([]Repo, error) {
	var allRepos []Repo
	url := fmt.Sprintf("https://api.github.com/orgs/%s/repos?per_page=100&type=public", org)

	for url != "" {
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("Accept", "application/vnd.github.v3+json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetch repos: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("GitHub API %d: %s", resp.StatusCode, string(body))
		}

		var repos []Repo
		if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
			return nil, fmt.Errorf("decode repos: %w", err)
		}
		allRepos = append(allRepos, repos...)

		// 解析 Link header 翻页
		url = parseNextLink(resp.Header.Get("Link"))
	}

	return allRepos, nil
}

func getLanguages(url, token string) (map[string]int64, error) {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var langs map[string]int64
	if err := json.NewDecoder(resp.Body).Decode(&langs); err != nil {
		return nil, err
	}
	return langs, nil
}

func parseNextLink(header string) string {
	if header == "" {
		return ""
	}
	for _, part := range strings.Split(header, ",") {
		if strings.Contains(part, `rel="next"`) {
			start := strings.Index(part, "<")
			end := strings.Index(part, ">")
			if start != -1 && end != -1 {
				return part[start+1 : end]
			}
		}
	}
	return ""
}

// ========== 统计 ==========

type LangStat struct {
	Name    string
	Bytes   int64
	Percent float64
	Color   string
}

func calcStats(repos []Repo, token string) ([]LangStat, int64) {
	totals := make(map[string]int64)

	for _, repo := range repos {
		if repo.Fork {
			continue
		}
		langs, err := getLanguages(repo.LanguagesURL, token)
		if err != nil {
			log.Printf("⚠️  skip %s: %v", repo.Name, err)
			continue
		}
		for lang, bytes := range langs {
			if excludeLangs[lang] {
				continue
			}
			totals[lang] += bytes
		}
		// 避免 rate limit
		time.Sleep(100 * time.Millisecond)
	}

	var stats []LangStat
	var totalBytes int64
	for _, bytes := range totals {
		totalBytes += bytes
	}

	for lang, bytes := range totals {
		color, ok := langColors[lang]
		if !ok {
			color = "#8b8b8b" // 默认灰色
		}
		stats = append(stats, LangStat{
			Name:    lang,
			Bytes:   bytes,
			Percent: float64(bytes) / float64(totalBytes) * 100,
			Color:   color,
		})
	}

	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Bytes > stats[j].Bytes
	})

	if len(stats) > maxLangs {
		// 把剩余的归为 Other
		var otherBytes int64
		for _, s := range stats[maxLangs:] {
			otherBytes += s.Bytes
		}
		stats = stats[:maxLangs]
		if otherBytes > 0 {
			stats = append(stats, LangStat{
				Name:    "Other",
				Bytes:   otherBytes,
				Percent: float64(otherBytes) / float64(totalBytes) * 100,
				Color:   "#8b8b8b",
			})
		}
	}

	return stats, totalBytes
}

// ========== SVG 生成 ==========

func formatBytes(bytes int64) string {
	if bytes >= 1_000_000 {
		return fmt.Sprintf("%.1f MB", float64(bytes)/1_000_000)
	}
	if bytes >= 1_000 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1_000)
	}
	return fmt.Sprintf("%d B", bytes)
}

func generateSVG(stats []LangStat, totalBytes int64) string {
	topBarHeight := 12.0
	topBarY := 100.0
	contentStartY := topBarY + topBarHeight + 30
	rowHeight := float64(barHeight + barPadding)

	// 底部备注行数：1行"统计单位" + excluded语言（若有）
	excludeNames := make([]string, 0, len(excludeLangs))
	for k := range excludeLangs {
		excludeNames = append(excludeNames, k)
	}
	sort.Strings(excludeNames)
	footerLines := 1 // "统计单位: 字节"
	if len(excludeNames) > 0 {
		footerLines++ // "已排除: ..."
	}
	height := contentStartY + float64(len(stats))*rowHeight + float64(footerLines)*16 + 20

	var sb strings.Builder

	// SVG 头
	sb.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%.0f" viewBox="0 0 %d %.0f">
`, svgWidth, height, svgWidth, height))

	// 读取字体 CSS
	fontCSS, err := os.ReadFile("../sub-font/output/font_face.css")
	if err != nil {
		log.Printf("⚠️  无法读取字体 CSS: %v (将使用 fallback 字体)", err)
		fontCSS = []byte{}
	}

	// 定义样式
	sb.WriteString(`<defs>
  <style>
`)
	sb.Write(fontCSS)
	sb.WriteString(`
    .bg { fill: #0d1117; rx: 10; }
    .title { font-family: 'LXGW WenKai Mono', 'Inter', 'Segoe UI', sans-serif; font-size: 16px; font-weight: 600; fill: #64b5f6; }
    .subtitle { font-family: 'LXGW WenKai Mono', 'Inter', 'Segoe UI', sans-serif; font-size: 11px; fill: #8b949e; }
    .lang-name { font-family: 'LXGW WenKai Mono', 'Inter', 'Segoe UI', sans-serif; font-size: 12px; fill: #e6edf3; }
    .lang-pct { font-family: 'LXGW WenKai Mono', 'Inter', 'Segoe UI', sans-serif; font-size: 11px; fill: #8b949e; }
    .bar-bg { fill: #161b22; rx: 4; }
    .top-bar-bg { fill: #161b22; rx: 6; }
    @keyframes fadeInRight { from { opacity: 0; transform: translateX(-8px); } to { opacity: 1; transform: translateX(0); } }
    @keyframes fadeInDown { from { opacity: 0; transform: translateY(-10px); } to { opacity: 1; transform: translateY(0); } }
    .row { animation: fadeInRight 0.6s ease forwards; opacity: 0; }
    .title-anim { animation: fadeInDown 0.8s ease forwards; opacity: 0; }
  </style>
</defs>
`)

	// 背景
	sb.WriteString(fmt.Sprintf(`<rect class="bg" width="%d" height="%.0f" rx="10"/>
`, svgWidth, height))

	// 标题
	sb.WriteString(fmt.Sprintf(`<text class="title title-anim" x="20" y="30">📊 %s · 语言分布</text>
`, org))
	tz, _ := time.LoadLocation("Asia/Shanghai")
	sb.WriteString(fmt.Sprintf(`<text class="subtitle title-anim" style="animation-delay: 0.2s;" x="20" y="48">共 %s 代码</text>
`, formatBytes(totalBytes)))
	sb.WriteString(fmt.Sprintf(`<text class="subtitle title-anim" style="animation-delay: 0.3s;" x="20" y="68">Auto update: %s (UTC)</text>
`, time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf(`<text class="subtitle title-anim" style="animation-delay: 0.4s;" x="20" y="88">自动更新于 %s (UTC+8)</text>
`, time.Now().In(tz).Format("2006-01-02 15:04:05")))

	// 顶部汇总条（带展开动画 + 圆角裁剪）
	sb.WriteString(fmt.Sprintf(`<clipPath id="top-bar-clip">
  <rect x="20" y="%.0f" width="%d" height="%.0f" rx="6"/>
</clipPath>
`, topBarY, svgWidth-40, topBarHeight))
	sb.WriteString(fmt.Sprintf(`<rect class="top-bar-bg" x="20" y="%.0f" width="%d" height="%.0f"/>
`, topBarY, svgWidth-40, topBarHeight))
	sb.WriteString(fmt.Sprintf(`<g clip-path="url(#top-bar-clip)">
`))
	barW := float64(svgWidth - 40)
	offsetX := 20.0
	for i, s := range stats {
		w := math.Max(barW*s.Percent/100, 1)
		delay := 0.3 + float64(i)*0.05
		sb.WriteString(fmt.Sprintf(`<rect x="%.1f" y="%.0f" width="0" height="%.0f" fill="%s">
  <animate attributeName="width" from="0" to="%.1f" dur="0.6s" begin="%.2fs" fill="freeze" />
</rect>
`, offsetX, topBarY, topBarHeight, s.Color, w, delay))
		offsetX += w
	}
	sb.WriteString(`</g>
`)

	// 每种语言的行
	for i, s := range stats {
		y := contentStartY + float64(i)*rowHeight
		delay := 0.4 + float64(i)*0.1

		sb.WriteString(fmt.Sprintf(`<g class="row" style="animation-delay: %.1fs;">
`, delay))

		// 语言颜色圆点
		sb.WriteString(fmt.Sprintf(`  <circle cx="30" cy="%.1f" r="5" fill="%s"/>
`, y+float64(barHeight)/2, s.Color))

		// 语言名
		sb.WriteString(fmt.Sprintf(`  <text class="lang-name" x="44" y="%.1f">%s</text>
`, y+float64(barHeight)/2+4, s.Name))

		// 进度条背景
		progX := 160.0
		progW := float64(svgWidth) - progX - 80
		sb.WriteString(fmt.Sprintf(`  <rect class="bar-bg" x="%.0f" y="%.1f" width="%.0f" height="%.0f"/>
`, progX, y+4, progW, float64(barHeight)-8))

		// 进度条（带 width 动画）—— 相对于最大值，第一名占满
		maxPercent := stats[0].Percent
		fillW := math.Max(progW*s.Percent/maxPercent, 2)
		sb.WriteString(fmt.Sprintf(`  <rect x="%.0f" y="%.1f" width="0" height="%.0f" rx="4" fill="%s" opacity="0.85">
    <animate attributeName="width" from="0" to="%.1f" dur="0.8s" begin="%.1fs" fill="freeze" />
  </rect>
`, progX, y+4, float64(barHeight)-8, s.Color, fillW, delay))

		// 百分比
		sb.WriteString(fmt.Sprintf(`  <text class="lang-pct" x="%.0f" y="%.1f">%.1f%%</text>
`, float64(svgWidth)-70, y+float64(barHeight)/2+4, s.Percent))

		sb.WriteString(`</g>
`)
	}

	// 底部备注
	footerY := contentStartY + float64(len(stats))*rowHeight + 14
	sb.WriteString(fmt.Sprintf(`<text class="subtitle" x="%d" y="%.0f" text-anchor="end">统计单位: 字节</text>
`, svgWidth-20, footerY))
	if len(excludeNames) > 0 {
		sb.WriteString(fmt.Sprintf(`<text class="subtitle" x="%d" y="%.0f" text-anchor="end">已排除: %s</text>
`, svgWidth-20, footerY+16, strings.Join(excludeNames, ", ")))
	}

	sb.WriteString(`</svg>`)
	return sb.String()
}

// ========== README 更新 ==========

func updateReadme(readmePath, svgFile string) error {
	content, err := os.ReadFile(readmePath)
	if err != nil {
		return fmt.Errorf("read README: %w", err)
	}

	text := string(content)

	// 标记区域
	startMarker := "<!-- ORG_LANG_STATS_START -->"
	endMarker := "<!-- ORG_LANG_STATS_END -->"

	newSection := fmt.Sprintf(`%s
### 📊 组织语言分布
<p align="center">
  <img src="%s" alt="Organization Language Stats" />
</p>
%s`, startMarker, svgFile, endMarker)

	if strings.Contains(text, startMarker) && strings.Contains(text, endMarker) {
		// 替换已有区域
		startIdx := strings.Index(text, startMarker)
		endIdx := strings.Index(text, endMarker) + len(endMarker)
		text = text[:startIdx] + newSection + text[endIdx:]
	} else {
		// 在 </div> 前插入
		divEnd := strings.LastIndex(text, "</div>")
		if divEnd != -1 {
			text = text[:divEnd] + "\n" + newSection + "\n\n" + text[divEnd:]
		} else {
			text += "\n\n" + newSection + "\n"
		}
	}

	return os.WriteFile(readmePath, []byte(text), 0644)
}

// ========== 主函数 ==========

// 全局变量
var (
	proxyURL        string
	useTmp          bool
	actualOutputDir string
	httpClient      *http.Client
)

func main() {
	flag.StringVar(&proxyURL, "proxy", "", "HTTP proxy URL (e.g. http://192.168.31.233:7890)")
	flag.BoolVar(&useTmp, "tmp", false, "Output to ../tmp instead of ../profile")
	flag.Parse()

	if useTmp {
		actualOutputDir = "../tmp"
		log.Println("📁 Output directory: ../tmp")
	} else {
		actualOutputDir = outputDir
	}

	httpClient = http.DefaultClient
	if proxyURL != "" {
		log.Printf("🌐 Using proxy: %s", proxyURL)
		parsed, err := url.Parse(proxyURL)
		if err == nil {
			httpClient = &http.Client{
				Transport: &http.Transport{Proxy: http.ProxyURL(parsed)},
			}
		}
	}

	token := os.Getenv("GH_TOKEN")
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}

	log.Printf("🔍 正在获取 %s 组织的仓库...", org)
	repos, err := getRepos(token)
	if err != nil {
		log.Fatalf("❌ 获取仓库失败: %v", err)
	}
	log.Printf("📦 找到 %d 个仓库", len(repos))

	log.Println("📊 正在统计语言...")
	stats, totalBytes := calcStats(repos, token)

	log.Println("🎨 正在生成 SVG...")
	svg := generateSVG(stats, totalBytes)

	// 写 SVG 文件
	svgPath := filepath.Join(actualOutputDir, svgFileName)
	if err := os.MkdirAll(actualOutputDir, 0755); err != nil {
		log.Fatalf("❌ 创建目录失败: %v", err)
	}
	if err := os.WriteFile(svgPath, []byte(svg), 0644); err != nil {
		log.Fatalf("❌ 写入 SVG 失败: %v", err)
	}
	log.Printf("✅ SVG 已保存: %s", svgPath)

	// 更新 README（--tmp 模式下跳过）
	if useTmp {
		log.Println("⏭️  --tmp 模式，跳过 README 更新")
	} else {
		readmePath := filepath.Join(actualOutputDir, "README.md")
		if err := updateReadme(readmePath, svgFileName); err != nil {
			log.Fatalf("❌ 更新 README 失败: %v", err)
		}
		log.Println("✅ README 已更新!")
	}

	// 打印统计结果
	log.Println("\n📊 语言统计排行：")
	for i, s := range stats {
		log.Printf("  %2d. %-15s %8s  %5.1f%%", i+1, s.Name, formatBytes(s.Bytes), s.Percent)
	}
	log.Printf("\n  总计: %s", formatBytes(totalBytes))
}
