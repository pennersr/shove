package email

import (
	"encoding/json"
	"errors"
)

type attachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content-type"`
	Content     []byte `json:"content"`
}
type email struct {
	Subject     string       `json:"subject"`
	To          []string     `json:"to"`
	From        string       `json:"from"`
	Text        string       `json:"text"`
	HTML        string       `json:"html"`
	Attachments []attachment `json:"attachments"`
	Digest      struct {
		Subject string `json:"subject"`
	} `json:"digest"`
}

func (es *EmailService) convert(data []byte) (email, error) {
	var em email
	if err := json.Unmarshal(data, &em); err != nil {
		return em, err
	}
	if len(em.To) == 0 {
		return em, errors.New("missing: `to`")
	}
	if len(em.To) != 1 {
		return em, errors.New("only one `to` is supported")
	}
	if len(em.From) == 0 {
		return em, errors.New("missing: `from`")
	}
	if len(em.Subject) == 0 {
		return em, errors.New("missing: `subject`")
	}
	return em, nil
}

func (es *EmailService) Validate(data []byte) error {
	_, err := es.convert(data)
	return err
}
