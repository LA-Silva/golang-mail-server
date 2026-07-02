package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

type S3Storage struct {
	client *s3.Client
	bucket string
}

func NewS3Client(ctx context.Context, region, bucket string) (*s3.Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config: %v", err)
	}

	return s3.NewFromConfig(cfg), nil
}

func NewS3Storage(client *s3.Client, bucket string) *S3Storage {
	return &S3Storage{
		client: client,
		bucket: bucket,
	}
}

// StoreEmail stores an email in S3
func (s *S3Storage) StoreEmail(ctx context.Context, username string, emailData []byte) (string, error) {
	emailID := uuid.New().String()
	key := fmt.Sprintf("emails/%s/%s", username, emailID)

	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(emailData),
	})

	if err != nil {
		return "", fmt.Errorf("failed to store email in S3: %v", err)
	}

	log.Printf("Stored email %s for user %s", emailID, username)
	return emailID, nil
}

// RetrieveEmail retrieves an email from S3
func (s *S3Storage) RetrieveEmail(ctx context.Context, username, emailID string) ([]byte, error) {
	key := fmt.Sprintf("emails/%s/%s", username, emailID)

	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve email from S3: %v", err)
	}

	defer result.Body.Close()
	return io.ReadAll(result.Body)
}

// ListEmails lists all emails for a user
func (s *S3Storage) ListEmails(ctx context.Context, username string) ([]string, error) {
	prefix := fmt.Sprintf("emails/%s/", username)
	var emailIDs []string

	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list emails from S3: %v", err)
		}

		for _, obj := range page.Contents {
			// Extract email ID from key (format: emails/username/emailID)
			emailID := (*obj.Key)[len(prefix):]
			emailIDs = append(emailIDs, emailID)
		}
	}

	return emailIDs, nil
}

// DeleteEmail deletes an email from S3
func (s *S3Storage) DeleteEmail(ctx context.Context, username, emailID string) error {
	key := fmt.Sprintf("emails/%s/%s", username, emailID)

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		return fmt.Errorf("failed to delete email from S3: %v", err)
	}

	log.Printf("Deleted email %s for user %s", emailID, username)
	return nil
}
