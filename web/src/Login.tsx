import { useState, type FormEvent } from "react";
import { api, ApiError } from "./api";

type Props = {
  notice: string;
  onSignIn: (token: string) => void;
};

export default function Login({ notice, onSignIn }: Props) {
  const [value, setValue] = useState("");
  const [error, setError] = useState("");
  const [checking, setChecking] = useState(false);

  async function submit(e: FormEvent) {
    e.preventDefault();
    const token = value.trim();
    if (!token) return;
    setChecking(true);
    setError("");
    try {
      await api.list(token);
      onSignIn(token);
    } catch (err) {
      setError(
        err instanceof ApiError && err.status === 401
          ? "token 不正确。请和服务器上的 UIV_TOKEN 核对。"
          : `无法连接 UIV：${(err as Error).message}`,
      );
    } finally {
      setChecking(false);
    }
  }

  return (
    <main className="grid min-h-dvh place-items-center p-6">
      <form
        onSubmit={submit}
        className="w-full max-w-sm rounded-[3px] bg-mount p-6 shadow-[0_1px_0_var(--color-line),0_12px_32px_-16px_rgba(27,36,48,.45)]"
      >
        <h1 className="font-mono text-2xl font-semibold tracking-tight">UIV</h1>
        <p className="mt-1 text-sm text-muted">代理上传的截图和录屏都在这里。</p>

        <label htmlFor="token" className="mt-6 block text-sm font-medium">
          访问 token
        </label>
        <input
          id="token"
          type="password"
          autoComplete="current-password"
          autoFocus
          value={value}
          onChange={(e) => setValue(e.target.value)}
          placeholder="UIV_TOKEN"
          className="mt-1.5 w-full rounded-[3px] border border-line bg-white px-3 py-2 font-mono text-sm outline-none focus-visible:border-ink focus-visible:ring-2 focus-visible:ring-ink/15"
        />

        {(error || notice) && (
          <p role="alert" className="mt-3 text-sm text-pencil">
            {error || notice}
          </p>
        )}

        <button
          type="submit"
          disabled={checking || !value.trim()}
          className="mt-5 w-full rounded-[3px] bg-ink px-4 py-2 text-sm font-medium text-mount transition-opacity hover:opacity-90 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ink disabled:opacity-40"
        >
          {checking ? "正在验证…" : "打开灯箱"}
        </button>
      </form>
    </main>
  );
}
