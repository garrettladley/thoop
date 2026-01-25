package xtempl

import (
	"fmt"
	"net/http"

	"github.com/a-h/templ"
	"github.com/garrettladley/thoop/internal/xhttp"
)

// Render writes a templ component to the response with proper HTML headers.
func Render(w http.ResponseWriter, r *http.Request, component templ.Component) error {
	xhttp.SetHeaderContentTypeTextHTMLCharsetUTF8(w)
	if err := component.Render(r.Context(), w); err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}
	return nil
}
