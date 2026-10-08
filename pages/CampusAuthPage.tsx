// 校园注册 / 找回密码：用户名 + 密码 + 手机号 + 短信验证。
// 注册成功后仅提示一次「是否下载到桌面」。
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { toast } from "react-hot-toast";
import { markDownloadAsked, promptInstall, shouldAskDownload } from "../settings";
import { campusApi } from "../api";

const CampusAuthPage = () => {
  const navigate = useNavigate();
  const [mode, setMode] = useState<"register" | "reset">("register");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [phone, setPhone] = useState("");
  const [code, setCode] = useState("");
  const [sending, setSending] = useState(false);
  const [busy, setBusy] = useState(false);

  const sendCode = async () => {
    if (!/^1\d{10}$/.test(phone)) {
      toast.error("请输入 11 位手机号");
      return;
    }
    setSending(true);
    try {
      const { devCode } = await campusApi.sendSms(phone, mode);
      if (devCode) {
        toast.success(`开发模式验证码：${devCode}`, { duration: 10000 });
      } else {
        toast.success("验证码已发送，请查收短信");
      }
    } catch (e) {
      toast.error((e as Error).message);
    } finally {
      setSending(false);
    }
  };

  const submit = async () => {
    setBusy(true);
    try {
      if (mode === "register") {
        await campusApi.register({ username: username.trim(), password, phone, code: code.trim() });
        toast.success("注册成功，请登录");
        // 注册成功后仅询问一次「是否下载到桌面」。
        if (await shouldAskDownload()) {
          await markDownloadAsked();
          const want = window.confirm("是否将「校园Memo」下载到手机/电脑桌面？（仅此次询问，之后可在 我的-设置-下载到本地 随时安装）");
          if (want) await promptInstall();
        }
        navigate("/auth");
      } else {
        await campusApi.resetPassword({ username: username.trim(), phone, code: code.trim(), newPassword: password });
        toast.success("密码已重置，请使用新密码登录");
        navigate("/auth");
      }
    } catch (e) {
      toast.error((e as Error).message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="mx-auto mt-10 w-full max-w-sm rounded-2xl border border-border bg-card p-6 shadow-sm">
      <div className="mb-4 flex rounded-lg bg-muted p-1 text-sm">
        <button onClick={() => setMode("register")} className={`flex-1 rounded-md py-1.5 ${mode === "register" ? "bg-background shadow" : ""}`}>
          注册
        </button>
        <button onClick={() => setMode("reset")} className={`flex-1 rounded-md py-1.5 ${mode === "reset" ? "bg-background shadow" : ""}`}>
          找回密码
        </button>
      </div>
      <div className="space-y-3">
        <input value={username} onChange={(e) => setUsername(e.target.value)} placeholder="用户名（不可重复）" className="h-10 w-full rounded-lg border border-border bg-background px-3 text-sm outline-none focus:border-primary" />
        <input value={phone} onChange={(e) => setPhone(e.target.value)} placeholder="手机号" maxLength={11} className="h-10 w-full rounded-lg border border-border bg-background px-3 text-sm outline-none focus:border-primary" />
        <div className="flex gap-2">
          <input value={code} onChange={(e) => setCode(e.target.value)} placeholder="短信验证码" maxLength={6} className="h-10 flex-1 rounded-lg border border-border bg-background px-3 text-sm outline-none focus:border-primary" />
          <button onClick={sendCode} disabled={sending} className="rounded-lg border border-border px-3 text-sm disabled:opacity-50">
            {sending ? "发送中…" : "获取验证码"}
          </button>
        </div>
        <input
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          placeholder={mode === "register" ? "密码（至少 6 位）" : "新密码（至少 6 位）"}
          className="h-10 w-full rounded-lg border border-border bg-background px-3 text-sm outline-none focus:border-primary"
        />
        <button onClick={submit} disabled={busy} className="h-10 w-full rounded-lg bg-primary text-sm text-primary-foreground disabled:opacity-50">
          {busy ? "提交中…" : mode === "register" ? "完成注册" : "重置密码"}
        </button>
        <p className="text-center text-xs text-muted-foreground">
          已有账号？<a href="/auth" className="text-primary">直接登录</a>
        </p>
      </div>
    </div>
  );
};

export default CampusAuthPage;
