import type { UivFile } from "./api";
import { formatSize, formatTime, typeLabel } from "./util";

export type CopyKind = "url" | "markdown";

type ActionsProps = {
  file: UivFile;
  copied: string;
  onCopy: (file: UivFile, kind: CopyKind) => void;
  onDelete: (file: UivFile) => void;
};

export function Actions({ file, copied, onCopy, onDelete }: ActionsProps) {
  const btn =
    "rounded-[3px] px-1.5 py-1 text-xs whitespace-nowrap font-medium transition-colors focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-ink";
  return (
    <div className="flex flex-wrap gap-1">
      <button type="button" className={`${btn} bg-table/70 hover:bg-table`} onClick={() => onCopy(file, "url")}>
        {copied === `${file.id}:url` ? "已复制链接" : "复制链接"}
      </button>
      <button type="button" className={`${btn} bg-table/70 hover:bg-table`} onClick={() => onCopy(file, "markdown")}>
        {copied === `${file.id}:markdown` ? "已复制 Markdown" : "复制 Markdown"}
      </button>
      <button type="button" className={`${btn} ml-auto text-muted hover:bg-pencil/10 hover:text-pencil`} onClick={() => onDelete(file)}>
        删除
      </button>
    </div>
  );
}

export function Media({ file, preview = false }: { file: UivFile; preview?: boolean }) {
  if (file.type.startsWith("video/")) {
    return preview ? (
      <video src={file.url} controls autoPlay playsInline className="max-h-full max-w-full" />
    ) : (
      // The #t fragment makes browsers decode and show an early frame instead of a blank box.
      <video src={`${file.url}#t=0.1`} preload="metadata" muted playsInline className="h-full w-full object-contain" />
    );
  }
  return preview ? (
    <img src={file.url} alt={file.name} className="max-h-full max-w-full object-contain" />
  ) : (
    <img src={file.url} alt="" loading="lazy" decoding="async" className="h-full w-full object-contain" />
  );
}

type CardProps = {
  file: UivFile;
  copied: string;
  onOpen: (file: UivFile) => void;
  onCopy: (file: UivFile, kind: CopyKind) => void;
  onDelete: (file: UivFile) => void;
};

export default function FileCard({ file, copied, onOpen, onCopy, onDelete }: CardProps) {
  const isVideo = file.type.startsWith("video/");
  return (
    <li className="group relative">
      <article className="relative rounded-[3px] bg-mount p-2.5 pb-3 shadow-[0_1px_0_var(--color-line),0_8px_20px_-12px_rgba(27,36,48,.4)]">
        <button
          type="button"
          onClick={() => onOpen(file)}
          aria-label={`预览 ${file.name}`}
          className="checker relative block aspect-[4/3] w-full overflow-hidden rounded-[1px] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ink"
        >
          <Media file={file} />
          {isVideo && (
            <span className="absolute bottom-1.5 left-1.5 rounded-[2px] bg-ink/80 px-1.5 py-0.5 font-mono text-[10px] text-mount">
              ▶ 视频
            </span>
          )}
        </button>

        <div className="mt-2.5 flex items-baseline justify-between gap-2 font-mono text-[11px] tracking-wide">
          <span className="select-all">{file.id}</span>
          <span className="text-muted">{typeLabel(file.type)}</span>
        </div>
        <p className="mt-1 truncate text-sm" title={file.name}>
          {file.name}
        </p>
        <p className="mt-0.5 text-xs text-muted">
          {formatSize(file.size)} · {formatTime(file.created_at)}
        </p>
        <div className="mt-2.5">
          <Actions file={file} copied={copied} onCopy={onCopy} onDelete={onDelete} />
        </div>
      </article>
      <span className="pencil-ring" aria-hidden="true" />
    </li>
  );
}
