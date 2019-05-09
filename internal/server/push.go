package server

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

func (s *Server) handlePush(w http.ResponseWriter, r *http.Request) {
	service := strings.TrimPrefix(r.URL.Path, "/api/push/")
	wrk, ok := s.workers[service]
	if !ok {
		http.NotFound(w, r)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "Invalid request method.", 405)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = wrk.push(body)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.WriteHeader(http.StatusAccepted)
	fmt.Fprintf(w, "OK")
}
