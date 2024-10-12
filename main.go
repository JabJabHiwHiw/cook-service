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
	"google.golang.org/grpc"
)

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

	db, err := sql.Open("postgres", "your-connection-string-here")
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
