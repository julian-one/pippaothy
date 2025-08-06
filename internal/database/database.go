package database

import (
	"database/sql"
	"fmt"
	"pippaothy/internal/recipes"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// DB wraps sqlx.DB to provide application-specific methods
type DB struct {
	*sqlx.DB
}

// NewDB creates a new database connection wrapper
func NewDB() (*DB, error) {
	sqlxDB, err := Create()
	if err != nil {
		return nil, err
	}
	return &DB{DB: sqlxDB}, nil
}

// RecipeExists checks if a recipe with the given source URL already exists
func (db *DB) RecipeExists(sourceURL string) (bool, error) {
	var count int
	err := db.Get(&count, `
		SELECT COUNT(*) FROM recipes 
		WHERE source_url = $1`,
		sourceURL,
	)
	if err != nil {
		return false, fmt.Errorf("failed to check recipe existence: %w", err)
	}
	return count > 0, nil
}

// CreateRecipe creates a new recipe in the database
func (db *DB) CreateRecipe(req recipes.CreateRequest) (int64, error) {
	return recipes.Create(db.DB, req)
}

// GetRecipeByID retrieves a recipe by its ID
func (db *DB) GetRecipeByID(recipeID int64) (*recipes.Recipe, error) {
	return recipes.GetByID(db.DB, recipeID)
}

// GetPublicRecipes retrieves public recipes with pagination
func (db *DB) GetPublicRecipes(limit, offset int) ([]recipes.Recipe, error) {
	return recipes.GetPublic(db.DB, limit, offset)
}

// SearchRecipesByTitle searches for recipes by title
func (db *DB) SearchRecipesByTitle(query string, onlyPublic bool) ([]recipes.Recipe, error) {
	return recipes.SearchByTitle(db.DB, query, onlyPublic)
}

// DeleteRecipesBySource deletes all recipes from a specific source
func (db *DB) DeleteRecipesBySource(sourceName string) (int64, error) {
	return recipes.DeleteBySource(db.DB, sourceName)
}

// CountRecipesBySource counts recipes from a specific source
func (db *DB) CountRecipesBySource(sourceName string) (int, error) {
	var count int
	err := db.Get(&count, `
		SELECT COUNT(*) FROM recipes 
		WHERE source_name = $1`,
		sourceName,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to count recipes by source: %w", err)
	}
	return count, nil
}

// GetRecipesBySource retrieves all recipes from a specific source
func (db *DB) GetRecipesBySource(sourceName string) ([]recipes.Recipe, error) {
	return recipes.GetBySource(db.DB, sourceName)
}

// BatchCreateRecipes creates multiple recipes in a single transaction
func (db *DB) BatchCreateRecipes(requests []recipes.CreateRequest) ([]int64, error) {
	tx, err := db.Beginx()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var recipeIDs []int64
	for _, req := range requests {
		recipeID, err := recipes.CreateInTx(tx, req)
		if err != nil {
			return nil, fmt.Errorf("failed to create recipe %s: %w", req.Title, err)
		}
		recipeIDs = append(recipeIDs, recipeID)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return recipeIDs, nil
}

// GetDuplicateSourceURLs checks which source URLs already exist in the database
func (db *DB) GetDuplicateSourceURLs(sourceURLs []string) (map[string]bool, error) {
	if len(sourceURLs) == 0 {
		return make(map[string]bool), nil
	}

	query := `
		SELECT source_url 
		FROM recipes 
		WHERE source_url = ANY($1) AND source_url IS NOT NULL`
	
	var existingURLs []sql.NullString
	err := db.Select(&existingURLs, query, pq.Array(sourceURLs))
	if err != nil {
		return nil, fmt.Errorf("failed to check duplicate URLs: %w", err)
	}

	duplicates := make(map[string]bool)
	for _, url := range existingURLs {
		if url.Valid {
			duplicates[url.String] = true
		}
	}

	return duplicates, nil
}