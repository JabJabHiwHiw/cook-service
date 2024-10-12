package table

import (
	"database/sql"
)

// InitializeDB sets up the database table if it does not exist
func InitializeDB(db *sql.DB) error {
	query := `
    CREATE TABLE IF NOT EXISTS cooks (
        id SERIAL PRIMARY KEY,
        username VARCHAR(255) UNIQUE NOT NULL,
        email VARCHAR(255) UNIQUE NOT NULL,
        password VARCHAR(255) NOT NULL,
        profile_picture TEXT
    );`
	_, err := db.Exec(query)
	if err != nil {
		return err
	}
	return nil
}
