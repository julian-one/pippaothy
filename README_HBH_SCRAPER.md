# Half Baked Harvest Recipe Scraper

A focused, efficient web scraper for importing recipes from Half Baked Harvest into a PostgreSQL database.

## Table of Contents
- [Features](#features)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Configuration](#configuration)
- [Usage](#usage)
- [Commands](#commands)
- [Database Schema](#database-schema)
- [Data Extraction](#data-extraction)
- [Performance](#performance)
- [Troubleshooting](#troubleshooting)

## Features

- **Batch Import**: Import multiple pages of recipes in a single operation
- **Dry Run Mode**: Validate scraping without database writes
- **Duplicate Detection**: Automatically skips recipes already in the database
- **Proper Data Parsing**: Correctly extracts individual ingredients and numbered instruction steps
- **Rate Limiting**: Configurable delays between requests to be respectful to the server
- **Progress Tracking**: Real-time feedback on import progress
- **Error Handling**: Graceful error handling with detailed error reporting

## Prerequisites

- Go 1.20 or higher
- PostgreSQL database
- Environment variables configured for database connection:
  - `DB_HOST`
  - `DB_PORT`
  - `DB_USER`
  - `DB_PASSWORD`
  - `DB_NAME`

## Installation

1. Build the scraper:
```bash
go build -o ./bin/hbh-scraper ./cmd/hbh_scraper.go
```

2. Ensure database is initialized with the schema (from `schema/model.sql`):
```sql
CREATE TABLE IF NOT EXISTS recipes (
    recipe_id SERIAL PRIMARY KEY,
    source_name TEXT,
    source_url TEXT UNIQUE,
    title TEXT NOT NULL,
    -- ... other fields
);
```

## Configuration

### Environment Variables

Set the following environment variables for database connection:

```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=your_user
export DB_PASSWORD=your_password
export DB_NAME=pippaothy
```

### Command Line Flags

Global flags available for all commands:
- `--delay, -d`: Delay between requests in seconds (default: 2)
- `--verbose, -v`: Enable verbose logging

## Usage

### Basic Import

Import recipes from a specific category:

```bash
./bin/hbh-scraper import \
  --category "https://www.halfbakedharvest.com/category/recipes/type-of-meal/main-course/" \
  --start 1 \
  --end 10
```

### Validation (Dry Run)

Test the scraper without writing to database:

```bash
./bin/hbh-scraper validate \
  --category "https://www.halfbakedharvest.com/category/recipes/type-of-meal/main-course/" \
  --start 1 \
  --end 5 \
  --verbose
```

### Import with Dry Run

Use the import command in dry-run mode:

```bash
./bin/hbh-scraper import \
  --category "https://www.halfbakedharvest.com/category/recipes/type-of-meal/main-course/" \
  --start 1 \
  --end 74 \
  --dry-run
```

## Commands

### `import`

Import recipes from Half Baked Harvest into the database.

**Flags:**
- `--category, -c` (required): Category URL to import from
- `--start, -s`: Start page number (default: 1)
- `--end, -e`: End page number (default: 1)
- `--dry-run`: Run without actually importing to database

**Example:**
```bash
./bin/hbh-scraper import \
  --category "https://www.halfbakedharvest.com/category/recipes/type-of-meal/appetizers-snacks/" \
  --start 1 \
  --end 20 \
  --delay 3
```

### `validate`

Validate scraping without importing (equivalent to dry run).

**Flags:**
- `--category, -c` (required): Category URL to validate
- `--start, -s`: Start page number (default: 1)
- `--end, -e`: End page number (default: 1)

**Example:**
```bash
./bin/hbh-scraper validate \
  --category "https://www.halfbakedharvest.com/category/recipes/type-of-meal/desserts/" \
  --start 1 \
  --end 5
```

## Database Schema

The scraper imports recipes into the following database structure:

### recipes table
- `recipe_id`: Primary key
- `source_name`: "Half Baked Harvest"
- `source_url`: Original recipe URL (unique constraint)
- `title`: Recipe title
- `description`: Recipe description
- `prep_time`: Preparation time in minutes
- `cook_time`: Cooking time in minutes
- `servings`: Number of servings
- `cuisine`: Cuisine type (defaults to "American")
- `tags`: Array of tags
- `image_url`: Featured image URL
- `is_public`: Boolean (defaults to true)

### recipe_ingredients table
- `ingredient_id`: Primary key
- `recipe_id`: Foreign key to recipes
- `ingredient_text`: Full ingredient text
- `order_index`: Order of ingredient in list

### recipe_instructions table
- `instruction_id`: Primary key
- `recipe_id`: Foreign key to recipes
- `instruction_text`: Instruction step text
- `order_index`: Step number

## Data Extraction

The scraper extracts the following data from each recipe:

1. **Title**: Recipe name from `h1.entry-title`
2. **Ingredients**: Individual items from `.wprm-recipe-ingredients-container li`
3. **Instructions**: Individual numbered steps from `.wprm-recipe-instruction-text span[style*='display: block']`
4. **Prep Time**: From `.wprm-recipe-prep-time`
5. **Cook Time**: From `.wprm-recipe-cook-time`
6. **Servings**: From `.wprm-recipe-servings`
7. **Image**: Featured image from recipe page
8. **Description**: From meta description or first paragraph

### Parsing Details

- **Instructions**: The scraper properly parses numbered instructions that are contained in `<span style="display: block;">` elements, removing the number prefixes
- **Time Parsing**: Extracts numeric values from time strings (e.g., "30 minutes" → 30)
- **Servings**: Extracts numeric value from servings text

## Performance

### Rate Limiting

The scraper includes built-in rate limiting:
- Default 2-second delay between requests
- Configurable via `--delay` flag
- No delay after the last item on the last page

### Batch Processing

- Processes 24 recipes per page (Half Baked Harvest's pagination)
- Can handle large ranges (e.g., 74 pages = ~1,776 recipes)
- Progress feedback for each recipe processed

### Example Import Times

With 2-second delay between recipes:
- 1 page (24 recipes): ~48 seconds
- 10 pages (240 recipes): ~8 minutes
- 74 pages (1,776 recipes): ~59 minutes

## Troubleshooting

### Common Issues

1. **Database Connection Failed**
   - Verify environment variables are set
   - Check PostgreSQL is running
   - Confirm database exists

2. **No Instructions Found**
   - Some recipe posts may be menu/collection pages
   - These are automatically skipped with validation errors

3. **Duplicate Recipe Errors**
   - The scraper checks for existing `source_url` values
   - Duplicates are automatically skipped and counted

4. **HTTP Errors**
   - Check internet connection
   - Verify the category URL is valid
   - Consider increasing delay if getting rate limited

### Error Reporting

The scraper provides detailed error reporting:
- Summary statistics after each run
- Last 10 errors displayed at completion
- Verbose mode (`-v`) for detailed logging

### Example Output

```
🚀 Starting Half Baked Harvest import
Category: https://www.halfbakedharvest.com/category/recipes/type-of-meal/main-course/
Pages: 1 to 5
Delay: 2 seconds
Mode: LIVE IMPORT
--------------------------------------------------

📄 Processing page 1/5
Found 24 recipes on page 1

  [1/24] Crockpot Brown Butter Tomato and Ricotta Pasta.
    ✅ Imported successfully

  [2/24] Skillet Mexican Tomatillo Chicken and Rice.
    ⏭️  Skipped (already exists)

==================================================
📊 Import Summary
Total Processed: 120
Imported: 95
Skipped (duplicates): 20
Failed: 5
```

## Half Baked Harvest Categories

Common category URLs for importing:

- **Main Courses**: `https://www.halfbakedharvest.com/category/recipes/type-of-meal/main-course/`
- **Appetizers**: `https://www.halfbakedharvest.com/category/recipes/type-of-meal/appetizers-snacks/`
- **Desserts**: `https://www.halfbakedharvest.com/category/recipes/type-of-meal/desserts/`
- **Breakfast**: `https://www.halfbakedharvest.com/category/recipes/type-of-meal/breakfast-brunch/`
- **Salads**: `https://www.halfbakedharvest.com/category/recipes/type-of-meal/salads/`
- **Soups**: `https://www.halfbakedharvest.com/category/recipes/type-of-meal/soups/`
- **Drinks**: `https://www.halfbakedharvest.com/category/recipes/type-of-meal/drinks/`

## License

This scraper is for personal use only. Please respect Half Baked Harvest's terms of service and robots.txt when using this tool.

## Support

For issues or questions, please check the error messages and troubleshooting section first. The scraper provides detailed error reporting to help diagnose problems.