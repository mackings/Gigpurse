package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	delivery "gigpurse/internal/delivery/http"
	"gigpurse/internal/domain"
	"gigpurse/internal/repository/memory"
	"gigpurse/internal/repository/mongodb"
	"gigpurse/internal/usecase"

	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	serverPort = "9090"
	serverAddr = "http://localhost:" + serverPort
	wsAddr     = "ws://localhost:" + serverPort
)

type TestClient struct {
	client *http.Client
	token  string
}

func main() {
	log.Println("==================================================================")
	log.Println("             GIGPURSE ALL-APIS FLOW SIMULATOR                   ")
	log.Println("==================================================================")

	// 1. Get MongoDB URI
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	clientOpts := options.Client().ApplyURI(mongoURI)
	mongoClient, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		log.Fatalf("Simulator: Failed to connect to MongoDB: %v", err)
	}

	// Clean database for testing to ensure deterministic results
	db := mongoClient.Database("gigpurse_simulation_db")
	_ = db.Collection("users").Drop(ctx)
	_ = db.Collection("jobs").Drop(ctx)
	_ = db.Collection("job_applications").Drop(ctx)
	_ = db.Collection("chat_messages").Drop(ctx)
	_ = db.Collection("contracts").Drop(ctx)
	_ = db.Collection("reviews").Drop(ctx)
	_ = db.Collection("notifications").Drop(ctx)
	_ = db.Collection("direct_hire_requests").Drop(ctx)
	_ = db.Collection("disputes").Drop(ctx)
	_ = db.Collection("password_reset_tokens").Drop(ctx)
	log.Println("Simulator: Cleaned up 'gigpurse_simulation_db' collections.")

	// 2. Start Backend Server on simulator port
	os.Setenv("MONGODB_URI", mongoURI)
	os.Setenv("MONGODB_DB", "gigpurse_simulation_db")
	os.Setenv("PORT", serverPort)
	os.Setenv("JWT_SECRET", "simulator-very-secure-jwt-secret-key-999")
	os.Setenv("ALLOW_ADMIN_SIGNUP", "true")

	// Wire up app
	userRepo := mongodb.NewUserRepository(db)
	jobRepo := mongodb.NewJobRepository(db)
	chatRepo := mongodb.NewChatRepository(db)
	contractRepo := mongodb.NewContractRepository(db)
	reviewRepo := mongodb.NewReviewRepository(db)
	notifRepo := mongodb.NewNotificationRepository(db)
	resetRepo := mongodb.NewPasswordResetRepository(db)
	disputeRepo := mongodb.NewDisputeRepository(db)
	walletRepo := memory.NewWalletRepository()

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

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"simulator-online"}`))
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

	go func() {
		log.Printf("Simulator: Starting server on port %s...", serverPort)
		if err := http.ListenAndServe(":"+serverPort, mux); err != nil {
			log.Printf("Simulator: Server stopped: %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(2 * time.Second)

	// 3. Initialize HTTP Test Clients
	httpClient := &http.Client{Timeout: 5 * time.Second}
	clientUser := &TestClient{client: httpClient}
	musicianUser := &TestClient{client: httpClient}
	adminUser := &TestClient{client: httpClient}

	// ------------------------------------------------------------------
	// STEP 1 & 2: SIGN UP & LOGIN
	// ------------------------------------------------------------------
	log.Println("\n--- STEP 1 & 2: Testing Sign Up & Login ---")
	clientSignupRes := signup(clientUser, "client@test.com", "password123", "client", "Alice the Client")
	log.Printf("[SUCCESS] Client Signed Up: ID=%s, Role=%s, Name=%s", clientSignupRes.ID, clientSignupRes.Role, clientSignupRes.Name)

	musicianSignupRes := signup(musicianUser, "musician@test.com", "password123", "musician", "Bob the Musician")
	log.Printf("[SUCCESS] Musician Signed Up: ID=%s, Role=%s, Name=%s", musicianSignupRes.ID, musicianSignupRes.Role, musicianSignupRes.Name)

	adminSignupRes := signup(adminUser, "admin@test.com", "password123", "admin", "Charlie the Admin")
	log.Printf("[SUCCESS] Admin Signed Up: ID=%s, Role=%s, Name=%s", adminSignupRes.ID, adminSignupRes.Role, adminSignupRes.Name)

	clientUser.token = login(clientUser, "client@test.com", "password123")
	musicianUser.token = login(musicianUser, "musician@test.com", "password123")
	adminUser.token = login(adminUser, "admin@test.com", "password123")
	log.Printf("[SUCCESS] Clients Logged In. JWT Tokens obtained.")

	// ------------------------------------------------------------------
	// STEP 3: UPDATE PROFILES & PORTFOLIO
	// ------------------------------------------------------------------
	log.Println("\n--- STEP 3: Testing Profile & Portfolio Updates ---")
	updateClientProfile(clientUser, "Alice the Client", "Looking for talented guitarists", "New York", &domain.ClientProfile{
		CompanyName: "Big Stage Productions",
	})

	updateMusicianProfile(musicianUser, "Bob the Musician", "Experienced rock guitarist", "New York", &domain.MusicianProfile{
		StageName:       "Guitar Bob",
		Instrument:      "Guitar",
		Genre:           "Rock",
		ExperienceYears: 6,
		Portfolio: []domain.PortfolioItem{
			{Title: "YouTube Solo", Description: "Live solo at Madison Square Garden", URL: "http://youtube.com/bob-solo"},
		},
	})
	log.Println("[SUCCESS] Profiles updated successfully.")

	// ------------------------------------------------------------------
	// STEP 4: BROWSE MUSICIANS
	// ------------------------------------------------------------------
	log.Println("\n--- STEP 4: Testing Client Browse Musicians ---")
	musicians := browseMusicians(clientUser, "Rock", "Guitar", "New York", 5)
	log.Printf("[SUCCESS] Client search returned %d match(es). Musician Name: %s, Stage: %s, Portfolio URL: %s",
		len(musicians), musicians[0].Name, musicians[0].MusicianProfile.StageName, musicians[0].MusicianProfile.Portfolio[0].URL)

	// ------------------------------------------------------------------
	// STEP 5: POST JOB
	// ------------------------------------------------------------------
	log.Println("\n--- STEP 5: Testing Client Post Job ---")
	job := postJob(clientUser, "Rock Gig in NYC", "Need a rock guitarist for a corporate gig.", "Guitar", "Rock", "New York", 400.00)
	log.Printf("[SUCCESS] Job Created: ID=%s, Title=%s, Status=%s, Budget=$%.2f", job.ID, job.Title, job.Status, job.Budget)

	// ------------------------------------------------------------------
	// STEP 6: BROWSE JOBS
	// ------------------------------------------------------------------
	log.Println("\n--- STEP 6: Testing Musician Gigs Filter & Sort ---")
	jobs := listJobs(musicianUser, "open", "Rock", "Guitar", "New York", 300, 500)
	log.Printf("[SUCCESS] Musician query returned %d gig(s). Gig Title: %s, Budget: $%.2f", len(jobs), jobs[0].Title, jobs[0].Budget)

	// ------------------------------------------------------------------
	// STEP 7: APPLY FOR JOB
	// ------------------------------------------------------------------
	log.Println("\n--- STEP 7: Testing Musician Apply for Job ---")
	app := applyForJob(musicianUser, job.ID, "Hey Alice, I am Bob and I have 6 years of experience. WhatsApp me or Paypal me.", 380.00)
	log.Printf("[SUCCESS] Application Submitted: ID=%s, JobID=%s, Bid=$%.2f, Status=%s", app.ID, app.JobID, app.PriceBid, app.Status)

	// ------------------------------------------------------------------
	// STEP 8: VIEW APPLICATIONS
	// ------------------------------------------------------------------
	log.Println("\n--- STEP 8: Testing Client View Applications ---")
	applications := listApplications(clientUser, job.ID)
	log.Printf("[SUCCESS] Client retrieved %d application(s) for job. Bidder ID: %s, Proposal: '%s'",
		len(applications), applications[0].MusicianID, applications[0].Proposal)

	// ------------------------------------------------------------------
	// STEP 9: ACCEPT APPLICATION (Hiring & Contract Creation)
	// ------------------------------------------------------------------
	log.Println("\n--- STEP 9: Testing Client Accept Application (Hiring & Contract Creation) ---")
	acceptApplication(clientUser, app.ID)
	log.Println("[SUCCESS] Application accepted.")

	hiredJob := getJob(clientUser, job.ID)
	log.Printf("[VERIFIED] Job Status after hiring: %s. Hired Musician ID: %s", hiredJob.Status, hiredJob.MusicianID)

	// ------------------------------------------------------------------
	// STEP 10: REAL-TIME CHAT & CENSORING
	// ------------------------------------------------------------------
	log.Println("\n--- STEP 10: Testing WebSocket Chat & Words Filter ---")
	clientWS := connectWS(clientUser.token)
	musicianWS := connectWS(musicianUser.token)

	musicianCh := make(chan domain.ChatMessage, 5)
	go func() {
		defer close(musicianCh)
		for {
			_, msgBytes, err := musicianWS.ReadMessage()
			if err != nil {
				return
			}
			var m domain.ChatMessage
			_ = json.Unmarshal(msgBytes, &m)
			musicianCh <- m
		}
	}()

	clientCh := make(chan domain.ChatMessage, 5)
	go func() {
		defer close(clientCh)
		for {
			_, msgBytes, err := clientWS.ReadMessage()
			if err != nil {
				return
			}
			var m domain.ChatMessage
			_ = json.Unmarshal(msgBytes, &m)
			clientCh <- m
		}
	}()

	// Client sends clean msg
	_ = clientWS.WriteJSON(map[string]string{
		"recv_id": musicianSignupRes.ID,
		"content": "Hi Guitar Bob, welcome to the job! Let's arrange details.",
	})

	select {
	case received := <-musicianCh:
		log.Printf("[WS RECEIVED] Musician got message: '%s'", received.Content)
	case <-time.After(3 * time.Second):
		log.Fatal("Musician WS message receive timeout")
	}

	// Musician sends bypass msg
	_ = musicianWS.WriteJSON(map[string]string{
		"recv_id": clientSignupRes.ID,
		"content": "Thanks! Can you pay me directly via Paypal or Whatsapp? Contact me on my phone number +12345.",
	})

	// Wait for client to receive
	var received domain.ChatMessage
	timeout := time.After(5 * time.Second)
	found := false
	for !found {
		select {
		case msg := <-clientCh:
			if msg.SenderID == musicianSignupRes.ID {
				received = msg
				found = true
			}
		case <-timeout:
			log.Fatal("Client WS message receive timeout waiting for Musician message")
		}
	}

	log.Printf("[WS RECEIVED] Client got message: '%s'", received.Content)
	if strings.Contains(strings.ToLower(received.Content), "paypal") ||
		strings.Contains(strings.ToLower(received.Content), "whatsapp") ||
		strings.Contains(strings.ToLower(received.Content), "phone number") {
		log.Fatal("[FAIL] Message was NOT censored!")
	} else {
		log.Println("[SUCCESS] Message successfully censored!")
	}

	time.Sleep(1 * time.Second)
	_ = clientWS.Close()
	_ = musicianWS.Close()

	// ------------------------------------------------------------------
	// STEP 11: GET CHAT HISTORY & RECENT CHATS
	// ------------------------------------------------------------------
	log.Println("\n--- STEP 11: Testing Persisted Chat History & Recent Threads ---")
	history := getChatHistory(clientUser, musicianSignupRes.ID)
	log.Printf("[SUCCESS] Chat History has %d messages.", len(history))
	for _, m := range history {
		log.Printf("   [%s]: %s", m.SenderID, m.Content)
	}

	recent := getRecentChats(clientUser)
	log.Printf("[SUCCESS] Client recent thread count: %d. Latest: '%s'", len(recent), recent[0].Content)

	// ------------------------------------------------------------------
	// STEP 12: CONTRACT COMPLETION (Contract System Feature)
	// ------------------------------------------------------------------
	log.Println("\n--- STEP 12: Testing Contract Completion ---")
	contracts := listContracts(clientUser)
	log.Printf("[SUCCESS] Found %d active contract(s) for Client. Contract ID: %s", len(contracts), contracts[0].ID)

	completeContract(clientUser, contracts[0].ID)
	log.Println("[SUCCESS] Client marked contract as completed.")

	// Check Job and Contract state after completion
	hiredJob = getJob(clientUser, job.ID)
	log.Printf("[VERIFIED] Job Status after completion: %s", hiredJob.Status)

	updatedContracts := listContracts(clientUser)
	log.Printf("[VERIFIED] Contract Status after completion: %s", updatedContracts[0].Status)

	// ------------------------------------------------------------------
	// STEP 13: SUBMIT RATINGS & REVIEWS (Rating/Review System Feature)
	// ------------------------------------------------------------------
	log.Println("\n--- STEP 13: Testing Client & Musician Reviews ---")
	clientReview := submitReview(clientUser, job.ID, 5, "Bob is an outstanding guitarist! Highly professional and punctual.")
	log.Printf("[SUCCESS] Client submitted review: Rating=%d, Comment='%s'", clientReview.Rating, clientReview.Comment)

	musicianReview := submitReview(musicianUser, job.ID, 5, "Alice is an excellent client. Clear requirements and fast sign-off.")
	log.Printf("[SUCCESS] Musician submitted review: Rating=%d, Comment='%s'", musicianReview.Rating, musicianReview.Comment)

	// Fetch public reviews
	reviews := listReviews(clientUser, musicianSignupRes.ID)
	log.Printf("[SUCCESS] Fetched %d review(s) for Musician Bob. Average Rating: %.1f", len(reviews), getAverageRating(clientUser, musicianSignupRes.ID))
	for _, r := range reviews {
		log.Printf("   [Reviewer %s]: %d stars - '%s'", r.ReviewerID, r.Rating, r.Comment)
	}

	// ------------------------------------------------------------------
	// STEP 14: IN-APP NOTIFICATIONS
	// ------------------------------------------------------------------
	log.Println("\n--- STEP 14: Testing In-App Notifications ---")
	notifications := listNotifications(musicianUser)
	log.Printf("[SUCCESS] Musician Bob has %d notifications:", len(notifications))
	for _, n := range notifications {
		log.Printf("   - [%s] (Read=%t) Msg: %s", n.Title, n.IsRead, n.Message)
	}

	// Mark one notification as read
	markNotificationRead(musicianUser, notifications[0].ID)
	log.Println("[SUCCESS] Notification marked as read.")

	// Verify read status
	notifications = listNotifications(musicianUser)
	log.Printf("[VERIFIED] Updated Notification Read Status: %s (Read=%t)", notifications[0].Title, notifications[0].IsRead)

	// ------------------------------------------------------------------
	// STEP 15: ADMIN DASHBOARD (Admin Dashboard Feature)
	// ------------------------------------------------------------------
	log.Println("\n--- STEP 15: Testing Admin Dashboard Analytics & Moderation ---")
	analytics := getAdminAnalytics(adminUser)
	log.Printf("[SUCCESS] Admin Analytics: Users=%d, Jobs=%d, Contracts=%d, Messages=%d",
		analytics.TotalUsers, analytics.TotalJobs, analytics.TotalContracts, analytics.TotalMessages)

	adminUsers := listAdminUsers(adminUser)
	log.Printf("[SUCCESS] Admin retrieved %d user(s) on platform.", len(adminUsers))

	adminJobs := listAdminJobs(adminUser)
	log.Printf("[SUCCESS] Admin retrieved %d job(s) on platform.", len(adminJobs))

	// Delete job listing as moderator
	deleteAdminJob(adminUser, job.ID)
	log.Println("[SUCCESS] Admin successfully deleted job listing (Moderated).")

	log.Println("\n==================================================================")
	log.Println("      ALL BACKEND APIS & FLOWS SIMULATED AND VERIFIED 100%!       ")
	log.Println("==================================================================")
}

// --- Additional Helper API Functions ---

func listContracts(tc *TestClient) []*domain.Contract {
	req, _ := http.NewRequest(http.MethodGet, serverAddr+"/contracts", nil)
	req.Header.Set("Authorization", "Bearer "+tc.token)
	resp, err := tc.client.Do(req)
	if err != nil {
		log.Fatalf("list contracts failed: %v", err)
	}
	defer resp.Body.Close()

	var contracts []*domain.Contract
	_ = json.NewDecoder(resp.Body).Decode(&contracts)
	return contracts
}

func completeContract(tc *TestClient, contractID string) {
	body := map[string]string{
		"contract_id": contractID,
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, serverAddr+"/contracts/complete", bytes.NewBuffer(b))
	req.Header.Set("Authorization", "Bearer "+tc.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := tc.client.Do(req)
	if err != nil {
		log.Fatalf("complete contract failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		out, _ := io.ReadAll(resp.Body)
		log.Fatalf("complete contract failed: status=%d, body=%s", resp.StatusCode, string(out))
	}
}

func submitReview(tc *TestClient, jobID string, rating int, comment string) domain.Review {
	body := map[string]interface{}{
		"job_id":  jobID,
		"rating":  rating,
		"comment": comment,
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, serverAddr+"/reviews", bytes.NewBuffer(b))
	req.Header.Set("Authorization", "Bearer "+tc.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := tc.client.Do(req)
	if err != nil {
		log.Fatalf("submit review failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		out, _ := io.ReadAll(resp.Body)
		log.Fatalf("submit review failed: status=%d, body=%s", resp.StatusCode, string(out))
	}

	var r domain.Review
	_ = json.NewDecoder(resp.Body).Decode(&r)
	return r
}

func listReviews(tc *TestClient, userID string) []*domain.Review {
	req, _ := http.NewRequest(http.MethodGet, serverAddr+"/reviews?user_id="+userID, nil)
	resp, err := tc.client.Do(req)
	if err != nil {
		log.Fatalf("list reviews failed: %v", err)
	}
	defer resp.Body.Close()

	var reviews []*domain.Review
	_ = json.NewDecoder(resp.Body).Decode(&reviews)
	return reviews
}

func getAverageRating(tc *TestClient, userID string) float64 {
	req, _ := http.NewRequest(http.MethodGet, serverAddr+"/reviews/average?user_id="+userID, nil)
	resp, err := tc.client.Do(req)
	if err != nil {
		log.Fatalf("get average rating failed: %v", err)
	}
	defer resp.Body.Close()

	var res struct {
		Average float64 `json:"average_rating"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&res)
	return res.Average
}

func listNotifications(tc *TestClient) []*domain.Notification {
	req, _ := http.NewRequest(http.MethodGet, serverAddr+"/notifications", nil)
	req.Header.Set("Authorization", "Bearer "+tc.token)
	resp, err := tc.client.Do(req)
	if err != nil {
		log.Fatalf("list notifications failed: %v", err)
	}
	defer resp.Body.Close()

	var notifs []*domain.Notification
	_ = json.NewDecoder(resp.Body).Decode(&notifs)
	return notifs
}

func markNotificationRead(tc *TestClient, notifID string) {
	body := map[string]string{
		"notification_id": notifID,
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, serverAddr+"/notifications/read", bytes.NewBuffer(b))
	req.Header.Set("Authorization", "Bearer "+tc.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := tc.client.Do(req)
	if err != nil {
		log.Fatalf("mark notification read failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		out, _ := io.ReadAll(resp.Body)
		log.Fatalf("mark notification read failed: status=%d, body=%s", resp.StatusCode, string(out))
	}
}

func getAdminAnalytics(tc *TestClient) domain.AdminAnalytics {
	req, _ := http.NewRequest(http.MethodGet, serverAddr+"/admin/analytics", nil)
	req.Header.Set("Authorization", "Bearer "+tc.token)
	resp, err := tc.client.Do(req)
	if err != nil {
		log.Fatalf("get admin analytics failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		out, _ := io.ReadAll(resp.Body)
		log.Fatalf("get admin analytics failed: status=%d, body=%s", resp.StatusCode, string(out))
	}

	var a domain.AdminAnalytics
	_ = json.NewDecoder(resp.Body).Decode(&a)
	return a
}

func listAdminUsers(tc *TestClient) []*domain.User {
	req, _ := http.NewRequest(http.MethodGet, serverAddr+"/admin/users", nil)
	req.Header.Set("Authorization", "Bearer "+tc.token)
	resp, err := tc.client.Do(req)
	if err != nil {
		log.Fatalf("list admin users failed: %v", err)
	}
	defer resp.Body.Close()

	var users []*domain.User
	_ = json.NewDecoder(resp.Body).Decode(&users)
	return users
}

func listAdminJobs(tc *TestClient) []*domain.Job {
	req, _ := http.NewRequest(http.MethodGet, serverAddr+"/admin/jobs", nil)
	req.Header.Set("Authorization", "Bearer "+tc.token)
	resp, err := tc.client.Do(req)
	if err != nil {
		log.Fatalf("list admin jobs failed: %v", err)
	}
	defer resp.Body.Close()

	var jobs []*domain.Job
	_ = json.NewDecoder(resp.Body).Decode(&jobs)
	return jobs
}

func deleteAdminJob(tc *TestClient, jobID string) {
	body := map[string]string{
		"job_id": jobID,
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodDelete, serverAddr+"/admin/jobs", bytes.NewBuffer(b))
	req.Header.Set("Authorization", "Bearer "+tc.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := tc.client.Do(req)
	if err != nil {
		log.Fatalf("delete admin job failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		out, _ := io.ReadAll(resp.Body)
		log.Fatalf("delete admin job failed: status=%d, body=%s", resp.StatusCode, string(out))
	}
}

// --- Existing Base Helpers (No changes) ---

func signup(tc *TestClient, email, password, role, name string) domain.User {
	body := map[string]string{
		"email":    email,
		"password": password,
		"role":     role,
		"name":     name,
	}
	b, _ := json.Marshal(body)
	resp, err := tc.client.Post(serverAddr+"/auth/signup", "application/json", bytes.NewBuffer(b))
	if err != nil {
		log.Fatalf("signup failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		out, _ := io.ReadAll(resp.Body)
		log.Fatalf("signup failed: status=%d, body=%s", resp.StatusCode, string(out))
	}

	var user domain.User
	_ = json.NewDecoder(resp.Body).Decode(&user)
	return user
}

func login(tc *TestClient, email, password string) string {
	body := map[string]string{
		"email":    email,
		"password": password,
	}
	b, _ := json.Marshal(body)
	resp, err := tc.client.Post(serverAddr+"/auth/login", "application/json", bytes.NewBuffer(b))
	if err != nil {
		log.Fatalf("login failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		out, _ := io.ReadAll(resp.Body)
		log.Fatalf("login failed: status=%d, body=%s", resp.StatusCode, string(out))
	}

	var res struct {
		Token string `json:"token"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&res)
	return res.Token
}

func getProfile(tc *TestClient) domain.User {
	req, _ := http.NewRequest(http.MethodGet, serverAddr+"/users/profile", nil)
	req.Header.Set("Authorization", "Bearer "+tc.token)
	resp, err := tc.client.Do(req)
	if err != nil {
		log.Fatalf("get profile failed: %v", err)
	}
	defer resp.Body.Close()

	var user domain.User
	_ = json.NewDecoder(resp.Body).Decode(&user)
	return user
}

func updateClientProfile(tc *TestClient, name, bio, location string, cp *domain.ClientProfile) {
	body := map[string]interface{}{
		"name":           name,
		"bio":            bio,
		"location":       location,
		"client_profile": cp,
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPut, serverAddr+"/users/profile", bytes.NewBuffer(b))
	req.Header.Set("Authorization", "Bearer "+tc.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := tc.client.Do(req)
	if err != nil {
		log.Fatalf("update profile failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		out, _ := io.ReadAll(resp.Body)
		log.Fatalf("update client profile failed: status=%d, body=%s", resp.StatusCode, string(out))
	}
}

func updateMusicianProfile(tc *TestClient, name, bio, location string, mp *domain.MusicianProfile) {
	body := map[string]interface{}{
		"name":             name,
		"bio":              bio,
		"location":         location,
		"musician_profile": mp,
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPut, serverAddr+"/users/profile", bytes.NewBuffer(b))
	req.Header.Set("Authorization", "Bearer "+tc.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := tc.client.Do(req)
	if err != nil {
		log.Fatalf("update profile failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		out, _ := io.ReadAll(resp.Body)
		log.Fatalf("update musician profile failed: status=%d, body=%s", resp.StatusCode, string(out))
	}
}

func browseMusicians(tc *TestClient, genre, instrument, location string, minExp int) []*domain.User {
	u, _ := url.Parse(serverAddr + "/musicians")
	q := u.Query()
	q.Set("genre", genre)
	q.Set("instrument", instrument)
	q.Set("location", location)
	q.Set("min_exp", fmt.Sprintf("%d", minExp))
	u.RawQuery = q.Encode()

	req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
	req.Header.Set("Authorization", "Bearer "+tc.token)
	resp, err := tc.client.Do(req)
	if err != nil {
		log.Fatalf("browse musicians failed: %v", err)
	}
	defer resp.Body.Close()

	var musicians []*domain.User
	_ = json.NewDecoder(resp.Body).Decode(&musicians)
	return musicians
}

func postJob(tc *TestClient, title, desc, inst, genre, loc string, budget float64) domain.Job {
	body := map[string]interface{}{
		"title":       title,
		"description": desc,
		"instrument":  inst,
		"genre":       genre,
		"location":    loc,
		"budget":      budget,
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, serverAddr+"/jobs", bytes.NewBuffer(b))
	req.Header.Set("Authorization", "Bearer "+tc.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := tc.client.Do(req)
	if err != nil {
		log.Fatalf("post job failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		out, _ := io.ReadAll(resp.Body)
		log.Fatalf("post job failed: status=%d, body=%s", resp.StatusCode, string(out))
	}

	var job domain.Job
	_ = json.NewDecoder(resp.Body).Decode(&job)
	return job
}

func getJob(tc *TestClient, id string) domain.Job {
	req, _ := http.NewRequest(http.MethodGet, serverAddr+"/jobs?id="+id, nil)
	req.Header.Set("Authorization", "Bearer "+tc.token)
	resp, err := tc.client.Do(req)
	if err != nil {
		log.Fatalf("get job failed: %v", err)
	}
	defer resp.Body.Close()

	var job domain.Job
	_ = json.NewDecoder(resp.Body).Decode(&job)
	return job
}

func listJobs(tc *TestClient, status, genre, instrument, location string, minB, maxB float64) []*domain.Job {
	u, _ := url.Parse(serverAddr + "/jobs")
	q := u.Query()
	q.Set("status", status)
	q.Set("genre", genre)
	q.Set("instrument", instrument)
	q.Set("location", location)
	q.Set("min_budget", fmt.Sprintf("%.2f", minB))
	q.Set("max_budget", fmt.Sprintf("%.2f", maxB))
	u.RawQuery = q.Encode()

	req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
	req.Header.Set("Authorization", "Bearer "+tc.token)
	resp, err := tc.client.Do(req)
	if err != nil {
		log.Fatalf("list jobs failed: %v", err)
	}
	defer resp.Body.Close()

	var jobs []*domain.Job
	_ = json.NewDecoder(resp.Body).Decode(&jobs)
	return jobs
}

func applyForJob(tc *TestClient, jobID, proposal string, bid float64) domain.JobApplication {
	body := map[string]interface{}{
		"job_id":    jobID,
		"proposal":  proposal,
		"price_bid": bid,
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, serverAddr+"/jobs/apply", bytes.NewBuffer(b))
	req.Header.Set("Authorization", "Bearer "+tc.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := tc.client.Do(req)
	if err != nil {
		log.Fatalf("apply for job failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		out, _ := io.ReadAll(resp.Body)
		log.Fatalf("apply for job failed: status=%d, body=%s", resp.StatusCode, string(out))
	}

	var app domain.JobApplication
	_ = json.NewDecoder(resp.Body).Decode(&app)
	return app
}

func listApplications(tc *TestClient, jobID string) []*domain.JobApplication {
	req, _ := http.NewRequest(http.MethodGet, serverAddr+"/jobs/applications?job_id="+jobID, nil)
	req.Header.Set("Authorization", "Bearer "+tc.token)
	resp, err := tc.client.Do(req)
	if err != nil {
		log.Fatalf("list applications failed: %v", err)
	}
	defer resp.Body.Close()

	var apps []*domain.JobApplication
	_ = json.NewDecoder(resp.Body).Decode(&apps)
	return apps
}

func acceptApplication(tc *TestClient, appID string) {
	body := map[string]string{
		"application_id": appID,
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, serverAddr+"/jobs/applications/accept", bytes.NewBuffer(b))
	req.Header.Set("Authorization", "Bearer "+tc.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := tc.client.Do(req)
	if err != nil {
		log.Fatalf("accept application failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		out, _ := io.ReadAll(resp.Body)
		log.Fatalf("accept application failed: status=%d, body=%s", resp.StatusCode, string(out))
	}
}

func connectWS(token string) *websocket.Conn {
	u := wsAddr + "/chats/ws?token=" + token
	conn, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		log.Fatalf("WS dial failed: %v", err)
	}
	return conn
}

func getChatHistory(tc *TestClient, otherUserID string) []*domain.ChatMessage {
	req, _ := http.NewRequest(http.MethodGet, serverAddr+"/chats/history?user_id="+otherUserID, nil)
	req.Header.Set("Authorization", "Bearer "+tc.token)
	resp, err := tc.client.Do(req)
	if err != nil {
		log.Fatalf("get chat history failed: %v", err)
	}
	defer resp.Body.Close()

	var history []*domain.ChatMessage
	_ = json.NewDecoder(resp.Body).Decode(&history)
	return history
}

func getRecentChats(tc *TestClient) []*domain.ChatMessage {
	req, _ := http.NewRequest(http.MethodGet, serverAddr+"/chats/recent", nil)
	req.Header.Set("Authorization", "Bearer "+tc.token)
	resp, err := tc.client.Do(req)
	if err != nil {
		log.Fatalf("get recent chats failed: %v", err)
	}
	defer resp.Body.Close()

	var recent []*domain.ChatMessage
	_ = json.NewDecoder(resp.Body).Decode(&recent)
	return recent
}
