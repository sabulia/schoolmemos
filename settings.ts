// 校园版界面个性化与通用工具（本项目新增）。
import { useEffect, useState } from "react";
import { campusApi } from "./api";

export const CAMPUS_SECTIONS = [
  { key: "friends", label: "好友" },
  { key: "groups", label: "群" },
  { key: "circle", label: "圈" },
  { key: "topics", label: "话题区" },
  { key: "recommend", label: "推荐" },
  { key: "themes", label: "主题分类" },
  { key: "mine", label: "我的" },
] as const;

export type SectionKey = (typeof CAMPUS_SECTIONS)[number]["key"];

/** 界面风格：default 简洁 / image:<url> 静态壁纸 / animated:<名称> 动态画面。 */
export type SectionStyle = string;

const STYLE_CACHE_KEY = "campus.section_styles";

export function readLocalStyles(): Record<string, string> {
  try {
    return JSON.parse(localStorage.getItem(STYLE_CACHE_KEY) ?? "{}");
  } catch {
    return {};
  }
}

export function writeLocalStyle(section: string, style: string) {
  const styles = readLocalStyles();
  styles[section] = style;
  localStorage.setItem(STYLE_CACHE_KEY, JSON.stringify(styles));
}

/** 从服务端同步一次界面个性化设置到本地缓存。 */
export async function syncSectionStyles(): Promise<void> {
  try {
    const { settings } = await campusApi.getSettings();
    const styles = readLocalStyles();
    for (const s of CAMPUS_SECTIONS) {
      const v = settings[`section_style_${s.key}`];
      if (v) styles[s.key] = v;
    }
    localStorage.setItem(STYLE_CACHE_KEY, JSON.stringify(styles));
  } catch {
    // 未登录或网络失败时沿用本地缓存
  }
}

/** 读取某分区的界面风格，并返回应施加到页面容器的 style/className。 */
export function useSectionStyle(section: SectionKey): { className: string; style: React.CSSProperties } {
  const [raw, setRaw] = useState<string>(() => readLocalStyles()[section] ?? "default");
  useEffect(() => {
    const onStorage = () => setRaw(readLocalStyles()[section] ?? "default");
    window.addEventListener("storage", onStorage);
    window.addEventListener("campus-style-changed", onStorage);
    return () => {
      window.removeEventListener("storage", onStorage);
      window.removeEventListener("campus-style-changed", onStorage);
    };
  }, [section]);

  if (raw.startsWith("image:")) {
    return {
      className: "campus-section campus-section-image",
      style: { backgroundImage: `url(${raw.slice(6)})`, backgroundSize: "cover", backgroundAttachment: "fixed" },
    };
  }
  if (raw.startsWith("animated:")) {
    return { className: `campus-section campus-animated-${raw.slice(9) || "aurora"}`, style: {} };
  }
  return { className: "campus-section", style: {} };
}

// ---- 下载到本地（PWA） ----

/** 捕获浏览器 beforeinstallprompt，供「下载到本地」按钮触发安装。 */
let deferredInstallPrompt: (Event & { prompt?: () => Promise<void>; userChoice?: Promise<{ outcome: string }> }) | null = null;

export function setupInstallPromptCapture() {
  window.addEventListener("beforeinstallprompt", (e) => {
    e.preventDefault();
    deferredInstallPrompt = e;
  });
}

/** 尝试触发浏览器安装；返回 false 表示需要给用户手动指引。 */
export async function promptInstall(): Promise<boolean> {
  if (!deferredInstallPrompt?.prompt) return false;
  await deferredInstallPrompt.prompt();
  const choice = await deferredInstallPrompt.userChoice;
  deferredInstallPrompt = null;
  return choice?.outcome === "accepted";
}

/** 注册成功后「是否下载」只问一次。 */
export async function shouldAskDownload(): Promise<boolean> {
  try {
    const { settings } = await campusApi.getSettings();
    return settings["download_prompt_shown"] !== "1";
  } catch {
    return false;
  }
}

export async function markDownloadAsked(): Promise<void> {
  try {
    await campusApi.putSettings({ download_prompt_shown: "1" });
  } catch {
    // ignore
  }
}
