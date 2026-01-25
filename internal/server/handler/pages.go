package handler

import (
	"net/http"

	"github.com/garrettladley/thoop/internal/server/templates"
	"github.com/garrettladley/thoop/internal/xtempl"
)

// HandlePrivacy serves the privacy policy page.
func HandlePrivacy(w http.ResponseWriter, r *http.Request) {
	_ = xtempl.Render(w, r, templates.Privacy())
}
