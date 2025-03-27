package recipes

import (
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type CreateRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Ingredients []string `json:"ingredients"`
	Steps       []string `json:"steps"`
	PrepTime    int64    `json:"prep_time"`
	CookTime    int64    `json:"cook_time"`
	Servings    int64    `json:"servings"`
}

func Create(db *sqlx.DB, req CreateRequest) (int, error) {
	query := `
		INSERT INTO recipes (name, description, ingredients, steps, prep_time, cook_time, servings)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING recipe_id
	`
	var id int
	err := db.QueryRow(
		query,
		req.Name,
		req.Description,
		pq.Array(req.Ingredients),
		pq.Array(req.Steps),
		req.PrepTime,
		req.CookTime,
		req.Servings,
	).Scan(&id)
	return id, err
}

type Recipe struct {
	RecipeId    string    `json:"recipe_id" db:"recipe_id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	Ingredients []string  `json:"ingredients" db:"ingredients"`
	Steps       []string  `json:"steps" db:"steps"`
	PrepTime    int64     `json:"prep_time" db:"prep_time"`
	CookTime    int64     `json:"cook_time" db:"cook_time"`
	Servings    int64     `json:"servings" db:"servings"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

func List(db *sqlx.DB) ([]Recipe, error) {
	rows, err := db.Query(`SELECT * FROM recipes ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rl []Recipe
	for rows.Next() {
		var r Recipe
		err := rows.Scan(
			&r.RecipeId,
			&r.Name,
			&r.Description,
			pq.Array(&r.Ingredients),
			pq.Array(&r.Steps),
			&r.PrepTime,
			&r.CookTime,
			&r.Servings,
			&r.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		rl = append(rl, r)
	}
	return rl, nil
}
