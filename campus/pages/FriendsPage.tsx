// 好友：参考微信（搜索添加、申请通过、好友列表、好友动态）。
import { UserPlus } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { toast } from "react-hot-toast";
import { campusApi, type CampusMemoCard, type FriendEntry } from "../api";
import MemoCardView from "../components/MemoCardView";
import { useSectionStyle } from "../settings";

const FriendsPage = () => {
  const [friends, setFriends] = useState<FriendEntry[]>([]);
  const [requests, setRequests] = useState<FriendEntry[]>([]);
  const [feed, setFeed] = useState<CampusMemoCard[]>([]);
  const [username, setUsername] = useState("");
  const shell = useSectionStyle("friends");

  const load = useCallback(async () => {
    try {
      const [{ friends, requests }, { memos }] = await Promise.all([campusApi.friends(), campusApi.feed("friends")]);
      setFriends(friends);
      setRequests(requests);
      setFeed(memos);
    } catch (e) {
      toast.error((e as Error).message);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const addFriend = async () => {
    if (!username.trim()) return;
    try {
      await campusApi.friendRequest(username.trim());
      toast.success("好友申请已发送");
      setUsername("");
    } catch (e) {
      toast.error((e as Error).message);
    }
  };

  const accept = async (name: string) => {
    try {
      await campusApi.friendAccept(name);
      toast.success("已添加为好友");
      load();
    } catch (e) {
      toast.error((e as Error).message);
    }
  };

  return (
    <div className={`mx-auto w-full max-w-2xl space-y-4 ${shell.className}`} style={shell.style}>
      <div className="flex gap-2">
        <input
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && addFriend()}
          placeholder="输入用户名添加好友"
          className="h-9 flex-1 rounded-lg border border-border bg-background px-3 text-sm outline-none focus:border-primary"
        />
        <button onClick={addFriend} className="flex items-center gap-1 rounded-lg bg-primary px-3 text-sm text-primary-foreground">
          <UserPlus className="h-4 w-4" />
          添加
        </button>
      </div>

      {requests.length > 0 && (
        <div className="rounded-xl border border-border bg-card p-4">
          <h3 className="mb-2 text-sm font-medium">新的朋友</h3>
          {requests.map((r) => (
            <div key={r.id} className="flex items-center justify-between py-1.5 text-sm">
              <span>{r.user.nickname || r.user.username}</span>
              <button onClick={() => accept(r.user.username)} className="rounded-full bg-primary px-3 py-0.5 text-xs text-primary-foreground">
                通过
              </button>
            </div>
          ))}
        </div>
      )}

      <div className="rounded-xl border border-border bg-card p-4">
        <h3 className="mb-2 text-sm font-medium">我的好友（{friends.length}）</h3>
        {friends.length === 0 && <p className="py-2 text-sm text-muted-foreground">还没有好友，先搜索用户名添加吧</p>}
        <div className="flex flex-wrap gap-2">
          {friends.map((f) => (
            <span key={f.id} className="flex items-center gap-1.5 rounded-full bg-muted px-3 py-1 text-sm">
              <span className="flex h-5 w-5 items-center justify-center rounded-full bg-primary/10 text-xs text-primary">
                {(f.user.nickname || "?").slice(0, 1)}
              </span>
              {f.user.nickname || f.user.username}
            </span>
          ))}
        </div>
      </div>

      <h3 className="text-sm font-medium">好友动态</h3>
      {feed.length === 0 && <p className="py-6 text-center text-sm text-muted-foreground">好友还没有动态</p>}
      {feed.map((m) => (
        <MemoCardView key={m.uid} memo={m} />
      ))}
    </div>
  );
};

export default FriendsPage;
