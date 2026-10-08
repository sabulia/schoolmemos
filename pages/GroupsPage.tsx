// 群：复用原项目的 Space（群组/圈子空间），界面参考微信群列表。
import { Plus, Users } from "lucide-react";
import { useState } from "react";
import { Link } from "react-router-dom";
import { toast } from "react-hot-toast";
import useCurrentUser from "@/hooks/useCurrentUser";
import { useCreateSpace, useSpaces } from "@/hooks/useSpaceQueries";
import { useSectionStyle } from "../settings";

const GroupsPage = () => {
  const currentUser = useCurrentUser();
  const viewerName = currentUser?.name;
  const { data: spaces, isLoading } = useSpaces(viewerName);
  const createSpace = useCreateSpace(viewerName ?? "");
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const shell = useSectionStyle("groups");

  const create = async () => {
    if (!title.trim()) {
      toast.error("请输入群名称");
      return;
    }
    try {
      await createSpace.mutateAsync({ title: title.trim(), description: description.trim() || undefined, spaceId: crypto.randomUUID() });
      toast.success("群已创建");
      setTitle("");
      setDescription("");
    } catch (e) {
      toast.error((e as Error).message);
    }
  };

  return (
    <div className={`mx-auto w-full max-w-2xl space-y-4 ${shell.className}`} style={shell.style}>
      <div className="rounded-xl border border-border bg-card p-4">
        <h3 className="mb-2 text-sm font-medium">创建群</h3>
        <div className="flex flex-col gap-2 sm:flex-row">
          <input
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder="群名称（如 24 级软件 3 班）"
            className="h-9 flex-1 rounded-lg border border-border bg-background px-3 text-sm outline-none focus:border-primary"
          />
          <input
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="群简介（可选）"
            className="h-9 flex-1 rounded-lg border border-border bg-background px-3 text-sm outline-none focus:border-primary"
          />
          <button onClick={create} className="flex items-center justify-center gap-1 rounded-lg bg-primary px-4 text-sm text-primary-foreground">
            <Plus className="h-4 w-4" />
            创建
          </button>
        </div>
      </div>

      <div className="rounded-xl border border-border bg-card">
        <h3 className="border-b border-border p-4 text-sm font-medium">我的群</h3>
        {isLoading && <p className="p-4 text-sm text-muted-foreground">加载中…</p>}
        {!isLoading && (spaces ?? []).length === 0 && <p className="p-4 text-sm text-muted-foreground">还没有加入任何群，创建一个或让同学邀请你</p>}
        {(spaces ?? []).map((space) => {
          const uid = space.name.split("/").pop() ?? "";
          return (
            <Link
              key={space.name}
              to={`/spaces/${uid}`}
              className="flex items-center gap-3 border-b border-border p-4 last:border-0 hover:bg-muted/50"
            >
              <span className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
                <Users className="h-5 w-5" />
              </span>
              <span className="flex-1">
                <span className="block text-sm font-medium">{space.title || uid}</span>
                {space.description && <span className="block text-xs text-muted-foreground">{space.description}</span>}
              </span>
              <span className="text-xs text-muted-foreground">进入 ›</span>
            </Link>
          );
        })}
      </div>
      <p className="text-center text-xs text-muted-foreground">群内动态与原项目 Space 完全打通，点进群即可发布与浏览</p>
    </div>
  );
};

export default GroupsPage;
