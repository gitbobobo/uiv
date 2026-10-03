// Runs the Go backend and the Vite dev server together. Open the Vite URL; it proxies /api and /f.
import { spawn } from "node:child_process";
import "./ensure-dist.mjs";

const env = {
  ...process.env,
  UIV_TOKEN: process.env.UIV_TOKEN ?? "dev",
  UIV_DATA_DIR: process.env.UIV_DATA_DIR ?? ".dev-data",
  UIV_ADDR: process.env.UIV_ADDR ?? "127.0.0.1:8080",
};

const children = [
  spawn("go", ["run", "./cmd/uiv"], { env, stdio: "inherit" }),
  spawn("pnpm", ["--filter", "web", "dev"], { env, stdio: "inherit", shell: process.platform === "win32" }),
];
console.log(`UIV dev: token is "${env.UIV_TOKEN}", data in ${env.UIV_DATA_DIR}`);

function stop() {
  for (const c of children) c.kill();
}
for (const c of children) {
  c.on("exit", (code) => {
    stop();
    process.exit(code ?? 0);
  });
}
process.on("SIGINT", stop);
process.on("SIGTERM", stop);
