package email

import (
	"log"

	"gitlab.com/pennersr/shove/internal/services"
)

const serviceID = "email"

type EmailConfig struct {
	EmailHost string
	EmailPort int
	Log       *log.Logger
}

type EmailService struct {
	config EmailConfig
}

func NewEmailService(config EmailConfig) (es *EmailService, err error) {
	es = &EmailService{
		config: config,
	}
	return
}

func (es *EmailService) Logger() *log.Logger {
	return es.config.Log
}

func (es *EmailService) ID() string {
	return serviceID
}

func (es *EmailService) String() string {
	return "Email"
}

func (es *EmailService) NewClient() (services.PumpClient, error) {
	return nil, nil
}

func (es *EmailService) SquashAndPushMessage(client services.PumpClient, smsgs []services.ServiceMessage, fc services.FeedbackCollector) services.PushStatus {
	emails := make([]email, len(smsgs))
	for i, smsg := range smsgs {
		emails[i] = smsg.(email)
	}
	body, err := encodeEmailDigest(emails)
	if err != nil {
		return services.PushStatusHardFail
	}
	return es.push(emails[0].From, emails[0].To, body, fc)
}

func (es *EmailService) PushMessage(pclient services.PumpClient, smsg services.ServiceMessage, fc services.FeedbackCollector) (status services.PushStatus) {
	email := smsg.(email)
	es.config.Log.Println("Sending email")
	body, err := encodeEmail(email)
	if err != nil {
		return services.PushStatusHardFail
	}
	return es.push(email.From, email.To, body, fc)
}
func (es *EmailService) push(from string, to []string, body []byte, fc services.FeedbackCollector) services.PushStatus {
	err := es.config.send(from, to, body, fc)
	if err != nil {
		es.config.Log.Printf("[ERROR] Sending email failed: %s", err)
		return services.PushStatusHardFail // TODO: smtp down is not a hard failure
	}
	return services.PushStatusSuccess
}
