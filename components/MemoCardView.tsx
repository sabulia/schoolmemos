// 校园版通用 memo 卡片（圈/推荐/话题/主题分类共用）。
import { CalendarClock, MapPin, Star } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { campusApi, type CampusMemoCard } from "../api";

const VISIBILITY_LABEL: Record<string, string> = {
  PRIVATE: "仅自己可见",
  FRIENDS: "仅好友可见",
  PROTECTED: "登录可见",
  PUBLIC: "完全公开",
};

const MemoCardView = ({ memo, showVisibility = false }: { memo: CampusMemoCard; showVisibility?: boolean }) => {
  const [favorited, setFavorited] = useState(false);
  const browsedRef = useRef(false);

  // 卡片进入视口时记录一次浏览（用于「我的-浏览记录」与推荐画像）。
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const el = ref.current;
    if (!el || browsedRef.current) return;
    const ob = new IntersectionObserver(
      (entries) => {
        if (entries[0]?.isIntersecting && !browsedRef.current) {
          browsedRef.current = true;
          campusApi.browse(memo.uid).catch(() => {});
          ob.disconnect();
        }
      },
      { threshold: 0.6 },
    );
    ob.observe(el);
    return () => ob.disconnect();
  }, [memo.uid]);

  const onFavorite = async () => {
    try {
      const { favorited: f } = await campusApi.toggleFavorite(memo.uid);
      setFavorited(f);
    } catch {
      // ignore
    }
  };

  return (
    <div ref={ref} className="rounded-xl border border-border bg-card p-4 shadow-sm">
      <div className="mb-2 flex items-center gap-2">
        {memo.anonymous && !memo.owned ? (
          <span className="flex h-7 w-7 items-center justify-center rounded-full bg-muted text-xs">匿</span>
        ) : (
          <span className="flex h-7 w-7 items-center justify-center rounded-full bg-primary/10 text-xs text-primary">
            {(memo.creator?.nickname || "?").slice(0, 1)}
          </span>
        )}
        <span className="text-sm font-medium">{memo.anonymous && !memo.owned ? "匿名同学" : memo.creator?.nickname || "未知"}</span>
        <span className="text-xs text-muted-foreground">{new Date(memo.createTime).toLocaleString("zh-CN")}</span>
        {showVisibility && <span className="ml-auto rounded bg-muted px-1.5 py-0.5 text-xs">{VISIBILITY_LABEL[memo.visibility]}</span>}
      </div>
      <Link to={`/memos/${memo.uid}`} className="block whitespace-pre-wrap text-sm leading-6 hover:text-primary">
        {memo.content}
      </Link>
      <div className="mt-2 flex flex-wrap items-center gap-1.5 text-xs">
        {memo.theme && <span className="rounded-full bg-primary/10 px-2 py-0.5 text-primary">{memo.theme}</span>}
        {memo.keywords?.slice(0, 5).map((k) => (
          <span key={k} className="rounded-full bg-muted px-2 py-0.5 text-muted-foreground">
            {k}
          </span>
        ))}
        {memo.location && (
          <span className="flex items-center gap-0.5 text-muted-foreground">
            <MapPin className="h-3 w-3" />
            {memo.location}
          </span>
        )}
        {memo.scheduleTs ? (
          <span className="flex items-center gap-0.5 text-amber-600">
            <CalendarClock className="h-3 w-3" />
            {new Date(memo.scheduleTs * 1000).toLocaleString("zh-CN")}
            {memo.remindMinutes ? `（提前${memo.remindMinutes}分钟提醒）` : ""}
          </span>
        ) : null}
        <button onClick={onFavorite} className={`ml-auto flex items-center gap-0.5 ${favorited ? "text-amber-500" : "text-muted-foreground"}`}>
          <Star className="h-3.5 w-3.5" fill={favorited ? "currentColor" : "none"} />
          {favorited ? "已收藏" : "收藏"}
        </button>
      </div>
    </div>
  );
};

export default MemoCardView;
