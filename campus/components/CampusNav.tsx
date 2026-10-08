// 校园版顶部导航（仿抖音顶部「推荐/关注/商城」栏）。
import { Search } from "lucide-react";
import { useState } from "react";
import { NavLink, useNavigate } from "react-router-dom";
import useCurrentUser from "@/hooks/useCurrentUser";
import { CAMPUS_SECTIONS } from "../settings";

const CampusNav = () => {
  const currentUser = useCurrentUser();
  const navigate = useNavigate();
  const [q, setQ] = useState("");
  if (!currentUser) return null;

  const onSearch = (e: React.FormEvent) => {
    e.preventDefault();
    if (q.trim()) navigate(`/campus/search?q=${encodeURIComponent(q.trim())}`);
  };

  return (
    <header className="sticky top-0 z-40 w-full border-b border-border bg-background/95 backdrop-blur">
      <div className="mx-auto flex h-12 w-full max-w-5xl items-center gap-2 px-3">
        <span className="mr-1 shrink-0 text-sm font-bold text-primary">校园Memo</span>
        <nav className="flex flex-1 items-center gap-1 overflow-x-auto">
          {CAMPUS_SECTIONS.map((s) => (
            <NavLink
              key={s.key}
              to={`/campus/${s.key}`}
              className={({ isActive }) =>
                `shrink-0 rounded-full px-3 py-1 text-sm transition-colors ${
                  isActive ? "bg-primary text-primary-foreground font-medium" : "text-muted-foreground hover:text-foreground"
                }`
              }
            >
              {s.label}
            </NavLink>
          ))}
        </nav>
        <form onSubmit={onSearch} className="relative hidden sm:block">
          <Search className="pointer-events-none absolute left-2 top-1.5 h-4 w-4 text-muted-foreground" />
          <input
            value={q}
            onChange={(e) => setQ(e.target.value)}
            placeholder="搜索内容 / 输入「下载」"
            className="h-8 w-44 rounded-full border border-border bg-muted/50 pl-7 pr-2 text-sm outline-none focus:w-56 focus:border-primary transition-all"
          />
        </form>
      </div>
    </header>
  );
};

export default CampusNav;
