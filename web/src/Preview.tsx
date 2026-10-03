import { useEffect, useRef } from "react";
import type { UivFile } from "./api";
import { Actions, Media, type CopyKind } from "./FileCard";
import { formatSize, formatTime, typeLabel } from "./util";

type Props = {
  file: UivFile;
  copied: string;
  onClose: () => void;
  onCopy: (file: UivFile, kind: CopyKind) => void;
  onDelete: (file: UivFile) => void;
};

export default function Preview({ file, copied, onClose, onCopy, onDelete }: Props) {
  const closeRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    const previous = document.activeElement as HTMLElement | null;
    closeRef.current?.focus();
    const overflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      document.body.style.overflow = overflow;
      previous?.focus();
    };
  }, []);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && onClose();
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  return (
    <div role="dialog" aria-modal="true" aria-label={file.name} className="fixed inset-0 z-50 flex flex-col bg-ink/95">
      <div
        className="flex min-h-0 flex-1 items-center justify-center p-4 sm:p-10"
        onClick={(e) => e.target === e.currentTarget && onClose()}
      >
        <Media file={file} preview />
      </div>
      <footer className="flex flex-wrap items-center gap-x-5 gap-y-2 bg-mount px-4 py-3 sm:px-6">
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-medium" title={file.name}>
            {file.name}
          </p>
          <p className="mt-0.5 font-mono text-[11px] text-muted">
            {file.id} · {typeLabel(file.type)} · {formatSize(file.size)} · {formatTime(file.created_at)}
          </p>
        </div>
        <div className="flex items-center gap-3">
          <Actions file={file} copied={copied} onCopy={onCopy} onDelete={onDelete} />
          <a
            href={file.url}
            target="_blank"
            rel="noreferrer"
            className="text-xs font-medium underline decoration-line underline-offset-4 hover:decoration-ink"
          >
            打开原文件
          </a>
          <button
            ref={closeRef}
            type="button"
            onClick={onClose}
            className="rounded-[3px] bg-ink px-3 py-1 text-xs font-medium text-mount focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ink"
          >
            关闭
          </button>
        </div>
      </footer>
    </div>
  );
}
