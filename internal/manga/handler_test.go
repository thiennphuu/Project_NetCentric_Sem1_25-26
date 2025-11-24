package manga

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"mangahub/pkg/models"
)

func TestMangaHandler_SearchManga(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	defer db.Close()

	repo := &MangaRepository{DB: db}
	handler := &MangaHandler{Repo: repo}

	// Seed data
	repo.CreateManga(models.Manga{ID: "one-piece", Title: "One Piece", Author: "Oda"})
	repo.CreateManga(models.Manga{ID: "naruto", Title: "Naruto", Author: "Kishimoto"})

	// Setup Router
	r := gin.Default()
	r.GET("/manga/search", handler.SearchManga)

	// Test Case 1: Successful Search
	req, _ := http.NewRequest("GET", "/manga/search?q=Piece", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	
	var response []models.Manga
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Len(t, response, 1)
	assert.Equal(t, "One Piece", response[0].Title)

	// Test Case 2: Empty Query
	req, _ = http.NewRequest("GET", "/manga/search?q=", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
