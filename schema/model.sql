CREATE TABLE IF NOT EXISTS users (
	user_id SERIAL PRIMARY KEY,
	first_name TEXT NOT NULL,
	last_name TEXT NOT NULL,
	email TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	salt BYTEA NOT NULL,
	last_login TIMESTAMP,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sessions (
	session_id TEXT PRIMARY KEY,
	user_id INTEGER NOT NULL,
	expires_at TIMESTAMP NOT NULL,
	flash_message TEXT,
	FOREIGN KEY (user_id) REFERENCES users (user_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS password_resets (
	reset_id SERIAL PRIMARY KEY,
	user_id INTEGER NOT NULL,
	token TEXT NOT NULL UNIQUE,
	expires_at TIMESTAMP NOT NULL,
	used BOOLEAN DEFAULT FALSE,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (user_id) REFERENCES users (user_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS password_reset_attempts (
	attempt_id SERIAL PRIMARY KEY,
	email TEXT NOT NULL,
	ip_address TEXT NOT NULL,
	created_at TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_email_created ON password_reset_attempts (email, created_at);
CREATE INDEX IF NOT EXISTS idx_ip_created ON password_reset_attempts (ip_address, created_at);

CREATE TABLE IF NOT EXISTS recipes (
	recipe_id SERIAL PRIMARY KEY,
	user_id INTEGER NOT NULL,
	title TEXT NOT NULL,
	description TEXT,
	prep_time INTEGER,
	cook_time INTEGER,
	servings INTEGER,
	difficulty TEXT CHECK (difficulty IN ('easy', 'medium', 'hard')),
	cuisine TEXT,
	category TEXT,
	image_url TEXT,
	is_public BOOLEAN DEFAULT TRUE,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (user_id) REFERENCES users (user_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS recipe_ingredients (
	ingredient_id SERIAL PRIMARY KEY,
	recipe_id INTEGER NOT NULL,
	ingredient_name TEXT NOT NULL,
	quantity TEXT,
	unit TEXT,
	notes TEXT,
	display_order INTEGER DEFAULT 0,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (recipe_id) REFERENCES recipes (recipe_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS recipe_instructions (
	instruction_id SERIAL PRIMARY KEY,
	recipe_id INTEGER NOT NULL,
	step_number INTEGER NOT NULL,
	instruction_text TEXT NOT NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (recipe_id) REFERENCES recipes (recipe_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_recipes_user ON recipes (user_id);
CREATE INDEX IF NOT EXISTS idx_recipes_public ON recipes (is_public);
CREATE INDEX IF NOT EXISTS idx_ingredients_recipe ON recipe_ingredients (recipe_id);
CREATE INDEX IF NOT EXISTS idx_instructions_recipe ON recipe_instructions (recipe_id);
