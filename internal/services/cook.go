package services

import (
	"context"
	"fmt"
	"log"

	"github.com/JabJabHiwHiw/cook-service/internal/models"
	"github.com/JabJabHiwHiw/cook-service/proto"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/metadata"
)

type CookService struct {
	proto.UnimplementedCookServiceServer
	DBPool *pgxpool.Pool
}

func (s *CookService) VerifyCookDetails(ctx context.Context, req *proto.Profile) (*proto.ProfileResponse, error) {
	// Check if the cook already exists by email or name
	var existingCookID int64
	err := s.DBPool.QueryRow(ctx,
		"SELECT id FROM cooks WHERE email=$1 OR name=$2",
		req.GetEmail(), req.GetName()).Scan(&existingCookID)
	if err == nil {
		return nil, fmt.Errorf("cook already exists with the given email or name")
	} else if err != pgx.ErrNoRows {
		return nil, fmt.Errorf("failed to check existing cook: %v", err)
	}

	// Hash the password
	hashedPassword := hashPassword(req.GetPassword())

	// Insert the new cook
	var newCookID int64
	err = s.DBPool.QueryRow(ctx,
		`INSERT INTO cooks (name, email, password, profile_picture)
         VALUES ($1, $2, $3, $4) RETURNING id`,
		req.GetName(), req.GetEmail(), hashedPassword, req.GetProfilePicture()).Scan(&newCookID)
	if err != nil {
		return nil, fmt.Errorf("failed to register cook: %v", err)
	}

	return &proto.ProfileResponse{
		Profile: &proto.Profile{
			Id:             fmt.Sprintf("%d", newCookID),
			Name:           req.GetName(),
			Email:          req.GetEmail(),
			ProfilePicture: req.GetProfilePicture(),
		},
	}, nil
}

func (s *CookService) ViewProfile(ctx context.Context, req *proto.Empty) (*proto.ProfileResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	var cookID string
	if ok && len(md.Get("cookID")) > 0 {
		cookID = md.Get("cookID")[0]
	} else {
		return nil, fmt.Errorf("unable to retrieve cook ID from context")
	}

	var cook models.Cook
	err := s.DBPool.QueryRow(ctx,
		"SELECT id, name, email, profile_picture FROM cooks WHERE id=$1", cookID).
		Scan(&cook.ID, &cook.Name, &cook.Email, &cook.ProfilePicture)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("cook not found")
		}
		return nil, err
	}

	profile := &proto.Profile{
		Id:             fmt.Sprintf("%d", cook.ID),
		Name:           cook.Name,
		Email:          cook.Email,
		ProfilePicture: cook.ProfilePicture,
	}

	return &proto.ProfileResponse{
		Profile: profile,
	}, nil
}

func (s *CookService) UpdateProfile(ctx context.Context, req *proto.Profile) (*proto.ProfileResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	var cookID string
	if ok && len(md.Get("cookID")) > 0 {
		cookID = md.Get("cookID")[0]
	} else {
		return nil, fmt.Errorf("invalid cook ID")
	}

	// Update the cook's profile
	_, err := s.DBPool.Exec(ctx,
		`UPDATE cooks SET name=$1, email=$2, profile_picture=$3 WHERE id=$4`,
		req.GetName(), req.GetEmail(), req.GetProfilePicture(), cookID)
	if err != nil {
		return nil, err
	}

	return s.ViewProfile(ctx, &proto.Empty{})
}

func hashPassword(password string) string {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}
	return string(bytes)
}
