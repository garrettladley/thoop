package xhttp

import (
	"net/http"
)

func Error(w http.ResponseWriter, status int) {
	http.Error(w, http.StatusText(status), status)
}
