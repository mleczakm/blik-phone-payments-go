package payments

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

// Message is a fetched e-mail.
type Message struct {
	UID     uint32
	From    string
	Subject string
	Date    time.Time
	Body    string // HTML part when present, otherwise plain text
}

// Mailbox reads unseen bank notifications over IMAP.
type Mailbox struct {
	Server   string // host:993
	Username string
	Password string
	Folder   string // e.g. INBOX or a Gmail label such as "ceramiza"
	From     string // optional sender address filter, e.g. powiadomienia@alior.pl
}

// Configured reports whether credentials were provided.
func (m *Mailbox) Configured() bool {
	return m != nil && m.Server != "" && m.Username != "" && m.Password != ""
}

// FetchUnseen passes every unseen message to handle and marks those it accepted (handle returned true) as seen.
func (m *Mailbox) FetchUnseen(handle func(Message) bool) (int, error) {
	c, err := client.DialTLS(m.Server, &tls.Config{MinVersion: tls.VersionTLS12})
	if err != nil {
		return 0, fmt.Errorf("imap dial: %w", err)
	}
	defer func() { _ = c.Logout() }()
	if err := c.Login(m.Username, m.Password); err != nil {
		return 0, fmt.Errorf("imap login: %w", err)
	}
	if _, err := c.Select(m.Folder, false); err != nil {
		return 0, fmt.Errorf("imap select %s: %w", m.Folder, err)
	}
	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{imap.SeenFlag}
	criteria.Since = time.Now().AddDate(0, 0, -14)
	if m.From != "" {
		criteria.Header.Add("From", m.From)
	}
	uids, err := c.UidSearch(criteria)
	if err != nil {
		return 0, fmt.Errorf("imap search: %w", err)
	}
	if len(uids) == 0 {
		return 0, nil
	}
	seq := new(imap.SeqSet)
	seq.AddNum(uids...)
	section := &imap.BodySectionName{Peek: true}
	msgs := make(chan *imap.Message, 16)
	done := make(chan error, 1)
	go func() {
		done <- c.UidFetch(seq, []imap.FetchItem{imap.FetchEnvelope, imap.FetchUid, section.FetchItem()}, msgs)
	}()

	var handled []uint32
	for msg := range msgs {
		parsed, err := parseMessage(msg, section)
		if err != nil {
			continue
		}
		if handle(parsed) {
			handled = append(handled, msg.Uid)
		}
	}
	if err := <-done; err != nil {
		return 0, fmt.Errorf("imap fetch: %w", err)
	}
	if len(handled) > 0 {
		seen := new(imap.SeqSet)
		seen.AddNum(handled...)
		if err := c.UidStore(seen, imap.FormatFlagsOp(imap.AddFlags, true), []any{imap.SeenFlag}, nil); err != nil {
			return len(handled), fmt.Errorf("imap mark seen: %w", err)
		}
	}
	return len(handled), nil
}

func parseMessage(msg *imap.Message, section *imap.BodySectionName) (Message, error) {
	out := Message{UID: msg.Uid}
	if msg.Envelope != nil {
		out.Subject = msg.Envelope.Subject
		out.Date = msg.Envelope.Date
		if len(msg.Envelope.From) > 0 {
			out.From = msg.Envelope.From[0].Address()
		}
	}
	r := msg.GetBody(section)
	if r == nil {
		return out, fmt.Errorf("no body")
	}
	parsed, err := mail.ReadMessage(io.LimitReader(r, 2<<20))
	if err != nil {
		return out, err
	}
	if out.Subject == "" {
		dec := new(mime.WordDecoder)
		out.Subject, _ = dec.DecodeHeader(parsed.Header.Get("Subject"))
	}
	html, text, err := readParts(parsed.Header.Get("Content-Type"), parsed.Header.Get("Content-Transfer-Encoding"), parsed.Body)
	if err != nil {
		return out, err
	}
	out.Body = html
	if out.Body == "" {
		out.Body = text
	}
	return out, nil
}

// readParts walks a (possibly multipart) MIME body returning the decoded HTML and plain-text parts.
func readParts(contentType, encoding string, body io.Reader) (html, text string, err error) {
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = "text/plain"
	}
	if strings.HasPrefix(mediaType, "multipart/") {
		mr := multipart.NewReader(body, params["boundary"])
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				return html, text, err
			}
			h, t, err := readParts(p.Header.Get("Content-Type"), p.Header.Get("Content-Transfer-Encoding"), p)
			if err != nil {
				return html, text, err
			}
			if html == "" {
				html = h
			}
			if text == "" {
				text = t
			}
		}
		return html, text, nil
	}
	decoded, err := decode(encoding, body)
	if err != nil {
		return "", "", err
	}
	decoded = toUTF8(params["charset"], decoded)
	if mediaType == "text/html" {
		return decoded, "", nil
	}
	return "", decoded, nil
}

func decode(encoding string, r io.Reader) (string, error) {
	var b []byte
	var err error
	switch strings.ToLower(strings.TrimSpace(encoding)) {
	case "quoted-printable":
		b, err = io.ReadAll(quotedprintable.NewReader(r))
	case "base64":
		raw, rerr := io.ReadAll(r)
		if rerr != nil {
			return "", rerr
		}
		b, err = decodeBase64(raw)
	default:
		b, err = io.ReadAll(r)
	}
	return string(b), err
}

func decodeBase64(raw []byte) ([]byte, error) {
	clean := bytes.Map(func(r rune) rune {
		if r == '\r' || r == '\n' || r == ' ' {
			return -1
		}
		return r
	}, raw)
	return base64Std.DecodeString(string(clean))
}
