package services

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CookService represents the service handling cook-related operations.
type CookService struct {
	DB *sql.DB
}

// GetFavoriteMenus retrieves the favorite menus for the authenticated user.
func (s *CookService) GetFavoriteMenus(c *gin.Context) {
	// Get the user ID from the context
	userID := c.MustGet("user").(string)

	// Query to get the favorite menu IDs for the user
	query := `SELECT menu_id FROM favorite_menus WHERE user_id = $1`
	rows, err := s.DB.Query(query, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error querying favorite menus",
			"userId": userID,
		})
		return
	}
	defer rows.Close()

	// Collect the menu IDs
	var menuIDs []string
	for rows.Next() {
		var menuID string
		if err := rows.Scan(&menuID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error scanning menu IDs"})
			return
		}
		menuIDs = append(menuIDs, menuID)
	}

	// Return the favorite menu IDs
	c.JSON(http.StatusOK, gin.H{"favorite_menus": menuIDs})
}

// AddFavoriteMenu adds a menu to the user's list of favorite menus.
func (s *CookService) AddFavoriteMenu(c *gin.Context) {
	// Get the user ID from the context
	userID := c.MustGet("user").(string)

	// Get the menu ID from the request body
	var requestData struct {
		MenuID string `json:"menu_id"`
	}
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Generate a new UUID for the favorite_menus entry
	entryID := uuid.New()

	// Insert the favorite menu into the database
	query := `INSERT INTO favorite_menus (id, user_id, menu_id) VALUES ($1, $2, $3)`
	_, err := s.DB.Exec(query, entryID, userID, requestData.MenuID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error adding favorite menu"})
		return
	}

	// Return success message
	c.JSON(http.StatusOK, gin.H{"message": "Favorite menu added"})
}

// RemoveFavoriteMenu removes a menu from the user's list of favorite menus.
func (s *CookService) RemoveFavoriteMenu(c *gin.Context) {
	// Get the user ID from the context
	userID := c.MustGet("user").(string)

	// Get the menu ID from the URL parameter
	menuID := c.Param("menu_id")

	// Delete the favorite menu from the database
	query := `DELETE FROM favorite_menus WHERE user_id = $1 AND menu_id = $2`
	result, err := s.DB.Exec(query, userID, menuID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error removing favorite menu"})
		return
	}

	// Check if any row was deleted
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking deletion result"})
		return
	}
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Favorite menu not found"})
		return
	}

	// Return success message
	c.JSON(http.StatusOK, gin.H{"message": "Favorite menu removed"})
}
