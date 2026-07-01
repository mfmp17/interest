# Deploy & Operate — Vercel + Namecheap (no Cloudflare)

This documents how the live setup is wired and how to reproduce / maintain it.

**Live now:**
- `curl -fsSL https://get.fred.cash | bash`  → installs the `interest` CLI
- `https://api.fred.cash/v1/status`          → the backend the CLI calls
- `https://fred.cash` (apex)                 → the Plinko game (separate project)

**Key fact:** DNS only maps *hostnames* to servers — it cannot route a *path*
like `/interest`. That's why we use the `get.` subdomain instead of
`fred.cash/interest`: the apex is already a different Vercel project (the game),
and a subdomain lets the installer live in its own project with zero risk to it.

---

## The moving parts

| Hostname | Serves | Vercel project |
|----------|--------|----------------|
| `get.fred.cash` | install script | `site` |
| `api.fred.cash` | API function | `site` |
| `fred.cash` (apex) | Plinko game | `fred-cash` (do not touch) |

The `site/` folder is the deployable unit:
```
site/
├── vercel.json          # host-based routing (get -> install.sh, api -> status)
├── public/install.sh    # the installer
└── api/status.go        # serverless Go function
```

---

## How it was set up (reference — already done)

### 1. GitHub repo + release (hosts the binaries)
```bash
gh auth login                                   # account: mfmp17
cd ~/fred
gh repo create mfmp17/interest --public --source=. --remote=origin --push
./build.sh 0.1.0
gh release create v0.1.0 dist/interest_darwin_arm64 dist/interest_darwin_amd64 \
    --title v0.1.0 --notes "first release"
```
Result — these URLs resolve automatically (installer pulls `latest`):
- https://github.com/mfmp17/interest/releases/latest/download/interest_darwin_arm64
- https://github.com/mfmp17/interest/releases/latest/download/interest_darwin_amd64

### 2. Deploy the site to Vercel
```bash
cd ~/fred/site
vercel --yes          # first time: links/creates the "site" project
vercel --prod --yes   # production deploy
```

### 3. Attach the subdomains to the `site` project
```bash
vercel domains add get.fred.cash site
vercel domains add api.fred.cash site
```
Both then reported "not configured properly" until the DNS records existed.

### 4. DNS in Namecheap
Namecheap → Domain List → `fred.cash` → **Manage → Advanced DNS** →
**Host Records**, add TWO A records (leave the existing `@` apex record alone —
that's the game):

| Type | Host | Value | TTL |
|------|------|-------|-----|
| A Record | `get` | `76.76.21.21` | Automatic |
| A Record | `api` | `76.76.21.21` | Automatic |

Vercel then issued TLS certs automatically (~1–3 min after DNS resolved).

---

## Verify end to end

```bash
curl -fsSL https://get.fred.cash | bash    # installs the CLI
interest                                     # connects to api.fred.cash
interest status
```

## Day-to-day operations

**Ship a new CLI version** (users auto-upgrade on next install):
```bash
cd ~/fred && ./build.sh 0.2.0
gh release create v0.2.0 dist/interest_darwin_arm64 dist/interest_darwin_amd64 \
    --title v0.2.0 --notes "..."
```

**Change the installer:**
```bash
# edit web/install.sh
cp web/install.sh site/public/install.sh
cd site && vercel --prod --yes
```

**Change the API** (e.g. real TVL, new endpoints):
```bash
# edit site/api/status.go
cd site && vercel --prod --yes
```

---

## Troubleshooting

- `dig @8.8.8.8 get.fred.cash +short` → should show `76.76.21.21`. Empty means
  DNS not propagated (or record missing). Your *local* resolver may lag public
  DNS by a while — trust `@8.8.8.8` / `@1.1.1.1` over the default.
- `SSL_ERROR_SYSCALL` on HTTPS but HTTP works → Vercel is still issuing the TLS
  cert. Wait 1–3 min and retry; re-running `vercel --prod` nudges it.
- `curl https://get.fred.cash` returns 404 → routing issue; check the host-based
  rewrite in `site/vercel.json`.
- Installer can't download the binary → confirm a release exists at
  https://github.com/mfmp17/interest/releases
- Don't attach `fred.cash` (apex) to the `site` project — it belongs to the
  `fred-cash` game project and moving it takes the game offline.
