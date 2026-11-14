package linkpolicy

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
)

// RuntimeGuard closes the default Wails HTTP Browser.OpenURL back door. None
// of our UI uses it: validated Go services perform the actual OS operation.
// Match the fixed alpha.74 transport's body-first/query-fallback semantics.
func RuntimeGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/wails/runtime" {
			if r.Body == nil {
				r.Body = http.NoBody
			}
			var object *int
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "invalid runtime request", http.StatusBadRequest)
				return
			}
			r.Body.Close()
			r.Body = io.NopCloser(bytes.NewReader(body))
			if len(body) > 0 {
				var request struct {
					Object *int `json:"object"`
				}
				if json.Unmarshal(body, &request) != nil {
					http.Error(w, "invalid runtime request", http.StatusBadRequest)
					return
				}
				object = request.Object
			} else if n, err := strconv.Atoi(r.URL.Query().Get("object")); err == nil {
				object = &n
			}
			if object != nil && *object == 9 {
				http.Error(w, "use validated link services", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
