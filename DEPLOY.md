# Deploying with Vercel + Namecheap DNS (no Cloudflare)

This gets you:
- `curl -fsSL https://fred.cash/interest | bash`  → installs the CLI
- `https://api.fred.cash/v1/status`               → the backend the CLI calls

**Key fact:** DNS only maps *hostnames* to servers — it cannot route a *path*
like `/interest`. So Vercel serves the path; Namecheap just points the domains
at Vercel. One Vercel project owns BOTH `fred.cash` and `api.fred.cash`.

The `site/` folder is the deployable unit:
```
site/
├── vercel.json        # routes /interest -> install.sh, /v1/status -> /api/status
├── public/install.sh  # the installer (served at fred.cash/interest)
└── api/status.go      # serverless API (served at api.fred.cash/v1/status)
```

---

## Step 1 — Deploy to Vercel

```bash
npm i -g vercel        # if you don't have it
cd ~/fred/site
vercel                 # first run: log in, link/create project "interest"
vercel --prod          # deploy to production
```

You'll get a URL like `https://interest-xxxx.vercel.app`. Test it:
```bash
curl -fsSL https://interest-xxxx.vercel.app/interest      # should print the script
curl -fsSL https://interest-xxxx.vercel.app/v1/status     # should print JSON
```

## Step 2 — Add your domains in Vercel

Vercel dashboard → your project → **Settings → Domains** → add BOTH:
- `fred.cash`
- `api.fred.cash`

Vercel will then show you the exact DNS records to create. They're usually:

| Host        | Type  | Value                    |
|-------------|-------|--------------------------|
| `fred.cash` (apex) | A     | `76.76.21.21`            |
| `api`       | CNAME | `cname.vercel-dns.com.`  |

(Use whatever Vercel shows you — it's authoritative. The apex A record IP can change.)

## Step 3 — Set those records in Namecheap

Namecheap dashboard → Domain List → `fred.cash` → **Manage** → **Advanced DNS**.

Under **Host Records**, add:

1. **Apex (root fred.cash) → Vercel**
   - Type: `A Record`
   - Host: `@`
   - Value: `76.76.21.21`  (the IP Vercel gave you)
   - TTL: Automatic

2. **api subdomain → Vercel**
   - Type: `CNAME Record`
   - Host: `api`
   - Value: `cname.vercel-dns.com.`  (with trailing dot)
   - TTL: Automatic

⚠️ Namecheap default has a "URL Redirect" / parking record on `@` and `www` —
**delete those** or they'll fight your A record.

Save. DNS propagates in ~5–30 min (sometimes up to a couple hours).

## Step 4 — Point the CLI at the real API

The CLI defaults to `https://api.fred.cash` already (see `cli/main.go`), so once
DNS is live, nothing to change. To rebuild/release after any code change:
```bash
cd ~/fred && ./build.sh 0.1.1
gh release create v0.1.1 dist/interest_darwin_arm64 dist/interest_darwin_amd64 --title v0.1.1 --notes "..."
```

## Step 5 — Verify end to end

```bash
curl -fsSL https://fred.cash/interest | bash    # installs
interest                                          # connects to api.fred.cash
```

---

## Checkpoints / troubleshooting

- `curl https://fred.cash/interest` returns the script text? → path routing OK.
- `curl https://api.fred.cash/v1/status` returns JSON? → API + subdomain OK.
- `dig fred.cash +short` shows Vercel's IP? → apex DNS propagated.
- `dig api.fred.cash +short` shows a vercel-dns CNAME? → subdomain propagated.
- Binary won't download during install? → check
  https://github.com/mfmp17/interest/releases (a release must exist).
