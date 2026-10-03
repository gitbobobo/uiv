import { execFileSync } from "node:child_process";

let version = process.env.UIV_VERSION;
if (!version) {
  try {
    version = execFileSync("git", ["describe", "--tags", "--always", "--dirty"], { encoding: "utf8" }).trim();
  } catch {
    version = "dev";
  }
}

execFileSync("go", ["build", "-trimpath", "-ldflags", `-s -w -X main.version=${version}`, "-o", "bin/", "./cmd/uiv"], {
  stdio: "inherit",
});
console.log(`built bin/uiv (${version})`);
