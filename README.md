# Fred — `fred.cash` CLI

Anonymous, no-accounts CLI. Users run one line on a Mac and they're connected:

```bash
curl -fsSL https://get.fred.cash | bash
fred.cash
```

## Architecture (as deployed)

- **`get.fred.cash`** → Vercel → serves the install script (`install.sh`)
- **`api.fred.cash`** → Railway persistent Go backend + watcher
- **`fred.cash`** (apex) → a separate Vercel project (the Plinko game), untouched
- **binaries** → GitHub Releases on `mfmp17/interest` (arm64 + Intel)
- **DNS** → Namecheap (two A records → Vercel), no Cloudflare

```
get.fred.cash ───► Vercel project "site" ───► install.sh
api.fred.cash ───► Railway "fred-backend" ──► API + Base watcher + volume
fred.cash (apex) ─► Vercel project "fred-cash" (Plinko game — separate)
```

## What's in this repo

```
fred/
├── cli/main.go         # the `fred.cash` command (Go, single static binary)
├── backend/            # persistent custodial API, watcher, payouts, state
├── web/install.sh      # source of the installer (edit here)
├── site/               # the Vercel deployment unit
│   ├── vercel.json     # host-based routing (see below)
│   └── public/install.sh   # installer served at get.fred.cash
├── build.sh            # cross-compiles CLI for arm64 + amd64 -> ./dist
├── dist/               # release binaries (uploaded to GitHub Releases)
├── README.md           # this file
└── DEPLOY.md           # step-by-step deploy / update guide
```

### How routing works (`site/vercel.json`)

One Vercel project ("site") owns both subdomains. Rewrites route by host/path:

- `get.fred.cash/`         → `/install.sh`  (host-based rewrite)
- `get.fred.cash/interest` → `/install.sh`  (alias, still works)

`api.fred.cash` is not routed through this Vercel project; it points to Railway.

The `install.sh` served at `get.fred.cash` is served as `text/plain` so
`curl … | bash` works.

## Local dev / testing

```bash
# terminal 1 — run the standalone API
cd backend && go build -o fred-backend . && PORT=8080 ./fred-backend

# terminal 2 — point the CLI at local API and run it
cd cli && go build -o fred.cash .
INTEREST_API=http://localhost:8080 ./fred.cash
INTEREST_API=http://localhost:8080 ./fred.cash status
```

The CLI defaults to `https://api.fred.cash` (see `apiBase()` in `cli/main.go`);
override with the `INTEREST_API` env var during development.

## Shipping a new CLI version

```bash
cd ~/fred
./build.sh 0.3.0
gh release create v0.3.0 \
    dist/fred.cash_darwin_arm64 dist/fred.cash_darwin_amd64 \
    dist/interest_darwin_arm64 dist/interest_darwin_amd64 \
    --title v0.3.0 --notes "what changed"
```

The installer always pulls the `latest` release, so users re-running the curl
line get the new binary automatically — no change to `install.sh` needed.

## Updating the installer or backend

```bash
# edit web/install.sh, then sync + deploy:
cp web/install.sh site/public/install.sh
cd site && vercel --prod

# deploy backend from the backend directory:
cd backend && railway up --detach --service fred-backend
```

See `DEPLOY.md` for the full deploy story and troubleshooting.


## Updating

```bash
fred.cash update
```

This downloads the latest GitHub Release asset for your Mac and keeps the legacy `interest` alias.

## Operations and support

```bash
fred.cash doctor                 # API, treasury, scanner, receipt, deposit state
fred.cash support                # redacted JSON support bundle
fred.cash positions              # list local anonymous positions
fred.cash use <position-id>      # select active position (prefix accepted)
fred.cash receipt export [path]  # private backup containing claim tokens
fred.cash receipt import <path>  # restore/merge a backup
```
