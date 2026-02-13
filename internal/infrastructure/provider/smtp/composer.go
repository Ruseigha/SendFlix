package smtp

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"mime"
	"strings"
	"time"

	"github.com/Ruseigha/SendFlix/internal/domain"
)

// MessageComposer composes RFC 5322 compliant email messages
//
// RFC 5322: Internet Message Format
// https://tools.ietf.org/html/rfc5322
//
// MESSAGE FORMAT:
// Headers (From, To, Subject, etc.)
// Empty line
// Body (text and/or HTML)
// Attachments (if any)
type MessageComposer struct{}

// NewMessageComposer creates new composer
func NewMessageComposer() *MessageComposer {
	return &MessageComposer{}
}

// Compose creates RFC 5322 message
//
// PROCESS:
// 1. Write headers (From, To, Subject, etc.)
// 2. Set MIME boundaries if multipart
// 3. Write body (text/html)
// 4. Write attachments
// 5. Close boundaries
//
// PARAMETERS:
// - email: Email to compose
//
// RETURNS:
// - []byte: Composed message
// - error: If composition fails
func (c *MessageComposer) Compose(email *domain.Email) ([]byte, error) {
	var buf bytes.Buffer

	// Write headers
	c.writeHeaders(&buf, email)

	// Determine content type
	hasHTML := email.BodyHTML != ""
	hasText := email.BodyText != ""
	hasAttachments := len(email.Attachments) > 0

	if hasAttachments || (hasHTML && hasText) {
		// Multipart message
		boundary := c.generateBoundary()
		buf.WriteString("MIME-Version: 1.0\r\n")
		buf.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n", boundary))
		buf.WriteString("\r\n")

		// Body parts
		if hasHTML && hasText {
			c.writeAlternativeParts(&buf, email, boundary)
		} else if hasHTML {
			c.writeHTMLPart(&buf, email, boundary)
		} else if hasText {
			c.writeTextPart(&buf, email, boundary)
		}

		// Attachments
		for _, attachment := range email.Attachments {
			c.writeAttachment(&buf, attachment, boundary)
		}

		// Close boundary
		buf.WriteString(fmt.Sprintf("\r\n--%s--\r\n", boundary))

	} else if hasHTML {
		// HTML only
		buf.WriteString("MIME-Version: 1.0\r\n")
		buf.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
		buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		buf.WriteString("\r\n")
		buf.WriteString(c.encodeQuotedPrintable(email.BodyHTML))

	} else {
		// Text only
		buf.WriteString("MIME-Version: 1.0\r\n")
		buf.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
		buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		buf.WriteString("\r\n")
		buf.WriteString(c.encodeQuotedPrintable(email.BodyText))
	}

	return buf.Bytes(), nil
}

// writeHeaders writes email headers
func (c *MessageComposer) writeHeaders(buf *bytes.Buffer, email *domain.Email) {
	// From
	buf.WriteString(fmt.Sprintf("From: %s\r\n", email.From))

	// To
	buf.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(email.To, ", ")))

	// CC
	if len(email.CC) > 0 {
		buf.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(email.CC, ", ")))
	}

	// Subject (encode if contains non-ASCII)
	subject := email.Subject
	if c.needsEncoding(subject) {
		subject = mime.BEncoding.Encode("UTF-8", subject)
	}
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))

	// Date
	buf.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))

	// Message-ID
	messageID := fmt.Sprintf("<%d.%s@sendflix.local>",
		time.Now().Unix(),
		c.generateBoundary()[:16])
	buf.WriteString(fmt.Sprintf("Message-ID: %s\r\n", messageID))

	// Reply-To
	if email.ReplyTo != "" {
		buf.WriteString(fmt.Sprintf("Reply-To: %s\r\n", email.ReplyTo))
	}

	// X-Priority
	switch email.Priority {
	case domain.EmailPriorityHigh:
		buf.WriteString("X-Priority: 1 (Highest)\r\n")
		buf.WriteString("Importance: high\r\n")
	case domain.EmailPriorityLow:
		buf.WriteString("X-Priority: 5 (Lowest)\r\n")
		buf.WriteString("Importance: low\r\n")
	}

	// X-Mailer
	buf.WriteString("X-Mailer: SendFlix v1.0\r\n")
}

// writeAlternativeParts writes text and HTML alternatives
func (c *MessageComposer) writeAlternativeParts(buf *bytes.Buffer, email *domain.Email, outerBoundary string) {
	altBoundary := c.generateBoundary()

	buf.WriteString(fmt.Sprintf("\r\n--%s\r\n", outerBoundary))
	buf.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", altBoundary))
	buf.WriteString("\r\n")

	// Text part
	buf.WriteString(fmt.Sprintf("\r\n--%s\r\n", altBoundary))
	buf.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
	buf.WriteString("\r\n")
	buf.WriteString(c.encodeQuotedPrintable(email.BodyText))

	// HTML part
	buf.WriteString(fmt.Sprintf("\r\n--%s\r\n", altBoundary))
	buf.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
	buf.WriteString("\r\n")
	buf.WriteString(c.encodeQuotedPrintable(email.BodyHTML))

	buf.WriteString(fmt.Sprintf("\r\n--%s--\r\n", altBoundary))
}

// writeTextPart writes plain text part
func (c *MessageComposer) writeTextPart(buf *bytes.Buffer, email *domain.Email, boundary string) {
	buf.WriteString(fmt.Sprintf("\r\n--%s\r\n", boundary))
	buf.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
	buf.WriteString("\r\n")
	buf.WriteString(c.encodeQuotedPrintable(email.BodyText))
}

// writeHTMLPart writes HTML part
func (c *MessageComposer) writeHTMLPart(buf *bytes.Buffer, email *domain.Email, boundary string) {
	buf.WriteString(fmt.Sprintf("\r\n--%s\r\n", boundary))
	buf.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
	buf.WriteString("\r\n")
	buf.WriteString(c.encodeQuotedPrintable(email.BodyHTML))
}

// writeAttachment writes attachment
func (c *MessageComposer) writeAttachment(buf *bytes.Buffer, attachment domain.Attachment, boundary string) {
	buf.WriteString(fmt.Sprintf("\r\n--%s\r\n", boundary))

	// Content-Type
	contentType := attachment.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	buf.WriteString(fmt.Sprintf("Content-Type: %s; name=\"%s\"\r\n", contentType, attachment.Filename))

	// Content-Disposition
	if attachment.Inline {
		buf.WriteString(fmt.Sprintf("Content-Disposition: inline; filename=\"%s\"\r\n", attachment.Filename))
		if attachment.ContentID != "" {
			buf.WriteString(fmt.Sprintf("Content-ID: <%s>\r\n", attachment.ContentID))
		}
	} else {
		buf.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n", attachment.Filename))
	}

	buf.WriteString("Content-Transfer-Encoding: base64\r\n")
	buf.WriteString("\r\n")

	// Encode content
	encoded := base64.StdEncoding.EncodeToString(attachment.Content)
	// Split into 76-character lines
	for i := 0; i < len(encoded); i += 76 {
		end := i + 76
		if end > len(encoded) {
			end = len(encoded)
		}
		buf.WriteString(encoded[i:end])
		buf.WriteString("\r\n")
	}
}

// generateBoundary generates MIME boundary
func (c *MessageComposer) generateBoundary() string {
	return fmt.Sprintf("boundary_%d", time.Now().UnixNano())
}

// needsEncoding checks if string needs MIME encoding
func (c *MessageComposer) needsEncoding(s string) bool {
	for _, r := range s {
		if r > 127 {
			return true
		}
	}
	return false
}

// encodeQuotedPrintable encodes text in quoted-printable
func (c *MessageComposer) encodeQuotedPrintable(text string) string {
	var buf bytes.Buffer
	lineLen := 0

	for i := 0; i < len(text); i++ {
		ch := text[i]

		// Check if needs encoding
		if ch == '=' || ch < 33 || ch > 126 || ch == '\r' || ch == '\n' {
			// Encode as =XX
			if ch == '\r' || ch == '\n' {
				buf.WriteByte(ch)
				lineLen = 0
			} else {
				encoded := fmt.Sprintf("=%02X", ch)
				buf.WriteString(encoded)
				lineLen += 3
			}
		} else {
			buf.WriteByte(ch)
			lineLen++
		}

		// Soft line break at 76 characters
		if lineLen >= 75 && i+1 < len(text) {
			buf.WriteString("=\r\n")
			lineLen = 0
		}
	}

	return buf.String()
}