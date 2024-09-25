package models

import "go.mongodb.org/mongo-driver/v2/bson"

type Cook struct {
	ID          bson.ObjectID `bson:"_id,omitempty"`
	UserName    string        `bson:"username"`
	Email       string        `bson:"email"`
	Password    string        `bson:"password"`
	Description string        `bson:"description"`
}