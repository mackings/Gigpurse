package http_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	delivery "gigpurse/internal/delivery/http"
	"gigpurse/internal/domain"
	"gigpurse/internal/repository/memory"
	"gigpurse/internal/usecase"

	"github.com/gorilla/websocket"
)

func TestSimulateClientMusicianAPIFlow(t *testing.T) {
	t.Setenv("JWT_SECRET", "api-simulation-secret")
	t.Setenv("ALLOW_ADMIN_SIGNUP", "true")

	app := newTestApp()
	server := httptest.NewServer(app.mux)
	defer server.Close()

	client := &apiClient{t: t, baseURL: server.URL, http: server.Client()}
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	clientUser := client.signup("client@example.com", "password123", "client", "Demo Client")
	musicianUser := client.signup("musician@example.com", "password123", "musician", "Demo Musician")
	adminUser := client.signup("admin@example.com", "password123", "admin", "Demo Admin")

	clientToken := client.login("client@example.com", "password123")
	musicianToken := client.login("musician@example.com", "password123")
	adminToken := client.login("admin@example.com", "password123")

	client.post("/auth/password-reset/request", "", map[string]any{"email": "client@example.com"}, http.StatusOK, nil)
	resetToken := "known-reset-token"
	resetHash := sha256.Sum256([]byte(resetToken))
	err := app.resetRepo.Create(context.Background(), &domain.PasswordResetToken{
		UserID:    clientUser.ID,
		TokenHash: hex.EncodeToString(resetHash[:]),
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("seed reset token: %v", err)
	}
	client.post("/auth/password-reset/confirm", "", map[string]any{"token": resetToken, "new_password": "password123"}, http.StatusOK, nil)

	client.get("/users/profile", clientToken, http.StatusOK, nil)
	client.put("/users/profile", clientToken, map[string]any{
		"name":     "Demo Client",
		"bio":      "Looking for reliable session musicians",
		"location": "Lagos",
		"client_profile": map[string]any{
			"company_name": "Gigpurse Events",
		},
	}, http.StatusOK, nil)
	client.put("/users/profile", musicianToken, map[string]any{
		"name":     "Demo Musician",
		"bio":      "Guitarist and producer",
		"location": "Lagos",
		"musician_profile": map[string]any{
			"stage_name":       "Demo Strings",
			"instrument":       "Guitar",
			"genre":            "Afrobeats",
			"experience_years": 7,
			"portfolio": []map[string]any{{
				"title":       "Live Session",
				"description": "Recorded guitar session",
				"url":         "https://example.com/session.mp4",
			}},
		},
	}, http.StatusOK, nil)

	var musicians []domain.User
	client.get("/musicians?genre=Afrobeats&instrument=Guitar&location=Lagos&min_exp=3&sort_by=experience", clientToken, http.StatusOK, &musicians)
	if len(musicians) != 1 || musicians[0].ID != musicianUser.ID {
		t.Fatalf("expected musician search to return the dummy musician, got %#v", musicians)
	}

	var job domain.Job
	client.post("/jobs", clientToken, map[string]any{
		"title":       "Afrobeats guitar session",
		"description": "Need guitar for a studio session",
		"instrument":  "Guitar",
		"genre":       "Afrobeats",
		"location":    "Lagos",
		"budget":      500,
	}, http.StatusCreated, &job)
	client.get("/jobs?id="+job.ID, clientToken, http.StatusOK, nil)
	client.get("/jobs?status=open&genre=Afrobeats&sort_by=budget&max_applications=5", musicianToken, http.StatusOK, nil)
	client.get("/jobs/recommended?limit=5", musicianToken, http.StatusOK, nil)

	var application domain.JobApplication
	client.post("/jobs/apply", musicianToken, map[string]any{
		"job_id":    job.ID,
		"proposal":  "I can deliver a clean session. WhatsApp and Paypal should be filtered in chat.",
		"price_bid": 450,
	}, http.StatusCreated, &application)
	client.get("/jobs/applications?job_id="+job.ID, clientToken, http.StatusOK, nil)
	client.get("/jobs/applications", musicianToken, http.StatusOK, nil)
	client.get("/jobs/mine?status=pending", musicianToken, http.StatusOK, nil)

	var chatMsg domain.ChatMessage
	client.post("/chats", clientToken, map[string]any{
		"recv_id": musicianUser.ID,
		"content": "Please do not ask for Paypal or WhatsApp outside the platform.",
	}, http.StatusCreated, &chatMsg)
	if strings.Contains(strings.ToLower(chatMsg.Content), "paypal") || strings.Contains(strings.ToLower(chatMsg.Content), "whatsapp") {
		t.Fatalf("expected chat content to be filtered, got %q", chatMsg.Content)
	}
	client.get("/chats/history?user_id="+musicianUser.ID, clientToken, http.StatusOK, nil)
	client.get("/chats/recent", clientToken, http.StatusOK, nil)

	clientWS := dialWS(t, wsURL+"/chats/ws?token="+clientToken)
	defer clientWS.Close()
	musicianWS := dialWS(t, wsURL+"/chats/ws?token="+musicianToken)
	defer musicianWS.Close()
	if err := clientWS.WriteJSON(map[string]string{"recv_id": musicianUser.ID, "content": "Realtime hello"}); err != nil {
		t.Fatalf("write websocket message: %v", err)
	}
	var wsSenderEcho domain.ChatMessage
	if err := clientWS.ReadJSON(&wsSenderEcho); err != nil {
		t.Fatalf("read websocket echo: %v", err)
	}
	var wsReceiverMsg domain.ChatMessage
	if err := musicianWS.ReadJSON(&wsReceiverMsg); err != nil {
		t.Fatalf("read websocket receiver message: %v", err)
	}

	client.post("/jobs/applications/accept", clientToken, map[string]any{"application_id": application.ID}, http.StatusOK, nil)
	client.get("/jobs/mine?status=active", musicianToken, http.StatusOK, nil)

	var contracts []domain.Contract
	client.get("/contracts", clientToken, http.StatusOK, &contracts)
	if len(contracts) == 0 {
		t.Fatal("expected accepted application to create a contract")
	}
	client.get("/contracts?id="+contracts[0].ID, clientToken, http.StatusOK, nil)

	var directHire domain.DirectHireRequest
	client.post("/direct-hires", clientToken, map[string]any{
		"musician_id": musicianUser.ID,
		"title":       "Private acoustic set",
		"description": "Direct hire for a private event",
		"price":       300,
	}, http.StatusCreated, &directHire)
	client.get("/direct-hires?status=pending", musicianToken, http.StatusOK, nil)
	client.post("/direct-hires/respond", musicianToken, map[string]any{
		"request_id": directHire.ID,
		"decision":   "accepted",
	}, http.StatusOK, nil)

	client.post("/contracts/complete", clientToken, map[string]any{"contract_id": contracts[0].ID}, http.StatusOK, nil)
	client.get("/jobs/mine?status=completed", musicianToken, http.StatusOK, nil)

	client.post("/reviews", clientToken, map[string]any{"job_id": job.ID, "rating": 5, "comment": "Excellent work"}, http.StatusCreated, nil)
	client.post("/reviews", musicianToken, map[string]any{"job_id": job.ID, "rating": 5, "comment": "Great client"}, http.StatusCreated, nil)
	client.get("/reviews?user_id="+musicianUser.ID, clientToken, http.StatusOK, nil)
	client.get("/reviews/average?user_id="+musicianUser.ID, clientToken, http.StatusOK, nil)

	var notifications []domain.Notification
	client.get("/notifications", musicianToken, http.StatusOK, &notifications)
	if len(notifications) > 0 {
		client.post("/notifications/read", musicianToken, map[string]any{"notification_id": notifications[0].ID}, http.StatusOK, nil)
	}

	client.get("/talent/dashboard", musicianToken, http.StatusOK, nil)

	var dispute domain.Dispute
	client.post("/disputes", clientToken, map[string]any{"contract_id": contracts[0].ID, "reason": "Need admin review"}, http.StatusCreated, &dispute)
	client.get("/disputes", musicianToken, http.StatusOK, nil)
	client.get("/admin/disputes?status=open", adminToken, http.StatusOK, nil)
	client.post("/admin/disputes/resolve", adminToken, map[string]any{"dispute_id": dispute.ID, "resolution": "Resolved after review"}, http.StatusOK, nil)

	client.post("/wallet", "", map[string]any{"user_id": clientUser.ID}, http.StatusCreated, nil)
	client.post("/wallet/deposit", "", map[string]any{"user_id": clientUser.ID, "amount": 100}, http.StatusOK, nil)
	client.get("/wallet?user_id="+clientUser.ID, "", http.StatusOK, nil)

	client.get("/admin/analytics", adminToken, http.StatusOK, nil)
	client.get("/admin/users", adminToken, http.StatusOK, nil)
	client.get("/admin/jobs", adminToken, http.StatusOK, nil)
	client.delete("/admin/jobs", adminToken, map[string]any{"job_id": job.ID}, http.StatusOK, nil)

	if clientUser.ID == "" || musicianUser.ID == "" || adminUser.ID == "" {
		t.Fatal("expected seeded users to have IDs")
	}
}

type testApp struct {
	mux       *http.ServeMux
	resetRepo *memoryPasswordResetRepo
}

func newTestApp() *testApp {
	userRepo := newMemoryUserRepo()
	jobRepo := newMemoryJobRepo()
	chatRepo := newMemoryChatRepo()
	contractRepo := newMemoryContractRepo()
	reviewRepo := newMemoryReviewRepo()
	notifRepo := newMemoryNotificationRepo()
	resetRepo := newMemoryPasswordResetRepo()
	disputeRepo := newMemoryDisputeRepo()
	walletRepo := memory.NewWalletRepository()

	userUsecase := usecase.NewUserUsecase(userRepo, resetRepo)
	jobUsecase := usecase.NewJobUsecase(jobRepo, userRepo, contractRepo, notifRepo)
	chatUsecase := usecase.NewChatUsecase(chatRepo, userRepo)
	contractUsecase := usecase.NewContractUsecase(contractRepo, jobRepo, notifRepo, userRepo)
	reviewUsecase := usecase.NewReviewUsecase(reviewRepo, jobRepo, notifRepo)
	notifUsecase := usecase.NewNotificationUsecase(notifRepo)
	disputeUsecase := usecase.NewDisputeUsecase(disputeRepo, contractRepo, notifRepo)
	dashboardUsecase := usecase.NewDashboardUsecase(jobUsecase, contractUsecase, reviewUsecase)
	adminUsecase := &memoryAdminUsecase{users: userRepo, jobs: jobRepo, chats: chatRepo, contracts: contractRepo, disputes: disputeRepo}
	walletUsecase := usecase.NewWalletUsecase(walletRepo)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"test-online"}`))
	})
	delivery.NewUserHandler(userUsecase).RegisterRoutes(mux)
	delivery.NewJobHandler(jobUsecase).RegisterRoutes(mux)
	delivery.NewChatHandler(chatUsecase).RegisterRoutes(mux)
	delivery.NewContractHandler(contractUsecase).RegisterRoutes(mux)
	delivery.NewReviewHandler(reviewUsecase).RegisterRoutes(mux)
	delivery.NewNotificationHandler(notifUsecase).RegisterRoutes(mux)
	delivery.NewDisputeHandler(disputeUsecase).RegisterRoutes(mux)
	delivery.NewDashboardHandler(dashboardUsecase).RegisterRoutes(mux)
	delivery.NewAdminHandler(adminUsecase).RegisterRoutes(mux)
	delivery.NewWalletHandler(walletUsecase).RegisterRoutes(mux)

	return &testApp{mux: mux, resetRepo: resetRepo}
}

type apiClient struct {
	t       *testing.T
	baseURL string
	http    *http.Client
}

func (c *apiClient) signup(email, password, role, name string) domain.User {
	var user domain.User
	c.post("/auth/signup", "", map[string]any{"email": email, "password": password, "role": role, "name": name}, http.StatusCreated, &user)
	return user
}

func (c *apiClient) login(email, password string) string {
	var res struct {
		Token string      `json:"token"`
		User  domain.User `json:"user"`
	}
	c.post("/auth/login", "", map[string]any{"email": email, "password": password}, http.StatusOK, &res)
	if res.Token == "" {
		c.t.Fatal("expected login token")
	}
	return res.Token
}

func (c *apiClient) get(path, token string, want int, out any) {
	c.request(http.MethodGet, path, token, nil, want, out)
}

func (c *apiClient) post(path, token string, body any, want int, out any) {
	c.request(http.MethodPost, path, token, body, want, out)
}

func (c *apiClient) put(path, token string, body any, want int, out any) {
	c.request(http.MethodPut, path, token, body, want, out)
}

func (c *apiClient) delete(path, token string, body any, want int, out any) {
	c.request(http.MethodDelete, path, token, body, want, out)
}

func (c *apiClient) request(method, path, token string, body any, want int, out any) {
	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			c.t.Fatalf("marshal request %s %s: %v", method, path, err)
		}
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, c.baseURL+path, reader)
	if err != nil {
		c.t.Fatalf("new request %s %s: %v", method, path, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		c.t.Fatalf("%s %s failed: %v", method, path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != want {
		c.t.Fatalf("%s %s status=%d want=%d", method, path, resp.StatusCode, want)
	}
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			c.t.Fatalf("decode %s %s: %v", method, path, err)
		}
	}
}

func dialWS(t *testing.T, url string) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	return conn
}

type memoryUserRepo struct {
	mu     sync.RWMutex
	next   int
	users  map[string]*domain.User
	emails map[string]string
}

func newMemoryUserRepo() *memoryUserRepo {
	return &memoryUserRepo{users: map[string]*domain.User{}, emails: map[string]string{}}
}

func (r *memoryUserRepo) Create(ctx context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.emails[user.Email]; exists {
		return errors.New("email already registered")
	}
	r.next++
	user.ID = fmt.Sprintf("usr_%d", r.next)
	cp := *user
	r.users[user.ID] = &cp
	r.emails[user.Email] = user.ID
	return nil
}

func (r *memoryUserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	user, ok := r.users[id]
	if !ok {
		return nil, errors.New("user not found")
	}
	cp := *user
	return &cp, nil
}

func (r *memoryUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.emails[email]
	if !ok {
		return nil, errors.New("user not found")
	}
	cp := *r.users[id]
	return &cp, nil
}

func (r *memoryUserRepo) Update(ctx context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *user
	r.users[user.ID] = &cp
	r.emails[user.Email] = user.ID
	return nil
}

func (r *memoryUserRepo) ListMusicians(ctx context.Context, filter domain.MusicianFilter) ([]*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var users []*domain.User
	for _, user := range r.users {
		if user.Role != "musician" {
			continue
		}
		if filter.Location != "" && !strings.Contains(strings.ToLower(user.Location), strings.ToLower(filter.Location)) {
			continue
		}
		if user.MusicianProfile != nil {
			if filter.Genre != "" && !strings.Contains(strings.ToLower(user.MusicianProfile.Genre), strings.ToLower(filter.Genre)) {
				continue
			}
			if filter.Instrument != "" && !strings.Contains(strings.ToLower(user.MusicianProfile.Instrument), strings.ToLower(filter.Instrument)) {
				continue
			}
			if filter.MinExp > 0 && user.MusicianProfile.ExperienceYears < filter.MinExp {
				continue
			}
		}
		cp := *user
		users = append(users, &cp)
	}
	return users, nil
}

func (r *memoryUserRepo) listAll() []*domain.User {
	r.mu.RLock()
	defer r.mu.RUnlock()
	users := make([]*domain.User, 0, len(r.users))
	for _, user := range r.users {
		cp := *user
		users = append(users, &cp)
	}
	return users
}

type memoryJobRepo struct {
	mu   sync.RWMutex
	next int
	jobs map[string]*domain.Job
	apps map[string]*domain.JobApplication
}

func newMemoryJobRepo() *memoryJobRepo {
	return &memoryJobRepo{jobs: map[string]*domain.Job{}, apps: map[string]*domain.JobApplication{}}
}

func (r *memoryJobRepo) Create(ctx context.Context, job *domain.Job) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.next++
	job.ID = fmt.Sprintf("job_%d", r.next)
	cp := *job
	r.jobs[job.ID] = &cp
	return nil
}

func (r *memoryJobRepo) GetByID(ctx context.Context, id string) (*domain.Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	job, ok := r.jobs[id]
	if !ok {
		return nil, errors.New("job not found")
	}
	cp := *job
	return &cp, nil
}

func (r *memoryJobRepo) Update(ctx context.Context, job *domain.Job) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *job
	r.jobs[job.ID] = &cp
	return nil
}

func (r *memoryJobRepo) List(ctx context.Context, filter domain.JobFilter) ([]*domain.Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var jobs []*domain.Job
	for _, job := range r.jobs {
		if filter.Status != "" && job.Status != filter.Status {
			continue
		}
		if filter.Genre != "" && !strings.Contains(strings.ToLower(job.Genre), strings.ToLower(filter.Genre)) {
			continue
		}
		if filter.Instrument != "" && !strings.Contains(strings.ToLower(job.Instrument), strings.ToLower(filter.Instrument)) {
			continue
		}
		if filter.Location != "" && !strings.Contains(strings.ToLower(job.Location), strings.ToLower(filter.Location)) {
			continue
		}
		if filter.MinBudget > 0 && job.Budget < filter.MinBudget {
			continue
		}
		if filter.MaxBudget > 0 && job.Budget > filter.MaxBudget {
			continue
		}
		cp := *job
		jobs = append(jobs, &cp)
	}
	return jobs, nil
}

func (r *memoryJobRepo) CreateApplication(ctx context.Context, app *domain.JobApplication) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.next++
	app.ID = fmt.Sprintf("app_%d", r.next)
	cp := *app
	r.apps[app.ID] = &cp
	return nil
}

func (r *memoryJobRepo) GetApplicationByID(ctx context.Context, id string) (*domain.JobApplication, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	app, ok := r.apps[id]
	if !ok {
		return nil, errors.New("application not found")
	}
	cp := *app
	return &cp, nil
}

func (r *memoryJobRepo) UpdateApplication(ctx context.Context, app *domain.JobApplication) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *app
	r.apps[app.ID] = &cp
	return nil
}

func (r *memoryJobRepo) ListApplications(ctx context.Context, jobID string) ([]*domain.JobApplication, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var apps []*domain.JobApplication
	for _, app := range r.apps {
		if app.JobID == jobID {
			cp := *app
			apps = append(apps, &cp)
		}
	}
	return apps, nil
}

func (r *memoryJobRepo) ListApplicationsByMusician(ctx context.Context, musicianID string) ([]*domain.JobApplication, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var apps []*domain.JobApplication
	for _, app := range r.apps {
		if app.MusicianID == musicianID {
			cp := *app
			apps = append(apps, &cp)
		}
	}
	return apps, nil
}

func (r *memoryJobRepo) delete(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.jobs, id)
}

type memoryChatRepo struct {
	mu       sync.RWMutex
	next     int
	messages []*domain.ChatMessage
}

func newMemoryChatRepo() *memoryChatRepo { return &memoryChatRepo{} }

func (r *memoryChatRepo) SaveMessage(ctx context.Context, msg *domain.ChatMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.next++
	msg.ID = fmt.Sprintf("msg_%d", r.next)
	cp := *msg
	r.messages = append(r.messages, &cp)
	return nil
}

func (r *memoryChatRepo) GetChatHistory(ctx context.Context, user1, user2 string) ([]*domain.ChatMessage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []*domain.ChatMessage
	for _, msg := range r.messages {
		if (msg.SenderID == user1 && msg.RecvID == user2) || (msg.SenderID == user2 && msg.RecvID == user1) {
			cp := *msg
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (r *memoryChatRepo) GetRecentChats(ctx context.Context, userID string) ([]*domain.ChatMessage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	latest := map[string]*domain.ChatMessage{}
	for _, msg := range r.messages {
		partner := msg.SenderID
		if msg.SenderID == userID {
			partner = msg.RecvID
		} else if msg.RecvID != userID {
			continue
		}
		if current, ok := latest[partner]; !ok || msg.Timestamp.After(current.Timestamp) {
			cp := *msg
			latest[partner] = &cp
		}
	}
	var out []*domain.ChatMessage
	for _, msg := range latest {
		out = append(out, msg)
	}
	return out, nil
}

func (r *memoryChatRepo) count() int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return int64(len(r.messages))
}

type memoryContractRepo struct {
	mu          sync.RWMutex
	next        int
	contracts   map[string]*domain.Contract
	directHires map[string]*domain.DirectHireRequest
}

func newMemoryContractRepo() *memoryContractRepo {
	return &memoryContractRepo{contracts: map[string]*domain.Contract{}, directHires: map[string]*domain.DirectHireRequest{}}
}

func (r *memoryContractRepo) Create(ctx context.Context, contract *domain.Contract) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.next++
	contract.ID = fmt.Sprintf("con_%d", r.next)
	cp := *contract
	r.contracts[contract.ID] = &cp
	return nil
}

func (r *memoryContractRepo) GetByID(ctx context.Context, id string) (*domain.Contract, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	contract, ok := r.contracts[id]
	if !ok {
		return nil, errors.New("contract not found")
	}
	cp := *contract
	return &cp, nil
}

func (r *memoryContractRepo) GetByJobID(ctx context.Context, jobID string) (*domain.Contract, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, contract := range r.contracts {
		if contract.JobID == jobID {
			cp := *contract
			return &cp, nil
		}
	}
	return nil, errors.New("contract not found")
}

func (r *memoryContractRepo) Update(ctx context.Context, contract *domain.Contract) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *contract
	r.contracts[contract.ID] = &cp
	return nil
}

func (r *memoryContractRepo) ListForUser(ctx context.Context, userID, role string) ([]*domain.Contract, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var contracts []*domain.Contract
	for _, contract := range r.contracts {
		if contract.ClientID == userID || contract.MusicianID == userID || role == "admin" {
			cp := *contract
			contracts = append(contracts, &cp)
		}
	}
	return contracts, nil
}

func (r *memoryContractRepo) CreateDirectHireRequest(ctx context.Context, req *domain.DirectHireRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.next++
	req.ID = fmt.Sprintf("dh_%d", r.next)
	cp := *req
	r.directHires[req.ID] = &cp
	return nil
}

func (r *memoryContractRepo) GetDirectHireRequestByID(ctx context.Context, id string) (*domain.DirectHireRequest, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	req, ok := r.directHires[id]
	if !ok {
		return nil, errors.New("direct hire request not found")
	}
	cp := *req
	return &cp, nil
}

func (r *memoryContractRepo) UpdateDirectHireRequest(ctx context.Context, req *domain.DirectHireRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *req
	r.directHires[req.ID] = &cp
	return nil
}

func (r *memoryContractRepo) ListDirectHireRequestsForUser(ctx context.Context, userID, role, status string) ([]*domain.DirectHireRequest, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var requests []*domain.DirectHireRequest
	for _, req := range r.directHires {
		if status != "" && req.Status != status {
			continue
		}
		if req.ClientID == userID || req.MusicianID == userID || role == "admin" {
			cp := *req
			requests = append(requests, &cp)
		}
	}
	return requests, nil
}

func (r *memoryContractRepo) count() int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return int64(len(r.contracts))
}

type memoryReviewRepo struct {
	mu      sync.RWMutex
	next    int
	reviews map[string]*domain.Review
}

func newMemoryReviewRepo() *memoryReviewRepo {
	return &memoryReviewRepo{reviews: map[string]*domain.Review{}}
}

func (r *memoryReviewRepo) Create(ctx context.Context, review *domain.Review) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.next++
	review.ID = fmt.Sprintf("rev_%d", r.next)
	cp := *review
	r.reviews[review.ID] = &cp
	return nil
}

func (r *memoryReviewRepo) ListByReviewee(ctx context.Context, revieweeID string) ([]*domain.Review, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var reviews []*domain.Review
	for _, review := range r.reviews {
		if review.RevieweeID == revieweeID {
			cp := *review
			reviews = append(reviews, &cp)
		}
	}
	return reviews, nil
}

func (r *memoryReviewRepo) GetByJobAndReviewer(ctx context.Context, jobID, reviewerID string) (*domain.Review, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, review := range r.reviews {
		if review.JobID == jobID && review.ReviewerID == reviewerID {
			cp := *review
			return &cp, nil
		}
	}
	return nil, errors.New("review not found")
}

type memoryNotificationRepo struct {
	mu     sync.RWMutex
	next   int
	notifs map[string]*domain.Notification
}

func newMemoryNotificationRepo() *memoryNotificationRepo {
	return &memoryNotificationRepo{notifs: map[string]*domain.Notification{}}
}

func (r *memoryNotificationRepo) Create(ctx context.Context, notif *domain.Notification) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.next++
	notif.ID = fmt.Sprintf("not_%d", r.next)
	cp := *notif
	r.notifs[notif.ID] = &cp
	return nil
}

func (r *memoryNotificationRepo) ListForUser(ctx context.Context, userID string) ([]*domain.Notification, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var notifs []*domain.Notification
	for _, notif := range r.notifs {
		if notif.UserID == userID {
			cp := *notif
			notifs = append(notifs, &cp)
		}
	}
	sort.Slice(notifs, func(i, j int) bool { return notifs[i].CreatedAt.After(notifs[j].CreatedAt) })
	return notifs, nil
}

func (r *memoryNotificationRepo) MarkAsRead(ctx context.Context, notifID, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	notif, ok := r.notifs[notifID]
	if !ok || notif.UserID != userID {
		return errors.New("notification not found")
	}
	notif.IsRead = true
	return nil
}

type memoryPasswordResetRepo struct {
	mu     sync.RWMutex
	tokens map[string]*domain.PasswordResetToken
}

func newMemoryPasswordResetRepo() *memoryPasswordResetRepo {
	return &memoryPasswordResetRepo{tokens: map[string]*domain.PasswordResetToken{}}
}

func (r *memoryPasswordResetRepo) Create(ctx context.Context, token *domain.PasswordResetToken) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	token.ID = fmt.Sprintf("rst_%d", len(r.tokens)+1)
	cp := *token
	r.tokens[token.TokenHash] = &cp
	return nil
}

func (r *memoryPasswordResetRepo) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.PasswordResetToken, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	token, ok := r.tokens[tokenHash]
	if !ok {
		return nil, errors.New("password reset token not found")
	}
	cp := *token
	return &cp, nil
}

func (r *memoryPasswordResetRepo) MarkUsed(ctx context.Context, id string, usedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, token := range r.tokens {
		if token.ID == id {
			token.UsedAt = usedAt
			return nil
		}
	}
	return errors.New("password reset token not found")
}

type memoryDisputeRepo struct {
	mu       sync.RWMutex
	next     int
	disputes map[string]*domain.Dispute
}

func newMemoryDisputeRepo() *memoryDisputeRepo {
	return &memoryDisputeRepo{disputes: map[string]*domain.Dispute{}}
}

func (r *memoryDisputeRepo) Create(ctx context.Context, dispute *domain.Dispute) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.next++
	dispute.ID = fmt.Sprintf("dsp_%d", r.next)
	cp := *dispute
	r.disputes[dispute.ID] = &cp
	return nil
}

func (r *memoryDisputeRepo) GetByID(ctx context.Context, id string) (*domain.Dispute, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	dispute, ok := r.disputes[id]
	if !ok {
		return nil, errors.New("dispute not found")
	}
	cp := *dispute
	return &cp, nil
}

func (r *memoryDisputeRepo) Update(ctx context.Context, dispute *domain.Dispute) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *dispute
	r.disputes[dispute.ID] = &cp
	return nil
}

func (r *memoryDisputeRepo) List(ctx context.Context, status string) ([]*domain.Dispute, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var disputes []*domain.Dispute
	for _, dispute := range r.disputes {
		if status == "" || dispute.Status == status {
			cp := *dispute
			disputes = append(disputes, &cp)
		}
	}
	return disputes, nil
}

func (r *memoryDisputeRepo) ListForUser(ctx context.Context, userID string) ([]*domain.Dispute, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var disputes []*domain.Dispute
	for _, dispute := range r.disputes {
		if dispute.ClientID == userID || dispute.MusicianID == userID || dispute.OpenedByID == userID {
			cp := *dispute
			disputes = append(disputes, &cp)
		}
	}
	return disputes, nil
}

func (r *memoryDisputeRepo) count() int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return int64(len(r.disputes))
}

type memoryAdminUsecase struct {
	users     *memoryUserRepo
	jobs      *memoryJobRepo
	chats     *memoryChatRepo
	contracts *memoryContractRepo
	disputes  *memoryDisputeRepo
}

func (u *memoryAdminUsecase) GetAnalytics(ctx context.Context) (*domain.AdminAnalytics, error) {
	return &domain.AdminAnalytics{
		TotalUsers:     int64(len(u.users.listAll())),
		TotalJobs:      int64(len(u.jobs.jobs)),
		TotalMessages:  u.chats.count(),
		TotalContracts: u.contracts.count(),
		TotalDisputes:  u.disputes.count(),
	}, nil
}

func (u *memoryAdminUsecase) ListAllUsers(ctx context.Context) ([]*domain.User, error) {
	return u.users.listAll(), nil
}

func (u *memoryAdminUsecase) ListAllJobs(ctx context.Context) ([]*domain.Job, error) {
	return u.jobs.List(ctx, domain.JobFilter{})
}

func (u *memoryAdminUsecase) DeleteJobListing(ctx context.Context, jobID string) error {
	u.jobs.delete(jobID)
	return nil
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
