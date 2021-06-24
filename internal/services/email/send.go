package email

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"time"

	"gitlab.com/pennersr/shove/internal/services"
)

func (ec EmailConfig) send(from string, to []string, body []byte, fc services.FeedbackCollector) error {
	t := time.Now()
	addr := fmt.Sprintf("%s:%d", ec.EmailHost, ec.EmailPort)
	var auth smtp.Auth

	var err error
	if !ec.TLS {
		err = smtp.SendMail(addr, auth, from, to, body)
	} else {
		err = ec.sendMailTLS(addr, auth, from, to, body)
	}

	duration := time.Since(t)
	fc.CountPush(serviceID, err == nil, duration)

	if err != nil {
		ec.Log.Printf("[ERROR] Send failed: %s", err)
		return err
	}
	return nil
}

func (ec EmailConfig) sendMailTLS(addr string, auth smtp.Auth, from string, to []string, body []byte) error {
	var t *tls.Config
	if ec.TLSInsecure {
		t = &tls.Config{InsecureSkipVerify: true}
	}
	c, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer c.Close()
	if err = c.Hello("localhost"); err != nil {
		return err
	}
	// Use TLS if available
	if ok, _ := c.Extension("STARTTLS"); ok {
		if err = c.StartTLS(t); err != nil {
			return err
		}
	}

	if auth != nil {
		if ok, _ := c.Extension("AUTH"); ok {
			if err = c.Auth(auth); err != nil {
				return err
			}
		}
	}
	if err = c.Mail(from); err != nil {
		return err
	}
	for _, addr := range to {
		if err = c.Rcpt(addr); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	_, err = w.Write(body)
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}
	return c.Quit()
}
