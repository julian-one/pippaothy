package route

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"pippaothy/internal/middleware"
	"pippaothy/internal/recipes"
	"pippaothy/internal/templates"

	"github.com/jmoiron/sqlx"
)

// GetRecipes returns a handler for listing recipes
func GetRecipes(db *sqlx.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user from context
		user := middleware.GetUserFromContext(r)
		if user == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		filter := recipes.ListRecipesFilter{
			UserId: &user.UserId,
			Limit:  50,
			Offset: 0,
		}

		recipeList, err := recipes.List(r.Context(), db, filter)
		if err != nil {
			logger.Error("failed to list recipes", "error", err)
			http.Error(w, "Failed to load recipes", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		templates.Layout(templates.RecipesList(recipeList), "Recipes", true).Render(r.Context(), w)
	}
}

// GetRecipe returns a handler for viewing a single recipe
func GetRecipe(db *sqlx.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user from context
		user := middleware.GetUserFromContext(r)
		if user == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		idStr := r.PathValue("id")
		recipeId, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid recipe ID", http.StatusBadRequest)
			return
		}

		recipe, err := recipes.GetByIdForUser(r.Context(), db, recipeId, user.UserId)
		if err != nil {
			logger.Error("failed to get recipe", "error", err, "recipeId", recipeId)
			http.Error(w, "Recipe not found", http.StatusNotFound)
			return
		}

		isOwner := recipe.Recipe.UserId == user.UserId

		w.Header().Set("Content-Type", "text/html")
		templates.Layout(templates.RecipeDetail(*recipe, isOwner), recipe.Recipe.Title, true).
			Render(r.Context(), w)
	}
}

// GetNewRecipe returns a handler for the new recipe form
func GetNewRecipe() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user from context
		user := middleware.GetUserFromContext(r)
		if user == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		templates.Layout(templates.RecipeForm(nil, false), "New Recipe", true).
			Render(r.Context(), w)
	}
}

// PostRecipe returns a handler for creating a new recipe
func PostRecipe(db *sqlx.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user from context
		user := middleware.GetUserFromContext(r)
		if user == nil {
			w.WriteHeader(http.StatusUnauthorized)
			templates.Error("Please login to create recipes").Render(r.Context(), w)
			return
		}

		if err := r.ParseForm(); err != nil {
			logger.Error("failed to parse recipe form", "error", err)
			w.WriteHeader(http.StatusBadRequest)
			templates.Error("Invalid form data. Please check your inputs and try again.").
				Render(r.Context(), w)
			return
		}

		req := parseRecipeForm(r)

		// Server-side validation
		if req.Title == "" {
			w.WriteHeader(http.StatusBadRequest)
			templates.Error("Recipe title is required").Render(r.Context(), w)
			return
		}

		if len(req.Ingredients) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			templates.Error("At least one ingredient is required").Render(r.Context(), w)
			return
		}

		if len(req.Instructions) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			templates.Error("At least one instruction step is required").Render(r.Context(), w)
			return
		}

		recipe, err := recipes.Create(r.Context(), db, user.UserId, req)
		if err != nil {
			logger.Error("failed to create recipe", "error", err)
			w.WriteHeader(http.StatusBadRequest)
			templates.Error(err.Error()).Render(r.Context(), w)
			return
		}

		w.Header().Set("HX-Redirect", fmt.Sprintf("/recipes/%d", recipe.Recipe.RecipeId))
		w.WriteHeader(http.StatusCreated)
	}
}

// GetEditRecipe returns a handler for the edit recipe form
func GetEditRecipe(db *sqlx.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user from context
		user := middleware.GetUserFromContext(r)
		if user == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		idStr := r.PathValue("id")
		recipeId, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid recipe ID", http.StatusBadRequest)
			return
		}

		recipe, err := recipes.GetById(r.Context(), db, recipeId)
		if err != nil {
			logger.Error("failed to get recipe", "error", err, "recipeId", recipeId)
			http.Error(w, "Recipe not found", http.StatusNotFound)
			return
		}

		if recipe.Recipe.UserId != user.UserId {
			http.Error(w, "Unauthorized", http.StatusForbidden)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		templates.Layout(templates.RecipeForm(recipe, true), "Edit Recipe", true).
			Render(r.Context(), w)
	}
}

// PutRecipe returns a handler for updating a recipe
func PutRecipe(db *sqlx.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user from context
		user := middleware.GetUserFromContext(r)
		if user == nil {
			w.WriteHeader(http.StatusUnauthorized)
			templates.Error("Please login to update recipes").Render(r.Context(), w)
			return
		}

		idStr := r.PathValue("id")
		recipeId, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			templates.Error("Invalid recipe ID").Render(r.Context(), w)
			return
		}

		if err := r.ParseForm(); err != nil {
			logger.Error("failed to parse recipe form", "error", err)
			w.WriteHeader(http.StatusBadRequest)
			templates.Error("Invalid form data. Please check your inputs and try again.").
				Render(r.Context(), w)
			return
		}

		req := parseUpdateRecipeForm(r)

		// Server-side validation
		if req.Title != nil && *req.Title == "" {
			w.WriteHeader(http.StatusBadRequest)
			templates.Error("Recipe title is required").Render(r.Context(), w)
			return
		}

		if len(req.Ingredients) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			templates.Error("At least one ingredient is required").Render(r.Context(), w)
			return
		}

		if len(req.Instructions) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			templates.Error("At least one instruction step is required").Render(r.Context(), w)
			return
		}

		recipe, err := recipes.Update(r.Context(), db, recipeId, user.UserId, req)
		if err != nil {
			logger.Error("failed to update recipe", "error", err)
			w.WriteHeader(http.StatusBadRequest)
			templates.Error(err.Error()).Render(r.Context(), w)
			return
		}

		w.Header().Set("HX-Redirect", fmt.Sprintf("/recipes/%d", recipe.Recipe.RecipeId))
		w.WriteHeader(http.StatusOK)
	}
}

// DeleteRecipe returns a handler for deleting a recipe
func DeleteRecipe(db *sqlx.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user from context
		user := middleware.GetUserFromContext(r)
		if user == nil {
			w.WriteHeader(http.StatusUnauthorized)
			templates.Error("Please login to delete recipes").Render(r.Context(), w)
			return
		}

		idStr := r.PathValue("id")
		recipeId, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			templates.Error("Invalid recipe ID").Render(r.Context(), w)
			return
		}

		if err := recipes.Delete(r.Context(), db, recipeId, user.UserId); err != nil {
			logger.Error("failed to delete recipe", "error", err)
			w.WriteHeader(http.StatusBadRequest)
			templates.Error(err.Error()).Render(r.Context(), w)
			return
		}

		w.Header().Set("HX-Redirect", "/recipes")
		w.WriteHeader(http.StatusOK)
	}
}

// parseRecipeForm parses the recipe form data
func parseRecipeForm(r *http.Request) recipes.CreateRecipeRequest {
	req := recipes.CreateRecipeRequest{
		Title: r.FormValue("title"),
	}

	if desc := r.FormValue("description"); desc != "" {
		req.Description = &desc
	}

	if prepTime := r.FormValue("prep_time"); prepTime != "" {
		if val, err := strconv.Atoi(prepTime); err == nil {
			req.PrepTime = &val
		}
	}

	if cookTime := r.FormValue("cook_time"); cookTime != "" {
		if val, err := strconv.Atoi(cookTime); err == nil {
			req.CookTime = &val
		}
	}

	if servings := r.FormValue("servings"); servings != "" {
		if val, err := strconv.Atoi(servings); err == nil {
			req.Servings = &val
		}
	}

	if difficulty := r.FormValue("difficulty"); difficulty != "" {
		req.Difficulty = &difficulty
	}

	if cuisine := r.FormValue("cuisine"); cuisine != "" {
		req.Cuisine = &cuisine
	}

	if category := r.FormValue("category"); category != "" {
		req.Category = &category
	}

	if imageUrl := r.FormValue("image_url"); imageUrl != "" {
		req.ImageUrl = &imageUrl
	}

	// Parse ingredients dynamically from form values
	req.Ingredients = recipes.ParseIngredients(r)

	// Parse instructions dynamically from form values
	req.Instructions = recipes.ParseInstructions(r)

	return req
}

// parseUpdateRecipeForm parses the update recipe form data
func parseUpdateRecipeForm(r *http.Request) recipes.UpdateRecipeRequest {
	req := recipes.UpdateRecipeRequest{}

	title := strings.TrimSpace(r.FormValue("title"))
	req.Title = &title

	if desc := r.FormValue("description"); desc != "" {
		req.Description = &desc
	}

	if prepTime := r.FormValue("prep_time"); prepTime != "" {
		if val, err := strconv.Atoi(prepTime); err == nil {
			req.PrepTime = &val
		}
	}

	if cookTime := r.FormValue("cook_time"); cookTime != "" {
		if val, err := strconv.Atoi(cookTime); err == nil {
			req.CookTime = &val
		}
	}

	if servings := r.FormValue("servings"); servings != "" {
		if val, err := strconv.Atoi(servings); err == nil {
			req.Servings = &val
		}
	}

	if difficulty := r.FormValue("difficulty"); difficulty != "" {
		req.Difficulty = &difficulty
	}

	if cuisine := r.FormValue("cuisine"); cuisine != "" {
		req.Cuisine = &cuisine
	}

	if category := r.FormValue("category"); category != "" {
		req.Category = &category
	}

	if imageUrl := r.FormValue("image_url"); imageUrl != "" {
		req.ImageUrl = &imageUrl
	}

	// Parse ingredients dynamically from form values
	ingredients := recipes.ParseIngredients(r)
	if len(ingredients) > 0 {
		req.Ingredients = ingredients
	}

	// Parse instructions dynamically from form values
	instructions := recipes.ParseInstructions(r)
	if len(instructions) > 0 {
		req.Instructions = instructions
	}

	return req
}

