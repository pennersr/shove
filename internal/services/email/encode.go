package email

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"net/textproto"

	jwemail "github.com/jordan-wright/email"
)

func encodeSMTPAddress(s string) (string, error) {
	addr, err := mail.ParseAddress(s)
	if err != nil {
		return "", err
	}
	return addr.Address, nil
}

func encodeSMTPAddresses(from string, to []string) (encFrom string, encTo []string, err error) {
	encFrom, err = encodeSMTPAddress(from)
	if err != nil {
		return
	}
	encTo = make([]string, 0, len(to))
	for _, t := range to {
		t, err = encodeSMTPAddress(t)
		if err != nil {
			return
		}
		encTo = append(encTo, t)
	}
	return
}

func encodeAddress(s string) (string, error) {
	addr, err := mail.ParseAddress(s)
	if err != nil {
		return "", err
	}
	return addr.String(), nil
}

func encodeRFC2047(s string) string {
	return mime.QEncoding.Encode("UTF-8", s)
}

func encodeEmail(em email) ([]byte, error) {
	e := jwemail.NewEmail()
	e.From = em.From
	e.To = em.To
	e.Subject = em.Subject
	e.Text = []byte(em.Text)
	e.HTML = []byte(em.HTML)
	for _, atm := range em.Attachments {
		e.Attachments = append(e.Attachments, &jwemail.Attachment{
			Filename:    atm.Filename,
			Content:     atm.Content,
			ContentType: atm.ContentType,
			Header:      textproto.MIMEHeader{},
		})
	}
	return e.Bytes()
}

func encodeEmailDigest(emails []email) ([]byte, error) {
	if len(emails) == 0 {
		return nil, errors.New("no emails specified")
	}
	if len(emails) == 1 {
		return encodeEmail(emails[0])
	}
	mixedContent := &bytes.Buffer{}
	mixedWriter := multipart.NewWriter(mixedContent)

	digWriter, err := nestedMultipart(mixedWriter, "multipart/digest")
	if err != nil {
		return nil, err
	}

	// Actual content alternatives (finally!)
	for i, em := range emails {
		filename := fmt.Sprintf("message-%d.eml", i+1)
		s, _ := encodeEmail(em)
		var childContent io.Writer
		childContent, _ = digWriter.CreatePart(textproto.MIMEHeader{
			"Content-Type":        {fmt.Sprintf("message/rfc822; name=\"%s\"", filename)},
			"Content-Disposition": {"inline; filename=" + filename},
		})
		childContent.Write([]byte(s))
	}
	digWriter.Close()
	mixedWriter.Close()

	headers := make(map[string]string)
	headers["To"] = emails[0].To[0]
	headers["From"] = emails[0].From
	subject := emails[0].Digest.Subject
	if subject == "" {
		subject = emails[0].Subject
	}
	headers["Subject"] = subject
	headers["Content-Type"] = "multipart/mixed; boundary=" + mixedWriter.Boundary()
	headers["MIME-Version"] = "1.0"

	var out bytes.Buffer
	err = encodeHeaders(&out, headers)
	if err != nil {
		return nil, err
	}
	out.WriteString(mixedContent.String())
	return out.Bytes(), nil

}

func encodeHeaders(out *bytes.Buffer, headers map[string]string) error {
	for k, v := range headers {
		out.WriteString(k)
		out.WriteString(": ")
		if k == "To" || k == "From" {
			var err error
			v, err = encodeAddress(v)
			if err != nil {
				return err
			}

		} else if k == "Subject" {
			v = encodeRFC2047(v)
		}
		out.WriteString(v)
		out.WriteString("\r\n")
	}
	out.WriteString("\r\n")
	return nil
}

func nestedMultipart(enclosingWriter *multipart.Writer, contentType string) (nestedWriter *multipart.Writer, err error) {
	boundary, err := randomBoundary()
	if err != nil {
		return
	}
	contentWithBoundary := contentType + "; boundary=\"" + boundary + "\""
	contentBuffer, err := enclosingWriter.CreatePart(textproto.MIMEHeader{"Content-Type": {contentWithBoundary}})
	if err != nil {
		return
	}

	nestedWriter = multipart.NewWriter(contentBuffer)
	nestedWriter.SetBoundary(boundary)
	return
}

func randomBoundary() (string, error) {
	var buf [30]byte
	_, err := io.ReadFull(rand.Reader, buf[:])
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", buf[:]), nil
}
