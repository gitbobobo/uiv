// go:embed fails to compile when web/dist is missing; create a placeholder so Go tooling works
// before the first frontend build. A real `pnpm build` overwrites it.
import { existsSync, mkdirSync, writeFileSync } from "node:fs";

const dir = new URL("../web/dist/", import.meta.url);
if (!existsSync(new URL("index.html", dir))) {
  mkdirSync(dir, { recursive: true });
  writeFileSync(
    new URL("index.html", dir),
    "<!doctype html><title>UIV</title><p>Admin UI not built. Run <code>pnpm build</code>, or use <code>pnpm dev</code>.</p>\n",
  );
}
