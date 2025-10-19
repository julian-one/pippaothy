package recipe

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

// ParseIngredients dynamically parses ingredients from form values
func ParseIngredients(r *http.Request) []CreateIngredientRequest {
	var ingredients []CreateIngredientRequest
	ingredientIndices := make(map[int]bool)

	// Collect all ingredient indices from form keys
	for key := range r.Form {
		if strings.HasPrefix(key, "ingredients[") && strings.Contains(key, "[name]") {
			// Extract index from key like "ingredients[0][name]"
			start := len("ingredients[")
			end := strings.Index(key[start:], "]")
			if end > 0 {
				if idx, err := strconv.Atoi(key[start : start+end]); err == nil {
					ingredientIndices[idx] = true
				}
			}
		}
	}

	// Sort indices to maintain order
	var indices []int
	for idx := range ingredientIndices {
		indices = append(indices, idx)
	}
	sort.Ints(indices)

	// Process each ingredient
	displayOrder := 0
	for _, i := range indices {
		name := strings.TrimSpace(r.FormValue(fmt.Sprintf("ingredients[%d][name]", i)))
		if name == "" {
			continue // Skip empty entries
		}

		ingredient := CreateIngredientRequest{
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

	return ingredients
}

// ParseInstructions dynamically parses instructions from form values
func ParseInstructions(r *http.Request) []CreateInstructionRequest {
	var instructions []CreateInstructionRequest
	instructionIndices := make(map[int]bool)

	// Collect all instruction indices from form keys
	for key := range r.Form {
		if strings.HasPrefix(key, "instructions[") && strings.HasSuffix(key, "]") {
			// Extract index from key like "instructions[0]"
			start := len("instructions[")
			end := strings.LastIndex(key, "]")
			if end > start {
				if idx, err := strconv.Atoi(key[start:end]); err == nil {
					instructionIndices[idx] = true
				}
			}
		}
	}

	// Sort indices to maintain order
	var indices []int
	for idx := range instructionIndices {
		indices = append(indices, idx)
	}
	sort.Ints(indices)

	// Process each instruction
	stepNumber := 1
	for _, i := range indices {
		text := strings.TrimSpace(r.FormValue(fmt.Sprintf("instructions[%d]", i)))
		if text == "" {
			continue // Skip empty entries
		}

		instructions = append(instructions, CreateInstructionRequest{
			StepNumber:      stepNumber,
			InstructionText: text,
		})
		stepNumber++
	}

	return instructions
}