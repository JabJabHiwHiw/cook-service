package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/JabJabHiwHiw/cook-service/internal/services"
	"github.com/clerkinc/clerk-sdk-go/clerk"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func InitializeDB(db *sql.DB) error {

	// Query to create the `favorite_menus` table
	favoriteMenusTableQuery := `
    CREATE TABLE IF NOT EXISTS favorite_menus (
        id UUID PRIMARY KEY,
        user_id VARCHAR NOT NULL,
        menu_id VARCHAR NOT NULL,
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

	// Load environment variables from .env file
	err = godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	}

	// Get Clerk secret key from environment variable
	key := os.Getenv("CLERK_SECRET_KEY")

	// Create a new Clerk client instance
	client, _ := clerk.NewClient(key)

	// Create a new Gin router instance
	router := gin.Default()

	// Configure CORS middleware
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"PUT", "PATCH", "GET", "POST", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// VerifyToken is a middleware function that verifies the session token
	// from the Authorization header
	verifyToken := func(c *gin.Context) {
		sessionToken := c.Request.Header.Get("Authorization")
		if sessionToken == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// Remove the "Bearer " prefix from the session token
		sessionToken = strings.TrimPrefix(sessionToken, "Bearer ")

		// Verify the session token using the Clerk client
		sessClaims, err := client.VerifyToken(sessionToken)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid session token"})
			return
		}

		// Print the session claims
		fmt.Println(sessClaims.Claims)

		// Store the user info in the context for later access
		c.Set("user", sessClaims.Claims.Subject)
	}

	// Protected is a handler function that returns a welcome message
	// to the authenticated user
	protected := func(c *gin.Context) {
		// Get the user info from the context
		userID := c.MustGet("user").(string)

		// Get the user's email addresses using the Clerk client
		email := client.Emails()

		fmt.Println(email) // print out emails

		// Get the user's profile using the Clerk client
		user, err := client.Users().Read(userID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving user"})
			return
		}

		// Return a welcome message to the user
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Welcome, %s!", *user.FirstName)})
	}

	// Register the protected route
	router.GET("/protected", verifyToken, protected)

	router.GET("/user/favorite-menus", verifyToken, cookService.GetFavoriteMenus)
	router.POST("/user/favorite-menus", verifyToken, cookService.AddFavoriteMenu)
	router.DELETE("/user/favorite-menus/:menu_id", verifyToken, cookService.RemoveFavoriteMenu)

	// Run the server
	fmt.Println("Server started on port :8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
