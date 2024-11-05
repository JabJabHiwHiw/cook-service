package services

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/JabJabHiwHiw/cook-service/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type CookService struct {
	DB *sql.DB
}

// Register a new cook
func (s *CookService) VerifyCookDetails(c *gin.Context) {
	var req models.Profile
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Check if the cook already exists
	var existingCookID uuid.UUID
	err := s.DB.QueryRow(
		"SELECT id FROM cooks WHERE email=$1 OR name=$2",
		req.Email, req.Name,
	).Scan(&existingCookID)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Cook already exists with the given email or name"})
		return
	} else if err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check existing cook"})
		return
	}

	// Hash the clerk_id
	hashedClerkId, err := hashClerkId(req.ClerkId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash clerk ID"})
		return
	}

	// Insert the new cook
	newCookID := uuid.New()
	_, err = s.DB.Exec(
		`INSERT INTO cooks (id, name, email, clerk_id, profile_picture)
         VALUES ($1, $2, $3, $4, $5)`,
		newCookID, req.Name, req.Email, hashedClerkId, req.ProfilePicture,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register cook"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"profile": req})
}

// View cook profile
func (s *CookService) ViewProfile(c *gin.Context) {
	cookIDStr := c.GetHeader("cookID")
	if cookIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cook ID is required in header"})
		return
	}

	cookID, err := uuid.Parse(cookIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid cook ID format"})
		return
	}

	var cook models.Cook
	err = s.DB.QueryRow(
		"SELECT id, name, email, profile_picture FROM cooks WHERE id=$1",
		cookID,
	).Scan(&cook.ID, &cook.Name, &cook.Email, &cook.ProfilePicture)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Cook not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"profile": cook})
}

// Update cook profile
func (s *CookService) UpdateProfile(c *gin.Context) {
	cookIDStr := c.GetHeader("cookID")
	if cookIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cook ID is required in header"})
		return
	}

	cookID, err := uuid.Parse(cookIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid cook ID format"})
		return
	}

	var req models.Profile
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	_, err = s.DB.Exec(
		`UPDATE cooks SET name=$1, email=$2, profile_picture=$3 WHERE id=$4`,
		req.Name, req.Email, req.ProfilePicture, cookID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	s.ViewProfile(c)
}

// Get favorite menus
func (s *CookService) GetFavoriteMenus(c *gin.Context) {
	cookIDStr := c.GetHeader("cookID")
	if cookIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cook ID is required in header"})
		return
	}

	cookID, err := uuid.Parse(cookIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid cook ID format"})
		return
	}

	rows, err := s.DB.Query(
		"SELECT id, menu_id FROM favorite_menus WHERE user_id=$1",
		cookID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve favorite menus"})
		return
	}
	defer rows.Close()

	var favoriteMenus []models.FavoriteMenu
	for rows.Next() {
		var fm models.FavoriteMenu
		err := rows.Scan(&fm.ID, &fm.MenuID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read favorite menu"})
			return
		}
		favoriteMenus = append(favoriteMenus, fm)
	}

	c.JSON(http.StatusOK, gin.H{"favorite_menus": favoriteMenus})
}

// Add a favorite menu
func (s *CookService) AddFavoriteMenu(c *gin.Context) {
	cookIDStr := c.GetHeader("cookID")
	if cookIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cook ID is required in header"})
		return
	}

	cookID, err := uuid.Parse(cookIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid cook ID format"})
		return
	}

	var req models.FavoriteMenu
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	favoriteMenuID := uuid.New()
	_, err = s.DB.Exec(
		`INSERT INTO favorite_menus (id, user_id, menu_id)
         VALUES ($1, $2, $3)`,
		favoriteMenuID, cookID, req.MenuID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add favorite menu"})
		return
	}

	s.GetFavoriteMenus(c)
}

// Remove a favorite menu
func (s *CookService) RemoveFavoriteMenu(c *gin.Context) {
	cookIDStr := c.GetHeader("cookID")
	if cookIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cook ID is required in header"})
		return
	}

	cookID, err := uuid.Parse(cookIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid cook ID format"})
		return
	}

	menuIDStr := c.Param("menu_id")
	menuID, err := uuid.Parse(menuIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid menu ID format"})
		return
	}

	_, err = s.DB.Exec(
		"DELETE FROM favorite_menus WHERE user_id=$1 AND menu_id=$2",
		cookID, menuID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove favorite menu"})
		return
	}

	s.GetFavoriteMenus(c)
}

// Helper function to hash the clerk ID
func hashClerkId(clerkId string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(clerkId), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash clerkId: %w", err)
	}
	return string(bytes), nil
}
