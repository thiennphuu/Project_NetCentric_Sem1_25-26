package library

import (
	"net/http"
	"mangahub/internal/auth"
	"mangahub/pkg/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type LibraryHandler struct {
	Repo *LibraryRepository
}

func (h *LibraryHandler) AddToLibrary(c *gin.Context) {
	userID := auth.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		MangaID string `json:"manga_id" binding:"required"`
		Status  string `json:"status"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if req.Status == "" {
		req.Status = "plan_to_read"
	}

	library := models.UserLibrary{
		ID:      uuid.New().String(),
		UserID:  userID,
		MangaID: req.MangaID,
		Status:  req.Status,
	}

	if err := h.Repo.AddToLibrary(library); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add to library"})
		return
	}

	c.JSON(http.StatusCreated, library)
}

func (h *LibraryHandler) GetUserLibrary(c *gin.Context) {
	userID := auth.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	libraries, err := h.Repo.GetUserLibrary(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch library"})
		return
	}

	c.JSON(http.StatusOK, libraries)
}

func (h *LibraryHandler) UpdateStatus(c *gin.Context) {
	userID := auth.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	mangaID := c.Param("id")
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.Repo.UpdateLibraryStatus(userID, mangaID, req.Status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Status updated successfully"})
}

func (h *LibraryHandler) RemoveFromLibrary(c *gin.Context) {
	userID := auth.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	mangaID := c.Param("id")
	if err := h.Repo.RemoveFromLibrary(userID, mangaID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove from library"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Removed from library successfully"})
}

