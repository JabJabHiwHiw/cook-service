package models

import "go.mongodb.org/mongo-driver/v2/bson"

type Cook struct {
	ID              bson.ObjectID `bson:"_id,omitempty"`
	Name            string        `bson:"name"`
	Email           string        `bson:"email"`
	Password        string        `bson:"password"`
	Profile_picture string        `bson:"profile_picture"`
}

type Menu struct {
	ID          bson.ObjectID `bson:"_id,omitempty"`
	Name        string        `bson:"name"`
	Description string        `bson:"description"`
	Ingredients []string      `bson:"ingredients"`
}
