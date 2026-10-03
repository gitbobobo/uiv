# UIV 架构

一个 Go 二进制，内嵌 React 管理后台，数据全部在一个目录里。

```
cmd/uiv/            入口：读配置、打开存储、启动 HTTP
internal/config/    环境变量解析
internal/media/     按文件头识别类型，决定是否接收
internal/store/     SQLite 元数据 + 磁盘文件
internal/server/    HTTP 路由：API、公开链接、静态资源
web/                React 管理后台（Vite + Tailwind），构建产物经 go:embed 打包
skills/uiv/         给代理用的 skill
```

## 数据

`UIV_DATA_DIR`（容器内 `/data`）下：

- `uiv.db`：SQLite，单表 `files(id, ext, name, size, type, created_at)`。
- `files/{id}.{ext}`：文件本体。
- `tmp/`：上传中的临时文件，启动时清空。

备份就是复制整个目录。

## 关键约束

- **链接**：`/f/{id}.{ext}`，`id` 是 10 位 base62 随机串（约 59 bit），不可猜测、不复用、不去重。因此文件响应可以带一年 `immutable` 缓存。
- **类型**：只接收图片和视频，按前 512 字节判断，不信任文件名和客户端 Content-Type。HTML 一律拒绝。SVG 允许，但响应带 `Content-Security-Policy: sandbox`，防止脚本在本域执行（管理后台的 token 在 localStorage 里）。所有文件响应带 `nosniff`。
- **鉴权**：只有一个 `UIV_TOKEN`，保护 `/api/files*`。`/f/*` 和 `/api/health` 公开。未设置 token 时拒绝启动。
- **链接域名**：数据库只存 ID，完整链接在每次请求时生成。设置了 `UIV_PUBLIC_URL` 就用它；否则按 `X-Forwarded-Proto/Host` → `Host` 推断。所以客户端用什么地址访问，就拿到什么地址的链接。代理必须通过公网（frp）地址上传；管理后台检测到私有地址时会提示。
- **上传**：multipart 流式写入临时文件再 rename，不整体读进内存。大小上限 `UIV_MAX_SIZE`。
- **生命周期**：永久保存，只能在管理后台手动删除。
- **前端状态**：不做乐观更新。列表只在服务端确认后变化（上传返回 201 后插入返回的记录，删除返回 204/404 后移除）。

## 构建与发布

- `pnpm build`：先构建 `web/dist`，再编译 `bin/uiv`，版本号来自 `git describe`。
- `web/dist` 不入库。Go 工具链需要它存在才能编译 `web` 包，`scripts/ensure-dist.mjs` 会在缺失时放一个占位页。
- 推送 `v*` 标签触发 `.github/workflows/release.yml`，构建 `linux/amd64`、`linux/arm64` 镜像推送到 `ghcr.io/gitbobobo/uiv`。Dockerfile 的构建阶段跑在构建机原生架构上，Go 交叉编译，最终镜像基于 `distroless/static`（默认 root）。
