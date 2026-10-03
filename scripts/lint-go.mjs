import { execFileSync } from "node:child_process";
import "./ensure-dist.mjs";

const unformatted = execFileSync("gofmt", ["-l", "cmd", "internal", "web/embed.go"], { encoding: "utf8" }).trim();
if (unformatted) {
  console.error(`gofmt needed:\n${unformatted}`);
  process.exit(1);
}
execFileSync("go", ["vet", "./..."], { stdio: "inherit" });
