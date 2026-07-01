# Fred — `interest` CLI

Anonymous, no-accounts CLI. Users run one line and they're connected:

```bash
curl -fsSL https://fred.cash/interest | bash
interest
```

## What's in this repo

```
fred/
├── cli/            # the `interest` command (Go)
│   └── main.go
├── api/            # backend API stub (Go) — serves /v1/status
│   └── main.go
├── web/
│   └── install.sh  # the script served at fred.cash/interest
├── build.sh        # cross-compiles CLI for arm64 + amd64 -> ./dist
└── dist/           # release binaries (upload these to GitHub Releases)
```

## Local dev / testing

```bash
# terminal 1 — run the API
cd api && go build -o fred-api . && PORT=8080 ./fred-api

# terminal 2 — point the CLI at local API and run it
cd cli && go build -o interest .
INTEREST_API=http://localhost:8080 ./interest
INTEREST_API=http://localhost:8080 ./interest status
```

---

# GOING LIVE — 3 things to wire up

You need three URLs to exist. Here's exactly how, cheapest/simplest path.

## 1. Publish the CLI binaries to GitHub Releases

This is what `install.sh` downloads. One-time repo setup, then a release per version.

```bash
# authenticate gh (interactive — pick GitHub.com, HTTPS, login via browser)
gh auth login

# create the repo on your personal account
cd ~/fred
gh repo create mfmp17/interest --public --source=. --remote=origin --push

# build fresh binaries and cut a release
./build.sh 0.1.0
gh release create v0.1.0 ./dist/interest_darwin_arm64 ./dist/interest_darwin_amd64 \
    --title "v0.1.0" --notes "first release"
```

After this, these URLs work automatically (note `latest`):
- https://github.com/mfmp17/interest/releases/latest/download/interest_darwin_arm64
- https://github.com/mfmp17/interest/releases/latest/download/interest_darwin_amd64

The installer always pulls `latest`, so future releases upgrade users with no script change.

## 2. Serve the install script at `https://fred.cash/interest`

Easiest zero-server option: **Cloudflare Workers** (free). It just returns the text of `web/install.sh`.

Alternative even-simpler options:
- **GitHub Pages**: put `install.sh` in a repo, enable Pages, then point `fred.cash/interest` there. (Path handling is fussier.)
- **Any static host / your own tiny server**: serve the file as `text/plain` at that path.

### Cloudflare Worker recipe (recommended)
1. Cloudflare dashboard → Workers & Pages → Create Worker.
2. Paste the worker below (it embeds the script).
3. Add a route: `fred.cash/interest*` → this worker (requires fred.cash on Cloudflare DNS).

```js
const SCRIPT = `PASTE THE ENTIRE CONTENTS OF web/install.sh HERE`;
export default {
  fetch() {
    return new Response(SCRIPT, {
      headers: { "content-type": "text/plain; charset=utf-8" },
    });
  },
};
```

Test: `curl -fsSL https://fred.cash/interest` should print the script.

## 3. Host the backend API at `https://api.fred.cash`

The CLI defaults to `https://api.fred.cash`. Deploy `api/` anywhere that runs a Go binary or container:

- **Fly.io** (simple, free tier): `fly launch` in the `api/` dir, it detects Go.
- **Railway / Render**: connect the repo, set start command to the built binary.
- **A VPS**: copy the `fred-api` binary, run behind nginx/caddy with TLS.

Then in your DNS (Cloudflare):
- `api.fred.cash` → your host (A/AAAA record or CNAME to Fly/Railway).

Until the API is live you can still demo everything with `INTEREST_API=http://localhost:8080 interest`.

---

## DNS summary (in your fred.cash registrar / Cloudflare)

| Record | Points to | Purpose |
|--------|-----------|---------|
| `fred.cash/interest` (Worker route) | Cloudflare Worker | serves install.sh |
| `api.fred.cash` (A/CNAME) | your API host | the backend |

## Cutting a new version later

```bash
./build.sh 0.2.0
gh release create v0.2.0 ./dist/interest_darwin_arm64 ./dist/interest_darwin_amd64 --title v0.2.0 --notes "..."
```
Users re-running the curl line get the new binary automatically.
