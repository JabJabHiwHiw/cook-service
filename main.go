package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/JabJabHiwHiw/cook-service/internal/services"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func InitializeDB(db *sql.DB) error {
	// Query to create the `cooks` table
	cooksTableQuery := `
    CREATE TABLE IF NOT EXISTS cooks (
        id UUID PRIMARY KEY,
        name VARCHAR(255) NOT NULL,
        email VARCHAR(255) UNIQUE NOT NULL,
        clerk_id VARCHAR(255) NOT NULL,
        profile_picture TEXT
    );`

	// Execute the query to create the `cooks` table
	if _, err := db.Exec(cooksTableQuery); err != nil {
		return err
	}

	// Query to create the `favorite_menus` table
	favoriteMenusTableQuery := `
    CREATE TABLE IF NOT EXISTS favorite_menus (
        id UUID PRIMARY KEY,
        user_id UUID NOT NULL REFERENCES cooks(id),
        menu_id UUID NOT NULL,
        added_at TIMESTAMP NOT NULL DEFAULT NOW(),
        UNIQUE(user_id, menu_id)  -- Ensure that each user-menu pair is unique
    );`

	// Execute the query to create the `favorite_menus` table
	if _, err := db.Exec(favoriteMenusTableQuery); err != nil {
		return err
	}

	return nil
}

func main() {
	connStr := "postgres://user:pass@localhost:5432/cook_service?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Error connecting to the database: ", err)
	}
	defer db.Close()

	fmt.Println("Connected to PostgreSQL")

	err = InitializeDB(db)
	if err != nil {
		log.Fatal("Error initializing database: ", err)
	}

	cookService := services.CookService{
		DB: db,
	}

	router := gin.Default()

	config := cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "https://hiw-hiw.vercel.app"}, // localhost 3000 for development, vercel app for production
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}

	router.Use(cors.New(config))

	router.GET("/user/", cookService.TestGet)
	router.GET("/user/profile", cookService.ViewProfile)
	router.PUT("/user/profile", cookService.UpdateProfile)
	router.POST("/user/oauth/google", cookService.HandleGoogleOAuth)
	router.GET("/user/favorite-menus", cookService.GetFavoriteMenus)
	router.POST("/user/favorite-menus", cookService.AddFavoriteMenu)
	router.DELETE("/user/favorite-menus/:menu_id", cookService.RemoveFavoriteMenu)

	fmt.Println("Server started on port :8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
