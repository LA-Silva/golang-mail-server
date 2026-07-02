package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Config struct {
	SMTPPort  string
	IMAPPort  string
	S3Bucket  string
	S3Region  string
	S3Client  *s3.Client
	Users     map[string]string
}

// LoadUsersFromFile loads users and passwords from a tab-separated file
// Format: username\tpassword
func LoadUsersFromFile(filepath string) (map[string]string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open password file: %w", err)
	}
	defer file.Close()

	users := make(map[string]string)
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid format at line %d: expected tab-separated username and password", lineNum)
		}

		username := strings.TrimSpace(parts[0])
		password := strings.TrimSpace(parts[1])

		if username == "" || password == "" {
			return nil, fmt.Errorf("empty username or password at line %d", lineNum)
		}

		users[username] = password
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading password file: %w", err)
	}

	return users, nil
}

// ValidateUser checks if the provided username and password are correct
func (c *Config) ValidateUser(username, password string) bool {
	pass, exists := c.Users[username]
	return exists && pass == password
}
