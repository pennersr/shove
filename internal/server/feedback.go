package server

import (
	"encoding/json"
	"log"
	"net/http"
)

type tokenFeedback struct {
	Service     string `json:"service"`
	Token       string `json:"token"`
	Replacement string `json:"replacement_token,omitempty"`
	Reason      string `json:"reason"`
}

func (s *Server) handleFeedback(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Invalid request method.", 405)
		return
	}
	s.feedbackLock.Lock()
	j, err := json.Marshal(struct {
		Feedback []tokenFeedback `json:"feedback"`
	}{Feedback: s.feedback})
	if err != nil {
		s.feedbackLock.Unlock()
		http.Error(w, err.Error(), 500)
		return
	}
	s.feedback = make([]tokenFeedback, 0)
	s.feedbackLock.Unlock()
	w.Header().Set("Content-Type", "application/json")
	w.Write(j)
}

// TokenInvalid ...
func (s *Server) TokenInvalid(serviceID, token string) {
	s.feedbackLock.Lock()
	s.feedback = append(s.feedback, tokenFeedback{serviceID, token, "", "invalid"})
	s.feedbackLock.Unlock()
	log.Println("Invalid", serviceID, "token:", token)
}

// ReplaceToken ...
func (s *Server) ReplaceToken(serviceID, token, replacement string) {
	s.feedbackLock.Lock()
	s.feedback = append(s.feedback, tokenFeedback{serviceID, token, replacement, "replaced"})
	s.feedbackLock.Unlock()
	log.Println(serviceID, "token replaced")
}
