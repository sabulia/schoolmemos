// 圈：全校动态流（参考 QQ 动态 / 朋友圈 / Instagram / 推特）。
import { useCallback, useEffect, useState } from "react";
import { campusApi, type CampusMemoCard } from "../api";
import CampusComposer from "../components/CampusComposer";
import MemoCardView from "../components/MemoCardView";
import { useSectionStyle } from "../settings";

const CirclePage = () => {
  const [memos, setMemos] = useState<CampusMemoCard[]>([]);
  const [error, setError] = useState("");
  const shell = useSectionStyle("circle");

  const load = useCallback(async () => {
    try {
      const { memos } = await campusApi.feed("circle");
      setMemos(memos);
    } catch (e) {
      setError((e as Error).message);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  return (
    <div className={`mx-auto w-full max-w-2xl space-y-4 ${shell.className}`} style={shell.style}>
      <CampusComposer defaultVisibility="PUBLIC" onPosted={load} />
      {error && <p className="text-sm text-destructive">{error}</p>}
      {memos.length === 0 && !error && <p className="py-12 text-center text-sm text-muted-foreground">圈里还没有动态，来发第一条吧</p>}
      {memos.map((m) => (
        <MemoCardView key={m.uid} memo={m} />
      ))}
    </div>
  );
};

export default CirclePage;
