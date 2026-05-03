# APEX — Credentials Setup Guide

One-time setup for external API credentials. Internal services (Postgres, Redis, MinIO, Neo4j) just need strong passwords — choose your own and set them in `.env`.

---

## Step 1 — Copy the env file

```bash
cp .env.example .env
```

Then fill in each section below.

---

## Section A — Internal Passwords (choose your own)

Edit `.env` and replace the placeholder values:

| Variable | What to set |
|---|---|
| `POSTGRES_PASSWORD` | Any strong password (e.g. `openssl rand -hex 16`) |
| `DATABASE_URL` | Replace `your_postgres_password_here` with same password |
| `NEO4J_AUTH` | Format `neo4j/<password>` — same password in both `NEO4J_AUTH` and `NEO4J_PASSWORD` |
| `MINIO_ROOT_PASSWORD` | Any strong password |

---

## Section B — Groq API Key (free tier)

1. Go to **console.groq.com** → Sign up / Log in
2. Click **API Keys** → **Create API Key**
3. Copy the key → set `GROQ_API_KEY=gsk_...` in `.env`

Free tier: 6,000 tokens/min on Llama 3.3 70B — sufficient for local demo.

---

## Section C — Google / Gmail OAuth

1. Go to **console.cloud.google.com** → Create or select a project
2. **APIs & Services → Enable APIs** → enable **Gmail API**
3. **APIs & Services → OAuth consent screen** → External → fill App name ("APEX"), your email, save
4. **APIs & Services → Credentials → Create Credentials → OAuth 2.0 Client ID**
   - Application type: **Web application**
   - Authorized redirect URIs: `http://localhost:8081/auth/google/callback`
   - Click **Create**
5. Copy **Client ID** and **Client Secret** → set in `.env`:
   ```
   GOOGLE_CLIENT_ID=<your-client-id>.apps.googleusercontent.com
   GOOGLE_CLIENT_SECRET=<your-client-secret>
   GOOGLE_REDIRECT_URI=http://localhost:8081/auth/google/callback
   ```
6. Create a Gmail label named `AP-Inbox` in your Google account — invoices sent there will be polled

**First run:** Visit `http://localhost:8081/auth/google` to complete OAuth and store the refresh token.

---

## Section D — Telegram Bot

1. Open Telegram → search **@BotFather** → `/newbot`
2. Follow prompts → copy the **HTTP API token**
3. Set `TELEGRAM_BOT_TOKEN=<token>` in `.env`
4. Set the webhook (after `docker-compose up`):
   ```bash
   curl "https://api.telegram.org/bot<TOKEN>/setWebhook?url=https://<your-public-url>/telegram/webhook&secret_token=<WEBHOOK_SECRET>"
   ```
   For local dev use **ngrok**: `ngrok http 8081` then use the ngrok URL.
5. Set `TELEGRAM_WEBHOOK_SECRET` to any random string (matches the `secret_token` above)
6. To receive approval notifications: start a chat with your bot, then get your chat ID:
   ```bash
   curl "https://api.telegram.org/bot<TOKEN>/getUpdates"
   ```
   Copy `result[0].message.chat.id` → set `TELEGRAM_ADMIN_CHAT_ID=<id>` in `.env`

---

## Section E — JWT Keys

For local dev, `JWT_SECRET` (HMAC) is used automatically. For production-grade RS256:

```bash
# Generate RSA key pair
openssl genrsa -out private.pem 2048
openssl rsa -in private.pem -pubout -out public.pem

# Base64-encode for env vars (no newlines)
JWT_PRIVATE_KEY=$(base64 -w 0 private.pem)
JWT_PUBLIC_KEY=$(base64 -w 0 public.pem)
```

Add to `.env`:
```
JWT_PRIVATE_KEY=<base64-private-key>
JWT_PUBLIC_KEY=<base64-public-key>
```

Remove `JWT_SECRET` line when using RSA keys. Clean up key files after.

---

## Section F — Final `.env` checklist

Before `docker-compose up`, verify:

- [ ] `POSTGRES_PASSWORD` and `DATABASE_URL` use the same password
- [ ] `NEO4J_AUTH=neo4j/<password>` and `NEO4J_PASSWORD=<password>` match
- [ ] `GROQ_API_KEY` is set (required for agent-service to function)
- [ ] `TELEGRAM_BOT_TOKEN` set (optional — Telegram features disabled if missing)
- [ ] `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET` set (optional — Gmail polling disabled if missing)
- [ ] `JWT_SECRET` set (or RSA keys) — used by api-gateway for auth

---

## Quick Start (after filling `.env`)

```bash
docker-compose up -d
# Wait ~30 seconds for all services to become healthy
docker-compose ps
# Visit http://localhost:3000 — dashboard
# Visit http://localhost:8080/health — api-gateway
```
