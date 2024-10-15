package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"

	"github.com/JabJabHiwHiw/cook-service/internal/services"
	"github.com/JabJabHiwHiw/cook-service/proto"
	"github.com/jackc/pgx/v4/pgxpool"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
)

func InitializeDB(db *sql.DB) error {
	// Query to create the `cooks` table
	cooksTableQuery := `
    CREATE TABLE IF NOT EXISTS cooks (
        id UUID PRIMARY KEY,
        name VARCHAR(255) UNIQUE NOT NULL,
        email VARCHAR(255) UNIQUE NOT NULL,
        password VARCHAR(255) NOT NULL,
        profile_picture TEXT
    );`

	// Execute the query to create the `cooks` table
	if _, err := db.Exec(cooksTableQuery); err != nil {
		return err
	}

	// Query to create the `favorite_menus` table
	favoriteMenusTableQuery := `
    CREATE TABLE IF NOT EXISTS favorite_menus (
        id UUID PRIMARY KEY,
        user_id UUID NOT NULL REFERENCES cooks(id),
        menu_id UUID NOT NULL,
        added_at TIMESTAMP NOT NULL DEFAULT NOW(),
        UNIQUE(user_id, menu_id)  -- Ensure that each user-menu pair is unique
    );`

	// Execute the query to create the `favorite_menus` table
	if _, err := db.Exec(favoriteMenusTableQuery); err != nil {
		return err
	}

	return nil
}

func main() {
	// Set up PostgreSQL connection
	connString := "postgres://user:pass@localhost:5432/cook_service"
	dbPool, err := pgxpool.Connect(context.Background(), connString)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer dbPool.Close()

	fmt.Println("Connected to PostgreSQL")

	cookService := services.CookService{
		DBPool: dbPool,
	}

	grpcServer := grpc.NewServer()
	proto.RegisterCookServiceServer(grpcServer, &cookService)

	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Server started on port :8080")

	connStr := "postgres://user:pass@localhost:5432/cook_service?sslmode=disable"

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Error connecting to the database: ", err)
	}

	err = InitializeDB(db)
	if err != nil {
		log.Fatal("Error initializing database: ", err)
	}

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatal(err)
	}
}
