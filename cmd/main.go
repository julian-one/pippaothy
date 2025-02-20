package main

import (
	"context"
	"net/http"
	"pippaothy/templates"
)

func main() {
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/", Home())
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}

func Home() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		comp := templates.Home()
		w.Header().Set("Content-Type", "text/html")

		if err := comp.Render(context.Background(), w); err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
		}
	}
}
