import "@github/relative-time-element";
import { QueryClientProvider } from "@tanstack/react-query";
import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import React, { useEffect, useRef } from "react";
import { createRoot } from "react-dom/client";
import { Toaster } from "react-hot-toast";
import { RouterProvider } from "react-router-dom";
import "./i18n";
import "./index.css";
import { ErrorBoundary } from "@/components/ErrorBoundary";
import { TooltipProvider } from "@/components/ui/tooltip";
import { refreshAccessToken } from "@/connect";
import { AuthProvider, useAuth } from "@/contexts/AuthContext";
import { InstanceProvider, useInstance } from "@/contexts/InstanceContext";
import { useLiveMemoRefresh } from "@/hooks/useLiveMemoRefresh";
import { useTokenRefreshOnFocus } from "@/hooks/useTokenRefreshOnFocus";
import { queryClient } from "@/lib/query-client";
import router from "./router";
import { setupInstallPromptCapture, syncSectionStyles } from "./campus/settings";
import { applyLocaleEarly } from "./utils/i18n";
import { applyThemeEarly } from "./utils/theme";

// Apply theme and locale early to prevent flash
applyThemeEarly();
applyLocaleEarly();

// 校园版（本项目新增）：注册 PWA Service Worker + 捕获「下载到桌面」安装提示。
if ("serviceWorker" in navigator) {
  window.addEventListener("load", () => {
    navigator.serviceWorker.register("/sw.js").catch(() => {});
  });
}
setupInstallPromptCapture();

// Inner component that initializes contexts
function AppInitializer({ children }: { children: React.ReactNode }) {
  const { isIdentityInitialized, initialize: initAuth, currentUser } = useAuth();
  const { isProfileInitialized, initialize: initInstance } = useInstance();
  const initStartedRef = useRef(false);

  // Initialize on mount - run in parallel for better performance
  useEffect(() => {
    if (initStartedRef.current) return;
    initStartedRef.current = true;

    const init = async () => {
      await Promise.all([initInstance(), initAuth()]);
      // 校园版：登录后同步各分区界面个性化设置。
      void syncSectionStyles();
    };
    init();
  }, [initAuth, initInstance]);

  // Proactively refresh token on window focus to prevent 401 errors
  // Only enabled when user is authenticated
  // Related: https://github.com/usememos/memos/issues/5589
  useTokenRefreshOnFocus(refreshAccessToken, !!currentUser);

  // Live refresh: listen for memo changes via SSE and invalidate caches.
  useLiveMemoRefresh();

  // Route loading and feed requests only need the verified identity and the
  // instance profile. Display-sensitive settings continue in the background;
  // PagedMemoList keeps memo content hidden until privacy settings have settled.
  if (!isIdentityInitialized || !isProfileInitialized) {
    return null;
  }

  return <>{children}</>;
}

function Main() {
  return (
    <ErrorBoundary>
      <QueryClientProvider client={queryClient}>
        <InstanceProvider>
          <AuthProvider>
            <TooltipProvider>
              <AppInitializer>
                <RouterProvider router={router} />
                <Toaster
                  position="top-right"
                  toastOptions={{
                    style: {
                      background: "var(--popover)",
                      color: "var(--popover-foreground)",
                    },
                  }}
                />
              </AppInitializer>
            </TooltipProvider>
          </AuthProvider>
        </InstanceProvider>
        <ReactQueryDevtools initialIsOpen={false} />
      </QueryClientProvider>
    </ErrorBoundary>
  );
}

const container = document.getElementById("root");
const root = createRoot(container as HTMLElement);
root.render(<Main />);
