package main

import (
	"context"
	"flag"
	"log"

	"mailserver/internal/config"
	"mailserver/internal/imap"
	"mailserver/internal/smtp"
	"mailserver/internal/storage"
)

func main() {
	smtpPort := flag.String("smtp-port", ":25", "SMTP server port")
	imapPort := flag.String("imap-port", ":143", "IMAP server port")
	s3Bucket := flag.String("s3-bucket", "", "S3 bucket name")
	s3Region := flag.String("s3-region", "us-east-1", "AWS region")
	passwordFile := flag.String("password-file", "", "Path to TSV file containing username and password (tab-separated)")
	flag.Parse()

	if *s3Bucket == "" {
		log.Fatal("S3 bucket name is required (--s3-bucket)")
	}

	if *passwordFile == "" {
		log.Fatal("Password file is required (--password-file)")
	}

	ctx := context.Background()

	// Load users from password file
	users, err := config.LoadUsersFromFile(*passwordFile)
	if err != nil {
		log.Fatalf("Failed to load users from password file: %v", err)
	}

	if len(users) == 0 {
		log.Fatal("No users found in password file")
	}

	log.Printf("Loaded %d users from %s", len(users), *passwordFile)

	// Initialize S3 storage
	s3Client, err := storage.NewS3Client(ctx, *s3Region, *s3Bucket)
	if err != nil {
		log.Fatalf("Failed to initialize S3 client: %v", err)
	}

	cfg := &config.Config{
		SMTPPort:  *smtpPort,
		IMAPPort:  *imapPort,
		S3Bucket:  *s3Bucket,
		S3Region:  *s3Region,
		S3Client:  s3Client,
		Users:     users,
	}

	// Start SMTP server
	go func() {
		smtpServer := smtp.NewServer(cfg)
		log.Printf("Starting SMTP server on %s", *smtpPort)
		if err := smtpServer.ListenAndServe(); err != nil {
			log.Printf("SMTP server error: %v", err)
		}
	}()

	// Start IMAP server
	imapServer := imap.NewServer(cfg)
	log.Printf("Starting IMAP server on %s", *imapPort)
	if err := imapServer.ListenAndServe(); err != nil {
		log.Printf("IMAP server error: %v", err)
	}
}
