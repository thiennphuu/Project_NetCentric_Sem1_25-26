package main

import (
	"context"
	"database/sql"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
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
)

func main() {
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
