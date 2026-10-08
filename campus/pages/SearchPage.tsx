// 校园搜索：搜「下载」时下载入口置顶；其余按内容匹配。
import { Download } from "lucide-react";
import { useEffect, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { campusApi, type CampusMemoCard } from "../api";
import MemoCardView from "../components/MemoCardView";

const SearchPage = () => {
  const [params] = useSearchParams();
  const q = params.get("q") ?? "";
  const [memos, setMemos] = useState<CampusMemoCard[]>([]);
  const isDownloadQuery = q.includes("下载");

  useEffect(() => {
    if (!q || isDownloadQuery) {
      setMemos([]);
      if (isDownloadQuery) return;
    }
    campusApi
      .feed("circle")
      .then((d) => setMemos(d.memos.filter((m) => m.content.includes(q))))
      .catch(() => {});
  }, [q, isDownloadQuery]);

  return (
    <div className="mx-auto w-full max-w-2xl space-y-4">
      <h3 className="text-sm text-muted-foreground">搜索：{q}</h3>
      {isDownloadQuery && (
        <div className="rounded-xl border-2 border-primary bg-card p-5 shadow">
          <h3 className="mb-1 flex items-center gap-1 font-medium text-primary">
            <Download className="h-4 w-4" />
            下载校园Memo 到本地（置顶推荐）
          </h3>
          <p className="mb-3 text-sm text-muted-foreground">安装后可从手机 / 电脑桌面直接打开，体验与 App 一致。</p>
          <div className="flex gap-2">
            <a href="/campus/mine" className="flex-1 rounded-lg bg-primary px-3 py-2 text-center text-sm text-primary-foreground">
              手机端安装
            </a>
            <a href="/campus/mine" className="flex-1 rounded-lg border border-border px-3 py-2 text-center text-sm">
              电脑端安装
            </a>
          </div>
        </div>
      )}
      {!isDownloadQuery && memos.length === 0 && <p className="py-8 text-center text-sm text-muted-foreground">没有找到相关内容</p>}
      {memos.map((m) => (
        <MemoCardView key={m.uid} memo={m} />
      ))}
    </div>
  );
};

export default SearchPage;
