// 话题区：#话题 热榜 + 话题内帖子流（参考小红书 / 贴吧）。
import { Flame } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { campusApi, type CampusMemoCard, type TagInfo } from "../api";
import CampusComposer from "../components/CampusComposer";
import MemoCardView from "../components/MemoCardView";
import { useSectionStyle } from "../settings";

const TopicsPage = () => {
  const [tags, setTags] = useState<TagInfo[]>([]);
  const [activeTag, setActiveTag] = useState("");
  const [memos, setMemos] = useState<CampusMemoCard[]>([]);
  const shell = useSectionStyle("topics");

  useEffect(() => {
    campusApi
      .feed("topics")
      .then((d) => setTags(d.tags ?? []))
      .catch(() => {});
  }, []);

  const openTag = useCallback(async (tag: string) => {
    setActiveTag(tag);
    const d = await campusApi.feed("topics", `&tag=${encodeURIComponent(tag)}`);
    setMemos(d.memos);
  }, []);

  return (
    <div className={`mx-auto w-full max-w-2xl space-y-4 ${shell.className}`} style={shell.style}>
      {!activeTag && (
        <div className="rounded-xl border border-border bg-card p-4">
          <h3 className="mb-3 flex items-center gap-1 text-sm font-medium">
            <Flame className="h-4 w-4 text-orange-500" />
            话题热榜
          </h3>
          {tags.length === 0 && <p className="text-sm text-muted-foreground">还没有话题，发布时带上 #话题 即可创建</p>}
          <div className="flex flex-col gap-1">
            {tags.map((t, i) => (
              <button key={t.tag} onClick={() => openTag(t.tag)} className="flex items-center gap-2 rounded-lg px-2 py-1.5 text-left text-sm hover:bg-muted">
                <span className={`w-5 text-center text-xs ${i < 3 ? "font-bold text-orange-500" : "text-muted-foreground"}`}>{i + 1}</span>
                <span className="flex-1">#{t.tag}</span>
                <span className="text-xs text-muted-foreground">{t.count} 条讨论</span>
              </button>
            ))}
          </div>
        </div>
      )}

      {activeTag && (
        <>
          <div className="flex items-center gap-2">
            <button onClick={() => setActiveTag("")} className="text-sm text-muted-foreground hover:text-foreground">
              ← 返回热榜
            </button>
            <h3 className="text-base font-semibold">#{activeTag}</h3>
          </div>
          <CampusComposer defaultVisibility="PUBLIC" onPosted={() => openTag(activeTag)} />
          {memos.length === 0 && <p className="py-8 text-center text-sm text-muted-foreground">该话题下还没有内容</p>}
          {memos.map((m) => (
            <MemoCardView key={m.uid} memo={m} />
          ))}
        </>
      )}
    </div>
  );
};

export default TopicsPage;
