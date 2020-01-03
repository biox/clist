package parsemail

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/mail"
	"strings"
	"time"
)

// Parse an email message read from io.Reader into parsemail.Email struct
func Parse(r io.Reader) (email Email, err error) {
	msg, err := mail.ReadMessage(r)
	if err != nil {
		return
	}

	email, err = createEmailFromHeader(msg.Header)
	if err != nil {
		return
	}

	body, _ := ioutil.ReadAll(msg.Body)
	email.Body = string(body)
	return
}

func createEmailFromHeader(header mail.Header) (email Email, err error) {
	// todo: strip this down enormously, replace with anon funcs for email (mail.Subject, mail.To, etc)
	hp := headerParser{header: &header}

	email.Subject = decodeMimeSentence(header.Get("Subject"))
	email.From = hp.parseAddressList(header.Get("From"))
	email.Sender = hp.parseAddress(header.Get("Sender"))
	email.ReplyTo = hp.parseAddressList(header.Get("Reply-To"))
	email.To = hp.parseAddressList(header.Get("To"))
	email.Cc = hp.parseAddressList(header.Get("Cc"))
	email.Bcc = hp.parseAddressList(header.Get("Bcc"))
	email.Date = hp.parseTime(header.Get("Date"))
	email.ResentFrom = hp.parseAddressList(header.Get("Resent-From"))
	email.ResentSender = hp.parseAddress(header.Get("Resent-Sender"))
	email.ResentTo = hp.parseAddressList(header.Get("Resent-To"))
	email.ResentCc = hp.parseAddressList(header.Get("Resent-Cc"))
	email.ResentBcc = hp.parseAddressList(header.Get("Resent-Bcc"))
	email.ResentMessageID = hp.parseMessageId(header.Get("Resent-Message-ID"))
	email.MessageID = hp.parseMessageId(header.Get("Message-ID"))
	email.InReplyTo = hp.parseMessageIdList(header.Get("In-Reply-To"))
	email.References = hp.parseMessageIdList(header.Get("References"))
	email.ResentDate = hp.parseTime(header.Get("Resent-Date"))

	if hp.err != nil {
		err = hp.err
		return
	}

	//decode whole header for easier access to extra fields
	//todo: should we decode? aren't only standard fields mime encoded?
	email.Header, err = decodeHeaderMime(header)
	if err != nil {
		return
	}

	return
}

func decodeMimeSentence(s string) string {
	result := []string{}
	ss := strings.Split(s, " ")

	for _, word := range ss {
		dec := new(mime.WordDecoder)
		w, err := dec.Decode(word)
		if err != nil {
			if len(result) == 0 {
				w = word
			} else {
				w = " " + word
			}
		}

		result = append(result, w)
	}

	return strings.Join(result, "")
}

func decodeHeaderMime(header mail.Header) (mail.Header, error) {
	parsedHeader := map[string][]string{}

	for headerName, headerData := range header {

		parsedHeaderData := []string{}
		for _, headerValue := range headerData {
			parsedHeaderData = append(parsedHeaderData, decodeMimeSentence(headerValue))
		}

		parsedHeader[headerName] = parsedHeaderData
	}

	return mail.Header(parsedHeader), nil
}

type headerParser struct {
	header *mail.Header
	err    error
}

func (hp headerParser) parseAddress(s string) (ma *mail.Address) {
	if hp.err != nil {
		return nil
	}

	if strings.Trim(s, " \n") != "" {
		ma, hp.err = mail.ParseAddress(s)

		return ma
	}

	return nil
}

func (hp headerParser) parseAddressList(s string) (ma []*mail.Address) {
	if hp.err != nil {
		return
	}

	if strings.Trim(s, " \n") != "" {
		ma, hp.err = mail.ParseAddressList(s)
		return
	}

	return
}

func (hp headerParser) parseTime(s string) (t time.Time) {
	if hp.err != nil || s == "" {
		return
	}

	t, hp.err = time.Parse(time.RFC1123Z, s)
	if hp.err == nil {
		return t
	}

	t, hp.err = time.Parse("Mon, 2 Jan 2006 15:04:05 -0700", s)

	return
}

func (hp headerParser) parseMessageId(s string) string {
	if hp.err != nil {
		return ""
	}

	return strings.Trim(s, "<> ")
}

func (hp headerParser) parseMessageIdList(s string) (result []string) {
	if hp.err != nil {
		return
	}

	for _, p := range strings.Split(s, " ") {
		if strings.Trim(p, " \n") != "" {
			result = append(result, hp.parseMessageId(p))
		}
	}

	return
}

func (e *Email) ToBytes() []byte {
	var buf bytes.Buffer

	// print the headers in whatever darn order, who cares
	for k, v := range e.Header {
		fmt.Fprintf(&buf, "%s: %s\r\n", k, strings.Join(v, " "))
	}

	fmt.Fprintf(&buf, "\r\n%s", e.Body)

	return buf.Bytes()
}

// Email with fields for all the headers defined in RFC5322 with it's attachments and
type Email struct {
	Header mail.Header

	Subject    string
	Sender     *mail.Address
	From       []*mail.Address
	ReplyTo    []*mail.Address
	To         []*mail.Address
	Cc         []*mail.Address
	Bcc        []*mail.Address
	Date       time.Time
	MessageID  string
	InReplyTo  []string
	References []string

	ResentFrom      []*mail.Address
	ResentSender    *mail.Address
	ResentTo        []*mail.Address
	ResentDate      time.Time
	ResentCc        []*mail.Address
	ResentBcc       []*mail.Address
	ResentMessageID string

	Body string
}
