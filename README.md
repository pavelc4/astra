<h1 align="center">
    Astra
</h1>
<p align="center">
    <img src="https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&colorA=363A4F&logo=go&logoColor=D9E0EE">
    <img src="https://img.shields.io/badge/Docker-2496ED?style=for-the-badge&colorA=363A4F&logo=docker&logoColor=D9E0EE">
    <img src="https://img.shields.io/badge/REST%20API-363A4F?style=for-the-badge&colorA=363A4F&logo=swagger&logoColor=D9E0EE">
</p>

---

## Astra Universal Downloader

Astra is a high-performance, concurrent-safe, and lightweight universal media downloader API backend written in Go. It supports extracting media (videos, photos, slides, and audio) from various major social media and sharing platforms through a single unified REST interface.

---

## API Documentation (Swagger / OpenAPI)

An OpenAPI 3.0 specification file is provided at [openapi.yaml](file:///home/sineva/prjkt/go/astra/openapi.yaml). You can:
1. Import [openapi.yaml](file:///home/sineva/prjkt/go/astra/openapi.yaml) directly into tools like **Postman**, **Insomnia**, or **Swagger Editor** to interactively test all endpoints.
2. Serve it locally using a Swagger UI docker container or ReDoc.

---

## Supported Platforms

1. **TikTok** (Videos, Slideshows, Music, Profiles)
2. **Instagram** (Reels/Videos, Posts/Photos, Profiles, Stories)
3. **Facebook** (Public Videos, Images, Group Content)
4. **Threads** (Posts, Videos, Photos)
5. **Reddit** (Videos, Images, Galleries)
6. **Twitter / X** (Videos, Photos)
7. **Pinterest** (Videos, Images)
8. **LinkedIn** (Videos, Public Images fallback crawler)
9. **TeraBox** (Files, Multi-file folders, and streaming m3u8 playlists)
10. **Spotify** (Track/Album metadata and downloads)
11. **Soundcloud** (Audio)
12. **CapCut** (Templates and videos)

---

## Architecture Overview

Astra follows Go best practices with an emphasis on concurrency safety and maintainability.

* **`cmd/server/`** — Entrypoint (`main.go`), route mapping (`routes.go`), and integration tests.
* **`internal/platform/`** — Independent platform scraper engines, isolated into subpackages (`meta`, `terabox`, `tiktok`, etc).
* **`internal/handler/`** — HTTP handlers wrapping platform execution logic via a generic `makeDownloadHandler` helper for uniform validation, execution, and JSON responses.
* **`internal/httpclient/`** — Tuned global `http.Client` with optimized connection pooling (`MaxIdleConnsPerHost = 100`) for connection reuse under scraping load.
* **`internal/types/`** — Shared data structures such as `DownloadItem` and `MediaType`.
* **`internal/errors/`** — Custom error types (`Validation`, `Upstream`, `RateLimit`) mapped to appropriate HTTP status codes.

**Design notes:**
* Go Generics unify 11+ handlers under a single template, reducing boilerplate.
* `context.Context` is propagated from the HTTP request down to every child scraper, enforcing timeouts and resource cleanup automatically.
* Graceful shutdown via `SIGINT`/`SIGTERM` interception, allowing in-flight requests to finish within a timeout window.
* Structured logging through Go's standard `slog`.
* Mutex-based rate-limit queues (e.g. for TikTok) are tuned to avoid blocking system threads under heavy traffic.

---

## Development Setup

### Prerequisites
- Go 1.20+

### Running Locally
```bash
go run ./cmd/server
```

### Building for Production
```bash
go build -ldflags="-s -w" -o astra ./cmd/server
```
This produces a stripped, optimized binary named `astra`.

---

## Configuration

Copy `.env.example` to `.env` and fill in the values:
```bash
cp .env.example .env
```

| Variable | Description | Default |
|---|---|---|
| `PORT` | Server listening port | `3000` |
| `HOST` | Server host interface | `0.0.0.0` |
| `INSTAGRAM_COOKIES` | Session cookies for Instagram API access | Optional |
| `FACEBOOK_COOKIES` | Session cookies for Facebook authenticated scraping (private/group posts, albums) | Optional |
| `PIXIV_COOKIES` | Session cookies for Pixiv authenticated scraping & R-18 access | Optional |

---

## Cookie Dumper Tool (`cmd/cookie-dumper`)

Astra includes a built-in CLI tool to extract session cookies directly from your browser (Firefox, Zen Browser, Chrome, Brave, Chromium) and update `.env` automatically:

```bash
# Build tool
go build -o cookie-dumper ./cmd/cookie-dumper

# Extract and write directly to .env
./cookie-dumper -platform instagram -write-env
./cookie-dumper -platform facebook -write-env
./cookie-dumper -platform pixiv -write-env
```

---

## Deployment

### Running as a systemd Service
```bash
sudo nano /etc/systemd/system/astra.service
```
```ini
[Unit]
Description=Astra Universal Downloader API Service
After=network.target

[Service]
Type=simple
User=sineva
WorkingDirectory=/home/sineva/prjkt/go/astra
ExecStart=/home/sineva/prjkt/go/astra/astra
Restart=always
RestartSec=5
Environment=PORT=3000
Environment=HOST=0.0.0.0

[Install]
WantedBy=multi-user.target
```
```bash
sudo systemctl daemon-reload
sudo systemctl enable astra
sudo systemctl start astra
sudo systemctl status astra
```

---

## Requirements
- **Go** – 1.20 or above
- **Linux/macOS/Windows** – cross-platform server binary

---

## License
Astra is open-sourced software licensed under the **MIT License**.
See the [LICENSE](LICENSE) file for more information.
