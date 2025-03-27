package route

import (
	"encoding/json"
	"fmt"
	"net/http"
	"pippaothy/internal/recipes"
	"pippaothy/internal/templates"

	"github.com/jmoiron/sqlx"
)

func CreateRecipe(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var req recipes.CreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Header().Set("Content-Type", "text/html")
			templates.RequestError().Render(ctx, w)
			return
		}

		_, err := recipes.Create(db, req)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("Content-Type", "text/html")
			templates.ServerError().Render(ctx, w)
			return
		}

		w.Header().Set("HX-Redirect", "/recipes/list")
		w.WriteHeader(http.StatusCreated)
	}
}

func Ingredient() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		templates.Ingredient().Render(r.Context(), w)
	}
}

func Step() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		templates.Step().Render(r.Context(), w)
	}
}

func Recipe() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		templates.Layout(templates.Recipe(), "recipe", true).Render(r.Context(), w)
	}
}

func ListRecipes(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		records, err := recipes.List(db)
		if err != nil {
			fmt.Println("error listing:", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("Content-Type", "text/html")
			templates.ServerError().Render(ctx, w)
			return
		}
		fmt.Println("records:", records)

		w.Header().Set("Content-Type", "text/html")
		templates.Layout(templates.RecipeList(records), "recipes", true).Render(r.Context(), w)
	}
}
