package recipes

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type Recipe struct {
	RecipeID     int64          `db:"recipe_id" json:"recipe_id"`
	UserID       sql.NullInt64  `db:"user_id" json:"user_id"`
	SourceName   sql.NullString `db:"source_name" json:"source_name"`
	SourceURL    sql.NullString `db:"source_url" json:"source_url"`
	Title        string         `db:"title" json:"title"`
	Description  sql.NullString `db:"description" json:"description"`
	Ingredients  []Ingredient   `json:"ingredients"`
	Instructions []Instruction  `json:"instructions"`
	PrepTime     sql.NullInt64  `db:"prep_time" json:"prep_time"`
	CookTime     sql.NullInt64  `db:"cook_time" json:"cook_time"`
	Servings     sql.NullInt64  `db:"servings" json:"servings"`
	Difficulty   sql.NullString `db:"difficulty" json:"difficulty"`
	Cuisine      sql.NullString `db:"cuisine" json:"cuisine"`
	Tags         pq.StringArray `db:"tags" json:"tags"`
	ImageURL     sql.NullString `db:"image_url" json:"image_url"`
	IsPublic     bool           `db:"is_public" json:"is_public"`
	CreatedAt    time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time      `db:"updated_at" json:"updated_at"`
}

type Ingredient struct {
	IngredientID   int64  `db:"ingredient_id" json:"ingredient_id"`
	RecipeID       int64  `db:"recipe_id" json:"recipe_id"`
	IngredientText string `db:"ingredient_text" json:"ingredient_text"`
	OrderIndex     int    `db:"order_index" json:"order_index"`
	Amount         string `db:"amount" json:"amount"`
	Unit           string `db:"unit" json:"unit"`
	Item           string `db:"item" json:"item"`
	Notes          string `db:"notes" json:"notes"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
}

type Instruction struct {
	InstructionID   int64 `db:"instruction_id" json:"instruction_id"`
	RecipeID        int64 `db:"recipe_id" json:"recipe_id"`
	InstructionText string `db:"instruction_text" json:"instruction_text"`
	OrderIndex      int   `db:"order_index" json:"order_index"`
	EstimatedTime   sql.NullInt64 `db:"estimated_time" json:"estimated_time"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
}

type CreateRequest struct {
	UserID       *int64        `json:"user_id,omitempty"`
	SourceName   string        `json:"source_name,omitempty"`
	SourceURL    string        `json:"source_url,omitempty"`
	Title        string        `json:"title"`
	Description  string        `json:"description"`
	Ingredients  []string      `json:"ingredients"`
	Instructions []string      `json:"instructions"`
	PrepTime     int           `json:"prep_time"`
	CookTime     int           `json:"cook_time"`
	Servings     int           `json:"servings"`
	Difficulty   string        `json:"difficulty"`
	Cuisine      string        `json:"cuisine"`
	Tags         []string      `json:"tags"`
	ImageURL     string        `json:"image_url"`
	IsPublic     bool          `json:"is_public"`
}

type UpdateRequest struct {
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Ingredients  []string `json:"ingredients"`
	Instructions []string `json:"instructions"`
	PrepTime     int      `json:"prep_time"`
	CookTime     int      `json:"cook_time"`
	Servings     int      `json:"servings"`
	Difficulty   string   `json:"difficulty"`
	Cuisine      string   `json:"cuisine"`
	Tags         []string `json:"tags"`
	ImageURL     string   `json:"image_url"`
	IsPublic     bool     `json:"is_public"`
}

func ValidateRecipe(title string, ingredients, instructions []string) error {
	if title == "" {
		return errors.New("title is required")
	}
	if len(title) > 200 {
		return errors.New("title is too long (max 200 characters)")
	}
	if len(ingredients) == 0 {
		return errors.New("ingredients are required")
	}
	if len(instructions) == 0 {
		return errors.New("instructions are required")
	}
	return nil
}

func ValidateDifficulty(difficulty string) error {
	if difficulty != "" && difficulty != "easy" && difficulty != "medium" && difficulty != "hard" {
		return errors.New("difficulty must be easy, medium, or hard")
	}
	return nil
}

func (req *CreateRequest) Validate() error {
	if err := ValidateRecipe(req.Title, req.Ingredients, req.Instructions); err != nil {
		return err
	}
	if err := ValidateDifficulty(req.Difficulty); err != nil {
		return err
	}
	if req.PrepTime < 0 {
		return errors.New("prep time cannot be negative")
	}
	if req.CookTime < 0 {
		return errors.New("cook time cannot be negative")
	}
	if req.Servings < 0 {
		return errors.New("servings cannot be negative")
	}
	
	// Validate that either user_id OR source fields are provided, but not both
	hasUserID := req.UserID != nil
	hasSource := req.SourceName != "" && req.SourceURL != ""
	
	if !hasUserID && !hasSource {
		return errors.New("either user_id or source_name/source_url must be provided")
	}
	if hasUserID && hasSource {
		return errors.New("cannot provide both user_id and source_name/source_url")
	}
	
	return nil
}

func (req *UpdateRequest) Validate() error {
	if err := ValidateRecipe(req.Title, req.Ingredients, req.Instructions); err != nil {
		return err
	}
	if err := ValidateDifficulty(req.Difficulty); err != nil {
		return err
	}
	if req.PrepTime < 0 {
		return errors.New("prep time cannot be negative")
	}
	if req.CookTime < 0 {
		return errors.New("cook time cannot be negative")
	}
	if req.Servings < 0 {
		return errors.New("servings cannot be negative")
	}
	return nil
}

func (req *CreateRequest) Sanitize() {
	req.Title = strings.TrimSpace(req.Title)
	req.Description = strings.TrimSpace(req.Description)
	for i, ingredient := range req.Ingredients {
		req.Ingredients[i] = strings.TrimSpace(ingredient)
	}
	for i, instruction := range req.Instructions {
		req.Instructions[i] = strings.TrimSpace(instruction)
	}
	req.Cuisine = strings.TrimSpace(req.Cuisine)
	req.ImageURL = strings.TrimSpace(req.ImageURL)
}

func (req *UpdateRequest) Sanitize() {
	req.Title = strings.TrimSpace(req.Title)
	req.Description = strings.TrimSpace(req.Description)
	for i, ingredient := range req.Ingredients {
		req.Ingredients[i] = strings.TrimSpace(ingredient)
	}
	for i, instruction := range req.Instructions {
		req.Instructions[i] = strings.TrimSpace(instruction)
	}
	req.Cuisine = strings.TrimSpace(req.Cuisine)
	req.ImageURL = strings.TrimSpace(req.ImageURL)
}

// Create inserts a new recipe into the database
func Create(db *sqlx.DB, req CreateRequest) (int64, error) {
	req.Sanitize()
	if err := req.Validate(); err != nil {
		return 0, err
	}

	tx, err := db.Beginx()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var recipeID int64
	err = tx.QueryRow(`
		INSERT INTO recipes (
			user_id, source_name, source_url, title, description,
			prep_time, cook_time, servings, difficulty, cuisine,
			tags, image_url, is_public
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING recipe_id`,
		nullInt64Ptr(req.UserID), nullString(req.SourceName), nullString(req.SourceURL),
		req.Title, nullString(req.Description),
		nullInt(req.PrepTime), nullInt(req.CookTime), nullInt(req.Servings),
		nullString(req.Difficulty), nullString(req.Cuisine),
		pq.Array(req.Tags), nullString(req.ImageURL), req.IsPublic,
	).Scan(&recipeID)

	if err != nil {
		return 0, fmt.Errorf("failed to create recipe: %w", err)
	}

	// Insert ingredients
	for i, ingredient := range req.Ingredients {
		if ingredient == "" {
			continue
		}
		_, err = tx.Exec(`
			INSERT INTO recipe_ingredients (recipe_id, ingredient_text, order_index, item)
			VALUES ($1, $2, $3, $4)`,
			recipeID, ingredient, i+1, ingredient)
		if err != nil {
			return 0, fmt.Errorf("failed to create ingredient: %w", err)
		}
	}

	// Insert instructions
	for i, instruction := range req.Instructions {
		if instruction == "" {
			continue
		}
		_, err = tx.Exec(`
			INSERT INTO recipe_instructions (recipe_id, instruction_text, order_index)
			VALUES ($1, $2, $3)`,
			recipeID, instruction, i+1)
		if err != nil {
			return 0, fmt.Errorf("failed to create instruction: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return recipeID, nil
}

// CreateInTx inserts a new recipe into the database within a transaction
func CreateInTx(tx *sqlx.Tx, req CreateRequest) (int64, error) {
	req.Sanitize()
	if err := req.Validate(); err != nil {
		return 0, err
	}

	var recipeID int64
	err := tx.QueryRow(`
		INSERT INTO recipes (
			user_id, source_name, source_url, title, description,
			prep_time, cook_time, servings, difficulty, cuisine,
			tags, image_url, is_public
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING recipe_id`,
		nullInt64Ptr(req.UserID), nullString(req.SourceName), nullString(req.SourceURL),
		req.Title, nullString(req.Description),
		nullInt(req.PrepTime), nullInt(req.CookTime), nullInt(req.Servings),
		nullString(req.Difficulty), nullString(req.Cuisine),
		pq.Array(req.Tags), nullString(req.ImageURL), req.IsPublic,
	).Scan(&recipeID)

	if err != nil {
		return 0, fmt.Errorf("failed to create recipe: %w", err)
	}

	// Insert ingredients
	for i, ingredient := range req.Ingredients {
		if ingredient == "" {
			continue
		}
		_, err = tx.Exec(`
			INSERT INTO recipe_ingredients (recipe_id, ingredient_text, order_index, item)
			VALUES ($1, $2, $3, $4)`,
			recipeID, ingredient, i+1, ingredient)
		if err != nil {
			return 0, fmt.Errorf("failed to create ingredient: %w", err)
		}
	}

	// Insert instructions
	for i, instruction := range req.Instructions {
		if instruction == "" {
			continue
		}
		_, err = tx.Exec(`
			INSERT INTO recipe_instructions (recipe_id, instruction_text, order_index)
			VALUES ($1, $2, $3)`,
			recipeID, instruction, i+1)
		if err != nil {
			return 0, fmt.Errorf("failed to create instruction: %w", err)
		}
	}

	return recipeID, nil
}

// GetByID retrieves a recipe by its ID
func GetByID(db *sqlx.DB, recipeID int64) (*Recipe, error) {
	var recipe Recipe
	err := db.Get(&recipe, `SELECT * FROM recipes WHERE recipe_id = $1`, recipeID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("recipe not found")
		}
		return nil, fmt.Errorf("failed to get recipe: %w", err)
	}
	
	// Load ingredients
	var ingredients []Ingredient
	err = db.Select(&ingredients, `
		SELECT * FROM recipe_ingredients 
		WHERE recipe_id = $1 
		ORDER BY order_index`, recipeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get recipe ingredients: %w", err)
	}
	recipe.Ingredients = ingredients
	
	// Load instructions
	var instructions []Instruction
	err = db.Select(&instructions, `
		SELECT * FROM recipe_instructions 
		WHERE recipe_id = $1 
		ORDER BY order_index`, recipeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get recipe instructions: %w", err)
	}
	recipe.Instructions = instructions
	
	return &recipe, nil
}

// loadRecipeDetails loads ingredients and instructions for multiple recipes
func loadRecipeDetails(db *sqlx.DB, recipes []Recipe) error {
	if len(recipes) == 0 {
		return nil
	}
	
	// Collect all recipe IDs
	recipeIDs := make([]int64, len(recipes))
	recipeMap := make(map[int64]*Recipe)
	for i := range recipes {
		recipeIDs[i] = recipes[i].RecipeID
		recipeMap[recipes[i].RecipeID] = &recipes[i]
	}
	
	// Load all ingredients
	var ingredients []Ingredient
	query, args, err := sqlx.In(`
		SELECT * FROM recipe_ingredients 
		WHERE recipe_id IN (?) 
		ORDER BY recipe_id, order_index`, recipeIDs)
	if err != nil {
		return fmt.Errorf("failed to build ingredients query: %w", err)
	}
	query = db.Rebind(query)
	
	err = db.Select(&ingredients, query, args...)
	if err != nil {
		return fmt.Errorf("failed to load ingredients: %w", err)
	}
	
	// Group ingredients by recipe
	for _, ingredient := range ingredients {
		if recipe := recipeMap[ingredient.RecipeID]; recipe != nil {
			recipe.Ingredients = append(recipe.Ingredients, ingredient)
		}
	}
	
	// Load all instructions
	var instructions []Instruction
	query, args, err = sqlx.In(`
		SELECT * FROM recipe_instructions 
		WHERE recipe_id IN (?) 
		ORDER BY recipe_id, order_index`, recipeIDs)
	if err != nil {
		return fmt.Errorf("failed to build instructions query: %w", err)
	}
	query = db.Rebind(query)
	
	err = db.Select(&instructions, query, args...)
	if err != nil {
		return fmt.Errorf("failed to load instructions: %w", err)
	}
	
	// Group instructions by recipe
	for _, instruction := range instructions {
		if recipe := recipeMap[instruction.RecipeID]; recipe != nil {
			recipe.Instructions = append(recipe.Instructions, instruction)
		}
	}
	
	return nil
}

// GetByUser retrieves all recipes for a specific user
func GetByUser(db *sqlx.DB, userID int64) ([]Recipe, error) {
	var recipes []Recipe
	err := db.Select(&recipes, `
		SELECT * FROM recipes 
		WHERE user_id = $1 
		ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user recipes: %w", err)
	}
	
	if err = loadRecipeDetails(db, recipes); err != nil {
		return nil, err
	}
	
	return recipes, nil
}

// GetBySource retrieves all recipes from a specific source
func GetBySource(db *sqlx.DB, sourceName string) ([]Recipe, error) {
	var recipes []Recipe
	err := db.Select(&recipes, `
		SELECT * FROM recipes 
		WHERE source_name = $1 
		ORDER BY created_at DESC`,
		sourceName,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get source recipes: %w", err)
	}
	
	if err = loadRecipeDetails(db, recipes); err != nil {
		return nil, err
	}
	
	return recipes, nil
}

// GetPublic retrieves all public recipes
func GetPublic(db *sqlx.DB, limit, offset int) ([]Recipe, error) {
	var recipes []Recipe
	err := db.Select(&recipes, `
		SELECT * FROM recipes 
		WHERE is_public = true 
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get public recipes: %w", err)
	}
	
	if err = loadRecipeDetails(db, recipes); err != nil {
		return nil, err
	}
	
	return recipes, nil
}

// Update updates an existing recipe (only works for user recipes)
func Update(db *sqlx.DB, recipeID, userID int64, req UpdateRequest) error {
	req.Sanitize()
	if err := req.Validate(); err != nil {
		return err
	}

	tx, err := db.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.Exec(`
		UPDATE recipes SET
			title = $1, description = $2,
			prep_time = $3, cook_time = $4, servings = $5, difficulty = $6,
			cuisine = $7, tags = $8, image_url = $9, is_public = $10,
			updated_at = CURRENT_TIMESTAMP
		WHERE recipe_id = $11 AND user_id = $12`,
		req.Title, nullString(req.Description),
		nullInt(req.PrepTime), nullInt(req.CookTime), nullInt(req.Servings),
		nullString(req.Difficulty), nullString(req.Cuisine),
		pq.Array(req.Tags), nullString(req.ImageURL), req.IsPublic,
		recipeID, userID,
	)

	if err != nil {
		return fmt.Errorf("failed to update recipe: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check update result: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("recipe not found or unauthorized")
	}

	// Delete existing ingredients and instructions
	_, err = tx.Exec(`DELETE FROM recipe_ingredients WHERE recipe_id = $1`, recipeID)
	if err != nil {
		return fmt.Errorf("failed to delete old ingredients: %w", err)
	}

	_, err = tx.Exec(`DELETE FROM recipe_instructions WHERE recipe_id = $1`, recipeID)
	if err != nil {
		return fmt.Errorf("failed to delete old instructions: %w", err)
	}

	// Insert new ingredients
	for i, ingredient := range req.Ingredients {
		if ingredient == "" {
			continue
		}
		_, err = tx.Exec(`
			INSERT INTO recipe_ingredients (recipe_id, ingredient_text, order_index, item)
			VALUES ($1, $2, $3, $4)`,
			recipeID, ingredient, i+1, ingredient)
		if err != nil {
			return fmt.Errorf("failed to insert ingredient: %w", err)
		}
	}

	// Insert new instructions
	for i, instruction := range req.Instructions {
		if instruction == "" {
			continue
		}
		_, err = tx.Exec(`
			INSERT INTO recipe_instructions (recipe_id, instruction_text, order_index)
			VALUES ($1, $2, $3)`,
			recipeID, instruction, i+1)
		if err != nil {
			return fmt.Errorf("failed to insert instruction: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Delete removes a recipe from the database (only works for user recipes)
func Delete(db *sqlx.DB, recipeID, userID int64) error {
	result, err := db.Exec(`
		DELETE FROM recipes 
		WHERE recipe_id = $1 AND user_id = $2`,
		recipeID, userID,
	)

	if err != nil {
		return fmt.Errorf("failed to delete recipe: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check delete result: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("recipe not found or unauthorized")
	}

	return nil
}

// DeleteBySource removes recipes from a specific source
func DeleteBySource(db *sqlx.DB, sourceName string) (int64, error) {
	result, err := db.Exec(`
		DELETE FROM recipes 
		WHERE source_name = $1`,
		sourceName,
	)

	if err != nil {
		return 0, fmt.Errorf("failed to delete recipes by source: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to check delete result: %w", err)
	}

	return rowsAffected, nil
}

// SearchByTitle searches for recipes by title
func SearchByTitle(db *sqlx.DB, query string, onlyPublic bool) ([]Recipe, error) {
	var recipes []Recipe
	searchQuery := "%" + strings.ToLower(query) + "%"
	
	baseQuery := `
		SELECT * FROM recipes 
		WHERE LOWER(title) LIKE $1`
	
	if onlyPublic {
		baseQuery += ` AND is_public = true`
	}
	
	baseQuery += ` ORDER BY created_at DESC`

	err := db.Select(&recipes, baseQuery, searchQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to search recipes: %w", err)
	}
	
	if err = loadRecipeDetails(db, recipes); err != nil {
		return nil, err
	}
	
	return recipes, nil
}

// SearchByTags searches for recipes containing any of the specified tags
func SearchByTags(db *sqlx.DB, tags []string, onlyPublic bool) ([]Recipe, error) {
	var recipes []Recipe
	
	baseQuery := `
		SELECT * FROM recipes 
		WHERE tags && $1`
	
	if onlyPublic {
		baseQuery += ` AND is_public = true`
	}
	
	baseQuery += ` ORDER BY created_at DESC`

	err := db.Select(&recipes, baseQuery, pq.Array(tags))
	if err != nil {
		return nil, fmt.Errorf("failed to search recipes by tags: %w", err)
	}
	
	if err = loadRecipeDetails(db, recipes); err != nil {
		return nil, err
	}
	
	return recipes, nil
}

// Helper functions for nullable fields
func nullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

func nullInt(i int) sql.NullInt64 {
	return sql.NullInt64{Int64: int64(i), Valid: i > 0}
}

func nullInt64Ptr(i *int64) sql.NullInt64 {
	if i == nil {
		return sql.NullInt64{Valid: false}
	}
	return sql.NullInt64{Int64: *i, Valid: true}
}