package main

import (
	"bufio"
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
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
			default:
				fmt.Println("Unknown auth command. Available: register, login")
			}
		} else {
			fmt.Println("Missing auth command. Available: register, login")
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

	// Generate token (we don't print it in the new format, but we could if needed)
	_, err = auth.GenerateToken(user)
	if err != nil {
		log.Fatalf("Failed to generate token: %v", err)
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
