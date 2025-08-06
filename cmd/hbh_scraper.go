package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"pippaothy/internal/database"
	"pippaothy/internal/recipes"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/spf13/cobra"
)

// RecipeData represents a scraped recipe
type RecipeData struct {
	Title        string   `json:"title"`
	URL          string   `json:"url"`
	Description  string   `json:"description"`
	Image        string   `json:"image"`
	Category     string   `json:"category"`
	Date         string   `json:"date"`
	Ingredients  []string `json:"ingredients"`
	Instructions []string `json:"instructions"`
	PrepTime     string   `json:"prep_time"`
	CookTime     string   `json:"cook_time"`
	Servings     string   `json:"servings"`
	Cuisine      string   `json:"cuisine"`
	Tags         []string `json:"tags"`
}

var (
	dryRun       bool
	delaySeconds int
	startPage    int
	endPage      int
	categoryURL  string
	verbose      bool
)

var rootCmd = &cobra.Command{
	Use:   "hbh-scraper",
	Short: "Half Baked Harvest recipe scraper",
	Long:  `A focused scraper for importing recipes from Half Baked Harvest`,
}

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import recipes from Half Baked Harvest",
	Long: `Import recipes from a specific category of Half Baked Harvest.
Example: hbh-scraper import --category https://www.halfbakedharvest.com/category/recipes/type-of-meal/main-course/ --start 1 --end 74`,
	Run: runImport,
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate scraping without importing (dry run)",
	Long: `Test the scraper and validate data extraction without saving to database.
Example: hbh-scraper validate --category https://www.halfbakedharvest.com/category/recipes/type-of-meal/main-course/ --start 1 --end 5`,
	Run: runValidate,
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().IntVarP(&delaySeconds, "delay", "d", 2, "Delay between requests in seconds")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")

	// Import command flags
	importCmd.Flags().StringVarP(&categoryURL, "category", "c", "", "Category URL to import from (required)")
	importCmd.Flags().IntVarP(&startPage, "start", "s", 1, "Start page number")
	importCmd.Flags().IntVarP(&endPage, "end", "e", 1, "End page number")
	importCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Run without actually importing to database")
	importCmd.MarkFlagRequired("category")

	// Validate command flags
	validateCmd.Flags().StringVarP(&categoryURL, "category", "c", "", "Category URL to validate (required)")
	validateCmd.Flags().IntVarP(&startPage, "start", "s", 1, "Start page number")
	validateCmd.Flags().IntVarP(&endPage, "end", "e", 1, "End page number")
	validateCmd.MarkFlagRequired("category")

	rootCmd.AddCommand(importCmd)
	rootCmd.AddCommand(validateCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runImport(cmd *cobra.Command, args []string) {
	fmt.Printf("🚀 Starting Half Baked Harvest import\n")
	fmt.Printf("Category: %s\n", categoryURL)
	fmt.Printf("Pages: %d to %d\n", startPage, endPage)
	fmt.Printf("Delay: %d seconds\n", delaySeconds)
	if dryRun {
		fmt.Println("Mode: DRY RUN (no database writes)")
	} else {
		fmt.Println("Mode: LIVE IMPORT")
	}
	fmt.Println(strings.Repeat("-", 50))

	// Initialize database connection only if not dry run
	var db *database.DB
	if !dryRun {
		var err error
		db, err = database.NewDB()
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}
		defer db.Close()
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	stats := ImportStats{}
	
	for page := startPage; page <= endPage; page++ {
		fmt.Printf("\n📄 Processing page %d/%d\n", page, endPage)
		
		recipes, err := scrapeRecipeList(client, categoryURL, page)
		if err != nil {
			log.Printf("❌ Error scraping page %d: %v", page, err)
			stats.Failed++
			continue
		}

		fmt.Printf("Found %d recipes on page %d\n", len(recipes), page)

		for i, recipe := range recipes {
			fmt.Printf("\n  [%d/%d] %s\n", i+1, len(recipes), recipe.Title)
			stats.Processed++

			// Get detailed recipe data
			detailedRecipe, err := scrapeRecipeDetail(client, recipe.URL)
			if err != nil {
				log.Printf("    ❌ Failed to get details: %v", err)
				stats.Failed++
				stats.Errors = append(stats.Errors, fmt.Sprintf("%s: %v", recipe.Title, err))
				continue
			}

			// Validate the recipe data
			if err := validateRecipe(detailedRecipe); err != nil {
				log.Printf("    ⚠️  Validation failed: %v", err)
				stats.Failed++
				stats.Errors = append(stats.Errors, fmt.Sprintf("%s: validation failed: %v", recipe.Title, err))
				continue
			}

			if verbose {
				fmt.Printf("    ✓ Title: %s\n", detailedRecipe.Title)
				fmt.Printf("    ✓ Ingredients: %d items\n", len(detailedRecipe.Ingredients))
				fmt.Printf("    ✓ Instructions: %d steps\n", len(detailedRecipe.Instructions))
			}

			// Import to database if not dry run
			if !dryRun && db != nil {
				req := convertToCreateRequest(detailedRecipe)
				
				// Check if recipe already exists
				existing, err := db.RecipeExists(req.SourceURL)
				if err != nil {
					log.Printf("    ❌ Database error: %v", err)
					stats.Failed++
					continue
				}
				
				if existing {
					if verbose {
						fmt.Printf("    ⏭️  Skipped (already exists)\n")
					}
					stats.Skipped++
					continue
				}

				// Create the recipe
				_, err = db.CreateRecipe(req)
				if err != nil {
					log.Printf("    ❌ Failed to save: %v", err)
					stats.Failed++
					stats.Errors = append(stats.Errors, fmt.Sprintf("%s: database save failed: %v", recipe.Title, err))
					continue
				}
				
				fmt.Printf("    ✅ Imported successfully\n")
				stats.Imported++
			} else if dryRun {
				fmt.Printf("    ✅ Valid (dry run)\n")
				stats.Imported++
			}

			// Delay between requests
			if i < len(recipes)-1 || page < endPage {
				time.Sleep(time.Duration(delaySeconds) * time.Second)
			}
		}
	}

	// Print summary
	fmt.Printf("\n%s\n", strings.Repeat("=", 50))
	fmt.Println("📊 Import Summary")
	fmt.Printf("Total Processed: %d\n", stats.Processed)
	if dryRun {
		fmt.Printf("Valid Recipes: %d\n", stats.Imported)
	} else {
		fmt.Printf("Imported: %d\n", stats.Imported)
		fmt.Printf("Skipped (duplicates): %d\n", stats.Skipped)
	}
	fmt.Printf("Failed: %d\n", stats.Failed)
	
	if len(stats.Errors) > 0 {
		fmt.Printf("\n⚠️  Errors encountered (%d):\n", len(stats.Errors))
		// Show last 10 errors
		start := 0
		if len(stats.Errors) > 10 {
			start = len(stats.Errors) - 10
			fmt.Printf("(Showing last 10 errors)\n")
		}
		for _, err := range stats.Errors[start:] {
			fmt.Printf("  - %s\n", err)
		}
	}
}

func runValidate(cmd *cobra.Command, args []string) {
	// Validation is just a dry run
	dryRun = true
	runImport(cmd, args)
}

func scrapeRecipeList(client *http.Client, categoryURL string, page int) ([]RecipeData, error) {
	var url string
	if page == 1 {
		url = categoryURL
	} else {
		url = strings.TrimSuffix(categoryURL, "/") + fmt.Sprintf("/page/%d/", page)
	}

	if verbose {
		log.Printf("Fetching: %s", url)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error parsing HTML: %w", err)
	}

	var recipes []RecipeData

	doc.Find(".post-summary").Each(func(i int, s *goquery.Selection) {
		recipe := RecipeData{}

		titleEl := s.Find(".post-summary__title a")
		recipe.Title = strings.TrimSpace(titleEl.Text())
		recipe.URL, _ = titleEl.Attr("href")

		imageEl := s.Find(".post-summary__image img")
		recipe.Image, _ = imageEl.Attr("src")
		if recipe.Image == "" {
			recipe.Image, _ = imageEl.Attr("data-src")
		}

		recipe.Category = strings.TrimSpace(s.Find(".entry-category").Text())
		recipe.Date = strings.TrimSpace(s.Find(".entry-date").Text())
		recipe.Description = strings.TrimSpace(s.Find(".post-summary__excerpt").Text())

		if recipe.Title != "" && recipe.URL != "" {
			recipes = append(recipes, recipe)
		}
	})

	return recipes, nil
}

func scrapeRecipeDetail(client *http.Client, recipeURL string) (*RecipeData, error) {
	if verbose {
		log.Printf("    Fetching details: %s", recipeURL)
	}

	req, err := http.NewRequest("GET", recipeURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching recipe: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error parsing HTML: %w", err)
	}

	recipe := &RecipeData{URL: recipeURL}

	// Extract title
	recipe.Title = strings.TrimSpace(doc.Find("h1.entry-title").Text())
	if recipe.Title == "" {
		recipe.Title = strings.TrimSpace(doc.Find(".page-header h1").Text())
	}
	if recipe.Title == "" {
		recipe.Title = strings.TrimSpace(doc.Find("h1").First().Text())
	}

	// Extract ingredients
	doc.Find(".wprm-recipe-ingredients-container li").Each(func(i int, s *goquery.Selection) {
		ingredient := strings.TrimSpace(s.Text())
		if ingredient != "" {
			recipe.Ingredients = append(recipe.Ingredients, ingredient)
		}
	})

	// Fallback ingredient selectors
	if len(recipe.Ingredients) == 0 {
		doc.Find(".recipe-ingredients li, .ingredients li").Each(func(i int, s *goquery.Selection) {
			ingredient := strings.TrimSpace(s.Text())
			if ingredient != "" {
				recipe.Ingredients = append(recipe.Ingredients, ingredient)
			}
		})
	}

	// Extract instructions - look for individual spans within instruction text
	doc.Find(".wprm-recipe-instruction-text span[style*='display: block']").Each(func(i int, s *goquery.Selection) {
		instruction := strings.TrimSpace(s.Text())
		// Remove leading numbers and dots (e.g., "1. " or "2. ")
		instruction = regexp.MustCompile(`^\d+\.\s*`).ReplaceAllString(instruction, "")
		if instruction != "" {
			recipe.Instructions = append(recipe.Instructions, instruction)
		}
	})

	// Fallback to li elements if no span blocks found
	if len(recipe.Instructions) == 0 {
		doc.Find(".wprm-recipe-instructions-container li").Each(func(i int, s *goquery.Selection) {
			instruction := strings.TrimSpace(s.Text())
			if instruction != "" {
				recipe.Instructions = append(recipe.Instructions, instruction)
			}
		})
	}

	// Another fallback for different markup
	if len(recipe.Instructions) == 0 {
		doc.Find(".wprm-recipe-instruction-text").Each(func(i int, s *goquery.Selection) {
			// Try to split by numbered patterns
			text := s.Text()
			// Split by patterns like "1. " or "2. "
			parts := regexp.MustCompile(`\d+\.\s+`).Split(text, -1)
			for _, part := range parts {
				instruction := strings.TrimSpace(part)
				if instruction != "" {
					recipe.Instructions = append(recipe.Instructions, instruction)
				}
			}
		})
	}

	// Extract metadata
	recipe.PrepTime = cleanTimeString(doc.Find(".wprm-recipe-prep-time").Text())
	recipe.CookTime = cleanTimeString(doc.Find(".wprm-recipe-cook-time").Text())
	recipe.Servings = strings.TrimSpace(doc.Find(".wprm-recipe-servings").Text())

	// Get image
	if recipe.Image == "" {
		imageEl := doc.Find(".entry-content img").First()
		recipe.Image, _ = imageEl.Attr("src")
		if recipe.Image == "" {
			recipe.Image, _ = imageEl.Attr("data-src")
		}
	}

	// Get description
	recipe.Description = strings.TrimSpace(doc.Find("meta[name='description']").AttrOr("content", ""))
	if recipe.Description == "" {
		recipe.Description = strings.TrimSpace(doc.Find(".entry-content p").First().Text())
	}

	// Set defaults
	recipe.Cuisine = "American"
	recipe.Tags = []string{"half-baked-harvest", "imported"}

	return recipe, nil
}

func validateRecipe(recipe *RecipeData) error {
	if recipe.Title == "" {
		return fmt.Errorf("missing title")
	}
	if len(recipe.Ingredients) == 0 {
		return fmt.Errorf("no ingredients found")
	}
	if len(recipe.Instructions) == 0 {
		return fmt.Errorf("no instructions found")
	}
	if recipe.URL == "" {
		return fmt.Errorf("missing URL")
	}
	return nil
}

func convertToCreateRequest(recipe *RecipeData) recipes.CreateRequest {
	return recipes.CreateRequest{
		SourceName:   "Half Baked Harvest",
		SourceURL:    recipe.URL,
		Title:        recipe.Title,
		Description:  recipe.Description,
		Ingredients:  recipe.Ingredients,
		Instructions: recipe.Instructions,
		PrepTime:     parseTimeMinutes(recipe.PrepTime),
		CookTime:     parseTimeMinutes(recipe.CookTime),
		Servings:     parseServings(recipe.Servings),
		Cuisine:      recipe.Cuisine,
		Tags:         recipe.Tags,
		ImageURL:     recipe.Image,
		IsPublic:     true,
	}
}

func cleanTimeString(timeStr string) string {
	timeStr = strings.ReplaceAll(timeStr, "minutes minutes", "minutes")
	timeStr = strings.ReplaceAll(timeStr, "Prep Time", "")
	timeStr = strings.ReplaceAll(timeStr, "Cook Time", "")
	return strings.TrimSpace(timeStr)
}

func parseTimeMinutes(timeStr string) int {
	if timeStr == "" {
		return 0
	}
	
	re := regexp.MustCompile(`\d+`)
	matches := re.FindAllString(timeStr, -1)
	
	if len(matches) > 0 {
		if minutes, err := strconv.Atoi(matches[0]); err == nil {
			return minutes
		}
	}
	
	return 0
}

func parseServings(servingsStr string) int {
	if servingsStr == "" {
		return 0
	}
	
	servingsStr = strings.ReplaceAll(servingsStr, "Servings:", "")
	servingsStr = strings.TrimSpace(servingsStr)
	
	re := regexp.MustCompile(`\d+`)
	matches := re.FindAllString(servingsStr, -1)
	
	if len(matches) > 0 {
		if servings, err := strconv.Atoi(matches[0]); err == nil {
			return servings
		}
	}
	
	return 0
}

// ImportStats tracks import statistics
type ImportStats struct {
	Processed int
	Imported  int
	Failed    int
	Skipped   int
	Errors    []string
}