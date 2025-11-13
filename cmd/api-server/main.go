// package main

// import (
// 	"github.com/gin-gonic/gin"
// 	"github.com/google/uuid"
// 	"mangahub/internal/auth"
// 	"mangahub/internal/user"
// 	"mangahub/pkg/database"
// )

// func main() {
// 	db := database.ConnectDB()
// 	userRepo := user.UserRepository{DB: db}

// 	r := gin.Default()

// 	r.POST("/auth/register", func(c *gin.Context) {
// 		var body struct {
// 			Username string `json:"username"`
// 			Password string `json:"password"`
// 		}
// 		if c.Bind(&body) != nil {
// 			c.JSON(400, gin.H{"error": "invalid request"})
// 			return
// 		}

// 		hash, _ := auth.HashPassword(body.Password)

// 		user := models.User{
// 			ID:           uuid.NewString(),
// 			Username:     body.Username,
// 			PasswordHash: hash,
// 		}

// 		userRepo.CreateUser(user)
// 		c.JSON(200, gin.H{"message": "registered successfully"})
// 	})

// 	r.POST("/auth/login", func(c *gin.Context) {
// 		var body struct {
// 			Username string `json:"username"`
// 			Password string `json:"password"`
// 		}
// 		c.Bind(&body)

// 		user, err := userRepo.GetUserByUsername(body.Username)
// 		if err != nil {
// 			c.JSON(401, gin.H{"error": "user not found"})
// 			return
// 		}

// 		if auth.CheckPassword(user.PasswordHash, body.Password) != nil {
// 			c.JSON(401, gin.H{"error": "invalid password"})
// 			return
// 		}

// 		token, _ := auth.GenerateToken(user)
// 		c.JSON(200, gin.H{"token": token})
// 	})

// 	r.Run(":8080")
// }
package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"mangahub/internal/manga"
)

func main() {
	// Load manga database from JSON
	mangaList, err := manga.LoadMangaData()
	if err != nil {
		log.Fatalf("Failed to load manga data: %v", err)
	}

	// Initialize Gin router
	router := gin.Default()

	// Simple test endpoint
	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "MangaHub API is running ✅"})
	})

	// Endpoint to return full manga list
	router.GET("/manga", func(c *gin.Context) {
		c.JSON(200, mangaList)
	})

	// *Optional*: get single manga by id
	router.GET("/manga/:id", func(c *gin.Context) {
		id := c.Param("id")
		for _, m := range mangaList {
			if m.ID == id {
				c.JSON(200, m)
				return
			}
		}
		c.JSON(404, gin.H{"error": "Manga not found"})
	})

	// Start server
	log.Println("Server running at http://localhost:8080 ✅")
	router.Run(":8080")
}
