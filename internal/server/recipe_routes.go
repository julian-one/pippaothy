package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"pippaothy/internal/recipes"
	"pippaothy/internal/templates"
)

func (s *Server) getRecipes(w http.ResponseWriter, r *http.Request) {
	user := s.getCtxUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	filter := recipes.ListRecipesFilter{
		UserId: &user.UserId,
		Limit:  50,
		Offset: 0,
	}

	if showPublic := r.URL.Query().Get("public"); showPublic == "true" {
		filter.UserId = nil
		isPublic := true
		filter.IsPublic = &isPublic
	}

	recipeList, err := recipes.List(r.Context(), s.db, filter)
	if err != nil {
		s.logger.Error("failed to list recipes", "error", err)
		http.Error(w, "Failed to load recipes", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	templates.Layout(templates.RecipesList(recipeList), "Recipes", true).Render(r.Context(), w)
}

func (s *Server) getRecipe(w http.ResponseWriter, r *http.Request) {
	user := s.getCtxUser(r)
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

	recipe, err := recipes.GetByIdForUser(r.Context(), s.db, recipeId, user.UserId)
	if err != nil {
		s.logger.Error("failed to get recipe", "error", err, "recipeId", recipeId)
		http.Error(w, "Recipe not found", http.StatusNotFound)
		return
	}

	isOwner := recipe.Recipe.UserId == user.UserId

	w.Header().Set("Content-Type", "text/html")
	templates.Layout(templates.RecipeDetail(*recipe, isOwner), recipe.Recipe.Title, true).Render(r.Context(), w)
}

func (s *Server) getNewRecipe(w http.ResponseWriter, r *http.Request) {
	user := s.getCtxUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	templates.Layout(templates.RecipeForm(nil, false), "New Recipe", true).Render(r.Context(), w)
}

func (s *Server) postRecipe(w http.ResponseWriter, r *http.Request) {
	user := s.getCtxUser(r)
	if user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`<div class="alert alert-error">
			<svg xmlns="http://www.w3.org/2000/svg" class="stroke-current flex-shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
			</svg>
			<span>Please login to create recipes</span>
		</div>`))
		return
	}

	if err := r.ParseForm(); err != nil {
		s.logger.Error("failed to parse recipe form", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`<div class="alert alert-error">
			<svg xmlns="http://www.w3.org/2000/svg" class="stroke-current flex-shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
			</svg>
			<span>Invalid form data. Please check your inputs and try again.</span>
		</div>`))
		return
	}

	req := parseRecipeForm(r)
	
	// Server-side validation
	if req.Title == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`<div class="alert alert-error">
			<svg xmlns="http://www.w3.org/2000/svg" class="stroke-current flex-shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
			</svg>
			<span>Recipe title is required</span>
		</div>`))
		return
	}
	
	if len(req.Ingredients) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`<div class="alert alert-error">
			<svg xmlns="http://www.w3.org/2000/svg" class="stroke-current flex-shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
			</svg>
			<span>At least one ingredient is required</span>
		</div>`))
		return
	}
	
	if len(req.Instructions) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`<div class="alert alert-error">
			<svg xmlns="http://www.w3.org/2000/svg" class="stroke-current flex-shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
			</svg>
			<span>At least one instruction step is required</span>
		</div>`))
		return
	}

	recipe, err := recipes.Create(r.Context(), s.db, user.UserId, req)
	if err != nil {
		s.logger.Error("failed to create recipe", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `<div class="alert alert-error">
			<svg xmlns="http://www.w3.org/2000/svg" class="stroke-current flex-shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
			</svg>
			<span>%s</span>
		</div>`, err.Error())
		return
	}

	w.Header().Set("HX-Redirect", fmt.Sprintf("/recipes/%d", recipe.Recipe.RecipeId))
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) getEditRecipe(w http.ResponseWriter, r *http.Request) {
	user := s.getCtxUser(r)
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

	recipe, err := recipes.GetById(r.Context(), s.db, recipeId)
	if err != nil {
		s.logger.Error("failed to get recipe", "error", err, "recipeId", recipeId)
		http.Error(w, "Recipe not found", http.StatusNotFound)
		return
	}

	if recipe.Recipe.UserId != user.UserId {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	templates.Layout(templates.RecipeForm(recipe, true), "Edit Recipe", true).Render(r.Context(), w)
}

func (s *Server) putRecipe(w http.ResponseWriter, r *http.Request) {
	user := s.getCtxUser(r)
	if user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`<div class="alert alert-error">
			<svg xmlns="http://www.w3.org/2000/svg" class="stroke-current flex-shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
			</svg>
			<span>Please login to update recipes</span>
		</div>`))
		return
	}

	idStr := r.PathValue("id")
	recipeId, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`<div class="alert alert-error">
			<svg xmlns="http://www.w3.org/2000/svg" class="stroke-current flex-shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
			</svg>
			<span>Invalid recipe ID</span>
		</div>`))
		return
	}

	if err := r.ParseForm(); err != nil {
		s.logger.Error("failed to parse recipe form", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`<div class="alert alert-error">
			<svg xmlns="http://www.w3.org/2000/svg" class="stroke-current flex-shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
			</svg>
			<span>Invalid form data. Please check your inputs and try again.</span>
		</div>`))
		return
	}

	req := parseUpdateRecipeForm(r)
	
	// Server-side validation
	if req.Title != nil && *req.Title == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`<div class="alert alert-error">
			<svg xmlns="http://www.w3.org/2000/svg" class="stroke-current flex-shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
			</svg>
			<span>Recipe title is required</span>
		</div>`))
		return
	}
	
	if len(req.Ingredients) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`<div class="alert alert-error">
			<svg xmlns="http://www.w3.org/2000/svg" class="stroke-current flex-shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
			</svg>
			<span>At least one ingredient is required</span>
		</div>`))
		return
	}
	
	if len(req.Instructions) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`<div class="alert alert-error">
			<svg xmlns="http://www.w3.org/2000/svg" class="stroke-current flex-shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
			</svg>
			<span>At least one instruction step is required</span>
		</div>`))
		return
	}

	recipe, err := recipes.Update(r.Context(), s.db, recipeId, user.UserId, req)
	if err != nil {
		s.logger.Error("failed to update recipe", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `<div class="alert alert-error">
			<svg xmlns="http://www.w3.org/2000/svg" class="stroke-current flex-shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
			</svg>
			<span>%s</span>
		</div>`, err.Error())
		return
	}

	w.Header().Set("HX-Redirect", fmt.Sprintf("/recipes/%d", recipe.Recipe.RecipeId))
	w.WriteHeader(http.StatusOK)
}

func (s *Server) deleteRecipe(w http.ResponseWriter, r *http.Request) {
	user := s.getCtxUser(r)
	if user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`<div class="error">Please login to delete recipes</div>`))
		return
	}

	idStr := r.PathValue("id")
	recipeId, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`<div class="error">Invalid recipe ID</div>`))
		return
	}

	if err := recipes.Delete(r.Context(), s.db, recipeId, user.UserId); err != nil {
		s.logger.Error("failed to delete recipe", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `<div class="error">%s</div>`, err.Error())
		return
	}

	w.Header().Set("HX-Redirect", "/recipes")
	w.WriteHeader(http.StatusOK)
}

func (s *Server) getRecipesAPI(w http.ResponseWriter, r *http.Request) {
	user := s.getCtxUser(r)
	if user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
		return
	}

	filter := recipes.ListRecipesFilter{
		Limit:  50,
		Offset: 0,
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
			filter.Limit = limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	if publicStr := r.URL.Query().Get("public"); publicStr != "" {
		isPublic := publicStr == "true"
		filter.IsPublic = &isPublic
		if !isPublic {
			filter.UserId = &user.UserId
		}
	} else {
		filter.UserId = &user.UserId
	}

	if search := r.URL.Query().Get("search"); search != "" {
		filter.Search = &search
	}

	if category := r.URL.Query().Get("category"); category != "" {
		filter.Category = &category
	}

	if cuisine := r.URL.Query().Get("cuisine"); cuisine != "" {
		filter.Cuisine = &cuisine
	}

	recipeList, err := recipes.List(r.Context(), s.db, filter)
	if err != nil {
		s.logger.Error("failed to list recipes", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to load recipes"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recipeList)
}

func parseRecipeForm(r *http.Request) recipes.CreateRecipeRequest {
	req := recipes.CreateRecipeRequest{
		Title:    r.FormValue("title"),
		IsPublic: r.FormValue("is_public") == "on",
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

	// Parse ingredients - check up to 100 to handle gaps from deleted items
	displayOrder := 0
	for i := 0; i < 100; i++ {
		name := strings.TrimSpace(r.FormValue(fmt.Sprintf("ingredients[%d][name]", i)))
		if name == "" {
			continue // Skip empty entries
		}

		ingredient := recipes.CreateIngredientRequest{
			IngredientName: name,
			DisplayOrder:   displayOrder,
		}
		displayOrder++

		if quantity := strings.TrimSpace(r.FormValue(fmt.Sprintf("ingredients[%d][quantity]", i))); quantity != "" {
			ingredient.Quantity = &quantity
		}

		if unit := strings.TrimSpace(r.FormValue(fmt.Sprintf("ingredients[%d][unit]", i))); unit != "" {
			ingredient.Unit = &unit
		}

		req.Ingredients = append(req.Ingredients, ingredient)
	}

	// Parse instructions - check up to 100 to handle gaps from deleted items
	stepNumber := 1
	for i := 0; i < 100; i++ {
		text := strings.TrimSpace(r.FormValue(fmt.Sprintf("instructions[%d]", i)))
		if text == "" {
			continue // Skip empty entries
		}

		req.Instructions = append(req.Instructions, recipes.CreateInstructionRequest{
			StepNumber:      stepNumber,
			InstructionText: text,
		})
		stepNumber++
	}

	return req
}

func parseUpdateRecipeForm(r *http.Request) recipes.UpdateRecipeRequest {
	req := recipes.UpdateRecipeRequest{}

	title := strings.TrimSpace(r.FormValue("title"))
	req.Title = &title

	isPublic := r.FormValue("is_public") == "on"
	req.IsPublic = &isPublic

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

	// Parse ingredients - check up to 100 to handle gaps from deleted items
	ingredients := []recipes.CreateIngredientRequest{}
	displayOrder := 0
	for i := 0; i < 100; i++ {
		name := strings.TrimSpace(r.FormValue(fmt.Sprintf("ingredients[%d][name]", i)))
		if name == "" {
			continue // Skip empty entries
		}

		ingredient := recipes.CreateIngredientRequest{
			IngredientName: name,
			DisplayOrder:   displayOrder,
		}
		displayOrder++

		if quantity := strings.TrimSpace(r.FormValue(fmt.Sprintf("ingredients[%d][quantity]", i))); quantity != "" {
			ingredient.Quantity = &quantity
		}

		if unit := strings.TrimSpace(r.FormValue(fmt.Sprintf("ingredients[%d][unit]", i))); unit != "" {
			ingredient.Unit = &unit
		}

		ingredients = append(ingredients, ingredient)
	}

	if len(ingredients) > 0 {
		req.Ingredients = ingredients
	}

	// Parse instructions - check up to 100 to handle gaps from deleted items
	instructions := []recipes.CreateInstructionRequest{}
	stepNumber := 1
	for i := 0; i < 100; i++ {
		text := strings.TrimSpace(r.FormValue(fmt.Sprintf("instructions[%d]", i)))
		if text == "" {
			continue // Skip empty entries
		}

		instructions = append(instructions, recipes.CreateInstructionRequest{
			StepNumber:      stepNumber,
			InstructionText: text,
		})
		stepNumber++
	}

	if len(instructions) > 0 {
		req.Instructions = instructions
	}

	return req
}