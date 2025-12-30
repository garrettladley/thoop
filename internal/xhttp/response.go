package xhttp

import (
	"net/http"

	go_json "github.com/goccy/go-json"
)

func WriteJSON(w http.ResponseWriter, status int, data any) {
	SetHeaderContentTypeApplicationJSON(w)
	w.WriteHeader(status)
	_ = go_json.NewEncoder(w).Encode(data)
}

func WriteNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func WriteOK(w http.ResponseWriter, data any) {
	WriteJSON(w, http.StatusOK, data)
}
