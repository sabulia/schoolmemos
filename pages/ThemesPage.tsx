// 主题分类：按关键词相关性自动归类（参考番茄免费小说的分类书架）。
import { useCallback, useEffect, useState } from "react";
import { campusApi, type CampusMemoCard, type ThemeInfo } from "../api";
import MemoCardView from "../components/MemoCardView";
import { useSectionStyle } from "../settings";

const ThemesPage = () => {
  const [themes, setThemes] = useState<ThemeInfo[]>([]);
  const [active, setActive] = useState("");
  const [memos, setMemos] = useState<CampusMemoCard[]>([]);
  const shell = useSectionStyle("themes");

  useEffect(() => {
    campusApi.themes().then((d) => setThemes(d.themes)).catch(() => {});
  }, []);

  const openTheme = useCallback(async (name: string) => {
    setActive(name);
    const d = await campusApi.feed("theme", `&theme=${encodeURIComponent(name)}`);
    setMemos(d.memos);
  }, []);

  if (active) {
    return (
      <div className={`mx-auto w-full max-w-2xl space-y-4 ${shell.className}`} style={shell.style}>
        <div className="flex items-center gap-2">
          <button onClick={() => setActive("")} className="text-sm text-muted-foreground hover:text-foreground">
            ← 返回分类
          </button>
          <h3 className="text-base font-semibold">{active}</h3>
        </div>
        {memos.length === 0 && <p className="py-8 text-center text-sm text-muted-foreground">该分类下还没有内容</p>}
        {memos.map((m) => (
          <MemoCardView key={m.uid} memo={m} />
        ))}
      </div>
    );
  }

  return (
    <div className={`mx-auto w-full max-w-2xl ${shell.className}`} style={shell.style}>
      <p className="mb-3 text-xs text-muted-foreground">系统自动提取每条内容的关键词，按相关性归入以下主题分类</p>
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
        {themes.map((t) => (
          <button
            key={t.name}
            onClick={() => openTheme(t.name)}
            className="rounded-xl border border-border bg-card p-4 text-left shadow-sm transition-shadow hover:shadow-md"
          >
            <div className="mb-1 flex items-center justify-between">
              <span className="font-medium">{t.name}</span>
              <span className="text-xs text-muted-foreground">{t.count}</span>
            </div>
            <div className="flex flex-wrap gap-1">
              {t.keywords.slice(0, 3).map((k) => (
                <span key={k} className="rounded bg-muted px-1.5 py-0.5 text-xs text-muted-foreground">
                  {k}
                </span>
              ))}
              {t.keywords.length === 0 && <span className="text-xs text-muted-foreground">暂无内容</span>}
            </div>
          </button>
        ))}
      </div>
    </div>
  );
};

export default ThemesPage;
