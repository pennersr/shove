package server

import (
	"encoding/json"
	"golang.org/x/exp/slog"
	"net/http"
)

const (
	tokenFeedbackReasonInvalid   = "invalid"
	tokenFeedbackReasonThrottled = "throttled"
	tokenFeedbackReasonReplaced  = "replaced"
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
	s.feedback = append(s.feedback, tokenFeedback{serviceID, token, "", tokenFeedbackReasonInvalid})
	s.feedbackLock.Unlock()
	slog.Info("Invalid token", "service", serviceID, "token", token)
}

// TokenThrottled ...
func (s *Server) TokenThrottled(serviceID, token string) {
	s.feedbackLock.Lock()
	s.feedback = append(s.feedback, tokenFeedback{serviceID, token, "", tokenFeedbackReasonThrottled})
	s.feedbackLock.Unlock()
	slog.Info("Throttled token", "service", serviceID, "token", token)
}

// ReplaceToken ...
func (s *Server) ReplaceToken(serviceID, token, replacement string) {
	s.feedbackLock.Lock()
	s.feedback = append(s.feedback, tokenFeedback{serviceID, token, replacement, tokenFeedbackReasonReplaced})
	s.feedbackLock.Unlock()
	slog.Info("Token replaced", "service", serviceID)
}
