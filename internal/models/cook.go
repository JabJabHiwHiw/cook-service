package models

import "go.mongodb.org/mongo-driver/v2/bson"

type Cook struct {
	ID             int64  `db:"id"`
	Name           string `db:"name"`
	Email          string `db:"email"`
	Password       string `db:"password"`
	ProfilePicture string `db:"profile_picture"`
}

type Menu struct {
	ID          bson.ObjectID `bson:"_id,omitempty"`
	Name        string        `bson:"name"`
	Description string        `bson:"description"`
	Ingredients []string      `bson:"ingredients"`
}
