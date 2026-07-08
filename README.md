# Fred — `fred.cash` CLI

Anonymous, no-accounts CLI. Users run one line on a Mac and they're connected:

```bash
curl -fsSL https://get.fred.cash | bash
interest
```

## Architecture (as deployed)

- **`get.fred.cash`** → Vercel → serves the install script (`install.sh`)
- **`api.fred.cash`** → Vercel serverless Go function → the API the CLI calls
- **`fred.cash`** (apex) → a separate Vercel project (the Plinko game), untouched
- **binaries** → GitHub Releases on `mfmp17/interest` (arm64 + Intel)
- **DNS** → Namecheap (two A records → Vercel), no Cloudflare

```
get.fred.cash ──┐
                ├─► Vercel project "site"  ──► install.sh + /api/status
api.fred.cash ──┘
fred.cash (apex) ─► Vercel project "fred-cash" (Plinko game — separate)
```

## What's in this repo

```
fred/
├── cli/main.go         # the `fred.cash` command (Go, single static binary)
├── api/main.go         # standalone API server (for local dev)
├── web/install.sh      # source of the installer (edit here)
├── site/               # the Vercel deployment unit
│   ├── vercel.json     # host-based routing (see below)
│   ├── public/install.sh   # copy of web/install.sh, served by Vercel
│   └── api/status.go   # serverless function -> api.fred.cash/v1/status
├── build.sh            # cross-compiles CLI for arm64 + amd64 -> ./dist
├── dist/               # release binaries (uploaded to GitHub Releases)
├── README.md           # this file
└── DEPLOY.md           # step-by-step deploy / update guide
```

### How routing works (`site/vercel.json`)

One Vercel project ("site") owns both subdomains. Rewrites route by host/path:

- `get.fred.cash/`         → `/install.sh`  (host-based rewrite)
- `get.fred.cash/interest` → `/install.sh`  (alias, still works)
- `api.fred.cash/v1/status`→ `/api/status`  (the Go function)

The `install.sh` served at `get.fred.cash` is served as `text/plain` so
`curl … | bash` works.

## Local dev / testing

```bash
# terminal 1 — run the standalone API
cd api && go build -o fred-api . && PORT=8080 ./fred-api

# terminal 2 — point the CLI at local API and run it
cd cli && go build -o fred.cash .
INTEREST_API=http://localhost:8080 ./interest
INTEREST_API=http://localhost:8080 ./fred.cash status
```

The CLI defaults to `https://api.fred.cash` (see `apiBase()` in `cli/main.go`);
override with the `INTEREST_API` env var during development.

## Shipping a new CLI version

```bash
cd ~/fred
./build.sh 0.2.0
gh release create v0.2.0 dist/interest_darwin_arm64 dist/interest_darwin_amd64 \
    --title v0.2.0 --notes "what changed"
```

The installer always pulls the `latest` release, so users re-running the curl
line get the new binary automatically — no change to `install.sh` needed.

## Updating the installer or the API

```bash
# edit web/install.sh, then sync + deploy:
cp web/install.sh site/public/install.sh
cd site && vercel --prod

# edit site/api/status.go, then:
cd site && vercel --prod
```

See `DEPLOY.md` for the full deploy story and troubleshooting.


## Updating

```bash
fred.cash update
```

This downloads the latest GitHub Release asset for your Mac and keeps the legacy `interest` alias.
