// 校园版前端 API 封装（本目录全部为本项目新增代码）。
// 对接后端 /api/campus 系列接口；鉴权复用原项目的会话 Cookie。

export interface CampusUserLite {
  username: string;
  nickname: string;
  avatarUrl?: string;
}

export interface CampusMemoCard {
  uid: string;
  content: string;
  createTime: string;
  updateTime: string;
  visibility: "PRIVATE" | "FRIENDS" | "PROTECTED" | "PUBLIC";
  pinned: boolean;
  anonymous: boolean;
  location?: string;
  theme: string;
  keywords: string[];
  scheduleTs?: number;
  remindMinutes?: number;
  owned: boolean;
  creator?: CampusUserLite;
}

export interface ThemeInfo {
  name: string;
  count: number;
  keywords: string[];
}

export interface TagInfo {
  tag: string;
  count: number;
}

export interface FriendEntry {
  id: number;
  user: CampusUserLite;
  status: "PENDING" | "ACCEPTED";
  direction: "in" | "out";
  createdAt: string;
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const resp = await fetch(`/api/campus${path}`, {
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    ...init,
  });
  if (!resp.ok) {
    let message = `请求失败（${resp.status}）`;
    try {
      const body = await resp.json();
      if (body?.message) message = body.message;
    } catch {
      // ignore
    }
    throw new Error(message);
  }
  return (await resp.json()) as T;
}

export const campusApi = {
  // memo 与信息流
  createMemo: (body: {
    content: string;
    visibility: string;
    anonymous: boolean;
    location?: string;
    scheduleTs?: number;
    remindMinutes?: number;
  }) => request<CampusMemoCard>("/memos", { method: "POST", body: JSON.stringify(body) }),
  feed: (box: string, extra = "") => request<{ memos: CampusMemoCard[]; needOptIn?: boolean; tags?: TagInfo[] }>(`/feed?box=${box}${extra}`),
  themes: () => request<{ themes: ThemeInfo[] }>("/themes"),
  browse: (uid: string) => request<{ ok: boolean }>(`/memos/${uid}/browse`, { method: "POST", body: "{}" }),
  toggleFavorite: (uid: string) => request<{ favorited: boolean }>(`/memos/${uid}/favorite`, { method: "POST", body: "{}" }),
  favorites: () => request<{ memos: CampusMemoCard[] }>("/favorites"),
  history: () => request<{ memos: CampusMemoCard[] }>("/history"),
  mine: () => request<{ memos: CampusMemoCard[] }>("/mine"),

  // 好友
  friends: () => request<{ friends: FriendEntry[]; requests: FriendEntry[] }>("/friends"),
  friendRequest: (username: string) => request<{ ok: boolean }>("/friends/request", { method: "POST", body: JSON.stringify({ username }) }),
  friendAccept: (username: string) => request<{ ok: boolean }>("/friends/accept", { method: "POST", body: JSON.stringify({ username }) }),
  friendDelete: (friendId: number) => request<{ ok: boolean }>(`/friends/${friendId}`, { method: "DELETE" }),

  // 设置
  getSettings: () => request<{ settings: Record<string, string> }>("/settings"),
  putSettings: (kv: Record<string, string>) => request<{ ok: boolean }>("/settings", { method: "PUT", body: JSON.stringify(kv) }),

  // 注册 / 找回密码
  sendSms: (phone: string, purpose: "register" | "reset") =>
    request<{ sent: boolean; devCode?: string }>("/auth/sms", { method: "POST", body: JSON.stringify({ phone, purpose }) }),
  register: (body: { username: string; password: string; phone: string; code: string }) =>
    request<{ ok: boolean; username: string }>("/auth/register", { method: "POST", body: JSON.stringify(body) }),
  resetPassword: (body: { username: string; phone: string; code: string; newPassword: string }) =>
    request<{ ok: boolean }>("/auth/reset-password", { method: "POST", body: JSON.stringify(body) }),
};

export const VISIBILITY_OPTIONS = [
  { value: "PRIVATE", label: "仅自己可见" },
  { value: "FRIENDS", label: "仅好友可见" },
  { value: "PROTECTED", label: "登录可见" },
  { value: "PUBLIC", label: "完全公开" },
] as const;
