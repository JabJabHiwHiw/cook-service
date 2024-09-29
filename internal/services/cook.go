package services

import (
	"context"
	"fmt"

	"github.com/JabJabHiwHiw/cook-service/internal/models"
	"github.com/JabJabHiwHiw/cook-service/proto"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"google.golang.org/grpc/metadata"
)

var _ proto.CookServiceServer = (*CookService)(nil)

type CookService struct {
	proto.UnimplementedCookServiceServer
	CookCollection *mongo.Collection
	MenuCollection *mongo.Collection
}

// ViewProfile retrieves the cook's profile based on the cook ID from the context.
func (s *CookService) ViewProfile(ctx context.Context, req *proto.Empty) (*proto.ProfileResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	var cookID string
	if ok {
		cookID = md.Get("cookID")[0]
	  }
	if !ok || cookID == "" {
		fmt.Println(cookID)
		return nil, fmt.Errorf("unable to retrieve cook ID from context")
	}

	// Convert cookID to ObjectID
	objectID, err := bson.ObjectIDFromHex(cookID)
	if err != nil {
		return nil, fmt.Errorf("invalid cook ID")
	}

	// Find the cook
	var cook models.Cook
	err = s.CookCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&cook)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("cook not found")
		}
		return nil, err
	}

	// Map models.Cook to proto.Profile
	profile := &proto.Profile{
		Id:       cook.ID.Hex(),
		Username: cook.UserName,
		Email:    cook.Email,
		Details:  cook.Details,
		// Do not include the password in the response
	}

	// Map FavoriteMenus
	for _, menu := range cook.FavoriteMenus {
		profile.FavoriteMenus = append(profile.FavoriteMenus, &proto.MenuItem{
			Id:          menu.ID.Hex(),
			Name:        menu.Name,
			Description: menu.Description,
			Ingredients: menu.Ingredients,
		})
	}

	return &proto.ProfileResponse{
		Profile: profile,
	}, nil
}

// UpdateProfile updates the cook's profile information.
func (s *CookService) UpdateProfile(ctx context.Context, req *proto.Profile) (*proto.ProfileResponse, error) {
	// Get cookID from context
	var cookID string
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		cookID = md.Get("cookID")[0]
	}

	objectID, err := bson.ObjectIDFromHex(cookID)
	if err != nil {
		return nil, fmt.Errorf("invalid cook ID")
	}

	// Build the update document
	update := bson.M{
		"$set": bson.M{
			"username": req.GetUsername(),
			"email":    req.GetEmail(),
			"details":  req.GetDetails(),
			// "password": req.Password, // If updating the password
		},
	}

	// Update the cook's profile
	_, err = s.CookCollection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	if err != nil {
		return nil, err
	}

	// Return the updated profile
	return s.ViewProfile(ctx, &proto.Empty{})
}

// VerifyCookDetails verifies the cook's login credentials.
func (s *CookService) VerifyCookDetails(ctx context.Context, req *proto.CookDetails) (*proto.CookDetailsResponse, error) {
	// Find the cook by email
	var cook models.Cook
	err := s.CookCollection.FindOne(ctx, bson.M{"email": req.GetEmail()}).Decode(&cook)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("cook not found")
		}
		return nil, err
	}

	// Verify the password (assuming plain text for simplicity; use hashing in production)
	if cook.Password != req.GetPassword() {
		return nil, fmt.Errorf("invalid password")
	}

	// Map models.Cook to proto.CookDetails
	cookDetails := &proto.CookDetails{
		Id:       cook.ID.Hex(),
		Username: cook.UserName,
		Email:    cook.Email,
		Details:  cook.Details,
		// Do not include the password in the response
	}

	// Map FavoriteMenus
	for _, menu := range cook.FavoriteMenus {
		cookDetails.FavoriteMenus = append(cookDetails.FavoriteMenus, &proto.MenuItem{
			Id:          menu.ID.Hex(),
			Name:        menu.Name,
			Description: menu.Description,
			Ingredients: menu.Ingredients,
		})
	}

	return &proto.CookDetailsResponse{
		Details: cookDetails,
	}, nil
}

// GetFavoriteMenus retrieves the cook's favorite menus.
func (s *CookService) GetFavoriteMenus(ctx context.Context, req *proto.Empty) (*proto.MenusResponse, error) {
	// Get cookID from context
	var cookID string
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		cookID = md.Get("cookID")[0]
	}

	objectID, err := bson.ObjectIDFromHex(cookID)
	if err != nil {
		return nil, fmt.Errorf("invalid cook ID")
	}

	// Find the cook
	var cook models.Cook
	err = s.CookCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&cook)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("cook not found")
		}
		return nil, err
	}

	// Map FavoriteMenus to proto.MenuItem
	var menus []*proto.MenuItem
	for _, menu := range cook.FavoriteMenus {
		menus = append(menus, &proto.MenuItem{
			Id:          menu.ID.Hex(),
			Name:        menu.Name,
			Description: menu.Description,
			Ingredients: menu.Ingredients,
		})
	}

	return &proto.MenusResponse{
		Menus: menus,
	}, nil
}

// AddFavoriteMenu adds a menu to the cook's list of favorite menus.
func (s *CookService) AddFavoriteMenu(ctx context.Context, req *proto.MenuItemRequest) (*proto.MenuItemResponse, error) {
	// Get cookID from context
	var cookID string
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		cookID = md.Get("cookID")[0]
	}

	cookObjectID, err := bson.ObjectIDFromHex(cookID)
	if err != nil {
		return nil, fmt.Errorf("invalid cook ID")
	}

	menuObjectID, err := bson.ObjectIDFromHex(req.GetId())
	if err != nil {
		return nil, fmt.Errorf("invalid menu ID")
	}

	// Find the menu
	var menu models.Menu
	err = s.MenuCollection.FindOne(ctx, bson.M{"_id": menuObjectID}).Decode(&menu)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("menu not found")
		}
		return nil, err
	}

	// Add the menu to the cook's favorite menus
	update := bson.M{
		"$addToSet": bson.M{
			"favorite_menus": menu,
		},
	}

	_, err = s.CookCollection.UpdateOne(ctx, bson.M{"_id": cookObjectID}, update)
	if err != nil {
		return nil, err
	}

	// Map to proto.MenuItem
	menuItem := &proto.MenuItem{
		Id:          menu.ID.Hex(),
		Name:        menu.Name,
		Description: menu.Description,
		Ingredients: menu.Ingredients,
	}

	return &proto.MenuItemResponse{
		Item: menuItem,
	}, nil
}

// RemoveFavoriteMenu removes a menu from the cook's list of favorite menus.
func (s *CookService) RemoveFavoriteMenu(ctx context.Context, req *proto.MenuItemRequest) (*proto.MenuItemResponse, error) {
	// Get cookID from context
	var cookID string
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		cookID = md.Get("cookID")[0]
	}

	cookObjectID, err := bson.ObjectIDFromHex(cookID)
	if err != nil {
		return nil, fmt.Errorf("invalid cook ID")
	}

	menuObjectID, err := bson.ObjectIDFromHex(req.GetId())
	if err != nil {
		return nil, fmt.Errorf("invalid menu ID")
	}

	// Remove the menu from the cook's favorite menus
	update := bson.M{
		"$pull": bson.M{
			"favorite_menus": bson.M{"_id": menuObjectID},
		},
	}

	_, err = s.CookCollection.UpdateOne(ctx, bson.M{"_id": cookObjectID}, update)
	if err != nil {
		return nil, err
	}

	// Optionally, return the removed menu item
	var menu models.Menu
	err = s.MenuCollection.FindOne(ctx, bson.M{"_id": menuObjectID}).Decode(&menu)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("menu not found")
		}
		return nil, err
	}

	menuItem := &proto.MenuItem{
		Id:          menu.ID.Hex(),
		Name:        menu.Name,
		Description: menu.Description,
		Ingredients: menu.Ingredients,
	}

	return &proto.MenuItemResponse{
		Item: menuItem,
	}, nil
}