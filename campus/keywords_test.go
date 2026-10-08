package campus

import "testing"

func TestExtractKeywordsChinese(t *testing.T) {
	content := "今天在三号教学楼捡到一张校园卡，校园卡上贴着动漫贴纸，请失主联系我认领 #失物招领"
	kws := ExtractKeywords(content, 8)
	if len(kws) == 0 {
		t.Fatal("expected keywords, got none")
	}
	found := map[string]bool{}
	for _, k := range kws {
		found[k.Text] = true
	}
	if !found["失物招领"] {
		t.Errorf("expected explicit tag 失物招领 to be extracted, got %v", kws)
	}
	if !found["校园卡"] {
		t.Errorf("expected 校园卡 (repeated n-gram) to rank high, got %v", kws)
	}
}

func TestExtractKeywordsEnglish(t *testing.T) {
	kws := ExtractKeywords("Anyone lost an iPhone near the library? iPhone with blue case", 5)
	found := map[string]bool{}
	for _, k := range kws {
		found[k.Text] = true
	}
	if !found["iphone"] {
		t.Errorf("expected iphone keyword, got %v", kws)
	}
}

func TestClassifyLostAndFound(t *testing.T) {
	content := "在食堂捡到一把雨伞和一张饭卡，请失主认领"
	kws := ExtractKeywords(content, 8)
	theme, score := Classify(content, kws)
	if theme != "失物招领" {
		t.Errorf("expected 失物招领, got %s (score %f)", theme, score)
	}
	if score <= 0 {
		t.Errorf("expected positive score, got %f", score)
	}
}

func TestClassifyStudy(t *testing.T) {
	content := "考研打卡第 30 天，今天在图书馆刷题背英语单词，坚持就是胜利"
	kws := ExtractKeywords(content, 8)
	theme, _ := Classify(content, kws)
	if theme != "学习打卡" {
		t.Errorf("expected 学习打卡, got %s", theme)
	}
}

func TestClassifyFallback(t *testing.T) {
	content := "zzzz ~~~"
	kws := ExtractKeywords(content, 8)
	theme, _ := Classify(content, kws)
	if theme != DefaultTheme {
		t.Errorf("expected fallback theme %s, got %s", DefaultTheme, theme)
	}
}
