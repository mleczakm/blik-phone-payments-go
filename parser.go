// Package blikpayments imports bank transfer notifications (BLIK to phone / bank transfer) from an IMAP
// mailbox and matches them to orders by the 4-character payment code in the transfer title.
package blikpayments

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

// Notification is a parsed incoming-transfer notification.
type Notification struct {
	Sender string
	Title  string
	Amount int64 // grosz
}

// Parser extracts a transfer from a bank notification e-mail; ok=false when the e-mail is not a notification it understands.
type Parser interface {
	Parse(subject, body string) (n Notification, ok bool)
}

// ErrBadAmount is returned for amounts that cannot be parsed.
var ErrBadAmount = errors.New("invalid amount")

// ParseAmount converts a Polish formatted amount ("1 234,56", "1234.5", "12") to grosz without float rounding.
func ParseAmount(s string) (int64, error) {
	s = strings.NewReplacer(" ", "", " ", "", "\t", "").Replace(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, ",", ".")
	if s == "" {
		return 0, ErrBadAmount
	}
	whole, frac, _ := strings.Cut(s, ".")
	if strings.Contains(frac, ".") || len(frac) > 2 {
		return 0, ErrBadAmount
	}
	for len(frac) < 2 {
		frac += "0"
	}
	w, err := strconv.ParseInt(whole, 10, 64)
	if err != nil || w < 0 {
		return 0, ErrBadAmount
	}
	f, err := strconv.ParseInt(frac, 10, 64)
	if err != nil || f < 0 || f > 99 {
		return 0, ErrBadAmount
	}
	if w > (int64(^uint64(0)>>1)-f)/100 {
		return 0, ErrBadAmount
	}
	return w*100 + f, nil
}

// AliorParser understands Alior Bank "Uznanie rachunku" notifications (the format used by kiddo).
type AliorParser struct{}

var (
	aliorSubject = regexp.MustCompile(`Uznanie rachunku [0-9.]+ kwotą ([0-9 ,.\x{00a0}]+) PLN`)
	aliorBody    = regexp.MustCompile(`(?s)kwotą ([0-9., \x{00a0}]+) PLN.*?Nadawca: (.*?)<br.*?Tytuł zlecenia: (.*?)<br`)
)

// Parse implements Parser.
func (AliorParser) Parse(subject, body string) (Notification, bool) {
	if !aliorSubject.MatchString(subject) {
		return Notification{}, false
	}
	m := aliorBody.FindStringSubmatch(body)
	if m == nil {
		return Notification{}, false
	}
	amount, err := ParseAmount(m[1])
	if err != nil {
		return Notification{}, false
	}
	return Notification{Sender: strings.TrimSpace(m[2]), Title: strings.TrimSpace(m[3]), Amount: amount}, true
}

// GenericParser understands "Uznanie rachunku - Kwota: 123,45 PLN - Nadawca: Jan Kowalski" subjects with a "Tytuł:" line in
// the body (the format used by cargo.mleczki.pl).
type GenericParser struct{}

var (
	genericAmount = regexp.MustCompile(`Kwota:\s*([\d\s,.]+?)\s*PLN`)
	genericSender = regexp.MustCompile(`Nadawca:\s*(.+?)(?:\s+-\s+|$)`)
	genericTitle  = regexp.MustCompile(`(?i)tytu[łl](?: przelewu| zlecenia)?:\s*(.+?)(?:<br|\r?\n|$)`)
)

// Parse implements Parser.
func (GenericParser) Parse(subject, body string) (Notification, bool) {
	m := genericAmount.FindStringSubmatch(subject)
	if m == nil {
		return Notification{}, false
	}
	amount, err := ParseAmount(m[1])
	if err != nil {
		return Notification{}, false
	}
	n := Notification{Amount: amount}
	if s := genericSender.FindStringSubmatch(subject); s != nil {
		n.Sender = strings.TrimSpace(s[1])
	}
	if t := genericTitle.FindStringSubmatch(body); t != nil {
		n.Title = strings.TrimSpace(t[1])
	} else if t := genericTitle.FindStringSubmatch(subject); t != nil {
		n.Title = strings.TrimSpace(t[1])
	}
	return n, true
}

// ChainParser tries parsers in order.
type ChainParser []Parser

// Parse implements Parser.
func (c ChainParser) Parse(subject, body string) (Notification, bool) {
	for _, p := range c {
		if n, ok := p.Parse(subject, body); ok {
			return n, true
		}
	}
	return Notification{}, false
}
