package campus

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ============================================================
// 关键词提取与主题分类（本文件全部为本项目新增代码）
//
// 算法说明：
//  1. 分词：英文/数字按单词切分；中文按连续 CJK 片段生成 2~4 元 n-gram，
//     通过「频度 + 词长 + 包含抑制」选出候选词（无需外部词典，零依赖）。
//  2. 显式标签：内容中的 #标签 直接计入关键词并加权。
//  3. 分类：内置 10 个校园主题词库，按「关键词—主题词相关性」打分，
//     得分最高的主题作为该 memo 的主题分类。
// ============================================================

// Theme 主题分类定义。
type Theme struct {
	Name     string   `json:"name"`
	Keywords []string `json:"keywords"` // 主题词库
}

// Themes 是内置的校园主题词库（可在后续版本中做成实例级配置）。
var Themes = []Theme{
	{Name: "失物招领", Keywords: []string{"丢失", "捡到", "失物", "招领", "寻物", "一卡通", "校园卡", "钥匙", "钱包", "耳机", "雨伞", "身份证", "饭卡", "认领", "失主"}},
	{Name: "学习打卡", Keywords: []string{"打卡", "自习", "背单词", "考研", "图书馆", "晨读", "刷题", "笔记", "复习", "学习计划", "坚持", "早起", "英语", "四级", "六级"}},
	{Name: "求助问答", Keywords: []string{"求助", "请问", "怎么", "如何", "有没有", "谁知道", "解答", "帮忙", "大佬", "求带", "答疑"}},
	{Name: "二手交易", Keywords: []string{"出", "收", "转让", "二手", "闲置", "便宜出", "九成新", "教材", "自提", "小刀", "包邮", "转手"}},
	{Name: "社团活动", Keywords: []string{"社团", "招新", "学生会", "协会", "活动", "排练", "路演", "纳新", "社员", "晚会", "比赛", "志愿者"}},
	{Name: "通知公告", Keywords: []string{"通知", "公告", "教务", "选课", "停水", "停电", "放假", "开学", "报到", "奖学金", "公示", " deadline"}},
	{Name: "考试升学", Keywords: []string{"考试", "期末", "期中", "绩点", "保研", "挂科", "补考", "成绩", "学分", "考研", "复试", "分数线"}},
	{Name: "生活分享", Keywords: []string{"食堂", "宿舍", "外卖", "天气", "猫", "校园", "操场", "晚霞", "奶茶", "日常", "分享", "好吃"}},
	{Name: "兴趣圈子", Keywords: []string{"摄影", "骑行", "篮球", "足球", "游戏", "音乐", "电影", "读书", "动漫", "健身", "跑步", "羽毛球"}},
	{Name: "其他", Keywords: []string{}},
}

// DefaultTheme 是无法归入任何主题时的兜底分类。
const DefaultTheme = "其他"

var (
	tagRegexp       = regexp.MustCompile(`#([^#\s，。！？；、,.!?;:'"【】\[\]()（）]{1,30})`)
	asciiWordRegexp = regexp.MustCompile(`[A-Za-z][A-Za-z0-9_\-]{1,31}`)
)

// 常见停用词（单字与高频虚词片段），用于过滤 n-gram 噪声。
var stopGrams = map[string]bool{
	"我们": true, "你们": true, "他们": true, "自己": true, "这个": true, "那个": true,
	"什么": true, "怎么": false, // 「怎么」保留，它是求助问答主题词
	"就是": true, "不是": true, "没有": false, "可以": true, "已经": true,
	"今天": false, // 「今天」保留，日程相关常用
	"的":  true, "了": true, "吗": true, "呢": true, "吧": true, "啊": true,
}

// Keyword 带权重关键词。
type Keyword struct {
	Text   string  `json:"text"`
	Weight float64 `json:"weight"`
}

// isCJK 判断字符是否为中日韩统一表意文字。
func isCJK(r rune) bool {
	return unicode.Is(unicode.Han, r)
}

// ExtractKeywords 从文本中提取 Top-K 关键词。
func ExtractKeywords(content string, topK int) []Keyword {
	if topK <= 0 {
		topK = 8
	}
	scores := map[string]float64{}

	// 1) 显式 #标签，权重最高。
	for _, m := range tagRegexp.FindAllStringSubmatch(content, -1) {
		tag := strings.TrimSpace(m[1])
		if tag != "" {
			scores[tag] += 10.0
		}
	}

	// 2) 英文单词 / 数字词。
	for _, w := range asciiWordRegexp.FindAllString(content, -1) {
		lw := strings.ToLower(w)
		if len(lw) >= 2 {
			scores[lw] += 2.0
		}
	}

	// 3) 中文 n-gram（2~3 元）。
	runes := []rune(stripMarkdown(content))
	var cjkRun []rune
	flush := func() {
		if len(cjkRun) >= 2 {
			countNgrams(cjkRun, scores)
		}
		cjkRun = cjkRun[:0]
	}
	for _, r := range runes {
		if isCJK(r) {
			cjkRun = append(cjkRun, r)
		} else {
			flush()
		}
	}
	flush()

	// 4) 停用词过滤 + 排序 + 子串去重（保留权重更高者）。
	kws := make([]Keyword, 0, len(scores))
	for text, w := range scores {
		if stopGrams[text] {
			continue
		}
		if utf8.RuneCountInString(text) < 2 && !isASCIIWord(text) {
			continue
		}
		kws = append(kws, Keyword{Text: text, Weight: w})
	}
	sort.Slice(kws, func(i, j int) bool {
		if kws[i].Weight == kws[j].Weight {
			// 权重相同优先长词（信息更完整），再按字典序保证确定性。
			li, lj := utf8.RuneCountInString(kws[i].Text), utf8.RuneCountInString(kws[j].Text)
			if li == lj {
				return kws[i].Text < kws[j].Text
			}
			return li > lj
		}
		return kws[i].Weight > kws[j].Weight
	})
	picked := make([]Keyword, 0, topK)
	for _, kw := range kws {
		dup := false
		for _, p := range picked {
			// 已选词与候选词互为子串时，保留权重更高的先选者。
			if strings.Contains(p.Text, kw.Text) || strings.Contains(kw.Text, p.Text) {
				dup = true
				break
			}
		}
		if dup {
			continue
		}
		picked = append(picked, kw)
		if len(picked) >= topK {
			break
		}
	}
	return picked
}

func isASCIIWord(s string) bool {
	for _, r := range s {
		if r > 127 {
			return false
		}
	}
	return len(s) >= 2
}

// countNgrams 对一个连续 CJK 片段生成 2~3 元 n-gram 并计分。
// 评分 = 出现次数 × 词长奖励 × 领域加成（命中主题词库的候选词加权 3 倍：
// 校园场景主题明确，用词库引导可显著降低 n-gram 噪声）。
func countNgrams(run []rune, scores map[string]float64) {
	counts := map[string]int{}
	for _, n := range []int{2, 3} {
		if len(run) < n {
			continue
		}
		for i := 0; i+n <= len(run); i++ {
			counts[string(run[i:i+n])]++
		}
	}
	for gram, c := range counts {
		gramLen := utf8.RuneCountInString(gram)
		weight := float64(c) * (1.0 + 0.5*float64(gramLen-2))
		switch lexiconMatch(gram) {
		case 2: // 词库精确命中：最高加成
			weight *= 5.0
		case 1: // 与词库词互相包含：中等加成
			weight *= 2.0
		}
		scores[gram] += weight
	}
}

// lexiconMatch 返回候选词与主题词库的匹配强度：2=精确命中，1=互相包含，0=未命中。
func lexiconMatch(gram string) int {
	result := 0
	for _, theme := range Themes {
		for _, tw := range theme.Keywords {
			tw = strings.TrimSpace(tw)
			if tw == "" {
				continue
			}
			if gram == tw {
				return 2
			}
			if len(gram) >= 2 && (strings.Contains(tw, gram) || strings.Contains(gram, tw)) {
				result = 1
			}
		}
	}
	return result
}

// markdown 语法噪声剥离（链接、图片、代码块标记等）。
var markdownNoise = regexp.MustCompile("([!\\[\\]()*_`>~|-]{1,}|https?://\\S+)")

func stripMarkdown(s string) string {
	return markdownNoise.ReplaceAllString(s, " ")
}

// Classify 按关键词与主题词库的相关性对内容分类。
// 返回主题名与得分；得分 <= 0 时返回 DefaultTheme。
func Classify(content string, keywords []Keyword) (string, float64) {
	text := content
	for _, kw := range keywords {
		text += " " + kw.Text
	}
	best, bestScore := DefaultTheme, 0.0
	for _, theme := range Themes {
		if theme.Name == DefaultTheme {
			continue
		}
		score := 0.0
		for _, tw := range theme.Keywords {
			tw = strings.TrimSpace(tw)
			if tw == "" {
				continue
			}
			// 直接命中内容：强相关。
			if strings.Contains(text, tw) {
				score += 2.0
				continue
			}
			// 与提取出的关键词互相包含：弱相关。
			for _, kw := range keywords {
				if strings.Contains(kw.Text, tw) || strings.Contains(tw, kw.Text) {
					score += 1.0 * (kw.Weight / 5.0)
				}
			}
		}
		if score > bestScore {
			best, bestScore = theme.Name, score
		}
	}
	return best, bestScore
}
