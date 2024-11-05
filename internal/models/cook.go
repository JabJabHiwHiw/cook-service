package models

import "github.com/google/uuid"

type Cook struct {
	ID             uuid.UUID `db:"id"`
	Name           string    `db:"name"`
	Email          string    `db:"email"`
	ClerkId        string    `db:"clerk_id"`
	ProfilePicture string    `db:"profile_picture"`
}

type FavoriteMenu struct {
	ID      uuid.UUID `db:"id"`
	UserID  uuid.UUID `db:"user_id"`
	MenuID  uuid.UUID `db:"menu_id"`
	AddedAt string    `db:"added_at"`
}
