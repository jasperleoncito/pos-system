# Deploying with a domain — Nginx Proxy Manager step by step

How to put the POS on a real domain using a VPS that already runs
**Portainer** and **Nginx Proxy Manager (NPM)**.

Throughout this guide the examples use:

| Thing | Example value (replace with yours) |
|---|---|
| Domain | `pos.jprserver.uk` |
| VPS public IP | `74.208.192.101` |
| App port on the VPS | `7642` (the stack's `WEB_PORT`) |

**You do NOT need a separate API subdomain** (no `api.pos.jprserver.uk`).
The stack ships its own internal nginx that serves the frontend, the API
(`/api/...`), and images (`/storage/...`) on one origin. NPM only adds
the domain + HTTPS in front of that single port.

```
Browser ── https://pos.jprserver.uk ──► NPM (TLS) ──► VPS:7642 (stack nginx)
                                                        ├── /api      → Go backend
                                                        ├── /storage  → MinIO images
                                                        └── /         → Next.js app
```

---

## Step 1 — DNS record

At your domain registrar (or DNS provider), create an **A record**:

| Type | Name | Value | TTL |
|---|---|---|---|
| A | `pos` | `74.208.192.101` | default |

Wait until `pos.jprserver.uk` resolves (usually minutes):
`nslookup pos.jprserver.uk` should return your VPS IP.

## Step 2 — Deploy the stack in Portainer

1. Portainer → **Stacks → Add stack**.
2. Name: `pos-system`.
3. Build method: **Repository** → paste your git repo URL
   (compose path: `docker-compose.yml`, branch `main`).
4. Scroll to **Environment variables** and add these
   (see `.env.example` — every production value is marked `PROD:`):

   ```env
   APP_ENV=production
   WEB_PORT=7642
   DB_PASSWORD=<long random>
   JWT_SECRET=<long random — openssl rand -base64 48>
   MINIO_ACCESS_KEY=<random>
   MINIO_SECRET_KEY=<long random>
   CORS_ORIGINS=https://pos.jprserver.uk
   APP_URL=https://pos.jprserver.uk
   MINIO_PUBLIC_BASE_URL=https://pos.jprserver.uk/storage
   # real mail provider (or omit to keep the bundled Mailpit for testing)
   SMTP_HOST=smtp.yourprovider.com
   SMTP_PORT=587
   SMTP_USER=...
   SMTP_PASSWORD=...
   SMTP_FROM=noreply@jprserver.uk
   ```

5. **Deploy the stack.** First build takes a few minutes.
6. Check it locally on the VPS: `curl http://localhost:7642/api/v1/health`
   should return `"success":true`.

> Do not publish ports 5432/6379/9000 anywhere — the compose file
> already keeps them internal. Only `7642` (app), `9673` (MinIO console)
> and `9284` (Mailpit) are published; you can firewall the last two to
> your own IP.

## Step 3 — Proxy host in Nginx Proxy Manager

NPM → **Hosts → Proxy Hosts → Add Proxy Host**.

**Details tab**

| Field | Value |
|---|---|
| Domain Names | `pos.jprserver.uk` |
| Scheme | `http` |
| Forward Hostname / IP | `74.208.192.101` (or the Docker gateway IP if NPM is on the same host) |
| Forward Port | `7642` |
| Cache Assets | OFF (Next.js sets its own caching) |
| Block Common Exploits | ON |
| Websockets Support | **ON** ← required for the live Kitchen Display stream |

**SSL tab**

| Field | Value |
|---|---|
| SSL Certificate | Request a new SSL Certificate (Let's Encrypt) |
| Force SSL | ON |
| HTTP/2 Support | ON |
| HSTS Enabled | ON (optional but recommended) |
| Email / Agree to ToS | fill in / tick |

Save. NPM fetches the certificate; `https://pos.jprserver.uk` is live.

## Step 4 — First data

Either register your own business at
`https://pos.jprserver.uk/register`, **or** load the demo data
(SSH on the VPS):

```bash
docker exec pos-system-backend-1 /app/seed
```

> The demo accounts all use password `password123` — on a public
> server change them immediately or skip seeding entirely.

## Step 5 — Verify everything works

- [ ] Log in at `https://pos.jprserver.uk`
- [ ] Upload a product photo in **Menu** — the image displays
      (proves `MINIO_PUBLIC_BASE_URL` is right)
- [ ] Open **Kitchen** in one tab, place a POS order in another —
      the ticket appears instantly (proves SSE through NPM)
- [ ] Use **Forgot password** — the email link points at
      `https://pos.jprserver.uk/reset-password?...`
      (proves `APP_URL`; check your inbox or Mailpit)
- [ ] `https://pos.jprserver.uk/api/v1/docs/index.html` is **absent**
      (Swagger is disabled when `APP_ENV=production`)

## Updating the app later

Push to `main`, then in Portainer open the stack → **Pull and redeploy**
(or `docker compose up -d --build` in the repo directory on the VPS).
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
| NPM shows **502 Bad Gateway** | Stack not up yet (`docker ps`), wrong forward IP/port, or a firewall blocking 7642 between NPM and the host. If NPM runs in Docker on the same VPS, forward to the host IP, not `localhost`. |
| Site loads but **images are broken** | `MINIO_PUBLIC_BASE_URL` doesn't match the domain — must be `https://pos.jprserver.uk/storage`. Re-deploy the backend after changing it. |
| **Kitchen Display doesn't update live** (only every 10 s) | Websockets Support is OFF on the NPM proxy host — turn it ON. (The app falls back to polling, so it still works, just slower.) |
| **Emailed links point at localhost** | `APP_URL` not set — set it and redeploy the backend. |
| Login works on the domain but API calls fail from another site | That's CORS doing its job — add the other origin to `CORS_ORIGINS` only if you really want to allow it. |
| Mixed-content warnings | Force SSL wasn't enabled in NPM, or something hardcodes `http://` — check `MINIO_PUBLIC_BASE_URL`. |
