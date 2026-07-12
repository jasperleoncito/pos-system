# Deploying with a domain — Portainer + Nginx Proxy Manager

How to put the POS on a real domain using a VPS that already runs
**Portainer** and **Nginx Proxy Manager (NPM)**. This reflects the actual
working setup — NPM reaches the stack **by container name over a shared
Docker network**, not via a published host port.

Examples below use (replace with your own):

| Thing | Example value |
|---|---|
| Domain | `pos.jprserver.uk` |
| VPS public IP | `72.61.151.141` |
| NPM's Docker network | `nginx-proxy_default` |
| Stack's internal nginx container | `pos-system-nginx-1` |

**You do NOT need an API subdomain.** The stack ships its own internal
nginx that serves the frontend, the API (`/api/...`), and images
(`/storage/...`) on one origin. NPM only adds the domain + HTTPS in front.

```
Browser ─ https://pos.jprserver.uk ─► Cloudflare ─► NPM (TLS :443)
             │
             └─ docker network: nginx-proxy_default ─► pos-system-nginx-1:80
                                                         ├── /api      → pos-backend:9137  (Go)
                                                         ├── /storage  → pos-minio:9000    (MinIO)
                                                         └── /         → pos-frontend:4519 (Next.js)
```

> **Key idea:** the stack's nginx joins NPM's network (`edge` → external
> `nginx-proxy_default`). NPM forwards to `pos-system-nginx-1:80` by name.
> The stack publishes **no** public ports — `7642`/`9673`/`9284` are bound to
> `127.0.0.1` only. This is the same pattern the sibling `rent-system` uses.

---

## Prerequisites

- Portainer + NPM already running on the VPS.
- NPM is attached to a Docker network (here `nginx-proxy_default`). Find it:
  ```bash
  docker inspect npm --format '{{json .NetworkSettings.Networks}}'
  ```
  If yours is named differently, set `NPM_NETWORK=<that-name>` in the stack
  env (Step 2). The compose defaults it to `nginx-proxy_default`.

## Step 1 — DNS record

`jprserver.uk` here is on **Cloudflare**. Add the subdomain:

| Type | Name | Value | Proxy status |
|---|---|---|---|
| A | `pos` | `72.61.151.141` | **DNS only (grey cloud)** ← during first cert issuance |

Grey-cloud it first so Let's Encrypt's HTTP-01 challenge reaches the origin
directly. After the cert is issued (Step 4) you can flip it back to
**Proxied (orange cloud)** to match your other subdomains.

Verify it resolves to the VPS before continuing:
```bash
getent hosts pos.jprserver.uk    # must show 72.61.151.141
```

## Step 2 — Deploy the stack in Portainer

1. Portainer → **Stacks → Add stack**, name `pos-system`.
2. Build method: **Repository** → your git repo URL, compose path
   `docker-compose.yml`, branch `main`.
3. Scroll to **Environment variables** → **Advanced mode** and paste the
   block below. The stack reads config from these (there is **no** `.env`
   file in the repo — it's gitignored). Fill the four secrets and your
   domain; the rest are safe defaults.

   ```env
   # ---- app ----
   APP_ENV=production
   APP_NAME=POS System
   WEB_PORT=7642
   HTTP_PORT=9137

   # ---- domain (all three must match your domain) ----
   CORS_ORIGINS=https://pos.jprserver.uk
   APP_URL=https://pos.jprserver.uk
   MINIO_PUBLIC_BASE_URL=https://pos.jprserver.uk/storage

   # ---- secrets (generate NEW ones — openssl rand -base64 48) ----
   DB_PASSWORD=<long-random>
   JWT_SECRET=<long-random>
   MINIO_ACCESS_KEY=<random>
   MINIO_SECRET_KEY=<long-random>

   # ---- postgres / redis / minio (service names — leave as-is) ----
   DB_HOST=postgres
   DB_PORT=5432
   DB_USER=pos
   DB_NAME=pos
   DB_SSLMODE=disable
   DB_MAX_CONNS=20
   REDIS_ADDR=redis:6379
   REDIS_PASSWORD=
   REDIS_DB=0
   MINIO_ENDPOINT=minio:9000
   MINIO_BUCKET=pos
   MINIO_USE_SSL=false

   # ---- jwt lifetimes ----
   JWT_ACCESS_TTL=15m
   JWT_REFRESH_TTL=720h

   # ---- mail (real provider for prod; Gmail SMTP shown) ----
   SMTP_HOST=smtp.gmail.com
   SMTP_PORT=587
   SMTP_USER=you@gmail.com
   SMTP_PASSWORD=<gmail app password>
   SMTP_FROM=noreply@jprserver.uk
   SMTP_FROM_NAME=POS System

   # ---- Xendit billing (OPTIONAL — leave blank to launch without billing) ----
   # App boots fine without these; subscription billing stays off until BOTH
   # are set (secret key creates invoices, webhook token confirms payments).
   XENDIT_SECRET_KEY=
   XENDIT_WEBHOOK_TOKEN=

   # ---- seeding ----
   SEED_ON_START=admin
   SEED_PASSWORD=<change-me>

   # ---- only if NPM's network isn't named nginx-proxy_default ----
   # NPM_NETWORK=your-npm-network
   ```

   Required at boot: `DB_PASSWORD`, `JWT_SECRET`, `MINIO_ACCESS_KEY`,
   `MINIO_SECRET_KEY`, `CORS_ORIGINS`. Xendit is **not** required.

4. **Deploy the stack.** First build takes a few minutes.
5. Confirm on the VPS (it listens on loopback only):
   ```bash
   curl http://127.0.0.1:7642/api/v1/health     # → {"success":true,...,"status":"healthy"}
   ```

> **Ports are localhost-only by design.** `7642` (app), `9673` (MinIO
> console) and `9284` (Mailpit) are bound to `127.0.0.1` — reach them via an
> SSH tunnel, e.g. `ssh -L 9673:127.0.0.1:9673 root@72.61.151.141`. Postgres,
> Redis and MinIO's API are never published.

## Step 3 — Proxy host in Nginx Proxy Manager

NPM → **Hosts → Proxy Hosts → Add Proxy Host**.

**Details tab**

| Field | Value |
|---|---|
| Domain Names | `pos.jprserver.uk` |
| Scheme | `http` |
| Forward Hostname / IP | `pos-system-nginx-1` ← **container name, NOT the public IP** |
| Forward Port | `80` |
| Cache Assets | OFF |
| Block Common Exploits | ON |
| Websockets Support | **ON** ← required for the live Kitchen Display stream |

> Forwarding to the public IP (`72.61.151.141:80`) does **not** work — port 80
> on that IP is NPM itself, so it loops. Always use the container name; NPM and
> the stack's nginx share the `nginx-proxy_default` network.

**SSL tab** (only after DNS from Step 1 resolves)

| Field | Value |
|---|---|
| SSL Certificate | Request a new Let's Encrypt certificate |
| Force SSL | ON |
| HTTP/2 Support | ON |
| HSTS | optional (the app already sends HSTS in production) |
| Email / Agree to ToS | fill in / tick |

Save. NPM fetches the cert; `https://pos.jprserver.uk` is live.

> ⚠️ Let's Encrypt allows only **5 failed validations per hostname per hour**.
> Make sure DNS resolves before requesting, or you'll be locked out for an hour.

## Step 4 — Re-enable Cloudflare proxy (optional)

Once the cert is issued, flip the `pos` record back to **Proxied (orange
cloud)** in Cloudflare for CDN/DDoS protection, matching your other subdomains.

## Step 5 — First data

Register your own business at `https://pos.jprserver.uk/register`, **or** load
demo data on the VPS:

```bash
docker exec pos-system-backend-1 /app/seed
```

> Demo accounts use password `password123` — change them or skip seeding on a
> public server. `SEED_ON_START=admin` already creates just the super admin.

## Step 6 — Verify

- [ ] Log in at `https://pos.jprserver.uk`
- [ ] Upload a product photo in **Menu** — the image displays (proves `MINIO_PUBLIC_BASE_URL`)
- [ ] Open **Kitchen** in one tab, place an order in another — the ticket appears instantly (proves SSE through NPM)
- [ ] **Forgot password** — the emailed link points at `https://pos.jprserver.uk/...` (proves `APP_URL`)
- [ ] `https://pos.jprserver.uk/api/v1/docs/index.html` is **absent** (Swagger off in production)

## Firewall (UFW)

Nothing extra to open. The app is served through NPM on **443**:

```
22/tcp  ALLOW   (SSH)
80      ALLOW   (NPM — Let's Encrypt + http→https redirect)
443     ALLOW   (NPM — HTTPS)
```

Do **not** add a UFW allow for `7642`/`9673`/`9284`. Note UFW does **not**
filter Docker-published ports (Docker writes its own iptables rules) — which is
exactly why this stack binds those ports to `127.0.0.1` instead of relying on a
firewall rule.

## Updating the app later

Push to `main`, then in Portainer open the stack → **Pull and redeploy**.
Database migrations run automatically when the backend starts.

## Backups

From the repo directory on the VPS:

```bash
make backup     # pg_dump + MinIO archive into ./backups/<timestamp>/
```

Restore notes are inside `scripts/backup.sh`.

---

## Troubleshooting

| Symptom | Likely cause / fix |
|---|---|
| NPM **"Internal Error"** on save | Let's Encrypt cert request failed — the domain doesn't resolve to the VPS yet (Step 1), or you hit the 5-fails/hour limit. Fix DNS, wait, retry once. |
| **502 Bad Gateway** via NPM | (a) Forward host is the public IP instead of `pos-system-nginx-1` (Step 3); or (b) the stack's nginx resolved another stack's `backend`/`minio` on the shared network — the compose fixes this with unique `pos-backend`/`pos-frontend`/`pos-minio` aliases, so redeploy after pulling; or (c) stack still starting (`docker ps`). |
| `curl 127.0.0.1:7642/api/v1/health` fails on the VPS | Backend crash-looping — check `docker logs pos-system-backend-1`. Usually a missing required secret (`DB_PASSWORD`/`JWT_SECRET`/`MINIO_*`). |
| **Images broken** | `MINIO_PUBLIC_BASE_URL` must be `https://<domain>/storage`. Redeploy after fixing. |
| **Kitchen Display not live** (updates every ~10 s) | Websockets Support is OFF on the NPM proxy host — turn it ON. |
| **Emailed links point at localhost** | `APP_URL` not set — set it and redeploy. |
| Emails don't arrive | `SMTP_HOST=mailpit` left in prod (Mailpit gets nothing) — use a real provider. Gmail rewrites `From` to the authenticated account and caps ~500/day. |
| Domain won't validate but resolves | Cloudflare proxy (orange) can interfere with HTTP-01 — set the record to **DNS only (grey)** for issuance, then re-enable proxy. |
