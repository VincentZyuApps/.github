package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// ========== 配置 ==========

const (
	org         = "VincentZyuApps"
	outputDir   = "../profile"
	svgFileName = "line-stats.svg"
	svgWidth    = 480
	barHeight   = 28
	barPadding  = 6
)

// 需要忽略的文件夹和文件
var ignoreDirs = map[string]bool{
	".git": true, ".github": true, ".vscode": true, ".idea": true,
	"node_modules": true, "vendor": true, "dist": true, "build": true,
	"__pycache__": true, "venv": true, "bin": true, "obj": true,
}

var ignoreExts = map[string]bool{
	".exe": true, ".dll": true, ".so": true, ".dylib": true, ".bin": true,
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".ico": true, ".svg": true,
	".zip": true, ".tar": true, ".gz": true, ".7z": true, ".rar": true,
	".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
	".lock": true, ".json": true, ".xml": true, ".yaml": true, ".yml": true, // 配置文件通常不算代码
	".md": true, ".txt": true,
}

// 语言扩展名映射 (可以根据需要扩展)
var extToLang = map[string]string{
	".go": "Go",
	".py": "Python",
	".js": "JavaScript", ".jsx": "JavaScript", ".mjs": "JavaScript",
	".ts": "TypeScript", ".tsx": "TSX",
	".java": "Java",
	".c":    "C", ".h": "C",
	".cpp": "C++", ".hpp": "C++", ".cc": "C++", ".cxx": "C++",
	".cs":   "C#",
	".html": "HTML", ".htm": "HTML",
	".css":  "CSS",
	".scss": "SCSS", ".sass": "SCSS",
	".vue": "Vue",
	".rs":  "Rust",
	".php": "PHP",
	".rb":  "Ruby",
	".sh":  "Shell", ".bash": "Shell",
	".bat": "Batch", ".cmd": "Batch",
	".ps1":   "PowerShell",
	".kt":    "Kotlin",
	".dart":  "Dart",
	".lua":   "Lua",
	".swift": "Swift",
}

// 语言颜色
var langColors = map[string]string{
	"Go":         "#00ADD8",
	"Python":     "#3572A5",
	"JavaScript": "#f1e05a",
	"TypeScript": "#3178c6",
	"TSX":        "#3178c6",
	"Java":       "#b07219",
	"C":          "#555555",
	"C++":        "#f34b7d",
	"C#":         "#178600",
	"HTML":       "#e34c26",
	"CSS":        "#563d7c",
	"SCSS":       "#c6538c",
	"Vue":        "#41b883",
	"Rust":       "#dea584",
	"PHP":        "#4F5D95",
	"Ruby":       "#701516",
	"Shell":      "#89e051",
	"Batch":      "#C1F12E",
	"PowerShell": "#012456",
	"Kotlin":     "#A97BFF",
	"Dart":       "#00B4AB",
	"Lua":        "#000080",
	"Swift":      "#F05138",
}

// ========== 结构体 ==========

type Repo struct {
	Name     string `json:"name"`
	CloneURL string `json:"clone_url"`
}

type LangStat struct {
	Name  string
	Lines int
	Color string
}

// ========== 主逻辑 ==========

// 全局变量
var proxyURL string
var useTmp bool
var actualOutputDir string

func main() {
	// 解析命令行参数
	flag.StringVar(&proxyURL, "proxy", "", "HTTP proxy URL (e.g. http://192.168.31.233:7890)")
	flag.BoolVar(&useTmp, "tmp", false, "Output to ../tmp instead of ../profile")
	flag.Parse()

	if useTmp {
		actualOutputDir = "../tmp"
		fmt.Println("Output directory: ../tmp")
	} else {
		actualOutputDir = outputDir
	}

	if proxyURL != "" {
		fmt.Printf("Using proxy: %s\n", proxyURL)
	}

	token := os.Getenv("GH_TOKEN")
	if token == "" {
		log.Println("Warning: GH_TOKEN is not set. API rate details might be limited.")
	}

	repos, err := getOrgRepos(token)
	if err != nil {
		log.Fatalf("Failed to get repos: %v", err)
	}

	fmt.Printf("Found %d repositories.\n", len(repos))

	// 创建临时目录用于 clone
	tempDir, err := os.MkdirTemp("", "line-stats-")
	if err != nil {
		log.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	stats := make(map[string]int)
	var wg sync.WaitGroup
	var mu sync.Mutex

	// 并发 Clone 和统计 (控制并发数，防止磁盘IO爆炸)
	sem := make(chan struct{}, 5) // 最多同时处理 5 个仓库

	for _, repo := range repos {
		wg.Add(1)
		go func(r Repo) {
			defer wg.Done()
			sem <- struct{}{}        // 获取令牌
			defer func() { <-sem }() // 释放令牌

			repoPath := filepath.Join(tempDir, r.Name)
			fmt.Printf("Cloning %s...\n", r.Name)

			// Git Clone (Depth 1 浅克隆加快速度)
			cmd := exec.Command("git", "clone", "--depth", "1", r.CloneURL, repoPath)
			// 如果设置了代理，通过环境变量传给 git
			if proxyURL != "" {
				cmd.Env = append(os.Environ(),
					"http_proxy="+proxyURL,
					"https_proxy="+proxyURL,
				)
			}
			if err := cmd.Run(); err != nil {
				log.Printf("Failed to clone %s: %v", r.Name, err)
				return
			}

			// 统计行数
			repoStats := countLines(repoPath)

			mu.Lock()
			for lang, count := range repoStats {
				stats[lang] += count
			}
			mu.Unlock()

			fmt.Printf("Analyzed %s\n", r.Name)
		}(repo)
	}

	wg.Wait()

	if len(stats) == 0 {
		log.Println("No lines counted.")
		return
	}

	// 转换并排序
	var finalStats []LangStat
	totalLines := 0
	for lang, lines := range stats {
		color := langColors[lang]
		if color == "" {
			color = "#cccccc" // 默认灰色
		}
		finalStats = append(finalStats, LangStat{Name: lang, Lines: lines, Color: color})
		totalLines += lines
	}

	sort.Slice(finalStats, func(i, j int) bool {
		return finalStats[i].Lines > finalStats[j].Lines
	})

	fmt.Printf("Total Lines: %d\n", totalLines)
	for _, s := range finalStats {
		fmt.Printf("%s: %d\n", s.Name, s.Lines)
	}

	// 生成 SVG
	if err := generateSVG(finalStats, totalLines); err != nil {
		log.Fatalf("Failed to generate SVG: %v", err)
	}
}

// ========== Helper Functions ==========

func getOrgRepos(token string) ([]Repo, error) {
	var allRepos []Repo
	apiURL := fmt.Sprintf("https://api.github.com/orgs/%s/repos?per_page=100&type=public", org)

	// 如果设置了代理，创建带代理的 HTTP Client
	client := http.DefaultClient
	if proxyURL != "" {
		parsed, err := url.Parse(proxyURL)
		if err == nil {
			client = &http.Client{
				Transport: &http.Transport{
					Proxy: http.ProxyURL(parsed),
				},
			}
		}
	}

	for apiURL != "" {
		req, _ := http.NewRequest("GET", apiURL, nil)
		req.Header.Set("Accept", "application/vnd.github.v3+json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("GitHub API %d: %s", resp.StatusCode, string(body))
		}

		var repos []Repo
		if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
			return nil, err
		}
		allRepos = append(allRepos, repos...)

		// Pagination
		link := resp.Header.Get("Link")
		apiURL = ""
		if link != "" {
			parts := strings.Split(link, ",")
			for _, p := range parts {
				if strings.Contains(p, `rel="next"`) {
					fs := strings.Split(p, ";")
					if len(fs) > 0 {
						apiURL = strings.Trim(fs[0], " <>")
					}
				}
			}
		}
	}
	return allRepos, nil
}

func countLines(root string) map[string]int {
	stats := make(map[string]int)

	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			if ignoreDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ignoreExts[ext] {
			return nil
		}

		lang, ok := extToLang[ext]
		if !ok {
			return nil
		}

		lines, err := countFileLines(path)
		if err != nil {
			return nil
		}

		stats[lang] += lines
		return nil
	})

	return stats
}

func countFileLines(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	// Create a buffer large enough to handle lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024) // 1MB max line size

	lines := 0
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text != "" && !strings.HasPrefix(text, "//") && !strings.HasPrefix(text, "#") {
			// 简单的非空行且非单行注释统计
			lines++
		}
	}
	return lines, scanner.Err()
}

func generateSVG(stats []LangStat, total int) error {
	totalHeight := len(stats) * (barHeight + barPadding)

	// 如果目录不存在，创建
	if _, err := os.Stat(actualOutputDir); os.IsNotExist(err) {
		os.MkdirAll(actualOutputDir, 0755)
	}

	f, err := os.Create(filepath.Join(actualOutputDir, svgFileName))
	if err != nil {
		return err
	}
	defer f.Close()

	// 格式化总行数 (例如 12,345)
	p := messagePrinter(total)

	// SVG 高度 = header(45) + subtitle(20) + bars + padding
	headerAreaHeight := 110
	svgContentHeight := headerAreaHeight + totalHeight + 40

	var sb strings.Builder

	// SVG 头
	sb.WriteString(fmt.Sprintf(`<svg width="%d" height="%d" viewBox="0 0 %d %d" fill="none" xmlns="http://www.w3.org/2000/svg">
`, svgWidth, svgContentHeight, svgWidth, svgContentHeight))

	// 读取字体 CSS
	fontCSS, err := os.ReadFile("../sub-font/output/font_face.css")
	if err != nil {
		log.Printf("⚠️  无法读取字体 CSS: %v (将使用 fallback 字体)", err)
		fontCSS = []byte{}
	}

	// 样式 + 动画
	sb.WriteString(`<defs>
  <style>
`)
	sb.Write(fontCSS)
	sb.WriteString(`
    .bg { fill: #0d1117; }
    .header { font-family: 'LXGW WenKai Mono', 'Inter', 'Segoe UI', sans-serif; font-size: 16px; font-weight: 600; fill: #64b5f6; }
    .subtitle { font-family: 'LXGW WenKai Mono', 'Inter', 'Segoe UI', sans-serif; font-size: 11px; fill: #8b949e; }
    .lang-name { font-family: 'LXGW WenKai Mono', 'Inter', 'Segoe UI', sans-serif; font-size: 12px; fill: #e6edf3; }
    .lang-count { font-family: 'LXGW WenKai Mono', 'Inter', 'Segoe UI', sans-serif; font-size: 12px; font-weight: 600; fill: #e6edf3; }
    .bar-bg { fill: #161b22; rx: 4; }
    @keyframes fadeInRight { from { opacity: 0; transform: translateX(-8px); } to { opacity: 1; transform: translateX(0); } }
    @keyframes fadeInDown { from { opacity: 0; transform: translateY(-10px); } to { opacity: 1; transform: translateY(0); } }
    .row { animation: fadeInRight 0.6s ease forwards; opacity: 0; }
    .title-anim { animation: fadeInDown 0.8s ease forwards; opacity: 0; }
  </style>
</defs>
`)

	// 背景
	sb.WriteString(fmt.Sprintf(`<rect class="bg" width="%d" height="%d" rx="10"/>
`, svgWidth, svgContentHeight))

	// 标题
	sb.WriteString(fmt.Sprintf(`<text class="header title-anim" x="20" y="30">📏 %s · 不同语言的代码行数排行捏</text>
`, org))
	tz, _ := time.LoadLocation("Asia/Shanghai")
	sb.WriteString(fmt.Sprintf(`<text class="subtitle title-anim" style="animation-delay: 0.2s;" x="20" y="48">共 %s 行代码（不含空行和注释）</text>
`, p))
	sb.WriteString(fmt.Sprintf(`<text class="subtitle title-anim" style="animation-delay: 0.3s;" x="20" y="68">Auto update: %s (UTC)</text>
`, time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf(`<text class="subtitle title-anim" style="animation-delay: 0.4s;" x="20" y="88">自动更新于 %s (UTC+8)</text>
`, time.Now().In(tz).Format("2006-01-02 15:04:05")))

	// 生成条形图内容
	maxLines := stats[0].Lines         // stats 已排序，第一个是最大值
	maxBarWidth := svgWidth - 120 - 70 // 条形起始x(120) 到 数字右边距(70)
	for i, stat := range stats {
		y := headerAreaHeight + i*(barHeight+barPadding)
		barWidth := int(float64(stat.Lines) / float64(maxLines) * float64(maxBarWidth))
		if barWidth < 2 {
			barWidth = 2
		}
		delay := 0.3 + float64(i)*0.1

		// 每行一个 group，带 fadeIn 延迟
		sb.WriteString(fmt.Sprintf(`<g class="row" style="animation-delay: %.1fs;">
`, delay))

		// 语言颜色圆点
		sb.WriteString(fmt.Sprintf(`  <circle cx="30" cy="%d" r="5" fill="%s"/>
`, y+barHeight/2, stat.Color))

		// 语言名称
		sb.WriteString(fmt.Sprintf(`  <text x="44" y="%d" class="lang-name" dominant-baseline="central">%s</text>
`, y+barHeight/2, stat.Name))

		// 进度条背景
		sb.WriteString(fmt.Sprintf(`  <rect class="bar-bg" x="120" y="%d" width="%d" height="%d"/>
`, y+4, maxBarWidth, barHeight-8))

		// 进度条前景 (带 width 动画)
		sb.WriteString(fmt.Sprintf(`  <rect x="120" y="%d" width="0" height="%d" fill="%s" rx="4" opacity="0.85">
    <animate attributeName="width" from="0" to="%d" dur="0.8s" begin="%.1fs" fill="freeze" />
  </rect>
`, y+4, barHeight-8, stat.Color, barWidth, delay))

		// 行数文字
		sb.WriteString(fmt.Sprintf(`  <text x="%d" y="%d" class="lang-count" dominant-baseline="central" text-anchor="end">%s</text>
`, svgWidth-20, y+barHeight/2, messagePrinter(stat.Lines)))

		sb.WriteString(`</g>
`)
	}

	// 底部备注
	footerY := headerAreaHeight + totalHeight + 20
	sb.WriteString(fmt.Sprintf(`<text class="subtitle" x="%d" y="%d" text-anchor="end">统计单位: 行数（不含空行和注释）</text>
`, svgWidth-20, footerY))

	sb.WriteString(`</svg>`)

	content := sb.String()

	_, err = f.WriteString(content)
	return err
}

func messagePrinter(n int) string {
	in := fmt.Sprintf("%d", n)
	numOfDigits := len(in)
	if n < 0 {
		numOfDigits--
	}
	numOfCommas := (numOfDigits - 1) / 3

	out := make([]byte, len(in)+numOfCommas)
	if n < 0 {
		in, out[0] = in[1:], '-'
	}

	for i, j, k := len(in)-1, len(out)-1, 0; ; i, j = i-1, j-1 {
		out[j] = in[i]
		if i == 0 {
			return string(out)
		}
		if k++; k == 3 {
			j, k = j-1, 0
			out[j] = ','
		}
	}
}
