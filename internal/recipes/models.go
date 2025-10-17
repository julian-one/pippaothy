package recipes

import (
	"time"
)

type Recipe struct {
	RecipeId    int64      `db:"recipe_id" json:"recipe_id"`
	UserId      int64      `db:"user_id" json:"user_id"`
	Title       string     `db:"title" json:"title"`
	Description *string    `db:"description" json:"description,omitempty"`
	PrepTime    *int       `db:"prep_time" json:"prep_time,omitempty"`
	CookTime    *int       `db:"cook_time" json:"cook_time,omitempty"`
	Servings    *int       `db:"servings" json:"servings,omitempty"`
	Difficulty  *string    `db:"difficulty" json:"difficulty,omitempty"`
	Cuisine     *string    `db:"cuisine" json:"cuisine,omitempty"`
	Category    *string    `db:"category" json:"category,omitempty"`
	ImageUrl    *string    `db:"image_url" json:"image_url,omitempty"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updated_at"`
}

type RecipeIngredient struct {
	IngredientId   int64     `db:"ingredient_id" json:"ingredient_id"`
	RecipeId       int64     `db:"recipe_id" json:"recipe_id"`
	IngredientName string    `db:"ingredient_name" json:"ingredient_name"`
	Quantity       *string   `db:"quantity" json:"quantity,omitempty"`
	Unit           *string   `db:"unit" json:"unit,omitempty"`
	Notes          *string   `db:"notes" json:"notes,omitempty"`
	DisplayOrder   int       `db:"display_order" json:"display_order"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
}

type RecipeInstruction struct {
	InstructionId   int64     `db:"instruction_id" json:"instruction_id"`
	RecipeId        int64     `db:"recipe_id" json:"recipe_id"`
	StepNumber      int       `db:"step_number" json:"step_number"`
	InstructionText string    `db:"instruction_text" json:"instruction_text"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
}

type RecipeWithDetails struct {
	Recipe       Recipe               `json:"recipe"`
	Ingredients  []RecipeIngredient   `json:"ingredients"`
	Instructions []RecipeInstruction  `json:"instructions"`
}

type CreateRecipeRequest struct {
	Title        string                     `json:"title"`
	Description  *string                    `json:"description,omitempty"`
	PrepTime     *int                       `json:"prep_time,omitempty"`
	CookTime     *int                       `json:"cook_time,omitempty"`
	Servings     *int                       `json:"servings,omitempty"`
	Difficulty   *string                    `json:"difficulty,omitempty"`
	Cuisine      *string                    `json:"cuisine,omitempty"`
	Category     *string                    `json:"category,omitempty"`
	ImageUrl     *string                    `json:"image_url,omitempty"`
	Ingredients  []CreateIngredientRequest  `json:"ingredients"`
	Instructions []CreateInstructionRequest `json:"instructions"`
}

type CreateIngredientRequest struct {
	IngredientName string  `json:"ingredient_name"`
	Quantity       *string `json:"quantity,omitempty"`
	Unit           *string `json:"unit,omitempty"`
	Notes          *string `json:"notes,omitempty"`
	DisplayOrder   int     `json:"display_order"`
}

type CreateInstructionRequest struct {
	StepNumber      int    `json:"step_number"`
	InstructionText string `json:"instruction_text"`
}

type UpdateRecipeRequest struct {
	Title        *string                    `json:"title,omitempty"`
	Description  *string                    `json:"description,omitempty"`
	PrepTime     *int                       `json:"prep_time,omitempty"`
	CookTime     *int                       `json:"cook_time,omitempty"`
	Servings     *int                       `json:"servings,omitempty"`
	Difficulty   *string                    `json:"difficulty,omitempty"`
	Cuisine      *string                    `json:"cuisine,omitempty"`
	Category     *string                    `json:"category,omitempty"`
	ImageUrl     *string                    `json:"image_url,omitempty"`
	Ingredients  []CreateIngredientRequest  `json:"ingredients,omitempty"`
	Instructions []CreateInstructionRequest `json:"instructions,omitempty"`
}

type ListRecipesFilter struct {
	UserId   *int64  `json:"user_id,omitempty"`
	Category *string `json:"category,omitempty"`
	Cuisine  *string `json:"cuisine,omitempty"`
	Search   *string `json:"search,omitempty"`
	Limit    int     `json:"limit"`
	Offset   int     `json:"offset"`
}