package server

import "net/http"

func NewRouter() http.Handler {
	mux := http.NewServeMux()

	// API эндпоинты
	mux.HandleFunc("/start", StartRun)
	mux.HandleFunc("/stop", StopRun)
	mux.HandleFunc("/stream", Stream)
	mux.HandleFunc("/export", ExportCSV)

	// статика
	// fs := http.FileServer(http.Dir("static"))
	// mux.Handle("/", fs)     // index.html по умолчанию
	// mux.Handle("/help", fs) // help.html

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
	})
	mux.HandleFunc("/help", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/help.html")
	})

	return mux
}
