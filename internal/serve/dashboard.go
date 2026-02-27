package serve

import (
	"embed"
	"net/http"
)

//go:embed static/dashboard.html
var dashboardFS embed.FS

// ServeDashboard serves the embedded single-file dashboard HTML.
func ServeDashboard(w http.ResponseWriter, r *http.Request) {
	data, err := dashboardFS.ReadFile("static/dashboard.html")
	if err != nil {
		http.Error(w, "dashboard not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}
