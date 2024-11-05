package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/JabJabHiwHiw/cook-service/internal/services"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func InitializeDB(db *sql.DB) error {
	// Your existing DB initialization logic
	// ...
	return nil
}

func main() {
	// PostgreSQL connection
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

	// Define RESTful routes
	router.GET("/profile", cookService.ViewProfile)
	router.PUT("/profile", cookService.UpdateProfile)
	router.POST("/register", cookService.VerifyCookDetails)
	router.GET("/favorite-menus", cookService.GetFavoriteMenus)
	router.POST("/favorite-menus", cookService.AddFavoriteMenu)
	router.DELETE("/favorite-menus/:menu_id", cookService.RemoveFavoriteMenu)

	fmt.Println("Server started on port :8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
