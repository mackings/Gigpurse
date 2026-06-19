package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	delivery "gigpurse/internal/delivery/http"
	"gigpurse/internal/repository/memory"
	"gigpurse/internal/repository/mongodb"
	"gigpurse/internal/usecase"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// 1. Initialize MongoDB Connection
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	log.Printf("Connecting to MongoDB (URI length: %d)...", len(mongoURI))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOpts := options.Client().ApplyURI(mongoURI)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		log.Fatalf("failed to connect to MongoDB: %v", err)
	}

	// Ping database to ensure connectivity
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("failed to ping MongoDB: %v", err)
	}
	log.Println("Connected to MongoDB successfully!")

	dbName := os.Getenv("MONGODB_DB")
	if dbName == "" {
		dbName = "gigpurse"
	}
	db := client.Database(dbName)

	// 2. Initialize Repositories
	userRepo := mongodb.NewUserRepository(db)
	jobRepo := mongodb.NewJobRepository(db)
	chatRepo := mongodb.NewChatRepository(db)
	contractRepo := mongodb.NewContractRepository(db)
	reviewRepo := mongodb.NewReviewRepository(db)
	notifRepo := mongodb.NewNotificationRepository(db)
	resetRepo := mongodb.NewPasswordResetRepository(db)
	disputeRepo := mongodb.NewDisputeRepository(db)
	walletRepo := memory.NewWalletRepository() // Keep memory wallet repo for now

	// 3. Initialize Usecases
	userUsecase := usecase.NewUserUsecase(userRepo, resetRepo)
	jobUsecase := usecase.NewJobUsecase(jobRepo, userRepo, contractRepo, notifRepo)
	chatUsecase := usecase.NewChatUsecase(chatRepo, userRepo)
	contractUsecase := usecase.NewContractUsecase(contractRepo, jobRepo, notifRepo, userRepo)
	reviewUsecase := usecase.NewReviewUsecase(reviewRepo, jobRepo, notifRepo)
	notifUsecase := usecase.NewNotificationUsecase(notifRepo)
	disputeUsecase := usecase.NewDisputeUsecase(disputeRepo, contractRepo, notifRepo)
	dashboardUsecase := usecase.NewDashboardUsecase(jobUsecase, contractUsecase, reviewUsecase)
	adminUsecase := usecase.NewAdminUsecase(db, userRepo, jobRepo)
	walletUsecase := usecase.NewWalletUsecase(walletRepo)

	// 4. Initialize Handlers
	userHandler := delivery.NewUserHandler(userUsecase)
	jobHandler := delivery.NewJobHandler(jobUsecase)
	chatHandler := delivery.NewChatHandler(chatUsecase)
	contractHandler := delivery.NewContractHandler(contractUsecase)
	reviewHandler := delivery.NewReviewHandler(reviewUsecase)
	notifHandler := delivery.NewNotificationHandler(notifUsecase)
	disputeHandler := delivery.NewDisputeHandler(disputeUsecase)
	dashboardHandler := delivery.NewDashboardHandler(dashboardUsecase)
	adminHandler := delivery.NewAdminHandler(adminUsecase)
	walletHandler := delivery.NewWalletHandler(walletUsecase)

	// 5. Register HTTP Routes
	mux := http.NewServeMux()

	// Add a root handler for health check / Render deployment checking
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"online", "service":"gigpurse-backend"}`))
	})

	userHandler.RegisterRoutes(mux)
	jobHandler.RegisterRoutes(mux)
	chatHandler.RegisterRoutes(mux)
	contractHandler.RegisterRoutes(mux)
	reviewHandler.RegisterRoutes(mux)
	notifHandler.RegisterRoutes(mux)
	disputeHandler.RegisterRoutes(mux)
	dashboardHandler.RegisterRoutes(mux)
	adminHandler.RegisterRoutes(mux)
	walletHandler.RegisterRoutes(mux)

	// 6. Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	serverAddr := ":" + port
	log.Printf("Gigpurse backend server is starting on port %s...", port)

	if err := http.ListenAndServe(serverAddr, mux); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
