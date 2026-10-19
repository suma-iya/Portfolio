// Package handler is the Vercel serverless version of the contact-form API.
//
// It mirrors backend/ (validation rules, email format, env variable names) but
// uses only the standard library so the function builds without extra
// dependencies. vercel.json rewrites /api/v1/contact here, so the frontend
// calls the same path on Vercel as it does behind Nginx on EC2.
package handler

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"mime"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

type contactRequest struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Subject string `json:"subject"`
	Message string `json:"message"`
}

const (
	maxBodyBytes = 64 << 10 // matches client_max_body_size in nginx/
	smtpTimeout  = 15 * time.Second
)

func Handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, "Method not allowed.")
		return
	}

	if !limiter.allow(clientIP(r), getEnvInt("RATE_LIMIT_PER_MINUTE", 5)) {
		writeJSON(w, http.StatusTooManyRequests, "Too many messages. Please wait a minute and try again.")
		return
	}

	var req contactRequest
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !valid(&req) {
		// The frontend shows this message to visitors, so keep internals out of it.
		writeJSON(w, http.StatusBadRequest, "Please check your name, email and message (at least 10 characters) and try again.")
		return
	}

	if err := sendContactMessage(req); err != nil {
		log.Printf("failed to send contact email: %v", err)
		writeJSON(w, http.StatusInternalServerError, "Could not send your message right now. Please try again later or email directly.")
		return
	}

	writeJSON(w, http.StatusOK, "Message sent successfully.")
}

// valid applies the same rules as the binding tags in backend/internal/models.
func valid(req *contactRequest) bool {
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(req.Email)
	req.Subject = strings.TrimSpace(req.Subject)
	req.Message = strings.TrimSpace(req.Message)

	if n := utf8.RuneCountInString(req.Name); n < 2 || n > 100 {
		return false
	}
	if n := utf8.RuneCountInString(req.Message); n < 10 || n > 5000 {
		return false
	}
	if utf8.RuneCountInString(req.Subject) > 200 || len(req.Email) > 255 {
		return false
	}
	// The address goes into the Reply-To header, so it must be a bare address
	// with no display name or line breaks.
	addr, err := mail.ParseAddress(req.Email)
	return err == nil && addr.Address == req.Email && !strings.ContainsAny(req.Email, "\r\n")
}

func sendContactMessage(req contactRequest) error {
	host := getEnv("SMTP_HOST", "smtp.gmail.com")
	port := getEnvInt("SMTP_PORT", 587)
	username := getEnv("SMTP_USERNAME", "")
	password := getEnv("SMTP_PASSWORD", "")
	from := getEnv("SMTP_FROM", username)
	to := getEnv("CONTACT_TO_EMAIL", "sumaiya.rahim234@gmail.com")
	if username == "" || password == "" {
		return fmt.Errorf("SMTP_USERNAME and SMTP_PASSWORD must be set")
	}

	// Collapse line breaks so a crafted subject can't spill into other headers.
	subject := strings.Join(strings.Fields(req.Subject), " ")
	if subject == "" {
		subject = "New portfolio contact message"
	}

	body := fmt.Sprintf(`
<div style="font-family: sans-serif; max-width: 560px; margin: 0 auto;">
  <h2 style="color:#0891b2;">New Contact Form Submission</h2>
  <p><strong>Name:</strong> %s</p>
  <p><strong>Email:</strong> %s</p>
  <p><strong>Subject:</strong> %s</p>
  <hr style="border:none;border-top:1px solid #eee;">
  <p style="white-space: pre-wrap;">%s</p>
</div>`,
		html.EscapeString(req.Name),
		html.EscapeString(req.Email),
		html.EscapeString(subject),
		html.EscapeString(req.Message),
	)

	var msg strings.Builder
	msg.WriteString("From: " + from + "\r\n")
	msg.WriteString("To: " + to + "\r\n")
	msg.WriteString("Reply-To: " + req.Email + "\r\n")
	msg.WriteString("Subject: " + mime.QEncoding.Encode("utf-8", "[Portfolio] "+subject) + "\r\n")
	msg.WriteString("Date: " + time.Now().UTC().Format(time.RFC1123Z) + "\r\n")
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
	msg.WriteString(strings.ReplaceAll(body, "\n", "\r\n"))

	return sendMail(host, port, username, password, from, to, msg.String())
}

// sendMail is smtp.SendMail with a deadline, and it refuses to send the
// password unless the connection has been upgraded to TLS.
func sendMail(host string, port int, username, password, from, to, msg string) error {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(port)), smtpTimeout)
	if err != nil {
		return err
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(smtpTimeout)); err != nil {
		return err
	}

	var c *smtp.Client
	if port == 465 {
		tlsConn := tls.Client(conn, &tls.Config{ServerName: host})
		if c, err = smtp.NewClient(tlsConn, host); err != nil {
			return err
		}
	} else {
		if c, err = smtp.NewClient(conn, host); err != nil {
			return err
		}
		if ok, _ := c.Extension("STARTTLS"); !ok {
			return fmt.Errorf("smtp server %s does not support STARTTLS", host)
		}
		if err := c.StartTLS(&tls.Config{ServerName: host}); err != nil {
			return err
		}
	}
	defer c.Close()

	if err := c.Auth(smtp.PlainAuth("", username, password, host)); err != nil {
		return err
	}
	if err := c.Mail(bareAddress(from)); err != nil {
		return err
	}
	if err := c.Rcpt(bareAddress(to)); err != nil {
		return err
	}
	wc, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := wc.Write([]byte(msg)); err != nil {
		return err
	}
	if err := wc.Close(); err != nil {
		return err
	}
	return c.Quit()
}

// bareAddress turns "Name <user@example.com>" into "user@example.com" for the
// SMTP envelope; the headers keep the full form.
func bareAddress(s string) string {
	if a, err := mail.ParseAddress(s); err == nil {
		return a.Address
	}
	return s
}

// rateLimiter allows a fixed number of requests per IP per minute. Vercel may
// run several instances at once, each with its own counts, so this is a
// best-effort guard against spam rather than a hard limit.
type rateLimiter struct {
	mu   sync.Mutex
	hits map[string][]time.Time
}

var limiter = &rateLimiter{hits: make(map[string][]time.Time)}

func (l *rateLimiter) allow(ip string, perMinute int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	cutoff := time.Now().Add(-time.Minute)
	for k, times := range l.hits {
		kept := times[:0]
		for _, t := range times {
			if t.After(cutoff) {
				kept = append(kept, t)
			}
		}
		if len(kept) == 0 {
			delete(l.hits, k)
		} else {
			l.hits[k] = kept
		}
	}

	if len(l.hits[ip]) >= perMinute {
		return false
	}
	l.hits[ip] = append(l.hits[ip], time.Now())
	return true
}

// clientIP uses the address set by Vercel's edge network, which overwrites
// any value the client sends.
func clientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-Ip"); ip != "" {
		return ip
	}
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.TrimSpace(strings.Split(fwd, ",")[0])
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	return host
}

func writeJSON(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": message})
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	n, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		return fallback
	}
	return n
}
