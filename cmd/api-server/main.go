package main

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"mangahub/api"
	"mangahub/internal/auth"
	grpcService "mangahub/internal/grpc"
	"mangahub/internal/library"
	"mangahub/internal/manga"
	"mangahub/internal/middleware"
	"mangahub/internal/progress"
	"mangahub/internal/tcp"
	"mangahub/internal/udp"
	"mangahub/internal/user"
	"mangahub/internal/websocket"
	"mangahub/pkg/database"
	"mangahub/pkg/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
)

func main() {
	if len(os.Args) < 2 {
		printHelp()
		return
	}

	switch os.Args[1] {
	case "start":
		if len(os.Args) > 2 && os.Args[2] == "server" {
			runServer()
		} else {
			printHelp()
		}
	case "auth":
		if len(os.Args) > 2 {
			switch os.Args[2] {
			case "register":
				handleRegister()
			case "login":
				handleLogin()
			case "logout":
				handleLogout()
			case "status":
				handleAuthStatus()
			case "change-password":
				handleChangePassword()
			default:
				fmt.Println("Unknown auth command. Available: register, login, logout, status, change-password")
			}
		} else {
			fmt.Println("Missing auth command. Available: register, login, logout, status, change-password")
		}
	case "manga":
		if len(os.Args) > 2 {
			switch os.Args[2] {
			case "search":
				handleMangaSearch()
			default:
				fmt.Println("Unknown manga command. Available: search")
			}
		} else {
			fmt.Println("Missing manga command. Available: search")
		}
	case "library":
		if len(os.Args) > 2 {
			switch os.Args[2] {
			case "add":
				handleLibraryAdd()
			default:
				fmt.Println("Unknown library command. Available: add")
			}
		} else {
			fmt.Println("Missing library command. Available: add")
		}
	case "progress":
		if len(os.Args) > 2 {
			switch os.Args[2] {
			case "update":
				handleProgressUpdate()
			default:
				fmt.Println("Unknown progress command. Available: update")
			}
		} else {
			fmt.Println("Missing progress command. Available: update")
		}
	default:
		printHelp()
	}
}

func printHelp() {
	fmt.Println("Usage:")
	fmt.Println("  mangahub start server")
	fmt.Println("  mangahub auth register --username <name> --email <email>")
	fmt.Println("  mangahub auth login --username <name> OR --email <email>")
	fmt.Println("  mangahub auth logout")
	fmt.Println("  mangahub auth status")
	fmt.Println("  mangahub auth change-password")
	fmt.Println("  mangahub manga search \"<query>\"")
	fmt.Println("  mangahub library add --manga-id <id> --status <status>")
	fmt.Println("  mangahub progress update --manga-id <id> --chapter <number>")
}

func handleMangaSearch() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: mangahub manga search \"<query>\"")
		return
	}

	query := os.Args[3]

	db := database.ConnectDB()
	defer db.Close()

	repo := &manga.MangaRepository{DB: db}
	results, err := repo.SearchManga(query)
	if err != nil {
		log.Fatalf("Failed to search manga: %v", err)
	}

	if len(results) == 0 {
		fmt.Printf("No manga found matching \"%s\"\n", query)
		return
	}

	fmt.Printf("Found %d results for \"%s\":\n", len(results), query)
	fmt.Println("--------------------------------------------------")
	for _, m := range results {
		fmt.Printf("ID: %s\n", m.ID)
		fmt.Printf("Title: %s\n", m.Title)
		fmt.Printf("Author: %s\n", m.Author)
		fmt.Printf("Status: %s\n", m.Status)
		fmt.Printf("Chapters: %d\n", m.TotalChapters)
		fmt.Printf("Genres: %v\n", m.Genres)
		fmt.Println("--------------------------------------------------")
	}
}

func handleRegister() {
	registerCmd := flag.NewFlagSet("register", flag.ExitOnError)
	username := registerCmd.String("username", "", "Username")
	email := registerCmd.String("email", "", "Email")

	if len(os.Args) < 4 {
		fmt.Println("Usage: mangahub auth register --username <name> --email <email>")
		return
	}
	registerCmd.Parse(os.Args[3:])

	if *username == "" || *email == "" {
		fmt.Println("Username and email are required")
		registerCmd.Usage()
		return
	}

	fmt.Print("Password: ")
	reader := bufio.NewReader(os.Stdin)
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)

	fmt.Print("Confirm password: ")
	confirmPassword, _ := reader.ReadString('\n')
	confirmPassword = strings.TrimSpace(confirmPassword)

	if password != confirmPassword {
		fmt.Println("Passwords do not match")
		return
	}

	if len(password) < 6 {
		fmt.Println("Password must be at least 6 characters")
		return
	}

	db := database.ConnectDB()
	defer db.Close()

	repo := &user.UserRepository{DB: db}

	// Check if user exists
	_, err := repo.GetUserByUsername(*username)
	if err == nil {
		fmt.Println("Username already exists")
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	newUser := models.User{
		ID:           uuid.New().String(),
		Username:     *username,
		Email:        *email,
		PasswordHash: string(hashedPassword),
	}

	if err := repo.CreateUser(newUser); err != nil {
		log.Fatalf("Failed to create user: %v", err)
	}

	fmt.Println("✓ Account created successfully!")
	fmt.Printf("User ID: %s\n", newUser.ID)
	fmt.Printf("Username: %s\n", newUser.Username)
	fmt.Printf("Email: %s\n", newUser.Email)
	fmt.Printf("Created: %s\n", time.Now().UTC().Format("2006-01-02 15:04:05 UTC"))
	fmt.Println("Please login to start using MangaHub:")
	fmt.Printf(" mangahub auth login --username %s\n", newUser.Username)
}

func handleLogin() {
	loginCmd := flag.NewFlagSet("login", flag.ExitOnError)
	username := loginCmd.String("username", "", "Username")
	email := loginCmd.String("email", "", "Email")

	if len(os.Args) < 4 {
		fmt.Println("Usage: mangahub auth login --username <name> OR --email <email>")
		return
	}
	loginCmd.Parse(os.Args[3:])

	if *username == "" && *email == "" {
		fmt.Println("Username or email is required")
		loginCmd.Usage()
		return
	}

	fmt.Print("Password: ")
	reader := bufio.NewReader(os.Stdin)
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)

	db := database.ConnectDB()
	defer db.Close()

	repo := &user.UserRepository{DB: db}

	var user models.User
	var err error

	if *username != "" {
		user, err = repo.GetUserByUsername(*username)
	} else {
		user, err = repo.GetUserByEmail(*email)
	}

	if err != nil {
		fmt.Println("Invalid username/email or password")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		fmt.Println("Invalid username/email or password")
		return
	}

	// Generate token and save it
	token, err := auth.GenerateToken(user)
	if err != nil {
		log.Fatalf("Failed to generate token: %v", err)
	}

	// Save token to file
	if err := saveToken(token); err != nil {
		log.Printf("Warning: Failed to save token: %v", err)
	}

	// Calculate expiry (24 hours from now)
	expiry := time.Now().Add(24 * time.Hour).UTC().Format("2006-01-02 15:04:05 UTC")

	fmt.Println("✓ Login successful!")
	fmt.Printf("Welcome back, %s!\n", user.Username)
	fmt.Println("Session Details:")
	fmt.Printf(" Token expires: %s (24 hours)\n", expiry)
	fmt.Println(" Permissions: read, write, sync")
	fmt.Println("")
	fmt.Println("Auto-sync: enabled")
	fmt.Println("Notifications: enabled")
	fmt.Println("Ready to use MangaHub! Try:")
	fmt.Println(" mangahub manga search \"your favorite manga\"")
}

// handleLogout removes the stored authentication token.
func handleLogout() {
	// Check if token exists
	if _, err := loadToken(); err != nil {
		fmt.Println("✗ Logout failed: Not logged in")
		fmt.Println("No active session found. You can login with:")
		fmt.Println("  mangahub auth login --username <username>")
		return
	}

	if err := deleteToken(); err != nil {
		fmt.Printf("✗ Logout failed: %v\n", err)
		return
	}

	fmt.Println("✓ Logged out successfully!")
	fmt.Println("Authentication token removed from local storage.")
}

// handleAuthStatus checks and prints current authentication status.
func handleAuthStatus() {
	token, err := loadToken()
	if err != nil || strings.TrimSpace(token) == "" {
		fmt.Println("✗ Not authenticated")
		fmt.Println("You are not logged in.")
		fmt.Println("Try: mangahub auth login --username <username>")
		return
	}

	userID, username, expiry, err := auth.ParseToken(token)
	if err != nil {
		fmt.Printf("✗ Authentication status: %v\n", err)
		fmt.Println("Stored token is invalid or expired. Please login again:")
		fmt.Println("  mangahub auth login --username <username>")
		return
	}

	// Check expiry
	if !expiry.IsZero() && time.Now().After(expiry) {
		fmt.Println("✗ Authentication status: Token expired")
		fmt.Println("Your session has expired. Please login again:")
		fmt.Println("  mangahub auth login --username <username>")
		return
	}

	// Fetch additional user info from DB if possible
	db := database.ConnectDB()
	defer db.Close()

	repo := &user.UserRepository{DB: db}
	var email string
	if userID != "" {
		if u, err := repo.GetUserByID(userID); err == nil {
			email = u.Email
		}
	}

	fmt.Println("✓ You are logged in.")
	fmt.Println("")
	fmt.Println("User Information:")
	fmt.Printf("  User ID: %s\n", userID)
	fmt.Printf("  Username: %s\n", username)
	if email != "" {
		fmt.Printf("  Email: %s\n", email)
	}
	fmt.Println("")
	fmt.Println("Session:")
	if !expiry.IsZero() {
		fmt.Printf("  Token expires: %s\n", expiry.UTC().Format("2006-01-02 15:04:05 UTC"))
	}
	fmt.Println("  Permissions: read, write, sync")
	fmt.Println("  Auto-sync: enabled")
	fmt.Println("  Notifications: enabled")
}

func runServer() {
	// Initialize database
	db := database.ConnectDB()
	defer db.Close()

	// Initialize repositories
	userRepo := &user.UserRepository{DB: db}
	mangaRepo := &manga.MangaRepository{DB: db}
	libraryRepo := &library.LibraryRepository{DB: db}
	progressRepo := &progress.ProgressRepository{DB: db}

	// Load initial manga data from JSON if database is empty
	loadInitialMangaData(db, mangaRepo)

	// Initialize network servers
	tcpServer := tcp.NewServer(":8081")
	udpServer := udp.NewServer(":8082", "127.0.0.1", 8083)
	wsHub := websocket.NewHub()

	// Initialize handlers
	userHandler := &user.UserHandler{Repo: userRepo}
	mangaHandler := &manga.MangaHandler{
		Repo:      mangaRepo,
		UDPServer: udpServer,
	}
	libraryHandler := &library.LibraryHandler{Repo: libraryRepo}
	progressHandler := &progress.ProgressHandler{
		Repo:      progressRepo,
		TCPServer: tcpServer,
		UDPServer: udpServer,
	}

	// Start network servers
	var wg sync.WaitGroup
	wg.Add(4)

	// Start TCP server
	go func() {
		defer wg.Done()
		if err := tcpServer.Start(); err != nil {
			log.Printf("TCP Server error: %v", err)
		}
	}()

	// Start UDP server
	go func() {
		defer wg.Done()
		if err := udpServer.Start(); err != nil {
			log.Printf("UDP Server error: %v", err)
		}
	}()

	// Start WebSocket hub
	go func() {
		defer wg.Done()
		wsHub.Run()
	}()

	// Start gRPC server
	grpcListener, err := net.Listen("tcp", ":8084")
	if err != nil {
		log.Fatalf("Failed to listen on gRPC port: %v", err)
	}
	grpcServer := grpc.NewServer()
	grpcServiceServer := &grpcService.MangaServiceServer{
		MangaRepo:    mangaRepo,
		ProgressRepo: progressRepo,
	}
	api.RegisterMangaServiceServer(grpcServer, grpcServiceServer)

	go func() {
		defer wg.Done()
		log.Println("gRPC Server listening on :8084")
		if err := grpcServer.Serve(grpcListener); err != nil {
			log.Printf("gRPC Server error: %v", err)
		}
	}()

	// Initialize HTTP router
	router := gin.Default()

	// Middleware
	router.Use(middleware.CORS())
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Health check
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "MangaHub API is running",
			"services": gin.H{
				"http":      "running",
				"tcp":       "running",
				"udp":       "running",
				"websocket": "running",
				"grpc":      "running",
			},
		})
	})

	// API routes
	api := router.Group("/api/v1")
	{
		// Auth routes (public)
		authGroup := api.Group("/auth")
		{
			authGroup.POST("/register", userHandler.Register)
			authGroup.POST("/login", userHandler.Login)
		}

		// Manga routes (public)
		mangaGroup := api.Group("/manga")
		{
			mangaGroup.GET("", mangaHandler.GetAllManga)
			mangaGroup.GET("/search", mangaHandler.SearchManga)
			mangaGroup.GET("/:id", mangaHandler.GetMangaByID)
			mangaGroup.POST("", auth.JWTAuthMiddleware(), mangaHandler.CreateManga) // Protected
		}

		// Library routes (protected)
		libraryGroup := api.Group("/library")
		libraryGroup.Use(auth.JWTAuthMiddleware())
		{
			libraryGroup.GET("", libraryHandler.GetUserLibrary)
			libraryGroup.POST("", libraryHandler.AddToLibrary)
			libraryGroup.PUT("/:id", libraryHandler.UpdateStatus)
			libraryGroup.DELETE("/:id", libraryHandler.RemoveFromLibrary)
		}

		// Progress routes (protected)
		progressGroup := api.Group("/progress")
		progressGroup.Use(auth.JWTAuthMiddleware())
		{
			progressGroup.GET("", progressHandler.GetUserProgress)
			progressGroup.GET("/:id", progressHandler.GetMangaProgress)
			progressGroup.POST("", progressHandler.UpdateProgress)
		}
	}

	// WebSocket endpoint
	router.GET("/ws", func(c *gin.Context) {
		websocket.HandleWebSocket(wsHub, c.Writer, c.Request)
	})

	// Start HTTP server
	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	go func() {
		log.Println("HTTP Server listening on :8080")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP Server error: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down servers...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Stop HTTP server
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("HTTP Server shutdown error: %v", err)
	}

	// Stop TCP server
	tcpServer.Stop()

	// Stop UDP server
	udpServer.Stop()

	// Stop gRPC server
	grpcServer.GracefulStop()

	log.Println("All servers stopped")
}

func loadInitialMangaData(db *sql.DB, mangaRepo *manga.MangaRepository) {
	// Check if manga table has data
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM manga").Scan(&count)
	if err != nil || count > 0 {
		return // Database already has data or error occurred
	}

	// Load from JSON file
	mangaList, err := manga.LoadMangaData()
	if err != nil {
		log.Printf("Warning: Could not load initial manga data: %v", err)
		return
	}

	// Insert into database
	for _, m := range mangaList {
		if err := mangaRepo.CreateManga(m); err != nil {
			log.Printf("Warning: Could not insert manga %s: %v", m.ID, err)
		}
	}

	log.Printf("Loaded %d manga from JSON file", len(mangaList))
}

// Token storage functions
func getTokenFilePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory
		return ".mangahub_token"
	}
	return filepath.Join(homeDir, ".mangahub_token")
}

func saveToken(token string) error {
	tokenFile := getTokenFilePath()
	return os.WriteFile(tokenFile, []byte(token), 0600)
}

func loadToken() (string, error) {
	tokenFile := getTokenFilePath()
	data, err := os.ReadFile(tokenFile)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// deleteToken removes the stored authentication token file.
func deleteToken() error {
	tokenFile := getTokenFilePath()
	if _, err := os.Stat(tokenFile); os.IsNotExist(err) {
		return err
	}
	return os.Remove(tokenFile)
}

// HTTP client helper for authenticated requests
func makeAuthenticatedRequest(method, url string, body interface{}) (*http.Response, error) {
	token, err := loadToken()
	if err != nil {
		return nil, fmt.Errorf("not authenticated. Please login first: %v", err)
	}

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %v", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	return client.Do(req)
}

func handleLibraryAdd() {
	addCmd := flag.NewFlagSet("add", flag.ExitOnError)
	mangaID := addCmd.String("manga-id", "", "Manga ID")
	status := addCmd.String("status", "", "Reading status (reading, plan_to_read, completed, dropped)")

	if len(os.Args) < 4 {
		fmt.Println("Usage: mangahub library add --manga-id <id> --status <status>")
		return
	}
	addCmd.Parse(os.Args[3:])

	if *mangaID == "" {
		fmt.Println("Error: --manga-id is required")
		addCmd.Usage()
		return
	}

	if *status == "" {
		*status = "plan_to_read" // Default status
	}

	// Validate status
	validStatuses := []string{"reading", "plan_to_read", "completed", "dropped"}
	valid := false
	for _, s := range validStatuses {
		if *status == s {
			valid = true
			break
		}
	}
	if !valid {
		fmt.Printf("Error: Invalid status '%s'. Valid statuses: %v\n", *status, validStatuses)
		return
	}

	// Make API request
	reqBody := map[string]string{
		"manga_id": *mangaID,
		"status":   *status,
	}

	resp, err := makeAuthenticatedRequest("POST", "http://localhost:8080/api/v1/library", reqBody)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}

	if resp.StatusCode != http.StatusCreated {
		var errorResp map[string]interface{}
		if err := json.Unmarshal(body, &errorResp); err == nil {
			if errMsg, ok := errorResp["error"].(string); ok {
				fmt.Printf("Error: %s\n", errMsg)
				return
			}
		}
		fmt.Printf("Error: Failed to add manga to library (Status: %d)\n", resp.StatusCode)
		fmt.Printf("Response: %s\n", string(body))
		return
	}

	var libraryEntry map[string]interface{}
	if err := json.Unmarshal(body, &libraryEntry); err != nil {
		fmt.Printf("Error parsing response: %v\n", err)
		return
	}

	fmt.Println("✓ Manga added to library successfully!")
	fmt.Printf("Manga ID: %s\n", *mangaID)
	fmt.Printf("Status: %s\n", *status)
	if id, ok := libraryEntry["id"].(string); ok {
		fmt.Printf("Library Entry ID: %s\n", id)
	}
}

func handleProgressUpdate() {
	updateCmd := flag.NewFlagSet("update", flag.ExitOnError)
	mangaID := updateCmd.String("manga-id", "", "Manga ID")
	chapter := updateCmd.Int("chapter", 0, "Chapter number")

	if len(os.Args) < 4 {
		fmt.Println("Usage: mangahub progress update --manga-id <id> --chapter <number>")
		return
	}
	updateCmd.Parse(os.Args[3:])

	if *mangaID == "" {
		fmt.Println("Error: --manga-id is required")
		updateCmd.Usage()
		return
	}

	if *chapter <= 0 {
		fmt.Println("Error: --chapter must be a positive number")
		updateCmd.Usage()
		return
	}

	// Make API request
	reqBody := map[string]interface{}{
		"manga_id": *mangaID,
		"chapter":  *chapter,
	}

	resp, err := makeAuthenticatedRequest("POST", "http://localhost:8080/api/v1/progress", reqBody)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}

	if resp.StatusCode != http.StatusOK {
		var errorResp map[string]interface{}
		if err := json.Unmarshal(body, &errorResp); err == nil {
			if errMsg, ok := errorResp["error"].(string); ok {
				fmt.Printf("Error: %s\n", errMsg)
				return
			}
		}
		fmt.Printf("Error: Failed to update progress (Status: %d)\n", resp.StatusCode)
		fmt.Printf("Response: %s\n", string(body))
		return
	}

	var progressEntry map[string]interface{}
	if err := json.Unmarshal(body, &progressEntry); err != nil {
		fmt.Printf("Error parsing response: %v\n", err)
		return
	}

	fmt.Println("✓ Reading progress updated successfully!")
	fmt.Printf("Manga ID: %s\n", *mangaID)
	fmt.Printf("Chapter: %d\n", *chapter)
	if id, ok := progressEntry["id"].(string); ok {
		fmt.Printf("Progress Entry ID: %s\n", id)
	}
}

// validatePasswordStrength enforces a stronger password policy.
func validatePasswordStrength(pw string) error {
	if len(pw) < 8 {
		return fmt.Errorf("Password must be at least 8 characters with mixed case and numbers")
	}

	var hasUpper, hasLower, hasDigit bool
	for _, r := range pw {
		switch {
		case r >= 'A' && r <= 'Z':
			hasUpper = true
		case r >= 'a' && r <= 'z':
			hasLower = true
		case r >= '0' && r <= '9':
			hasDigit = true
		}
	}

	if !hasUpper || !hasLower || !hasDigit {
		return fmt.Errorf("Password must be at least 8 characters with mixed case and numbers")
	}

	return nil
}

// handleChangePassword allows an authenticated user to change their password.
func handleChangePassword() {
	// Require existing valid token
	token, err := loadToken()
	if err != nil || strings.TrimSpace(token) == "" {
		fmt.Println("✗ Change password failed: Not authenticated")
		fmt.Println("Please login before changing password:")
		fmt.Println("  mangahub auth login --username <username>")
		return
	}

	userID, _, _, err := auth.ParseToken(token)
	if err != nil || userID == "" {
		fmt.Println("✗ Change password failed: Invalid or expired session")
		fmt.Println("Please login again:")
		fmt.Println("  mangahub auth login --username <username>")
		return
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Current password: ")
	currentPassword, _ := reader.ReadString('\n')
	currentPassword = strings.TrimSpace(currentPassword)

	fmt.Print("New password: ")
	newPassword, _ := reader.ReadString('\n')
	newPassword = strings.TrimSpace(newPassword)

	fmt.Print("Confirm new password: ")
	confirmPassword, _ := reader.ReadString('\n')
	confirmPassword = strings.TrimSpace(confirmPassword)

	if newPassword != confirmPassword {
		fmt.Println("✗ Change password failed: Passwords do not match")
		fmt.Println("New password and confirmation do not match. Please try again.")
		return
	}

	if err := validatePasswordStrength(newPassword); err != nil {
		fmt.Printf("✗ Change password failed: %s\n", err.Error())
		return
	}

	db := database.ConnectDB()
	defer db.Close()

	repo := &user.UserRepository{DB: db}
	u, err := repo.GetUserByID(userID)
	if err != nil {
		fmt.Println("✗ Change password failed: User not found")
		return
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(currentPassword)); err != nil {
		fmt.Println("✗ Change password failed: Invalid current password")
		fmt.Println("The current password you entered is incorrect.")
		return
	}

	// Hash new password and update
	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println("✗ Change password failed: Internal error")
		return
	}

	if err := repo.UpdatePassword(userID, string(newHash)); err != nil {
		fmt.Println("✗ Change password failed: Internal error")
		return
	}

	fmt.Println("✓ Password changed successfully!")
	fmt.Println("Your new password is now active.")
	fmt.Println("For security, you may need to login again in some clients.")
}
