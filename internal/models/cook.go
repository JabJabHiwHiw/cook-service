package models

import "go.mongodb.org/mongo-driver/v2/bson"

type Cook struct {
	ID          bson.ObjectID `bson:"_id,omitempty"`
	UserName    string        `bson:"username"`
	Email       string        `bson:"email"`
	Password    string        `bson:"password"`
	Details    	string        `bson:"details"`
	FavoriteMenus []Menu `bson:"favorite_menus"`
}

type Menu struct {
	ID          bson.ObjectID `bson:"_id,omitempty"`
	Name        string        `bson:"name"`
	Description string        `bson:"description"`
	Ingredients []string      `bson:"ingredients"`
}