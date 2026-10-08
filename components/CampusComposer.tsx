// 校园版发布器：可见性四档 + 匿名开关 + 地点 + 日程提醒。
import { useState } from "react";
import { toast } from "react-hot-toast";
import { campusApi, VISIBILITY_OPTIONS } from "../api";

const CampusComposer = ({ defaultVisibility = "PUBLIC", onPosted }: { defaultVisibility?: string; onPosted?: () => void }) => {
  const [content, setContent] = useState("");
  const [visibility, setVisibility] = useState(defaultVisibility);
  const [anonymous, setAnonymous] = useState(false);
  const [location, setLocation] = useState("");
  const [scheduleAt, setScheduleAt] = useState("");
  const [remindMinutes, setRemindMinutes] = useState(30);
  const [posting, setPosting] = useState(false);

  const submit = async () => {
    if (!content.trim()) {
      toast.error("内容不能为空");
      return;
    }
    setPosting(true);
    try {
      await campusApi.createMemo({
        content: content.trim(),
        visibility,
        anonymous,
        location: location.trim() || undefined,
        scheduleTs: scheduleAt ? Math.floor(new Date(scheduleAt).getTime() / 1000) : undefined,
        remindMinutes: scheduleAt ? remindMinutes : undefined,
      });
      setContent("");
      setLocation("");
      setScheduleAt("");
      toast.success("发布成功");
      onPosted?.();
    } catch (e) {
      toast.error((e as Error).message);
    } finally {
      setPosting(false);
    }
  };

  return (
    <div className="rounded-xl border border-border bg-card p-4 shadow-sm">
      <textarea
        value={content}
        onChange={(e) => setContent(e.target.value)}
        rows={3}
        placeholder="分享此刻… 支持 #话题 标签，系统会自动提取关键词并归入主题分类"
        className="w-full resize-y rounded-lg border border-border bg-background p-2 text-sm outline-none focus:border-primary"
      />
      <div className="mt-2 flex flex-wrap items-center gap-2 text-sm">
        <select value={visibility} onChange={(e) => setVisibility(e.target.value)} className="rounded-lg border border-border bg-background px-2 py-1">
          {VISIBILITY_OPTIONS.map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </select>
        <label className="flex items-center gap-1 text-muted-foreground">
          <input type="checkbox" checked={anonymous} onChange={(e) => setAnonymous(e.target.checked)} />
          匿名发布
        </label>
        <input
          value={location}
          onChange={(e) => setLocation(e.target.value)}
          placeholder="地点（如 三号教学楼）"
          className="w-36 rounded-lg border border-border bg-background px-2 py-1"
        />
        <input
          type="datetime-local"
          value={scheduleAt}
          onChange={(e) => setScheduleAt(e.target.value)}
          title="日程时间（可选）"
          className="rounded-lg border border-border bg-background px-2 py-1"
        />
        {scheduleAt && (
          <label className="flex items-center gap-1 text-muted-foreground">
            提前
            <input
              type="number"
              min={0}
              value={remindMinutes}
              onChange={(e) => setRemindMinutes(Number(e.target.value))}
              className="w-16 rounded-lg border border-border bg-background px-2 py-1"
            />
            分钟提醒
          </label>
        )}
        <button
          onClick={submit}
          disabled={posting}
          className="ml-auto rounded-full bg-primary px-4 py-1.5 text-primary-foreground disabled:opacity-50"
        >
          {posting ? "发布中…" : "发布"}
        </button>
      </div>
    </div>
  );
};

export default CampusComposer;
