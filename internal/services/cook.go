package services

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/JabJabHiwHiw/cook-service/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type CookService struct {
	DB *sql.DB
}

// HandleGoogleOAuth handles Google OAuth authentication via Clerk
func (s *CookService) HandleGoogleOAuth(c *gin.Context) {
	// Define a struct to hold the incoming data
	type OAuthRequest struct {
		ClerkUserID    string `json:"clerk_user_id" binding:"required"`
		Name           string `json:"name" binding:"required"`
		Email          string `json:"email" binding:"required"`
		ProfilePicture string `json:"profile_picture"`
	}

	var req OAuthRequest

	// Bind the JSON body to the struct
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	var existingUser models.Cook
	err := s.DB.QueryRow(
		"SELECT id, name, email, clerk_id, profile_picture FROM cooks WHERE clerk_id=$1",
		req.ClerkUserID,
	).Scan(&existingUser.ID, &existingUser.Name, &existingUser.Email, &existingUser.ClerkId, &existingUser.ProfilePicture)

	if err == sql.ErrNoRows {
		// User does not exist; register them
		newUserID := uuid.New()

		_, err = s.DB.Exec(
			`INSERT INTO cooks (id, name, email, clerk_id, profile_picture)
             VALUES ($1, $2, $3, $4, $5)`,
			newUserID, req.Name, req.Email, req.ClerkUserID, req.ProfilePicture,
		)
		if err != nil {
			// Log the error for debugging
			fmt.Printf("Error inserting user: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user"})
			return
		}

		existingUser = models.Cook{
			ID:             newUserID,
			Name:           req.Name,
			Email:          req.Email,
			ClerkId:        req.ClerkUserID,
			ProfilePicture: req.ProfilePicture,
		}
	} else if err != nil {
		// Database error
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	} else {
		// User exists; update their clerk_id if different
		if existingUser.ClerkId != req.ClerkUserID {
			_, err = s.DB.Exec(
				"UPDATE cooks SET clerk_id=$1 WHERE id=$2",
				req.ClerkUserID, existingUser.ID,
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
				return
			}
			existingUser.ClerkId = req.ClerkUserID
		}
	}

	// Return the user's backend userId (UUID)
	c.JSON(http.StatusOK, gin.H{"user_id": existingUser.ID})
}

func (s *CookService) TestGet(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Cook service is up and running"})
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
	// Securely retrieve cookID from context
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Begin transaction
	tx, err := s.DB.Begin()
	if err != nil {
		// log.Errorf("Failed to begin transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	defer tx.Rollback()

	// Check for existing email or name
	if req.Email != "" || req.Name != "" {
		conditions := []string{}
		args := []interface{}{}
		argID := 1

		if req.Email != "" {
			conditions = append(conditions, fmt.Sprintf("email=$%d", argID))
			args = append(args, req.Email)
			argID++
		}
		if req.Name != "" {
			conditions = append(conditions, fmt.Sprintf("name=$%d", argID))
			args = append(args, req.Name)
			argID++
		}

		args = append(args, cookID)
		checkQuery := fmt.Sprintf("SELECT id FROM cooks WHERE (%s) AND id != $%d", strings.Join(conditions, " OR "), argID)

		var existingCookID uuid.UUID
		err := tx.QueryRow(checkQuery, args...).Scan(&existingCookID)
		if err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Email or name is already in use"})
			return
		} else if err != sql.ErrNoRows {
			// log.Errorf("Error checking existing cook: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}
	}

	// Build the UPDATE statement
	updates := []string{}
	args := []interface{}{}
	argID := 1

	if req.Name != "" {
		updates = append(updates, fmt.Sprintf("name=$%d", argID))
		args = append(args, req.Name)
		argID++
	}
	if req.Email != "" {
		updates = append(updates, fmt.Sprintf("email=$%d", argID))
		args = append(args, req.Email)
		argID++
	}
	if req.ProfilePicture != "" {
		updates = append(updates, fmt.Sprintf("profile_picture=$%d", argID))
		args = append(args, req.ProfilePicture)
		argID++
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	args = append(args, cookID)
	query := fmt.Sprintf("UPDATE cooks SET %s WHERE id=$%d", strings.Join(updates, ", "), argID)

	// Execute the update
	_, err = tx.Exec(query, args...)
	if err != nil {
		// log.Errorf("Failed to update profile for cookID %s: %v", cookID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		// log.Errorf("Failed to commit transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Retrieve and return the updated profile
	var updatedProfile models.Profile
	err = s.DB.QueryRow(
		"SELECT name, email, profile_picture FROM cooks WHERE id=$1",
		cookID,
	).Scan(&updatedProfile.Name, &updatedProfile.Email, &updatedProfile.ProfilePicture)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve updated profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"profile": updatedProfile})
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
