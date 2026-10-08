// 推荐：首次进入询问是否接受个性化推荐；同意后按点赞/收藏/浏览记录推送。
import { useCallback, useEffect, useState } from "react";
import { campusApi, type CampusMemoCard } from "../api";
import MemoCardView from "../components/MemoCardView";
import { useSectionStyle } from "../settings";

const RecommendPage = () => {
  const [needOptIn, setNeedOptIn] = useState<boolean | null>(null);
  const [memos, setMemos] = useState<CampusMemoCard[]>([]);
  const shell = useSectionStyle("recommend");

  const load = useCallback(async () => {
    const d = await campusApi.feed("recommend");
    setNeedOptIn(d.needOptIn ?? false);
    setMemos(d.memos);
  }, []);

  useEffect(() => {
    load().catch(() => setNeedOptIn(null));
  }, [load]);

  const agree = async () => {
    await campusApi.putSettings({ recommend_optin: "1" });
    await load();
  };

  if (needOptIn === null) {
    return <p className="py-12 text-center text-sm text-muted-foreground">加载中…</p>;
  }

  if (needOptIn) {
    return (
      <div className="mx-auto mt-16 w-full max-w-md rounded-2xl border border-border bg-card p-6 text-center shadow-sm">
        <h2 className="mb-2 text-lg font-semibold">开启个性化推荐？</h2>
        <p className="mb-4 text-sm leading-6 text-muted-foreground">
          开启后，我们会根据你点赞、收藏和浏览过的内容提取关键词，为你推送相关的校园内容与话题。
          你的行为数据仅用于推荐，可随时在「我的-设置」中关闭。
        </p>
        <div className="flex justify-center gap-3">
          <button onClick={agree} className="rounded-full bg-primary px-6 py-2 text-sm text-primary-foreground">
            同意并开启
          </button>
          <a href="/campus/circle" className="rounded-full border border-border px-6 py-2 text-sm">
            暂不使用
          </a>
        </div>
      </div>
    );
  }

  return (
    <div className={`mx-auto w-full max-w-2xl space-y-4 ${shell.className}`} style={shell.style}>
      <p className="text-xs text-muted-foreground">根据你的点赞、收藏与浏览记录推荐</p>
      {memos.length === 0 && <p className="py-12 text-center text-sm text-muted-foreground">还没有可推荐的内容，多去圈里逛逛吧</p>}
      {memos.map((m) => (
        <MemoCardView key={m.uid} memo={m} />
      ))}
    </div>
  );
};

export default RecommendPage;
