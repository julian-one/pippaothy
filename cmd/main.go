package main

import (
	"net/http"
	"pippaothy/internal/database"
	"pippaothy/internal/route"
)

func main() {
	db, err := database.Create()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	mux := http.NewServeMux()
	router := route.NewRouter(db)
	router.RegisterRoutes(mux)

	if err := http.ListenAndServe(":8080", mux); err != nil {
		panic(err)
	}
}
