package smtp

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/emersion/go-smtp"
	"mailserver/internal/config"
	"mailserver/internal/storage"
)

type Backend struct {
	cfg *config.Config
}

type Session struct {
	username string
	cfg      *config.Config
}

func NewServer(cfg *config.Config) *smtp.Server {
	backend := &Backend{cfg: cfg}
	server := smtp.NewServer(backend)
	server.Addr = cfg.SMTPPort
	server.AllowInsecureAuth = true
	return server
}

// Implement Backend interface
func (b *Backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	return &Session{cfg: b.cfg}, nil
}

// Implement Session interface
func (sess *Session) AuthPlain(username, password string) error {
	if !sess.cfg.ValidateUser(username, password) {
		return fmt.Errorf("invalid credentials")
	}

	sess.username = username
	log.Printf("SMTP: User %s authenticated", username)
	return nil
}

func (sess *Session) Mail(from string, opts *smtp.MailOptions) error {
	if sess.username == "" {
		return fmt.Errorf("not authenticated")
	}

	log.Printf("SMTP: User %s sending email from %s", sess.username, from)
	return nil
}

func (sess *Session) Rcpt(to string, opts *smtp.RcptOptions) error {
	log.Printf("SMTP: Email will be delivered to %s", to)
	return nil
}

func (sess *Session) Data(r io.Reader) error {
	if sess.username == "" {
		return fmt.Errorf("not authenticated")
	}

	s3Storage := storage.NewS3Storage(sess.cfg.S3Client, sess.cfg.S3Bucket)

	// Read email data
	emailData, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("failed to read email data: %v", err)
	}

	// Store in S3
	ctx := context.Background()
	emailID, err := s3Storage.StoreEmail(ctx, sess.username, emailData)
	if err != nil {
		return fmt.Errorf("failed to store email: %v", err)
	}

	log.Printf("SMTP: Email %s stored successfully for user %s", emailID, sess.username)
	return nil
}

func (sess *Session) Reset() {
	log.Printf("SMTP: Session reset for user %s", sess.username)
}

func (sess *Session) Logout() error {
	log.Printf("SMTP: User %s logged out", sess.username)
	return nil
}
