package services

import (
	"context"
	"fmt"
	"log"

	"github.com/JabJabHiwHiw/cook-service/internal/models"
	"github.com/JabJabHiwHiw/cook-service/proto"
	"github.com/google/uuid"
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
	var existingCookID uuid.UUID
	err := s.DBPool.QueryRow(ctx,
		"SELECT id FROM cooks WHERE email=$1 OR name=$2",
		req.GetEmail(), req.GetName()).Scan(&existingCookID)
	if err == nil {
		return nil, fmt.Errorf("cook already exists with the given email or name")
	} else if err != pgx.ErrNoRows {
		return nil, fmt.Errorf("failed to check existing cook (check): %v", err)
	}

	// Hash the cid
	hashedClerkId := hashClerkId(req.GetClerkId())

	// Insert the new cook
	newCookID := uuid.New() // Generate a new UUID
	err = s.DBPool.QueryRow(ctx,
		`INSERT INTO cooks (id, name, email, clerk_id, profile_picture)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`, // Include id in the insert
		newCookID, req.GetName(), req.GetEmail(), hashedClerkId, req.GetProfilePicture()).Scan(&newCookID)
	if err != nil {
		return nil, fmt.Errorf("failed to register cook (insert): %v", err)
	}

	return &proto.ProfileResponse{
		Profile: &proto.Profile{
			Id:             newCookID.String(),
			Name:           req.GetName(),
			Email:          req.GetEmail(),
			ProfilePicture: req.GetProfilePicture(),
		},
	}, nil
}

func (s *CookService) ViewProfile(ctx context.Context, req *proto.Empty) (*proto.ProfileResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	var cookIDStr string
	if ok && len(md.Get("cookID")) > 0 {
		cookIDStr = md.Get("cookID")[0]
	} else {
		return nil, fmt.Errorf("unable to retrieve cook ID from context")
	}

	cookID, err := uuid.Parse(cookIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid cook ID format")
	}

	var cook models.Cook
	err = s.DBPool.QueryRow(ctx,
		"SELECT id, name, email, profile_picture FROM cooks WHERE id=$1", cookID).
		Scan(&cook.ID, &cook.Name, &cook.Email, &cook.ProfilePicture)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("cook not found")
		}
		return nil, err
	}

	profile := &proto.Profile{
		Id:             cook.ID.String(), // Ensure the UUID is converted to string
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
	var cookIDStr string
	if ok && len(md.Get("cookID")) > 0 {
		cookIDStr = md.Get("cookID")[0]
	} else {
		return nil, fmt.Errorf("invalid cook ID")
	}

	cookID, err := uuid.Parse(cookIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid cook ID format")
	}

	// Update the cook's profile
	_, err = s.DBPool.Exec(ctx,
		`UPDATE cooks SET name=$1, email=$2, profile_picture=$3 WHERE id=$4`,
		req.GetName(), req.GetEmail(), req.GetProfilePicture(), cookID)
	if err != nil {
		return nil, err
	}

	return s.ViewProfile(ctx, &proto.Empty{})
}

func hashClerkId(clerkId string) string {
	bytes, err := bcrypt.GenerateFromPassword([]byte(clerkId), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash clerkId: %v", err)
	}
	return string(bytes)
}

func (s *CookService) GetFavoriteMenus(ctx context.Context, req *proto.Empty) (*proto.FavoriteMenusResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	var cookIDStr string
	if ok && len(md.Get("cookID")) > 0 {
		cookIDStr = md.Get("cookID")[0]
	} else {
		return nil, fmt.Errorf("invalid cook ID")
	}

	cookID, err := uuid.Parse(cookIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid cook ID format")
	}

	rows, err := s.DBPool.Query(ctx,
		`SELECT id, user_id, menu_id FROM favorite_menus WHERE user_id=$1`, cookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var favoriteMenus []*proto.FavoriteMenu
	for rows.Next() {
		var favoriteMenu models.FavoriteMenu
		err = rows.Scan(&favoriteMenu.ID, &favoriteMenu.UserID, &favoriteMenu.MenuID)
		if err != nil {
			return nil, err
		}

		favoriteMenus = append(favoriteMenus, &proto.FavoriteMenu{
			Id:     favoriteMenu.ID.String(),
			MenuId: favoriteMenu.MenuID.String(),
		})
	}

	return &proto.FavoriteMenusResponse{
		FavoriteMenus: favoriteMenus,
	}, nil
}

func (s *CookService) AddFavoriteMenu(ctx context.Context, req *proto.FavoriteMenu) (*proto.FavoriteMenusResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	var cookIDStr string
	if ok && len(md.Get("cookID")) > 0 {
		cookIDStr = md.Get("cookID")[0]
	} else {
		return nil, fmt.Errorf("invalid cook ID")
	}

	cookID, err := uuid.Parse(cookIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid cook ID format")
	}

	// Check if the menu exists
	// var menuID uuid.UUID
	// err = s.DBPool.QueryRow(ctx,
	// 	"SELECT id FROM menus WHERE id=$1", req.GetMenuId()).Scan(&menuID)
	// if err != nil {
	// 	return nil, fmt.Errorf("menu not found")
	// }

	// Insert the favorite menu
	favoriteMenuID := uuid.New()
	err = s.DBPool.QueryRow(ctx,
		`INSERT INTO favorite_menus (id, user_id, menu_id)
		 VALUES ($1, $2, $3) RETURNING id`,
		favoriteMenuID, cookID, req.GetMenuId()).Scan(&favoriteMenuID)
	if err != nil {
		return nil, fmt.Errorf("failed to add favorite menu")
	}

	return s.GetFavoriteMenus(ctx, &proto.Empty{})
}

func (s *CookService) RemoveFavoriteMenu(ctx context.Context, req *proto.FavoriteMenu) (*proto.FavoriteMenusResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	var cookIDStr string
	if ok && len(md.Get("cookID")) > 0 {
		cookIDStr = md.Get("cookID")[0]
	} else {
		return nil, fmt.Errorf("invalid cook ID")
	}

	cookID, err := uuid.Parse(cookIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid cook ID format")
	}

	// Delete the favorite menu
	_, err = s.DBPool.Exec(ctx,
		"DELETE FROM favorite_menus WHERE user_id=$1 AND menu_id=$2",
		cookID, req.GetMenuId())
	if err != nil {
		return nil, fmt.Errorf("failed to remove favorite menu")
	}

	return s.GetFavoriteMenus(ctx, &proto.Empty{})
}
