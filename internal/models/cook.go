package models

import "github.com/google/uuid"

type Cook struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	Email          string    `json:"email"`
	ClerkId        string    `json:"clerk_id,omitempty"`
	ProfilePicture string    `json:"profile_picture"`
}

type FavoriteMenu struct {
	ID     uuid.UUID `json:"id"`
	UserID uuid.UUID `json:"user_id,omitempty"`
	MenuID uuid.UUID `json:"menu_id"`
}

type Profile struct {
	ID             uuid.UUID `json:"id,omitempty"`
	Name           string    `json:"name"`
	Email          string    `json:"email"`
	ClerkId        string    `json:"clerk_id,omitempty"`
	ProfilePicture string    `json:"profile_picture"`
}
