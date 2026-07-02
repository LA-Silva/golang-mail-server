package imap

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/backend"
	imapserver "github.com/emersion/go-imap/server"
	"mailserver/internal/config"
	"mailserver/internal/storage"
)

type Backend struct {
	cfg *config.Config
}

type User struct {
	username string
	cfg      *config.Config
}

type Mailbox struct {
	name     string
	username string
	cfg      *config.Config
}

// BytesLiteral implements the imap.Literal interface
type BytesLiteral struct {
	data []byte
}

func (b *BytesLiteral) Len() int {
	return len(b.data)
}

func (b *BytesLiteral) Read(p []byte) (int, error) {
	return bytes.NewReader(b.data).Read(p)
}

func NewServer(cfg *config.Config) *imapserver.Server {
	s := imapserver.New(NewBackend(cfg))
	s.Addr = cfg.IMAPPort
	s.AllowInsecureAuth = true
	return s
}

func NewBackend(cfg *config.Config) backend.Backend {
	return &Backend{cfg: cfg}
}

func (b *Backend) Login(connInfo *imap.ConnInfo, username, password string) (backend.User, error) {
	if !b.cfg.ValidateUser(username, password) {
		return nil, fmt.Errorf("invalid credentials")
	}

	log.Printf("IMAP: User %s logged in from %s", username, connInfo.RemoteAddr)
	return &User{username: username, cfg: b.cfg}, nil
}

// Implement backend.User interface
func (u *User) Username() string {
	return u.username
}

func (u *User) ListMailboxes(subscribed bool) (mailboxes []backend.Mailbox, err error) {
	inbox := &Mailbox{name: "INBOX", username: u.username, cfg: u.cfg}
	return []backend.Mailbox{inbox}, nil
}

func (u *User) GetMailbox(name string) (backend.Mailbox, error) {
	if name != "INBOX" {
		return nil, fmt.Errorf("mailbox not found")
	}
	return &Mailbox{name: name, username: u.username, cfg: u.cfg}, nil
}

func (u *User) CreateMailbox(name string) error {
	return fmt.Errorf("not implemented")
}

func (u *User) DeleteMailbox(name string) error {
	return fmt.Errorf("not implemented")
}

func (u *User) RenameMailbox(existingName, newName string) error {
	return fmt.Errorf("not implemented")
}

func (u *User) Logout() error {
	log.Printf("IMAP: User %s logged out", u.username)
	return nil
}

// Implement backend.Mailbox interface
func (mb *Mailbox) Name() string {
	return mb.name
}

func (mb *Mailbox) Info() (*imap.MailboxInfo, error) {
	return &imap.MailboxInfo{
		Attributes: nil,
		Delimiter:  "/",
		Name:       mb.name,
	}, nil
}

func (mb *Mailbox) Status(items []imap.StatusItem) (*imap.MailboxStatus, error) {
	s3Storage := storage.NewS3Storage(mb.cfg.S3Client, mb.cfg.S3Bucket)
	ctx := context.Background()

	emails, err := s3Storage.ListEmails(ctx, mb.username)
	if err != nil {
		return nil, err
	}

	status := &imap.MailboxStatus{
		Name: mb.name,
	}

	for _, item := range items {
		switch item {
		case imap.StatusMessages:
			status.Messages = uint32(len(emails))
		case imap.StatusRecent:
			status.Recent = uint32(len(emails))
		case imap.StatusUnseen:
			status.Unseen = uint32(len(emails))
		}
	}

	return status, nil
}

func (mb *Mailbox) SetSubscribed(subscribed bool) error {
	return nil
}

func (mb *Mailbox) Check() error {
	return nil
}

func (mb *Mailbox) ListMessages(uid bool, seqSet *imap.SeqSet, items []imap.FetchItem, ch chan<- *imap.Message) error {
	defer close(ch)

	s3Storage := storage.NewS3Storage(mb.cfg.S3Client, mb.cfg.S3Bucket)
	ctx := context.Background()

	emails, err := s3Storage.ListEmails(ctx, mb.username)
	if err != nil {
		return err
	}

	for i, emailID := range emails {
		emailData, err := s3Storage.RetrieveEmail(ctx, mb.username, emailID)
		if err != nil {
			log.Printf("Error retrieving email %s: %v", emailID, err)
			continue
		}

		msg := &imap.Message{
			SeqNum: uint32(i + 1),
			Uid:    uint32(i + 1),
			Body:   make(map[*imap.BodySectionName]imap.Literal),
		}

		// Process requested items - store with proper keys
		for _, item := range items {
			switch item {
			case imap.FetchBody, imap.FetchBodyStructure, imap.FetchRFC822:
				// For these items, store the literal data with empty BodySectionName
				msg.Body[&imap.BodySectionName{}] = &BytesLiteral{data: emailData}
			}
		}

		select {
		case ch <- msg:
		}
	}

	return nil
}

func (mb *Mailbox) SearchMessages(uid bool, criteria *imap.SearchCriteria) ([]uint32, error) {
	s3Storage := storage.NewS3Storage(mb.cfg.S3Client, mb.cfg.S3Bucket)
	ctx := context.Background()

	emails, err := s3Storage.ListEmails(ctx, mb.username)
	if err != nil {
		return nil, err
	}

	var ids []uint32
	for i := range emails {
		ids = append(ids, uint32(i+1))
	}
	return ids, nil
}

func (mb *Mailbox) CreateMessage(flags []string, date time.Time, body imap.Literal) error {
	return fmt.Errorf("not implemented")
}

func (mb *Mailbox) UpdateMessagesFlags(uid bool, seqSet *imap.SeqSet, operation imap.FlagsOp, flags []string) error {
	return nil
}

func (mb *Mailbox) CopyMessages(uid bool, seqSet *imap.SeqSet, dest string) error {
	return fmt.Errorf("not implemented")
}

func (mb *Mailbox) Expunge() error {
	return nil
}
