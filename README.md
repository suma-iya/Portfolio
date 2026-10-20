# Sumaiya Rahim — Portfolio

**Live site: [sumaiya-rahim.vercel.app](https://sumaiya-rahim.vercel.app)**

A backend engineering portfolio. Static frontend (no framework),
Go/Gin backend for a Gmail-SMTP contact form — no third-party form services.

## Structure

```
portfolio/
├── frontend/          # Static site: HTML5, CSS3, vanilla JS
│   ├── index.html
│   ├── css/style.css
│   ├── js/main.js
│   └── assets/
├── api/contact.go      # Vercel serverless contact-form function (Go)
├── vercel.json         # Vercel config: serves frontend/, routes /api
├── backend/            # Go + Gin REST API (for EC2)
│   ├── cmd/server/     # main.go — entrypoint
│   └── internal/
│       ├── config/     # env-driven configuration
│       ├── handlers/    # HTTP handlers
│       ├── services/    # SMTP mailer
│       ├── middleware/  # CORS, logging, rate limiting
│       ├── routes/      # route registration
│       └── models/      # request/response types
├── nginx/              # Nginx site: serves frontend/, proxies /api to Go
├── deploy/             # deploy.sh, server setup, systemd unit, guide
└── docker/             # optional containerized backend
```

## Local development

**Frontend:**
```bash
cd frontend
python3 -m http.server 8765
# → http://localhost:8765
```

**Backend:**
```bash
cd backend
cp .env.example .env   # fill in SMTP credentials (Gmail App Password)
set -a; source .env; set +a   # the server reads env vars, not the file
go run ./cmd/server
# → http://localhost:8080/health
```

On `localhost` the frontend posts to `http://localhost:8080`; anywhere else
it posts to its own origin (`/api/v1/contact`), which Nginx proxies to Go.
Override with `window.PORTFOLIO_API_BASE_URL` set before `main.js` loads.

## Deployment

### Vercel (free, no credit card)

The site and contact form run on Vercel's free Hobby plan. `vercel.json`
serves `frontend/` and routes `/api/v1/contact` to the Go function in
`api/contact.go`, a dependency-free version of `backend/` that uses the same
validation, email format and environment variables.

1. Sign in at [vercel.com](https://vercel.com) with GitHub (no card needed).
2. **Add New → Project**, import this repository, keep the defaults, deploy.
3. **Settings → Environment Variables**, add `SMTP_USERNAME` and
   `SMTP_PASSWORD` (a Gmail App Password), then **Redeploy**.
   Optional: `SMTP_FROM`, `CONTACT_TO_EMAIL`, `RATE_LIMIT_PER_MINUTE`.

Every push to `master` redeploys automatically.

### AWS EC2

Everything runs on one free-tier AWS EC2 instance (Ubuntu 24.04): Nginx
serves the site and proxies `/api` to the Go service; Let's Encrypt for HTTPS.

```bash
cp deploy/deploy.env.example deploy/deploy.env   # set EC2_HOST, SSH_KEY
./deploy/deploy.sh                               # first deploy and every update
```

Step-by-step (AWS setup, Gmail App Password, domain/HTTPS):
[`deploy/DEPLOYMENT.md`](deploy/DEPLOYMENT.md).

## Stack

- **Frontend:** HTML5, CSS3, vanilla JS — no React/Vue/Bootstrap/Tailwind
- **Backend:** Go, Gin, structured JSON logging, per-IP rate limiting, graceful shutdown
- **Email:** Gmail SMTP relay (App Password), no Formspree/Netlify Forms/EmailJS/Firebase
- **Database:** none required currently (contact form is stateless); PostgreSQL/SQLite ready to add under `backend/internal/models` if persistence is needed later
- **Deployment:** single AWS EC2 instance: Nginx (static site + reverse proxy) + systemd + Let's Encrypt
