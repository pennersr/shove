package server

import (
	"encoding/json"
	"fmt"
	"gitlab.com/pennersr/shove/internal/types"
	"net/http"
)

type pushRequest struct {
	Service string          `json:"service"`
	Tokens  []string        `json:"tokens"`
	Topic   string          `json:"topic"`
	Payload json.RawMessage `json:"payload"`
}

func (s *Server) handlePush(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Invalid request method.", 405)
		return
	}
	var pr pushRequest
	if err := json.NewDecoder(r.Body).Decode(&pr); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if len(pr.Tokens) == 0 {
		http.Error(w, "No tokens specified.", 400)
		return
	}
	wrk, ok := s.workers[pr.Service]
	if !ok {
		http.Error(w, "Unknown service.", 400)
		return
	}
	msg := types.PushMessage{
		Tokens:  pr.Tokens,
		Topic:   pr.Topic,
		Payload: pr.Payload,
	}
	err := wrk.push(msg)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.WriteHeader(http.StatusAccepted)
	fmt.Fprintf(w, "OK")
}
