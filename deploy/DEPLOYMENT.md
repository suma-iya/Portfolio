# Deployment Guide — one AWS EC2 instance

Everything runs on a single free-tier EC2 instance:

```
Browser ──► Nginx :80/:443 ──┬─► /var/www/portfolio   (frontend/ — static files)
                             └─► /api/*, /health ──► Go backend 127.0.0.1:8080 ──► Gmail SMTP ──► your inbox
```

- **Nginx** serves the static site and forwards `/api/*` to the Go backend.
  The site and API share one address, so no CORS or separate API domain is needed.
- **Go backend** runs as a systemd service under an unprivileged `portfolio`
  user. It listens on localhost only, so port 8080 is never exposed.
- **Let's Encrypt** gives free HTTPS once you point a domain at the instance.

You build on your Mac and upload a ready-made binary. The server never needs
Go installed, and a 1 GB free-tier instance never has to compile anything.

| File | Runs where | Purpose |
|---|---|---|
| `deploy/deploy.sh` | your Mac | Builds the Go binary, uploads it plus the site and configs, runs the setup script |
| `deploy/server-setup.sh` | EC2 (as root) | Installs Nginx/certbot, the service, the site and HTTPS. Safe to re-run. |
| `deploy/env.production.example` | EC2 | Template for `/opt/portfolio-api/.env` (SMTP secrets) |
| `deploy/portfolio-api.service` | EC2 | systemd unit for the backend |
| `nginx/*.conf` | EC2 | Nginx site (HTTP, and HTTPS once a certificate exists) |

---

## Step 1 — Get a Gmail App Password (5 min)

The backend sends each contact-form message through Gmail's SMTP server.

1. Choose the Gmail account that will *send* the messages: your own, or a
   new account made just for this (safer, since a leaked password then can't
   expose your main inbox).
2. Turn on **2-Step Verification** for that account: https://myaccount.google.com/security
3. Open https://myaccount.google.com/apppasswords, create one named
   `portfolio`, and copy the 16-character password. Remove the spaces.

Messages are delivered to `CONTACT_TO_EMAIL` (default `sumaiya.rahim234@gmail.com`).
When you reply to one, the reply goes to the visitor, because the backend sets `Reply-To` to their address.

## Step 2 — AWS account and a billing alarm (5 min)

1. Sign in to (or create) an AWS account at https://aws.amazon.com.
2. **Set a budget alert first** so a mistake can't cost you money:
   Billing and Cost Management → **Budgets** → Create budget → *Use a template* →
   **Zero spend budget** → enter your email → Create.

> **Free tier:** New AWS accounts (created after 15 July 2025) get a Free
> plan with credits for about 6 months. Older accounts get 750 hours/month of
> a `t2.micro`/`t3.micro` for 12 months. One small instance running 24/7 fits
> either. AWS also bills public IPv4 addresses per hour; the free tier/credits
> cover one on a running instance. The zero-spend budget will email you if
> anything starts to cost money.

## Step 3 — Launch the EC2 instance (10 min)

EC2 console → pick a **region** near your visitors (top-right; e.g.
*Asia Pacific (Mumbai) ap-south-1* or *Singapore*) → **Launch instance**:

| Setting | Value |
|---|---|
| Name | `portfolio` |
| AMI | **Ubuntu Server 24.04 LTS**, 64-bit (x86) |
| Instance type | the one marked **Free tier eligible** (`t3.micro` or `t2.micro`) |
| Key pair | **Create new key pair** → name `portfolio-key`, RSA, `.pem` → it downloads |
| Network settings | ✅ Allow SSH from **My IP** · ✅ Allow HTTP · ✅ Allow HTTPS |
| Storage | 8–30 GiB gp3 (30 GiB is free) |

Click **Launch instance**.

Move the key somewhere safe and lock its permissions (SSH refuses keys other users can read):

```bash
mkdir -p ~/.ssh
mv ~/Downloads/portfolio-key.pem ~/.ssh/
chmod 400 ~/.ssh/portfolio-key.pem
```

## Step 4 — Give it a permanent IP (Elastic IP) (2 min)

A normal public IP changes every time the instance stops. EC2 →
**Elastic IPs** → **Allocate Elastic IP address** → Allocate → select it →
**Actions → Associate** → choose the `portfolio` instance → Associate.

Write the IP down (e.g. `13.233.10.20`). From now on it's your server's address.

> An Elastic IP that is *not* attached to a running instance is billed. If you
> ever delete the instance, release the Elastic IP too.

Test SSH:

```bash
ssh -i ~/.ssh/portfolio-key.pem ubuntu@13.233.10.20
# type "yes" the first time, then `exit`
```

## Step 5 — First deploy (5 min)

On your Mac, from the repo root:

```bash
cp deploy/deploy.env.example deploy/deploy.env
nano deploy/deploy.env       # set EC2_HOST to your Elastic IP; SSH_KEY is already ~/.ssh/portfolio-key.pem
./deploy/deploy.sh
```

This first run installs Nginx and certbot, creates the `portfolio` user,
publishes the site and installs the backend. It then **stops with a yellow
warning**, because the server doesn't have your Gmail App Password yet. That's
expected: the password is never stored in the repo.

## Step 6 — Put your SMTP credentials on the server (2 min)

```bash
ssh -i ~/.ssh/portfolio-key.pem ubuntu@13.233.10.20
sudo nano /opt/portfolio-api/.env
```

Set these two lines (save with Ctrl+O, Enter, then exit with Ctrl+X):

```
SMTP_USERNAME=the-sending-account@gmail.com
SMTP_PASSWORD=abcdefghijklmnop
```

Start the backend and check it:

```bash
sudo systemctl restart portfolio-api
systemctl status portfolio-api --no-pager      # should say "active (running)"
curl http://127.0.0.1:8080/health              # {"status":"ok"}
exit
```

## Step 7 — Test it

Open `http://13.233.10.20` in a browser, fill in the contact form and send
it. The message should reach your inbox within a few seconds.

At this point the site works but is plain **HTTP**, so browsers show "Not
secure". Step 8 fixes that.

## Step 8 — Domain + free HTTPS (recommended, 10 min)

HTTPS needs a domain name. Pick one:

- **Free:** https://www.duckdns.org → sign in → create a subdomain such as
  `sumaiya-rahim` → put your Elastic IP in the *current ip* box → update ip.
  Your domain is then `sumaiya-rahim.duckdns.org`.
- **Paid (~$10/yr, looks more professional):** buy a domain (Namecheap,
  Cloudflare, Route 53…) and create an **A record** for `@` (and optionally
  `www`) pointing to your Elastic IP.

Check the DNS has updated (it should print your Elastic IP):

```bash
dig +short sumaiya-rahim.duckdns.org
```

Then add the domain to `deploy/deploy.env`:

```bash
DOMAINS=sumaiya-rahim.duckdns.org
# or, for your own domain:  DOMAINS="example.com www.example.com"
LETSENCRYPT_EMAIL=sumaiya.rahim234@gmail.com
```

and run `./deploy/deploy.sh` again. It gets a Let's Encrypt certificate,
switches Nginx to HTTPS and redirects HTTP to HTTPS. Certbot's timer renews the
certificate automatically, and re-running deploys never undoes this.

## Updating the site later

Edit anything in `frontend/` or `backend/`, then:

```bash
./deploy/deploy.sh
```

That's the whole release process. Your server `.env` and HTTPS certificate are left untouched.

## Turn off the GitHub Pages copy

GitHub Pages still serves the old Formspree version of the site from the
repository root. To keep EC2 as the only copy: GitHub repo → **Settings →
Pages** → set *Source/Branch* to **None** (or **Unpublish site**). Update the
link on your CV and LinkedIn to the new address.

---

## Troubleshooting

All of these run on the server (`ssh -i ~/.ssh/portfolio-key.pem ubuntu@<IP>`).

| Symptom | Check |
|---|---|
| Form says "Could not send your message" | `sudo journalctl -u portfolio-api -n 50`. `535 Username and Password not accepted` means a wrong App Password (or 2-Step Verification is off). |
| `502 Bad Gateway` on the form | Backend is down: `systemctl status portfolio-api`, then the journal command above |
| Site doesn't load at all | EC2 security group must allow inbound 80 and 443 from `0.0.0.0/0`; `sudo nginx -t`; `sudo tail /var/log/nginx/error.log` |
| `ssh: Connection timed out` | Your home IP changed. EC2 → Security Groups → edit the SSH rule → *My IP* again |
| Certificate request failed | `dig +short <domain>` must print your Elastic IP, and port 80 must be open |
| "Too many requests" | Rate limit: 5 submissions/min per visitor (`RATE_LIMIT_PER_MINUTE` in `.env`) |

Useful commands:

```bash
sudo systemctl restart portfolio-api        # after editing /opt/portfolio-api/.env
sudo journalctl -u portfolio-api -f         # live backend logs (JSON)
sudo tail -f /var/log/nginx/access.log      # live web traffic
sudo certbot certificates                   # certificate expiry
```

## Optional: Docker

`docker/docker-compose.yml` runs the backend in a container for local
testing. The EC2 setup above deliberately doesn't use Docker: a plain systemd
service uses less of the 1 GB of RAM and has fewer moving parts.
