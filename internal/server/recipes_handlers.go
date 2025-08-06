package server

import (
	"net/http"
	"strconv"
	"strings"

	"pippaothy/internal/recipes"
	"pippaothy/internal/templates"
)

func (s *Server) handleRecipesList(w http.ResponseWriter, r *http.Request) {
	user := s.getCtxUser(r)

	userRecipes, err := recipes.GetByUser(s.db, user.UserId)
	if err != nil {
		s.logger.Error("Failed to get user recipes", "error", err)
		http.Error(w, "Failed to load recipes", http.StatusInternalServerError)
		return
	}

	csrfToken := s.getCSRFToken(r)

	component := templates.Layout(
		templates.RecipesList(userRecipes, true),
		"My Recipes",
		true,
		csrfToken,
	)

	if err := component.Render(r.Context(), w); err != nil {
		s.logger.Error("Failed to render recipes list", "error", err)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}

func (s *Server) handlePublicRecipes(w http.ResponseWriter, r *http.Request) {
	user := s.getCtxUser(r)
	loggedIn := user != nil

	limit := 50
	offset := 0
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	publicRecipes, err := recipes.GetPublic(s.db, limit, offset)
	if err != nil {
		s.logger.Error("Failed to get public recipes", "error", err)
		http.Error(w, "Failed to load recipes", http.StatusInternalServerError)
		return
	}

	csrfToken := s.getCSRFToken(r)

	component := templates.Layout(
		templates.RecipesList(publicRecipes, false),
		"Discover Recipes",
		loggedIn,
		csrfToken,
	)

	if err := component.Render(r.Context(), w); err != nil {
		s.logger.Error("Failed to render public recipes", "error", err)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}

func (s *Server) handleRecipeDetail(w http.ResponseWriter, r *http.Request) {
	user := s.getCtxUser(r)
	
	recipeIDStr := r.PathValue("id")
	recipeID, err := strconv.ParseInt(recipeIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid recipe ID", http.StatusBadRequest)
		return
	}

	recipe, err := recipes.GetByID(s.db, recipeID)
	if err != nil {
		if err.Error() == "recipe not found" {
			http.Error(w, "Recipe not found", http.StatusNotFound)
		} else {
			s.logger.Error("Failed to get recipe", "error", err)
			http.Error(w, "Failed to load recipe", http.StatusInternalServerError)
		}
		return
	}

	isOwner := false
	loggedIn := false
	if user != nil {
		loggedIn = true
		isOwner = recipe.UserID.Valid && recipe.UserID.Int64 == user.UserId
	}

	if !recipe.IsPublic && !isOwner {
		http.Error(w, "Recipe not found", http.StatusNotFound)
		return
	}

	csrfToken := s.getCSRFToken(r)

	component := templates.Layout(
		templates.RecipeDetail(*recipe, isOwner, csrfToken),
		recipe.Title,
		loggedIn,
		csrfToken,
	)

	if err := component.Render(r.Context(), w); err != nil {
		s.logger.Error("Failed to render recipe detail", "error", err)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}

func (s *Server) handleRecipeNew(w http.ResponseWriter, r *http.Request) {
	csrfToken := s.getCSRFToken(r)

	component := templates.Layout(
		templates.RecipeForm(nil, false, csrfToken),
		"New Recipe",
		true,
		csrfToken,
	)

	if err := component.Render(r.Context(), w); err != nil {
		s.logger.Error("Failed to render new recipe form", "error", err)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}

func (s *Server) handleRecipeCreate(w http.ResponseWriter, r *http.Request) {
	user := s.getCtxUser(r)

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	prepTime, _ := strconv.Atoi(r.FormValue("prep_time"))
	cookTime, _ := strconv.Atoi(r.FormValue("cook_time"))
	servings, _ := strconv.Atoi(r.FormValue("servings"))

	tags := []string{}
	if tagsStr := r.FormValue("tags"); tagsStr != "" {
		for _, tag := range strings.Split(tagsStr, ",") {
			if trimmed := strings.TrimSpace(tag); trimmed != "" {
				tags = append(tags, trimmed)
			}
		}
	}

	req := recipes.CreateRequest{
		UserID:       &user.UserId,
		Title:        r.FormValue("title"),
		Description:  r.FormValue("description"),
		Ingredients:  strings.Split(strings.TrimSpace(r.FormValue("ingredients")), "\n"),
		Instructions: strings.Split(strings.TrimSpace(r.FormValue("instructions")), "\n"),
		PrepTime:     prepTime,
		CookTime:     cookTime,
		Servings:     servings,
		Difficulty:   r.FormValue("difficulty"),
		Cuisine:      r.FormValue("cuisine"),
		Tags:         tags,
		ImageURL:     r.FormValue("image_url"),
		IsPublic:     r.FormValue("is_public") == "on",
	}

	recipeID, err := recipes.Create(s.db, req)
	if err != nil {
		s.logger.Error("Failed to create recipe", "error", err)
		http.Error(w, "Failed to create recipe", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Redirect", "/recipes/"+strconv.FormatInt(recipeID, 10))
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) handleRecipeEdit(w http.ResponseWriter, r *http.Request) {
	user := s.getCtxUser(r)

	recipeIDStr := r.PathValue("id")
	recipeID, err := strconv.ParseInt(recipeIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid recipe ID", http.StatusBadRequest)
		return
	}

	recipe, err := recipes.GetByID(s.db, recipeID)
	if err != nil {
		if err.Error() == "recipe not found" {
			http.Error(w, "Recipe not found", http.StatusNotFound)
		} else {
			s.logger.Error("Failed to get recipe", "error", err)
			http.Error(w, "Failed to load recipe", http.StatusInternalServerError)
		}
		return
	}

	if !recipe.UserID.Valid || recipe.UserID.Int64 != user.UserId {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	csrfToken := s.getCSRFToken(r)

	component := templates.Layout(
		templates.RecipeForm(recipe, true, csrfToken),
		"Edit Recipe",
		true,
		csrfToken,
	)

	if err := component.Render(r.Context(), w); err != nil {
		s.logger.Error("Failed to render edit recipe form", "error", err)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}

func (s *Server) handleRecipeUpdate(w http.ResponseWriter, r *http.Request) {
	user := s.getCtxUser(r)

	recipeIDStr := r.PathValue("id")
	recipeID, err := strconv.ParseInt(recipeIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid recipe ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	prepTime, _ := strconv.Atoi(r.FormValue("prep_time"))
	cookTime, _ := strconv.Atoi(r.FormValue("cook_time"))
	servings, _ := strconv.Atoi(r.FormValue("servings"))

	tags := []string{}
	if tagsStr := r.FormValue("tags"); tagsStr != "" {
		for _, tag := range strings.Split(tagsStr, ",") {
			if trimmed := strings.TrimSpace(tag); trimmed != "" {
				tags = append(tags, trimmed)
			}
		}
	}

	req := recipes.UpdateRequest{
		Title:        r.FormValue("title"),
		Description:  r.FormValue("description"),
		Ingredients:  strings.Split(strings.TrimSpace(r.FormValue("ingredients")), "\n"),
		Instructions: strings.Split(strings.TrimSpace(r.FormValue("instructions")), "\n"),
		PrepTime:     prepTime,
		CookTime:     cookTime,
		Servings:     servings,
		Difficulty:   r.FormValue("difficulty"),
		Cuisine:      r.FormValue("cuisine"),
		Tags:         tags,
		ImageURL:     r.FormValue("image_url"),
		IsPublic:     r.FormValue("is_public") == "on",
	}

	err = recipes.Update(s.db, recipeID, user.UserId, req)
	if err != nil {
		if err.Error() == "recipe not found or unauthorized" {
			http.Error(w, "Recipe not found or unauthorized", http.StatusNotFound)
		} else {
			s.logger.Error("Failed to update recipe", "error", err)
			http.Error(w, "Failed to update recipe", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("HX-Redirect", "/recipes/"+strconv.FormatInt(recipeID, 10))
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleRecipeDelete(w http.ResponseWriter, r *http.Request) {
	user := s.getCtxUser(r)

	recipeIDStr := r.PathValue("id")
	recipeID, err := strconv.ParseInt(recipeIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid recipe ID", http.StatusBadRequest)
		return
	}

	err = recipes.Delete(s.db, recipeID, user.UserId)
	if err != nil {
		if err.Error() == "recipe not found or unauthorized" {
			http.Error(w, "Recipe not found or unauthorized", http.StatusNotFound)
		} else {
			s.logger.Error("Failed to delete recipe", "error", err)
			http.Error(w, "Failed to delete recipe", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("HX-Redirect", "/recipes")
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleRecipeSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Search query required", http.StatusBadRequest)
		return
	}

	user := s.getCtxUser(r)
	onlyPublic := user == nil

	results, err := recipes.SearchByTitle(s.db, query, onlyPublic)
	if err != nil {
		s.logger.Error("Failed to search recipes", "error", err)
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}

	csrfToken := s.getCSRFToken(r)
	loggedIn := user != nil

	component := templates.Layout(
		templates.RecipesList(results, false),
		"Search Results",
		loggedIn,
		csrfToken,
	)

	if err := component.Render(r.Context(), w); err != nil {
		s.logger.Error("Failed to render search results", "error", err)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}