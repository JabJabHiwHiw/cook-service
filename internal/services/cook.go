package services

import (
	"context"
	"fmt"

	"github.com/JabJabHiwHiw/cook-service/internal/models"
	"github.com/JabJabHiwHiw/cook-service/proto"
	"github.com/jackc/pgx/v4/pgxpool"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"google.golang.org/grpc/metadata"
)

var _ proto.CookServiceServer = (*CookService)(nil)

type CookService struct {
	proto.UnimplementedCookServiceServer
	CookCollection *mongo.Collection
	MenuCollection *mongo.Collection
	DBPool         *pgxpool.Pool
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
		Id:             cook.ID.Hex(),
		Name:           cook.Name,
		Email:          cook.Email,
		ProfilePicture: cook.Profile_picture,
		// Do not include the password in the response
	}

	return &proto.ProfileResponse{
		Profile: profile,
	}, nil
}

// UpdateProfile updates the cook's profile information.
func (s *CookService) UpdateProfile(ctx context.Context, req *proto.Profile) (*proto.ProfileResponse, error) {
	var cookID string
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		cookID = md.Get("cookID")[0]
	}

	objectID, err := bson.ObjectIDFromHex(cookID)
	if err != nil {
		return nil, fmt.Errorf("invalid cook ID")
	}
	update := models.Cook{
		Name:            req.GetName(),
		Email:           req.GetEmail(),
		Profile_picture: req.GetProfilePicture(),
	}

	_, err = s.CookCollection.UpdateOne(ctx, bson.M{"_id": objectID}, bson.M{"$set": update})
	if err != nil {
		return nil, err
	}

	return s.ViewProfile(ctx, &proto.Empty{})
}

func (s *CookService) VerifyCookDetails(ctx context.Context, req *proto.Profile) (*proto.ProfileResponse, error) {
	var existingCook models.Cook
	filter := bson.M{"$or": []bson.M{{"email": req.GetEmail()}, {"username": req.GetName()}}}
	err := s.CookCollection.FindOne(ctx, filter).Decode(&existingCook)
	if err == nil {
		return nil, fmt.Errorf("cook already exists with the given email or username")
	}

	cook := models.Cook{
		Name:            req.GetName(),
		Email:           req.GetEmail(),
		Password:        hashPassword(req.GetPassword()),
		Profile_picture: req.GetProfilePicture(),
	}

	result, err := s.CookCollection.InsertOne(ctx, cook)
	if err != nil {
		return nil, fmt.Errorf("failed to register cook: %v", err)
	}

	return &proto.ProfileResponse{
		Profile: &proto.Profile{
			Id:             result.InsertedID.(bson.ObjectID).Hex(),
			Name:           cook.Name,
			Email:          cook.Email,
			ProfilePicture: cook.Profile_picture,
			// Password:       cook.Password,
		},
	}, nil
}

func hashPassword(password string) string {
	return password
}
