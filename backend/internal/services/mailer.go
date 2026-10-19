package services

import (
	"fmt"
	"html"
	"strings"

	"github.com/suma-iya/portfolio-backend/internal/config"
	"github.com/suma-iya/portfolio-backend/internal/models"
	"gopkg.in/gomail.v2"
)

// Mailer sends contact-form submissions to the site owner's Gmail inbox via SMTP.
type Mailer struct {
	cfg *config.Config
}

func NewMailer(cfg *config.Config) *Mailer {
	return &Mailer{cfg: cfg}
}

func (m *Mailer) SendContactMessage(req models.ContactRequest) error {
	// Collapse line breaks so a crafted subject can't spill into other headers.
	subject := strings.Join(strings.Fields(req.Subject), " ")
	if subject == "" {
		subject = "New portfolio contact message"
	}

	body := fmt.Sprintf(`
<div style="font-family: sans-serif; max-width: 560px; margin: 0 auto;">
  <h2 style="color:#00d4aa;">New Contact Form Submission</h2>
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

	msg := gomail.NewMessage()
	msg.SetHeader("From", m.cfg.SMTPFrom)
	msg.SetHeader("To", m.cfg.ContactTo)
	msg.SetHeader("Reply-To", req.Email)
	msg.SetHeader("Subject", fmt.Sprintf("[Portfolio] %s", subject))
	msg.SetBody("text/html", body)

	dialer := gomail.NewDialer(m.cfg.SMTPHost, m.cfg.SMTPPort, m.cfg.SMTPUsername, m.cfg.SMTPPassword)
	return dialer.DialAndSend(msg)
}
