package models

import "github.com/google/uuid"

type FavoriteMenu struct {
	ID     uuid.UUID `json:"id"`
	UserID uuid.UUID `json:"user_id,omitempty"`
	MenuID uuid.UUID `json:"menu_id"`
}
