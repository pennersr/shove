package email

import (
	"fmt"
	"log"
	"net/smtp"
)

func (ec EmailConfig) send(from string, to []string, body []byte) error {
	err := smtp.SendMail(fmt.Sprintf("%s:%d", ec.EmailHost, ec.EmailPort), nil, from, to, body)
	if err != nil {
		log.Printf("[ERROR] Send mail failed: %s", err)
		return err
	}
	return nil
}
