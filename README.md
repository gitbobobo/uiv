# UIV

Upload Image Video：部署在 NAS 上的图片/视频服务。代理把端到端截图、录屏上传上来，拿到公开链接贴进 PR，方便人和代理协作。

- 上传：`POST /api/files`，返回链接和可直接粘贴的 Markdown
- 访问：`/f/{id}.{ext}`，公开，无需 token
- 管理后台：浏览、预览、复制链接、删除、拖拽或粘贴上传
- 给代理用的说明：[`skills/uiv/SKILL.md`](skills/uiv/SKILL.md)

## 部署

```yaml
# compose.yaml
services:
  uiv:
    image: ghcr.io/gitbobobo/uiv:latest
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      UIV_TOKEN: "换成一个长随机串" # openssl rand -hex 32
    volumes:
      - ./data:/data
    # 不想以 root 运行时，指定数据目录的属主：
    # user: "1000:1000"
```

```bash
docker compose up -d
```

浏览器打开 `http://NAS地址:8080`，输入 `UIV_TOKEN` 进入管理后台。

### 配置

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `UIV_TOKEN` | 无，必填 | 上传、列表、删除用的 token。未设置时拒绝启动 |
| `UIV_PUBLIC_URL` | 空 | 生成链接用的外部地址，如 `https://uiv.example.com`。留空时按请求地址自动识别 |
| `UIV_MAX_SIZE` | `200MB` | 单文件上限，支持 `KB`/`MB`/`GB` |
| `UIV_DATA_DIR` | `/data` | 数据目录：`uiv.db` + `files/`。备份时复制整个目录 |
| `UIV_ADDR` | `:8080` | 监听地址 |

## 通过 frp 暴露到公网

GitHub 通过自己的服务器拉取 PR 里的图片，所以链接必须能从公网访问。局域网和普通 Tailscale 地址生成的链接在 PR 里会是裂图。

UIV 默认从请求地址推断链接，不需要配置域名：代理用哪个公网地址上传，返回的就是哪个地址的链接。

**方式一：TCP 端口（不需要域名）**

```toml
# frpc.toml
serverAddr = "frps 的公网 IP"
serverPort = 7000
auth.token = "frp 的 token"

[[proxies]]
name = "uiv"
type = "tcp"
localIP = "127.0.0.1"
localPort = 8080
remotePort = 7080
```

代理的 `UIV_URL` 设为 `http://frps公网IP:7080`。

**方式二：HTTP 虚拟主机（有域名时）**

```toml
[[proxies]]
name = "uiv"
type = "http"
localIP = "127.0.0.1"
localPort = 8080
customDomains = ["uiv.example.com"]
```

frps 需要配置 `vhostHTTPPort`。如果 frps 前面还有 nginx 或 Caddy 终止 HTTPS，确保它转发 `X-Forwarded-Proto`（Caddy 默认会；nginx 加上 `proxy_set_header X-Forwarded-Proto $scheme;` 和 `proxy_set_header Host $host;`），这样生成的链接才是 `https://`。

frpc 也跑在 compose 里时，把 `localIP` 换成 `uiv` 服务名。

## 给代理配置

在代理的环境里设置：

```bash
UIV_URL=http://frps公网IP:7080   # 必须是公网地址
UIV_TOKEN=与服务端一致
```

然后把 [`skills/uiv`](skills/uiv) 目录安装为代理的 skill。

## 开发

需要 Go 和 Node（pnpm）。

```bash
pnpm install
pnpm dev     # Go 后端 :8080 + Vite 前端，token 为 dev，数据在 .dev-data/
pnpm lint    # ESLint + gofmt + go vet
pnpm test    # go test
pnpm build   # 构建前端并编译 bin/uiv
```

架构说明见 [`docs/internal/architecture.md`](docs/internal/architecture.md)。

## 发布

推送 `v*` 标签（如 `v0.1.0`）后，GitHub Actions 构建 `linux/amd64` 和 `linux/arm64` 镜像，推送到 `ghcr.io/gitbobobo/uiv`，打上 `0.1.0`、`0.1` 和 `latest` 标签。

## 许可证

MIT
