# Golang IMAP/SMTP Mail Server

A simple mail server implementation in Go that uses Amazon S3 as backend storage.

## Features

- SMTP server for receiving emails
- IMAP server for accessing emails
- Basic authentication from TSV password file
- Amazon S3 backend storage
- Support for multiple users
- No TLS/SSL support (plain connections)

## Requirements

- Go 1.21+
- AWS account with S3 access
- AWS credentials configured locally

## Setup

1. Clone the repository:
```bash
git clone https://github.com/LA-Silva/golang-mail-server.git
cd golang-mail-server
```

2. Install dependencies:
```bash
go mod download
```

3. Configure AWS credentials:
```bash
aws configure
```

4. Create password file with TSV format:
```bash
cat > /etc/mypas.tsv << EOF
user1	password1
user2	password2
EOF
```

5. Run the server:
```bash
go run main.go \
  --s3-bucket your-bucket-name \
  --s3-region us-east-1 \
  --password-file /etc/mypas.tsv
```

## Configuration

### Command-line flags:
- `-smtp-port` (default: `:25`) - SMTP server port
- `-imap-port` (default: `:143`) - IMAP server port
- `-s3-bucket` (required) - S3 bucket name for storing emails
- `-s3-region` (default: `us-east-1`) - AWS region
- `-password-file` (required) - Path to TSV file with usernames and passwords

### Password file format

Tab-separated values: `username\tpassword`

```
# Comments and empty lines are ignored
user1	password1
user2	password2
admin	adminpass123
```

## Usage

### Sending emails (SMTP)

Use any SMTP client with:
- Host: localhost
- Port: 25 (or custom with `-smtp-port`)
- Username: user1
- Password: password1

### Receiving emails (IMAP)

Use any IMAP client with:
- Host: localhost
- Port: 143 (or custom with `-imap-port`)
- Username: user1
- Password: password1

## Architecture

- **SMTP Backend**: Receives emails and stores them in S3
- **IMAP Backend**: Provides access to stored emails via IMAP protocol
- **S3 Storage**: Stores emails with structure: `emails/{username}/{emailID}`
- **Password File**: TSV file containing user credentials

## Limitations

- No TLS/SSL support
- Basic authentication only
- Simplified IMAP implementation
- No support for multiple mailboxes beyond INBOX
- Password file must be readable by the server process
