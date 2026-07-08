package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	delivery "gigpurse/internal/delivery/http"
	"gigpurse/internal/repository/mongodb"
	"gigpurse/internal/usecase"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// Loads backend/.env into the process environment if present — a no-op
	// (and not an error) when it's missing, which is the normal case in
	// production, where env vars come from the host platform instead.
	_ = godotenv.Load()

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
	hub := delivery.NewHub()
	userRepo := mongodb.NewUserRepository(db)
	jobRepo := mongodb.NewJobRepository(db)
	chatRepo := mongodb.NewChatRepository(db)
	contractRepo := mongodb.NewContractRepository(db)
	reviewRepo := mongodb.NewReviewRepository(db)
	pushSubRepo := mongodb.NewPushSubscriptionRepository(db)
	vapidPublicKey := os.Getenv("VAPID_PUBLIC_KEY")
	vapidPrivateKey := os.Getenv("VAPID_PRIVATE_KEY")
	vapidSubject := os.Getenv("VAPID_SUBJECT")
	if vapidSubject == "" {
		vapidSubject = "mailto:support@gigpurse.app"
	}
	pushSender := delivery.NewPushSender(pushSubRepo, vapidPublicKey, vapidPrivateKey, vapidSubject)
	if vapidPublicKey == "" || vapidPrivateKey == "" {
		log.Println("VAPID_PUBLIC_KEY/VAPID_PRIVATE_KEY not set — Web Push notifications are disabled (in-app realtime notifications still work).")
	}
	notifRepo := delivery.NewBroadcastingNotificationRepository(mongodb.NewNotificationRepository(db), hub, pushSender)
	resetRepo := mongodb.NewPasswordResetRepository(db)
	emailVerifyRepo := mongodb.NewEmailVerificationRepository(db)
	disputeRepo := mongodb.NewDisputeRepository(db)
	walletRepo := mongodb.NewWalletRepository(db)
	milestoneRepo := mongodb.NewMilestoneRepository(db)

	// 3. Initialize Usecases
	userUsecase := usecase.NewUserUsecaseWithVerification(userRepo, resetRepo, emailVerifyRepo)
	jobUsecase := usecase.NewJobUsecase(jobRepo, userRepo, contractRepo, notifRepo)
	chatUsecase := usecase.NewChatUsecase(chatRepo, userRepo, notifRepo)
	contractUsecase := usecase.NewContractUsecase(contractRepo, jobRepo, notifRepo, userRepo)
	reviewUsecase := usecase.NewReviewUsecase(reviewRepo, contractRepo, notifRepo)
	notifUsecase := usecase.NewNotificationUsecase(notifRepo)
	disputeUsecase := usecase.NewDisputeUsecase(disputeRepo, contractRepo, notifRepo)
	dashboardUsecase := usecase.NewDashboardUsecase(jobUsecase, contractUsecase, reviewUsecase)
	adminUsecase := usecase.NewAdminUsecase(db, userRepo, jobRepo)
	walletUsecase := usecase.NewWalletUsecase(walletRepo)
	milestoneUsecase := usecase.NewMilestoneUsecase(milestoneRepo, contractRepo, walletRepo, notifRepo)

	uploadDir := os.Getenv("MEDIA_UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "uploads"
	}
	publicURL := os.Getenv("PUBLIC_BASE_URL")

	// 4. Initialize Handlers
	userHandler := delivery.NewUserHandler(userUsecase)
	jobHandler := delivery.NewJobHandler(jobUsecase)
	chatHandler := delivery.NewChatHandler(chatUsecase, hub)
	contractHandler := delivery.NewContractHandler(contractUsecase)
	reviewHandler := delivery.NewReviewHandler(reviewUsecase)
	notifHandler := delivery.NewNotificationHandler(notifUsecase)
	disputeHandler := delivery.NewDisputeHandler(disputeUsecase)
	dashboardHandler := delivery.NewDashboardHandler(dashboardUsecase)
	adminHandler := delivery.NewAdminHandler(adminUsecase)
	walletHandler := delivery.NewWalletHandler(walletUsecase)
	milestoneHandler := delivery.NewMilestoneHandler(milestoneUsecase)
	mediaHandler := delivery.NewMediaHandler(uploadDir, publicURL)
	pushHandler := delivery.NewPushHandler(pushSubRepo, vapidPublicKey)

	// 5. Register HTTP Routes
	mux := http.NewServeMux()

	// Add a root handler for health check / Render deployment checking
	mux.HandleFunc("/{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true,"status":"success","status_code":200,"message":"service online","data":{"status":"online","service":"gigpurse-backend"}}`))
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
	milestoneHandler.RegisterRoutes(mux)
	mediaHandler.RegisterRoutes(mux)
	pushHandler.RegisterRoutes(mux)

	// Serve uploaded media files
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir(uploadDir))))

	// 6. Keep-alive: Render's free plan spins a web service down after ~15
	// minutes with no inbound traffic. Self-pinging (and pinging the
	// frontend) well under that window keeps both alive with zero external
	// dependencies. KEEPALIVE_URLS is a comma-separated list; unset/empty
	// disables this entirely (e.g. for local dev).
	startKeepAlive(os.Getenv("KEEPALIVE_URLS"), 10*time.Minute)

	// 7. Start server
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

// startKeepAlive periodically GETs each URL in a comma-separated list. It
// deliberately does not log successful pings — only real failures — so it
// doesn't spam the logs every few minutes forever.
func startKeepAlive(rawURLs string, interval time.Duration) {
	var urls []string
	for _, u := range strings.Split(rawURLs, ",") {
		if u = strings.TrimSpace(u); u != "" {
			urls = append(urls, u)
		}
	}
	if len(urls) == 0 {
		return
	}

	client := &http.Client{Timeout: 15 * time.Second}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			for _, u := range urls {
				resp, err := client.Get(u)
				if err != nil {
					log.Printf("keep-alive ping to %s failed: %v", u, err)
					continue
				}
				resp.Body.Close()
			}
		}
	}()
}
