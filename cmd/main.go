package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"pippaothy/internal/database"
	"pippaothy/internal/templates"
	"time"
)

func main() {
	db, err := database.Create()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	accessLog, err := os.OpenFile("access.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer accessLog.Close()
	accessLogger := log.New(accessLog, "", 0)

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/", Middleware(Home(), accessLogger))
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}

func Middleware(handler http.HandlerFunc, logger *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		now := time.Now().Format("2006-01-02 15:04:05")
		entry := fmt.Sprintf(
			"[%s] [INFO] | IP: %s | Method: %-6s | Path: %-20s | User-Agent: %s",
			now,
			r.RemoteAddr,
			r.Method,
			r.URL.Path,
			r.UserAgent(),
		)
		logger.Println(entry)
		handler(w, r)
	}
}

func Home() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		comp := templates.Layout(templates.Home(), "home")
		w.Header().Set("Content-Type", "text/html")

		if err := comp.Render(context.Background(), w); err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
		}
	}
}
