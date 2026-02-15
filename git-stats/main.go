package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ========== 配置 ==========

const (
	org         = "VincentZyuApps"
	user        = "VincentZyu233" // 个人账号（用于查询个人贡献数据）
	svgWidth    = 480
	outputDir   = "../profile"
	svgFileName = "github-stats.svg"
)

// ========== GitHub API 数据结构 ==========

type Repo struct {
	Name            string `json:"name"`
	Fork            bool   `json:"fork"`
	StargazersCount int    `json:"stargazers_count"`
	ForksCount      int    `json:"forks_count"`
	OpenIssuesCount int    `json:"open_issues_count"` // issues + PRs
	Size            int    `json:"size"`              // KB
}

// GraphQL 响应结构
type GraphQLResponse struct {
	Data struct {
		User struct {
			ContributionsCollection struct {
				TotalCommitContributions            int `json:"totalCommitContributions"`
				TotalPullRequestContributions       int `json:"totalPullRequestContributions"`
				TotalIssueContributions             int `json:"totalIssueContributions"`
				TotalPullRequestReviewContributions int `json:"totalPullRequestReviewContributions"`
				ContributionCalendar                struct {
					TotalContributions int `json:"totalContributions"`
				} `json:"contributionCalendar"`
				RestrictedContributionsCount int `json:"restrictedContributionsCount"`
			} `json:"contributionsCollection"`
			Repositories struct {
				TotalCount int `json:"totalCount"`
			} `json:"repositories"`
			RepositoriesContributedTo struct {
				TotalCount int `json:"totalCount"`
			} `json:"repositoriesContributedTo"`
			Followers struct {
				TotalCount int `json:"totalCount"`
			} `json:"followers"`
		} `json:"user"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// ========== 统计数据 ==========

type GitStats struct {
	TotalStars         int
	TotalForks         int
	TotalRepos         int
	TotalCommits       int
	TotalPRs           int
	TotalIssues        int
	TotalPRReviews     int
	TotalContributions int
	ContributedTo      int
	Followers          int
	Rank               string
	RankPercent        float64
}

// ========== GitHub REST API ==========

func getOrgRepos(token string) ([]Repo, error) {
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
		url = parseNextLink(resp.Header.Get("Link"))
	}

	return allRepos, nil
}

func getUserRepos(token string) ([]Repo, error) {
	var allRepos []Repo
	url := fmt.Sprintf("https://api.github.com/users/%s/repos?per_page=100&type=owner", user)

	for url != "" {
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("Accept", "application/vnd.github.v3+json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetch user repos: %w", err)
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
		url = parseNextLink(resp.Header.Get("Link"))
	}

	return allRepos, nil
}

// ========== GitHub GraphQL API ==========

func getUserContributions(token string) (*GraphQLResponse, error) {
	query := `{
		user(login: "` + user + `") {
			contributionsCollection {
				totalCommitContributions
				totalPullRequestContributions
				totalIssueContributions
				totalPullRequestReviewContributions
				restrictedContributionsCount
				contributionCalendar {
					totalContributions
				}
			}
			repositories(ownerAffiliations: OWNER, first: 1) {
				totalCount
			}
			repositoriesContributedTo(contributionTypes: [COMMIT, ISSUE, PULL_REQUEST, REPOSITORY], first: 1) {
				totalCount
			}
			followers {
				totalCount
			}
		}
	}`

	body := fmt.Sprintf(`{"query": %q}`, query)
	req, _ := http.NewRequest("POST", "https://api.github.com/graphql", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("graphql request: %w", err)
	}
	defer resp.Body.Close()

	var result GraphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode graphql: %w", err)
	}

	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("graphql error: %s", result.Errors[0].Message)
	}

	return &result, nil
}

// ========== 辅助函数 ==========

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

// 计算 rank（简化版，参考 github-readme-stats 的算法）
func calcRank(stats *GitStats) {
	// 基于 exponential cdf 的简化评分
	// 参考: https://github.com/anuraghazra/github-readme-stats/blob/master/src/calculateRank.js
	commits := float64(stats.TotalCommits)
	prs := float64(stats.TotalPRs)
	issues := float64(stats.TotalIssues)
	reviews := float64(stats.TotalPRReviews)
	stars := float64(stats.TotalStars)
	followers := float64(stats.Followers)

	// 加权得分（简化）
	score := (commits*1.0 + prs*3.0 + issues*2.0 + reviews*1.0 + stars*5.0 + followers*1.0) / 100.0

	// 段位判定
	switch {
	case score >= 100:
		stats.Rank = "S+"
		stats.RankPercent = 1
	case score >= 50:
		stats.Rank = "S"
		stats.RankPercent = 5
	case score >= 25:
		stats.Rank = "A++"
		stats.RankPercent = 10
	case score >= 15:
		stats.Rank = "A+"
		stats.RankPercent = 25
	case score >= 8:
		stats.Rank = "A"
		stats.RankPercent = 40
	case score >= 4:
		stats.Rank = "B+"
		stats.RankPercent = 55
	case score >= 2:
		stats.Rank = "B"
		stats.RankPercent = 70
	default:
		stats.Rank = "C"
		stats.RankPercent = 90
	}
}

func formatNumber(n int) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fk", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}

// ========== SVG 生成 ==========

type StatRow struct {
	Icon  string
	Label string
	Value string
	Color string
}

func generateSVG(stats *GitStats) string {
	rows := []StatRow{
		{"⭐", "Total Stars", formatNumber(stats.TotalStars), "#ffd700"},
		{"📦", "Total Repos", formatNumber(stats.TotalRepos), "#64b5f6"},
		{"🔀", "Total Forks", formatNumber(stats.TotalForks), "#41b883"},
		{"📝", "Total Commits", formatNumber(stats.TotalCommits), "#e6edf3"},
		{"🔃", "Total PRs", formatNumber(stats.TotalPRs), "#3178c6"},
		{"❗", "Total Issues", formatNumber(stats.TotalIssues), "#f34b7d"},
		{"👀", "PR Reviews", formatNumber(stats.TotalPRReviews), "#dea584"},
		{"🤝", "Contributed To", formatNumber(stats.ContributedTo), "#00ADD8"},
	}

	rowHeight := 34.0
	headerHeight := 55.0
	height := headerHeight + float64(len(rows))*rowHeight + 30

	// Rank 圆环参数
	rankCX := float64(svgWidth) - 60
	rankCY := height/2 + 10
	rankR := 40.0
	circumference := 2 * 3.14159 * rankR
	progress := circumference * (1 - stats.RankPercent/100)

	var sb strings.Builder

	// SVG 头
	sb.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%.0f" viewBox="0 0 %d %.0f">
`, svgWidth, height, svgWidth, height))

	// 读取字体 CSS
	fontCSS, fontErr := os.ReadFile("../sub-font/output/font_face.css")
	if fontErr != nil {
		log.Printf("⚠️  无法读取字体 CSS: %v (将使用 fallback 字体)", fontErr)
		fontCSS = []byte{}
	}

	// 样式
	sb.WriteString(`<defs>
  <style>
`)
	sb.Write(fontCSS)
	sb.WriteString(fmt.Sprintf(`
    .bg { fill: #0d1117; }
    .title { font-family: 'LXGW WenKai Mono', 'Inter', 'Segoe UI', sans-serif; font-size: 16px; font-weight: 600; fill: #64b5f6; }
    .subtitle { font-family: 'LXGW WenKai Mono', 'Inter', 'Segoe UI', sans-serif; font-size: 11px; fill: #8b949e; }
    .stat-label { font-family: 'LXGW WenKai Mono', 'Inter', 'Segoe UI', sans-serif; font-size: 13px; fill: #8b949e; }
    .stat-value { font-family: 'LXGW WenKai Mono', 'Inter', 'Segoe UI', sans-serif; font-size: 13px; font-weight: 600; fill: #e6edf3; }
    .rank-text { font-family: 'LXGW WenKai Mono', 'Inter', 'Segoe UI', sans-serif; font-size: 22px; font-weight: 700; fill: #64b5f6; }
    .rank-label { font-family: 'LXGW WenKai Mono', 'Inter', 'Segoe UI', sans-serif; font-size: 10px; fill: #8b949e; }
    .rank-ring-bg { fill: none; stroke: #161b22; stroke-width: 6; }
    .rank-ring { fill: none; stroke: #64b5f6; stroke-width: 6; stroke-linecap: round;
      stroke-dasharray: %.1f; stroke-dashoffset: %.1f;
      transform-origin: %.0fpx %.0fpx; transform: rotate(-90deg);
      animation: rankAnim 1.5s ease-in-out forwards; }
    @keyframes rankAnim {
      from { stroke-dashoffset: %.1f; }
      to { stroke-dashoffset: %.1f; }
    }
    .stat-icon { font-size: 14px; }
    @keyframes fadeIn { from { opacity: 0; transform: translateX(-5px); } to { opacity: 1; transform: translateX(0); } }
    @keyframes fadeInDown { from { opacity: 0; transform: translateY(-10px); } to { opacity: 1; transform: translateY(0); } }
    .stat-row { animation: fadeIn 0.5s ease forwards; opacity: 0; }
    .title-anim { animation: fadeInDown 0.8s ease forwards; opacity: 0; }
  </style>
</defs>
`, circumference, progress, rankCX, rankCY, circumference, progress))

	// 背景
	sb.WriteString(fmt.Sprintf(`<rect class="bg" width="%d" height="%.0f" rx="10"/>
`, svgWidth, height))

	// 标题
	sb.WriteString(fmt.Sprintf(`<text class="title title-anim" x="20" y="28">📊 %s · GitHub Stats</text>
`, user))
	sb.WriteString(fmt.Sprintf(`<text class="subtitle title-anim" style="animation-delay: 0.2s;" x="20" y="44">%s + %s · 自动更新于 %s</text>
`, user, org, time.Now().Format("2006-01-02")))

	// 统计行
	for i, row := range rows {
		y := headerHeight + float64(i)*rowHeight
		delay := float64(i) * 0.1

		sb.WriteString(fmt.Sprintf(`<g class="stat-row" style="animation-delay: %.1fs;">
`, delay))
		sb.WriteString(fmt.Sprintf(`  <text class="stat-icon" x="24" y="%.1f">%s</text>
`, y+16, row.Icon))
		sb.WriteString(fmt.Sprintf(`  <text class="stat-label" x="46" y="%.1f">%s</text>
`, y+16, row.Label))
		sb.WriteString(fmt.Sprintf(`  <text class="stat-value" x="%.0f" y="%.1f" text-anchor="end">%s</text>
`, float64(svgWidth)-130, y+16, row.Value))
		sb.WriteString(`</g>
`)
	}

	// Rank 圆环
	sb.WriteString(fmt.Sprintf(`<circle class="rank-ring-bg" cx="%.0f" cy="%.0f" r="%.0f"/>
`, rankCX, rankCY, rankR))
	sb.WriteString(fmt.Sprintf(`<circle class="rank-ring" cx="%.0f" cy="%.0f" r="%.0f"/>
`, rankCX, rankCY, rankR))
	sb.WriteString(fmt.Sprintf(`<text class="rank-text" x="%.0f" y="%.0f" text-anchor="middle" dominant-baseline="central">%s</text>
`, rankCX, rankCY-4, stats.Rank))
	sb.WriteString(fmt.Sprintf(`<text class="rank-label" x="%.0f" y="%.0f" text-anchor="middle">Top %.0f%%</text>
`, rankCX, rankCY+18, stats.RankPercent))

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

	startMarker := "<!-- GIT_STATS_START -->"
	endMarker := "<!-- GIT_STATS_END -->"

	newSection := fmt.Sprintf(`%s
### 📈 GitHub 统计
<p align="center">
  <img src="%s" alt="GitHub Stats" />
</p>
%s`, startMarker, svgFile, endMarker)

	if strings.Contains(text, startMarker) && strings.Contains(text, endMarker) {
		startIdx := strings.Index(text, startMarker)
		endIdx := strings.Index(text, endMarker) + len(endMarker)
		text = text[:startIdx] + newSection + text[endIdx:]
	} else {
		// 替换原来注释掉的 GitHub 统计部分
		oldHeader := "### 📈 GitHub 统计\n<!-- ![GitHub Stats]"
		if idx := strings.Index(text, oldHeader); idx != -1 {
			// 找到这个部分的结尾（下一个空行或下一个 marker）
			endIdx := strings.Index(text[idx:], "\n\n")
			if endIdx != -1 {
				text = text[:idx] + newSection + text[idx+endIdx:]
			}
		} else {
			// 在 <div align="center"> 后插入
			divStart := strings.Index(text, "<div align=\"center\">")
			if divStart != -1 {
				insertPos := divStart + len("<div align=\"center\">")
				text = text[:insertPos] + "\n\n" + newSection + text[insertPos:]
			}
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

	stats := &GitStats{}

	// 1. 获取组织仓库统计
	log.Printf("🔍 正在获取 %s 组织仓库...", org)
	orgRepos, err := getOrgRepos(token)
	if err != nil {
		log.Printf("⚠️  获取组织仓库失败: %v", err)
	} else {
		for _, repo := range orgRepos {
			if !repo.Fork {
				stats.TotalStars += repo.StargazersCount
				stats.TotalForks += repo.ForksCount
				stats.TotalRepos++
			}
		}
		log.Printf("📦 组织: %d 仓库, ⭐ %d stars, 🔀 %d forks", stats.TotalRepos, stats.TotalStars, stats.TotalForks)
	}

	// 2. 获取个人仓库统计（合并）
	log.Printf("🔍 正在获取 %s 个人仓库...", user)
	userRepos, err := getUserRepos(token)
	if err != nil {
		log.Printf("⚠️  获取个人仓库失败: %v", err)
	} else {
		for _, repo := range userRepos {
			if !repo.Fork {
				stats.TotalStars += repo.StargazersCount
				stats.TotalForks += repo.ForksCount
				stats.TotalRepos++
			}
		}
		log.Printf("📦 合计: %d 仓库, ⭐ %d stars, 🔀 %d forks", stats.TotalRepos, stats.TotalStars, stats.TotalForks)
	}

	// 3. 获取个人贡献数据（GraphQL）
	if token != "" {
		log.Printf("📊 正在获取 %s 贡献数据...", user)
		contrib, err := getUserContributions(token)
		if err != nil {
			log.Printf("⚠️  获取贡献数据失败: %v (需要 token 有 read:user 权限)", err)
		} else {
			uc := contrib.Data.User.ContributionsCollection
			stats.TotalCommits = uc.TotalCommitContributions
			stats.TotalPRs = uc.TotalPullRequestContributions
			stats.TotalIssues = uc.TotalIssueContributions
			stats.TotalPRReviews = uc.TotalPullRequestReviewContributions
			stats.TotalContributions = uc.ContributionCalendar.TotalContributions
			stats.ContributedTo = contrib.Data.User.RepositoriesContributedTo.TotalCount
			stats.Followers = contrib.Data.User.Followers.TotalCount
			log.Printf("✅ Commits: %d, PRs: %d, Issues: %d, Reviews: %d",
				stats.TotalCommits, stats.TotalPRs, stats.TotalIssues, stats.TotalPRReviews)
		}
	} else {
		log.Println("⚠️  没有 token，跳过 GraphQL 查询（commits/PRs/issues 将为 0）")
	}

	// 4. 计算 rank
	calcRank(stats)
	log.Printf("🏆 Rank: %s (Top %.0f%%)", stats.Rank, stats.RankPercent)

	// 5. 生成 SVG
	log.Println("🎨 正在生成 SVG...")
	svg := generateSVG(stats)

	svgPath := filepath.Join(actualOutputDir, svgFileName)
	if err := os.MkdirAll(actualOutputDir, 0755); err != nil {
		log.Fatalf("❌ 创建目录失败: %v", err)
	}
	if err := os.WriteFile(svgPath, []byte(svg), 0644); err != nil {
		log.Fatalf("❌ 写入 SVG 失败: %v", err)
	}
	log.Printf("✅ SVG 已保存: %s", svgPath)

	// 6. 更新 README（--tmp 模式下跳过）
	if useTmp {
		log.Println("⏭️  --tmp 模式，跳过 README 更新")
	} else {
		readmePath := filepath.Join(actualOutputDir, "README.md")
		if err := updateReadme(readmePath, svgFileName); err != nil {
			log.Fatalf("❌ 更新 README 失败: %v", err)
		}
		log.Println("✅ README 已更新!")
	}

	// 打印总结
	log.Println("\n📊 GitHub Stats 总结：")
	log.Printf("  ⭐ Stars:     %d", stats.TotalStars)
	log.Printf("  📦 Repos:     %d", stats.TotalRepos)
	log.Printf("  🔀 Forks:     %d", stats.TotalForks)
	log.Printf("  📝 Commits:   %d", stats.TotalCommits)
	log.Printf("  🔃 PRs:       %d", stats.TotalPRs)
	log.Printf("  ❗ Issues:    %d", stats.TotalIssues)
	log.Printf("  👀 Reviews:   %d", stats.TotalPRReviews)
	log.Printf("  🤝 Contrib:   %d", stats.ContributedTo)
	log.Printf("  🏆 Rank:      %s (Top %.0f%%)", stats.Rank, stats.RankPercent)
}
