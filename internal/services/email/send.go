package email

import (
	"fmt"
	"net/smtp"
)

func (ec EmailConfig) send(from string, to []string, body []byte) error {
	err := smtp.SendMail(fmt.Sprintf("%s:%d", ec.EmailHost, ec.EmailPort), nil, from, to, body)
	if err != nil {
		ec.Log.Printf("[ERROR] Send failed: %s", err)
		return err
	}
	return nil
}
