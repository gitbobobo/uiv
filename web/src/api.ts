export type UivFile = {
  id: string;
  url: string;
  markdown: string;
  name: string;
  size: number;
  type: string;
  created_at: string;
};

export type FilePage = {
  files: UivFile[];
  next_cursor: string;
  base_url: string;
};

export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

async function request<T>(path: string, token: string | null, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  if (token) headers.set("Authorization", `Bearer ${token}`);
  const res = await fetch(path, { ...init, headers });
  if (!res.ok) {
    const body = await res.json().catch(() => null);
    throw new ApiError(res.status, body?.error ?? `请求失败（HTTP ${res.status}）`);
  }
  if (res.status === 204) return undefined as T;
  return res.json();
}

export const api = {
  list(token: string, cursor = "") {
    const q = cursor ? `?cursor=${encodeURIComponent(cursor)}` : "";
    return request<FilePage>(`/api/files${q}`, token);
  },
  upload(token: string, file: File) {
    const form = new FormData();
    form.append("file", file);
    return request<UivFile>("/api/files", token, { method: "POST", body: form });
  },
  remove(token: string, id: string) {
    return request<void>(`/api/files/${encodeURIComponent(id)}`, token, { method: "DELETE" });
  },
  health() {
    return request<{ version: string }>("/api/health", null);
  },
};
