package recipe

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

func ValidateRecipeRequest(req *CreateRecipeRequest) error {
	if req.Title == "" {
		return errors.New("title is required")
	}
	if len(req.Title) > 255 {
		return errors.New("title is too long (max 255 characters)")
	}
	
	if req.Difficulty != nil {
		difficulty := *req.Difficulty
		if difficulty != "easy" && difficulty != "medium" && difficulty != "hard" {
			return errors.New("difficulty must be 'easy', 'medium', or 'hard'")
		}
	}
	
	if req.PrepTime != nil && *req.PrepTime < 0 {
		return errors.New("prep time cannot be negative")
	}
	
	if req.CookTime != nil && *req.CookTime < 0 {
		return errors.New("cook time cannot be negative")
	}
	
	if req.Servings != nil && *req.Servings <= 0 {
		return errors.New("servings must be positive")
	}
	
	if len(req.Ingredients) == 0 {
		return errors.New("at least one ingredient is required")
	}
	
	for i, ingredient := range req.Ingredients {
		if ingredient.IngredientName == "" {
			return fmt.Errorf("ingredient %d: name is required", i+1)
		}
	}
	
	if len(req.Instructions) == 0 {
		return errors.New("at least one instruction is required")
	}
	
	for i, instruction := range req.Instructions {
		if instruction.InstructionText == "" {
			return fmt.Errorf("instruction %d: text is required", i+1)
		}
	}
	
	return nil
}

func Create(ctx context.Context, db *sqlx.DB, userId int64, req CreateRecipeRequest) (*RecipeWithDetails, error) {
	if err := ValidateRecipeRequest(&req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	var recipeId int64
	err = tx.QueryRowContext(ctx, `
		INSERT INTO recipes (
			user_id, title, description, prep_time, cook_time,
			servings, difficulty, cuisine, category, image_url
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING recipe_id`,
		userId, req.Title, req.Description, req.PrepTime, req.CookTime,
		req.Servings, req.Difficulty, req.Cuisine, req.Category, req.ImageUrl,
	).Scan(&recipeId)
	if err != nil {
		return nil, fmt.Errorf("failed to create recipe: %w", err)
	}
	
	for _, ingredient := range req.Ingredients {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO recipe_ingredients (
				recipe_id, ingredient_name, quantity, unit, notes, display_order
			) VALUES ($1, $2, $3, $4, $5, $6)`,
			recipeId, ingredient.IngredientName, ingredient.Quantity,
			ingredient.Unit, ingredient.Notes, ingredient.DisplayOrder,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create ingredient: %w", err)
		}
	}
	
	for _, instruction := range req.Instructions {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO recipe_instructions (
				recipe_id, step_number, instruction_text
			) VALUES ($1, $2, $3)`,
			recipeId, instruction.StepNumber, instruction.InstructionText,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create instruction: %w", err)
		}
	}
	
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	return GetById(ctx, db, recipeId)
}

func GetById(ctx context.Context, db *sqlx.DB, recipeId int64) (*RecipeWithDetails, error) {
	var recipe Recipe
	err := db.GetContext(ctx, &recipe, `SELECT * FROM recipes WHERE recipe_id = $1`, recipeId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("recipe not found")
		}
		return nil, fmt.Errorf("failed to get recipe: %w", err)
	}
	
	var ingredients []RecipeIngredient
	err = db.SelectContext(ctx, &ingredients, `
		SELECT * FROM recipe_ingredients 
		WHERE recipe_id = $1 
		ORDER BY display_order, ingredient_id`,
		recipeId,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get ingredients: %w", err)
	}
	
	var instructions []RecipeInstruction
	err = db.SelectContext(ctx, &instructions, `
		SELECT * FROM recipe_instructions 
		WHERE recipe_id = $1 
		ORDER BY step_number`,
		recipeId,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get instructions: %w", err)
	}
	
	return &RecipeWithDetails{
		Recipe:       recipe,
		Ingredients:  ingredients,
		Instructions: instructions,
	}, nil
}

func GetByIdForUser(ctx context.Context, db *sqlx.DB, recipeId, userId int64) (*RecipeWithDetails, error) {
	recipe, err := GetById(ctx, db, recipeId)
	if err != nil {
		return nil, err
	}

	if recipe.Recipe.UserId != userId {
		return nil, errors.New("recipe not found")
	}

	return recipe, nil
}

func Update(ctx context.Context, db *sqlx.DB, recipeId, userId int64, req UpdateRecipeRequest) (*RecipeWithDetails, error) {
	var recipe Recipe
	err := db.GetContext(ctx, &recipe, `SELECT * FROM recipes WHERE recipe_id = $1`, recipeId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("recipe not found")
		}
		return nil, fmt.Errorf("failed to get recipe: %w", err)
	}
	
	if recipe.UserId != userId {
		return nil, errors.New("unauthorized to update this recipe")
	}
	
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	var updates []string
	var args []interface{}
	argCount := 1
	
	if req.Title != nil {
		updates = append(updates, fmt.Sprintf("title = $%d", argCount))
		args = append(args, *req.Title)
		argCount++
	}
	if req.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argCount))
		args = append(args, *req.Description)
		argCount++
	}
	if req.PrepTime != nil {
		updates = append(updates, fmt.Sprintf("prep_time = $%d", argCount))
		args = append(args, *req.PrepTime)
		argCount++
	}
	if req.CookTime != nil {
		updates = append(updates, fmt.Sprintf("cook_time = $%d", argCount))
		args = append(args, *req.CookTime)
		argCount++
	}
	if req.Servings != nil {
		updates = append(updates, fmt.Sprintf("servings = $%d", argCount))
		args = append(args, *req.Servings)
		argCount++
	}
	if req.Difficulty != nil {
		updates = append(updates, fmt.Sprintf("difficulty = $%d", argCount))
		args = append(args, *req.Difficulty)
		argCount++
	}
	if req.Cuisine != nil {
		updates = append(updates, fmt.Sprintf("cuisine = $%d", argCount))
		args = append(args, *req.Cuisine)
		argCount++
	}
	if req.Category != nil {
		updates = append(updates, fmt.Sprintf("category = $%d", argCount))
		args = append(args, *req.Category)
		argCount++
	}
	if req.ImageUrl != nil {
		updates = append(updates, fmt.Sprintf("image_url = $%d", argCount))
		args = append(args, *req.ImageUrl)
		argCount++
	}

	if len(updates) > 0 {
		updates = append(updates, "updated_at = CURRENT_TIMESTAMP")
		args = append(args, recipeId)
		query := fmt.Sprintf("UPDATE recipes SET %s WHERE recipe_id = $%d", 
			strings.Join(updates, ", "), argCount)
		_, err = tx.ExecContext(ctx, query, args...)
		if err != nil {
			return nil, fmt.Errorf("failed to update recipe: %w", err)
		}
	}
	
	if req.Ingredients != nil {
		_, err = tx.ExecContext(ctx, `DELETE FROM recipe_ingredients WHERE recipe_id = $1`, recipeId)
		if err != nil {
			return nil, fmt.Errorf("failed to delete ingredients: %w", err)
		}
		
		for _, ingredient := range req.Ingredients {
			_, err = tx.ExecContext(ctx, `
				INSERT INTO recipe_ingredients (
					recipe_id, ingredient_name, quantity, unit, notes, display_order
				) VALUES ($1, $2, $3, $4, $5, $6)`,
				recipeId, ingredient.IngredientName, ingredient.Quantity,
				ingredient.Unit, ingredient.Notes, ingredient.DisplayOrder,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to create ingredient: %w", err)
			}
		}
	}
	
	if req.Instructions != nil {
		_, err = tx.ExecContext(ctx, `DELETE FROM recipe_instructions WHERE recipe_id = $1`, recipeId)
		if err != nil {
			return nil, fmt.Errorf("failed to delete instructions: %w", err)
		}
		
		for _, instruction := range req.Instructions {
			_, err = tx.ExecContext(ctx, `
				INSERT INTO recipe_instructions (
					recipe_id, step_number, instruction_text
				) VALUES ($1, $2, $3)`,
				recipeId, instruction.StepNumber, instruction.InstructionText,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to create instruction: %w", err)
			}
		}
	}
	
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	return GetById(ctx, db, recipeId)
}

func Delete(ctx context.Context, db *sqlx.DB, recipeId, userId int64) error {
	var recipe Recipe
	err := db.GetContext(ctx, &recipe, `SELECT * FROM recipes WHERE recipe_id = $1`, recipeId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("recipe not found")
		}
		return fmt.Errorf("failed to get recipe: %w", err)
	}
	
	if recipe.UserId != userId {
		return errors.New("unauthorized to delete this recipe")
	}
	
	_, err = db.ExecContext(ctx, `DELETE FROM recipes WHERE recipe_id = $1`, recipeId)
	if err != nil {
		return fmt.Errorf("failed to delete recipe: %w", err)
	}
	
	return nil
}

func List(ctx context.Context, db *sqlx.DB, filter ListRecipesFilter) ([]Recipe, error) {
	query := `SELECT * FROM recipes WHERE 1=1`
	var args []interface{}
	argCount := 1
	
	if filter.UserId != nil {
		query += fmt.Sprintf(" AND user_id = $%d", argCount)
		args = append(args, *filter.UserId)
		argCount++
	}

	if filter.Category != nil {
		query += fmt.Sprintf(" AND category = $%d", argCount)
		args = append(args, *filter.Category)
		argCount++
	}
	
	if filter.Cuisine != nil {
		query += fmt.Sprintf(" AND cuisine = $%d", argCount)
		args = append(args, *filter.Cuisine)
		argCount++
	}
	
	if filter.Search != nil && *filter.Search != "" {
		query += fmt.Sprintf(" AND (title ILIKE $%d OR description ILIKE $%d)", argCount, argCount)
		searchTerm := "%" + *filter.Search + "%"
		args = append(args, searchTerm)
		argCount++
	}
	
	query += " ORDER BY created_at DESC"
	
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filter.Limit)
		argCount++
	}
	
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, filter.Offset)
		argCount++
	}
	
	var recipes []Recipe
	err := db.SelectContext(ctx, &recipes, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list recipes: %w", err)
	}
	
	return recipes, nil
}

func GetUserRecipeCount(ctx context.Context, db *sqlx.DB, userId int64) (int, error) {
	var count int
	err := db.GetContext(ctx, &count, `SELECT COUNT(*) FROM recipes WHERE user_id = $1`, userId)
	if err != nil {
		return 0, fmt.Errorf("failed to get recipe count: %w", err)
	}
	return count, nil
}

func GetCategories(ctx context.Context, db *sqlx.DB) ([]string, error) {
	var categories []string
	err := db.SelectContext(ctx, &categories, `
		SELECT DISTINCT category FROM recipes
		WHERE category IS NOT NULL
		ORDER BY category`)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}
	return categories, nil
}

func GetCuisines(ctx context.Context, db *sqlx.DB) ([]string, error) {
	var cuisines []string
	err := db.SelectContext(ctx, &cuisines, `
		SELECT DISTINCT cuisine FROM recipes
		WHERE cuisine IS NOT NULL
		ORDER BY cuisine`)
	if err != nil {
		return nil, fmt.Errorf("failed to get cuisines: %w", err)
	}
	return cuisines, nil
}