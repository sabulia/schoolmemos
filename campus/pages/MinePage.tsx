// 我的：日期表（日历）、我的发布、浏览记录、收藏、设置（隐私/界面/下载/导出）。
import { CalendarDays, Download, FileDown, Settings2, ShieldCheck } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { toast } from "react-hot-toast";
import useCurrentUser from "@/hooks/useCurrentUser";
import { campusApi, type CampusMemoCard } from "../api";
import MemoCardView from "../components/MemoCardView";
import { CAMPUS_SECTIONS, promptInstall, readLocalStyles, useSectionStyle, writeLocalStyle } from "../settings";

type Tab = "calendar" | "memos" | "history" | "favorites" | "settings";

const PRIVACY_ITEMS = [
  { key: "privacy_show_calendar", label: "日期表" },
  { key: "privacy_show_memos", label: "我发布的 memo" },
  { key: "privacy_show_history", label: "浏览记录" },
  { key: "privacy_show_favorites", label: "我的收藏" },
];

const ANIMATED_OPTIONS = [
  { value: "animated:aurora", label: "动态·极光" },
  { value: "animated:ocean", label: "动态·海洋" },
  { value: "animated:sunset", label: "动态·日落" },
];

const MinePage = () => {
  const currentUser = useCurrentUser();
  const [tab, setTab] = useState<Tab>("calendar");
  const [mine, setMine] = useState<CampusMemoCard[]>([]);
  const [history, setHistory] = useState<CampusMemoCard[]>([]);
  const [favorites, setFavorites] = useState<CampusMemoCard[]>([]);
  const [settings, setSettings] = useState<Record<string, string>>({});
  const [styles, setStyles] = useState<Record<string, string>>(() => readLocalStyles());
  const shell = useSectionStyle("mine");

  const load = useCallback(async () => {
    try {
      const [m, h, f, s] = await Promise.all([campusApi.mine(), campusApi.history(), campusApi.favorites(), campusApi.getSettings()]);
      setMine(m.memos);
      setHistory(h.memos);
      setFavorites(f.memos);
      setSettings(s.settings);
    } catch (e) {
      toast.error((e as Error).message);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  // ---- 日期表：当月日历，标注有发布/日程的日期 ----
  const [month, setMonth] = useState(() => {
    const d = new Date();
    return new Date(d.getFullYear(), d.getMonth(), 1);
  });
  const dayMarks = useMemo(() => {
    const marks = new Map<string, { posts: number; schedules: number }>();
    for (const m of mine) {
      const day = m.createTime.slice(0, 10);
      const entry = marks.get(day) ?? { posts: 0, schedules: 0 };
      entry.posts++;
      if (m.scheduleTs) entry.schedules++;
      marks.set(day, entry);
    }
    return marks;
  }, [mine]);

  const calendarCells = useMemo(() => {
    const year = month.getFullYear();
    const m = month.getMonth();
    const first = new Date(year, m, 1);
    const days = new Date(year, m + 1, 0).getDate();
    const cells: ({ day: number; key: string } | null)[] = [];
    for (let i = 0; i < first.getDay(); i++) cells.push(null);
    for (let d = 1; d <= days; d++) {
      const key = `${year}-${String(m + 1).padStart(2, "0")}-${String(d).padStart(2, "0")}`;
      cells.push({ day: d, key });
    }
    return cells;
  }, [month]);

  const togglePrivacy = async (key: string) => {
    const next = settings[key] === "1" ? "0" : "1";
    setSettings((s) => ({ ...s, [key]: next }));
    await campusApi.putSettings({ [key]: next }).catch(() => {});
  };

  const applyStyle = async (section: string, value: string) => {
    const next = { ...styles, [section]: value };
    setStyles(next);
    writeLocalStyle(section, value);
    window.dispatchEvent(new Event("campus-style-changed"));
    await campusApi.putSettings({ [`section_style_${section}`]: value }).catch(() => {});
  };

  const onInstall = async (platform: "mobile" | "desktop") => {
    const ok = await promptInstall();
    if (!ok) {
      toast(
        platform === "mobile"
          ? "手机端：用浏览器菜单选择「添加到主屏幕」即可安装"
          : "电脑端：点击浏览器地址栏右侧的「安装」图标，或菜单中选择「安装应用」",
        { duration: 5000 },
      );
    } else {
      toast.success("已开始安装到桌面");
    }
  };

  const tabs: { key: Tab; label: string }[] = [
    { key: "calendar", label: "日期表" },
    { key: "memos", label: "我的发布" },
    { key: "history", label: "浏览记录" },
    { key: "favorites", label: "收藏" },
    { key: "settings", label: "设置" },
  ];

  return (
    <div className={`mx-auto w-full max-w-2xl space-y-4 ${shell.className}`} style={shell.style}>
      <div className="flex items-center gap-3 rounded-xl border border-border bg-card p-4">
        <span className="flex h-12 w-12 items-center justify-center rounded-full bg-primary/10 text-lg text-primary">
          {(currentUser?.displayName || currentUser?.username || "?").slice(0, 1)}
        </span>
        <div>
          <p className="font-medium">{currentUser?.displayName || currentUser?.username}</p>
          <p className="text-xs text-muted-foreground">@{currentUser?.username}</p>
        </div>
      </div>

      <div className="flex gap-1 overflow-x-auto rounded-xl border border-border bg-card p-1">
        {tabs.map((t) => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            className={`flex-1 whitespace-nowrap rounded-lg px-3 py-1.5 text-sm ${tab === t.key ? "bg-primary text-primary-foreground" : "text-muted-foreground"}`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {tab === "calendar" && (
        <div className="rounded-xl border border-border bg-card p-4">
          <div className="mb-3 flex items-center justify-between">
            <button className="text-sm" onClick={() => setMonth(new Date(month.getFullYear(), month.getMonth() - 1, 1))}>
              ← 上月
            </button>
            <h3 className="flex items-center gap-1 text-sm font-medium">
              <CalendarDays className="h-4 w-4" />
              {month.getFullYear()} 年 {month.getMonth() + 1} 月
            </h3>
            <button className="text-sm" onClick={() => setMonth(new Date(month.getFullYear(), month.getMonth() + 1, 1))}>
              下月 →
            </button>
          </div>
          <div className="grid grid-cols-7 gap-1 text-center text-xs">
            {["日", "一", "二", "三", "四", "五", "六"].map((d) => (
              <div key={d} className="py-1 text-muted-foreground">
                {d}
              </div>
            ))}
            {calendarCells.map((cell, i) => {
              if (!cell) return <div key={`empty-${i}`} />;
              const mark = dayMarks.get(cell.key);
              return (
                <div key={cell.key} className={`rounded-lg py-1.5 ${mark ? "bg-primary/10 font-medium text-primary" : ""}`}>
                  {cell.day}
                  {mark?.schedules ? <div className="mx-auto mt-0.5 h-1 w-1 rounded-full bg-amber-500" /> : null}
                </div>
              );
            })}
          </div>
          <a href="/api/campus/calendar.ics" className="mt-3 block text-center text-xs text-primary underline">
            导出我的日程到系统日历（ICS，含到时提醒）
          </a>
        </div>
      )}

      {tab === "memos" && (
        <div className="space-y-3">
          {mine.length === 0 && <p className="py-8 text-center text-sm text-muted-foreground">还没有发布过内容</p>}
          {mine.map((m) => (
            <MemoCardView key={m.uid} memo={m} showVisibility />
          ))}
        </div>
      )}

      {tab === "history" && (
        <div className="space-y-3">
          {history.length === 0 && <p className="py-8 text-center text-sm text-muted-foreground">暂无浏览记录</p>}
          {history.map((m) => (
            <MemoCardView key={m.uid} memo={m} />
          ))}
        </div>
      )}

      {tab === "favorites" && (
        <div className="space-y-3">
          {favorites.length === 0 && <p className="py-8 text-center text-sm text-muted-foreground">暂无收藏</p>}
          {favorites.map((m) => (
            <MemoCardView key={m.uid} memo={m} />
          ))}
        </div>
      )}

      {tab === "settings" && (
        <div className="space-y-4">
          <section className="rounded-xl border border-border bg-card p-4">
            <h3 className="mb-2 flex items-center gap-1 text-sm font-medium">
              <ShieldCheck className="h-4 w-4" />
              隐私设置（选择个人主页公开内容）
            </h3>
            {PRIVACY_ITEMS.map((item) => (
              <label key={item.key} className="flex items-center justify-between py-1.5 text-sm">
                <span>{item.label}</span>
                <input type="checkbox" checked={settings[item.key] === "1"} onChange={() => togglePrivacy(item.key)} />
              </label>
            ))}
            <p className="mt-1 text-xs text-muted-foreground">勾选表示公开；每条 memo 的可见范围在发布时单独设置。</p>
          </section>

          <section className="rounded-xl border border-border bg-card p-4">
            <h3 className="mb-2 flex items-center gap-1 text-sm font-medium">
              <Settings2 className="h-4 w-4" />
              界面个性化（默认简洁，可按分区设置）
            </h3>
            {CAMPUS_SECTIONS.map((s) => {
              const current = styles[s.key] ?? "default";
              const isImage = current.startsWith("image:");
              return (
                <div key={s.key} className="flex flex-col gap-1 py-1.5 text-sm">
                  <div className="flex items-center justify-between gap-2">
                    <span>{s.label}</span>
                    <select
                      value={isImage ? "image" : current}
                      onChange={(e) => applyStyle(s.key, e.target.value === "image" ? "image:" : e.target.value)}
                      className="rounded-lg border border-border bg-background px-2 py-1 text-xs"
                    >
                      <option value="default">默认简洁</option>
                      <option value="image">静态图片壁纸</option>
                      {ANIMATED_OPTIONS.map((o) => (
                        <option key={o.value} value={o.value}>
                          {o.label}
                        </option>
                      ))}
                    </select>
                  </div>
                  {(isImage || current === "image:") && (
                    <input
                      defaultValue={isImage ? current.slice(6) : ""}
                      placeholder="输入图片 URL，回车生效（如 https://example.com/bg.jpg）"
                      onKeyDown={(e) => {
                        if (e.key === "Enter") applyStyle(s.key, `image:${(e.target as HTMLInputElement).value.trim()}`);
                      }}
                      className="rounded-lg border border-border bg-background px-2 py-1 text-xs"
                    />
                  )}
                </div>
              );
            })}
          </section>

          <section className="rounded-xl border border-border bg-card p-4">
            <h3 className="mb-2 flex items-center gap-1 text-sm font-medium">
              <Download className="h-4 w-4" />
              下载到本地
            </h3>
            <div className="flex gap-2">
              <button onClick={() => onInstall("mobile")} className="flex-1 rounded-lg bg-primary px-3 py-2 text-sm text-primary-foreground">
                手机端
              </button>
              <button onClick={() => onInstall("desktop")} className="flex-1 rounded-lg border border-border px-3 py-2 text-sm">
                电脑端
              </button>
            </div>
            <p className="mt-1 text-xs text-muted-foreground">安装后可直接从手机/电脑桌面打开，体验与 App 一致。</p>
          </section>

          <section className="rounded-xl border border-border bg-card p-4">
            <h3 className="mb-2 flex items-center gap-1 text-sm font-medium">
              <FileDown className="h-4 w-4" />
              数据导出（用于向学校汇报 / 成果展示）
            </h3>
            <div className="flex gap-2">
              <a href="/api/campus/export.json" className="flex-1 rounded-lg border border-border px-3 py-2 text-center text-sm">
                导出 JSON
              </a>
              <a href="/api/campus/export.csv" className="flex-1 rounded-lg border border-border px-3 py-2 text-center text-sm">
                导出 CSV
              </a>
            </div>
            <label className="mt-3 flex items-center justify-between text-sm">
              <span>个性化推荐（根据点赞/浏览推送）</span>
              <input
                type="checkbox"
                checked={settings["recommend_optin"] === "1"}
                onChange={() => togglePrivacy("recommend_optin")}
              />
            </label>
          </section>
        </div>
      )}
    </div>
  );
};

export default MinePage;
