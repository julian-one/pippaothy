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

CREATE TABLE IF NOT EXISTS recipes (
	recipe_id SERIAL PRIMARY KEY,
	user_id INTEGER,
	source_name TEXT,
	source_url TEXT UNIQUE,
	title TEXT NOT NULL,
	description TEXT,
	prep_time INTEGER,
	cook_time INTEGER,
	servings INTEGER,
	difficulty TEXT CHECK (difficulty IN ('easy', 'medium', 'hard')),
	cuisine TEXT,
	tags TEXT[],
	image_url TEXT,
	is_public BOOLEAN DEFAULT false,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (user_id) REFERENCES users (user_id) ON DELETE CASCADE,
	CHECK ((user_id IS NOT NULL AND source_name IS NULL AND source_url IS NULL) OR 
	       (user_id IS NULL AND source_name IS NOT NULL AND source_url IS NOT NULL))
);

CREATE TABLE IF NOT EXISTS recipe_ingredients (
	ingredient_id SERIAL PRIMARY KEY,
	recipe_id INTEGER NOT NULL,
	ingredient_text TEXT NOT NULL,
	order_index INTEGER NOT NULL,
	amount TEXT,
	unit TEXT,
	item TEXT NOT NULL,
	notes TEXT,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (recipe_id) REFERENCES recipes (recipe_id) ON DELETE CASCADE,
	UNIQUE(recipe_id, order_index)
);

CREATE TABLE IF NOT EXISTS recipe_instructions (
	instruction_id SERIAL PRIMARY KEY,
	recipe_id INTEGER NOT NULL,
	instruction_text TEXT NOT NULL,
	order_index INTEGER NOT NULL,
	estimated_time INTEGER,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (recipe_id) REFERENCES recipes (recipe_id) ON DELETE CASCADE,
	UNIQUE(recipe_id, order_index)
);


