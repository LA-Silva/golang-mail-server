package smtp

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"

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
	return smtp.NewServer(backend)
}

func (s *Backend) Login(state *smtp.ConnectionState, username, password string) (smtp.Session, error) {
	if !s.cfg.ValidateUser(username, password) {
		return nil, smtp.ErrAuthFailed
	}

	log.Printf("SMTP: User %s logged in", username)
	return &Session{username: username, cfg: s.cfg}, nil
}

func (s *Backend) AnonymousLogin(state *smtp.ConnectionState) (smtp.Session, error) {
	return nil, smtp.ErrAuthRequired
}

func (sess *Session) Mail(from string, opts *smtp.MailOptions) error {
	log.Printf("SMTP: Receiving email from %s to user %s", from, sess.username)
	return nil
}

func (sess *Session) Rcpt(to string, opts *smtp.RcptOptions) error {
	// For simplicity, accept all recipients
	log.Printf("SMTP: Email will be delivered to %s", to)
	return nil
}

func (sess *Session) Data(r io.Reader) error {
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

func (s *Backend) ListenAndServe() error {
	return s.Listen("tcp", ":25", func(c net.Conn) error {
		_, err := s.HandleConn(c)
		return err
	})
}

func (s *Backend) Listen(network, addr string, callback func(net.Conn) error) error {
	listener, err := net.Listen(network, addr)
	if err != nil {
		return err
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}

		go func() {
			if err := callback(conn); err != nil {
				log.Printf("SMTP error: %v", err)
			}
		}()
	}
}

func (s *Backend) HandleConn(c net.Conn) (error, error) {
	session, err := smtp.NewSession(c, s)
	if err != nil {
		return err, nil
	}
	return session.Serve(), nil
}
