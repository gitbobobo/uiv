import { useCallback, useEffect, useEffectEvent, useRef, useState, type DragEvent } from "react";
import { api, ApiError, type UivFile } from "./api";
import FileCard, { type CopyKind } from "./FileCard";
import Preview from "./Preview";
import { copyText, isPrivateHost } from "./util";

type Status = { tone: "info" | "error"; text: string } | null;

type Props = {
  token: string;
  onSignOut: (reason?: string) => void;
};

export default function Library({ token, onSignOut }: Props) {
  const [files, setFiles] = useState<UivFile[]>([]);
  const [nextCursor, setNextCursor] = useState("");
  const [baseUrl, setBaseUrl] = useState("");
  const [loaded, setLoaded] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [dragging, setDragging] = useState(false);
  const [status, setStatus] = useState<Status>(null);
  const [copied, setCopied] = useState("");
  const [preview, setPreview] = useState<UivFile | null>(null);
  const [version, setVersion] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);

  function fail(err: unknown) {
    if (err instanceof ApiError && err.status === 401) {
      onSignOut("token 已失效，请重新输入。");
      return;
    }
    setStatus({ tone: "error", text: (err as Error).message });
  }

  const onLoadError = useEffectEvent(fail);

  useEffect(() => {
    let cancelled = false;
    api
      .list(token)
      .then((page) => {
        if (cancelled) return;
        setFiles(page.files);
        setNextCursor(page.next_cursor);
        setBaseUrl(page.base_url);
        setLoaded(true);
      })
      .catch((err) => !cancelled && onLoadError(err));
    api
      .health()
      .then((h) => !cancelled && setVersion(h.version))
      .catch(() => {});
    return () => {
      cancelled = true;
    };
  }, [token]);

  async function loadMore() {
    setLoadingMore(true);
    try {
      const page = await api.list(token, nextCursor);
      setFiles((prev) => [...prev, ...page.files.filter((f) => !prev.some((p) => p.id === f.id))]);
      setNextCursor(page.next_cursor);
    } catch (err) {
      fail(err);
    } finally {
      setLoadingMore(false);
    }
  }

  async function upload(list: File[]) {
    if (list.length === 0 || uploading) return;
    setUploading(true);
    const failed: string[] = [];
    for (const [i, file] of list.entries()) {
      setStatus({
        tone: "info",
        text: list.length > 1 ? `正在上传 ${i + 1}/${list.length}：${file.name}` : `正在上传 ${file.name}…`,
      });
      try {
        const saved = await api.upload(token, file);
        setFiles((prev) => [saved, ...prev]);
      } catch (err) {
        if (err instanceof ApiError && err.status === 401) {
          fail(err);
          return;
        }
        failed.push(`${file.name}：${(err as Error).message}`);
      }
    }
    setUploading(false);
    if (failed.length > 0) {
      setStatus({ tone: "error", text: `${failed.length} 个文件上传失败。${failed.join("；")}` });
    } else {
      setStatus({ tone: "info", text: list.length > 1 ? `已上传 ${list.length} 个文件。` : `已上传 ${list[0].name}。` });
    }
  }

  const onPaste = useEffectEvent((e: ClipboardEvent) => {
    const pasted = Array.from(e.clipboardData?.files ?? []);
    if (pasted.length > 0) {
      e.preventDefault();
      upload(pasted);
    }
  });

  useEffect(() => {
    const handler = (e: ClipboardEvent) => onPaste(e);
    window.addEventListener("paste", handler);
    return () => window.removeEventListener("paste", handler);
  }, []);

  async function copy(file: UivFile, kind: CopyKind) {
    const key = `${file.id}:${kind}`;
    try {
      await copyText(kind === "url" ? file.url : file.markdown);
      setCopied(key);
      setTimeout(() => setCopied((c) => (c === key ? "" : c)), 1600);
    } catch (err) {
      setStatus({ tone: "error", text: `复制失败：${(err as Error).message}` });
    }
  }

  async function remove(file: UivFile) {
    if (!window.confirm(`删除「${file.name}」？\n删除后链接立即失效，贴过这个链接的地方会显示为裂图。`)) return;
    try {
      await api.remove(token, file.id);
    } catch (err) {
      // 404 means the server no longer has it, which is the state we want to show.
      if (!(err instanceof ApiError && err.status === 404)) {
        fail(err);
        return;
      }
    }
    setFiles((prev) => prev.filter((f) => f.id !== file.id));
    setPreview((p) => (p?.id === file.id ? null : p));
    setStatus({ tone: "info", text: `已删除 ${file.name}。` });
  }

  const closePreview = useCallback(() => setPreview(null), []);

  function onDragOver(e: DragEvent) {
    if (!e.dataTransfer.types.includes("Files")) return;
    e.preventDefault();
    setDragging(true);
  }

  function onDragLeave(e: DragEvent) {
    if (!e.currentTarget.contains(e.relatedTarget as Node | null)) setDragging(false);
  }

  function onDrop(e: DragEvent) {
    e.preventDefault();
    setDragging(false);
    upload(Array.from(e.dataTransfer.files));
  }

  const base = URL.canParse(baseUrl) ? new URL(baseUrl) : null;
  const host = base?.host ?? "";
  const privateLinks = base !== null && isPrivateHost(base.hostname);

  return (
    <div
      onDragOver={onDragOver}
      onDragLeave={onDragLeave}
      onDrop={onDrop}
      className={`flex min-h-dvh flex-col transition-colors duration-200 ${dragging ? "bg-table-lit" : ""}`}
    >
      <header className="flex flex-wrap items-center gap-x-4 gap-y-2 px-4 pt-5 pb-4 sm:px-8">
        <div className="flex min-w-0 items-baseline gap-3">
          <h1 className="font-mono text-xl font-semibold tracking-tight">UIV</h1>
          {host && <span className="truncate font-mono text-[11px] text-muted">{host}</span>}
        </div>
        <div className="ml-auto flex items-center gap-2">
          <input
            ref={inputRef}
            type="file"
            multiple
            accept="image/*,video/*"
            className="hidden"
            onChange={(e) => {
              upload(Array.from(e.target.files ?? []));
              e.target.value = "";
            }}
          />
          <button
            type="button"
            disabled={uploading}
            onClick={() => inputRef.current?.click()}
            className="rounded-[3px] bg-ink px-3.5 py-1.5 text-sm font-medium text-mount transition-opacity hover:opacity-90 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ink disabled:opacity-40"
          >
            {uploading ? "正在上传…" : "上传文件"}
          </button>
          <button
            type="button"
            onClick={() => onSignOut()}
            className="rounded-[3px] px-2.5 py-1.5 text-sm text-muted hover:text-ink focus-visible:outline-2 focus-visible:outline-ink"
          >
            退出
          </button>
        </div>
      </header>

      {privateLinks && (
        <p className="mx-4 mb-3 border-l-[3px] border-pencil bg-mount/70 px-3 py-2 text-sm sm:mx-8">
          当前链接只在内网能打开，贴到 GitHub 会显示为裂图。请通过 frp 公网地址访问本页，或在服务端设置 UIV_PUBLIC_URL。
        </p>
      )}

      {status && (
        <p
          role={status.tone === "error" ? "alert" : "status"}
          className={`mx-4 mb-3 text-sm sm:mx-8 ${status.tone === "error" ? "text-pencil" : "text-muted"}`}
        >
          {status.text}
        </p>
      )}

      <main className="flex-1 px-4 pb-10 sm:px-8">
        {loaded && files.length === 0 && (
          <div className="grid min-h-[50vh] place-items-center rounded-[3px] border-2 border-dashed border-line text-center">
            <div className="max-w-sm px-6">
              <p className="text-base font-medium">灯箱还是空的</p>
              <p className="mt-2 text-sm text-muted">
                把截图或录屏拖到这里，或者直接粘贴。代理可以按仓库里 skills/uiv 的说明自己上传。
              </p>
            </div>
          </div>
        )}

        {files.length > 0 && (
          <ul className="grid grid-cols-[repeat(auto-fill,minmax(250px,1fr))] gap-x-6 gap-y-7">
            {files.map((f) => (
              <FileCard key={f.id} file={f} copied={copied} onOpen={setPreview} onCopy={copy} onDelete={remove} />
            ))}
          </ul>
        )}

        {nextCursor && (
          <div className="mt-8 flex justify-center">
            <button
              type="button"
              disabled={loadingMore}
              onClick={loadMore}
              className="rounded-[3px] border border-line bg-mount px-4 py-1.5 text-sm hover:border-ink focus-visible:outline-2 focus-visible:outline-ink disabled:opacity-50"
            >
              {loadingMore ? "正在加载…" : "加载更早的文件"}
            </button>
          </div>
        )}
      </main>

      <footer className="px-4 pb-4 font-mono text-[10px] text-muted sm:px-8">UIV {version}</footer>

      {dragging && (
        <div className="pointer-events-none fixed inset-3 z-40 grid place-items-center rounded-[6px] border-[3px] border-dashed border-pencil">
          <p className="rounded-[3px] bg-mount px-4 py-2 text-sm font-medium text-pencil shadow">松开即可上传</p>
        </div>
      )}

      {preview && (
        <Preview file={preview} copied={copied} onClose={closePreview} onCopy={copy} onDelete={remove} />
      )}
    </div>
  );
}
