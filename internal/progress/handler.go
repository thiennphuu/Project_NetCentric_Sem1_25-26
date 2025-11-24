package progress

import (
	"net/http"
	"mangahub/internal/auth"
	"mangahub/internal/tcp"
	"mangahub/internal/udp"
	"mangahub/pkg/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ProgressHandler struct {
	Repo      *ProgressRepository
	TCPServer *tcp.Server
	UDPServer *udp.Server
}

func (h *ProgressHandler) UpdateProgress(c *gin.Context) {
	userID := auth.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		MangaID string `json:"manga_id" binding:"required"`
		Chapter int    `json:"chapter" binding:"required"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	progress := models.UserProgress{
		ID:      uuid.New().String(),
		UserID:  userID,
		MangaID: req.MangaID,
		Chapter: req.Chapter,
	}

	if err := h.Repo.UpdateProgress(progress); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update progress"})
		return
	}

	// Broadcast progress update via TCP and UDP
	if h.TCPServer != nil {
		h.TCPServer.BroadcastProgress(userID, req.MangaID, req.Chapter)
	}

	if h.UDPServer != nil {
		h.UDPServer.BroadcastUpdate(
			"Progress updated",
			map[string]interface{}{
				"user_id":  userID,
				"manga_id": req.MangaID,
				"chapter":  req.Chapter,
			},
		)
	}

	c.JSON(http.StatusOK, progress)
}

func (h *ProgressHandler) GetUserProgress(c *gin.Context) {
	userID := auth.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	progresses, err := h.Repo.GetUserProgress(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch progress"})
		return
	}

	c.JSON(http.StatusOK, progresses)
}

func (h *ProgressHandler) GetMangaProgress(c *gin.Context) {
	userID := auth.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	mangaID := c.Param("id")
	progress, err := h.Repo.GetMangaProgress(userID, mangaID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Progress not found"})
		return
	}

	c.JSON(http.StatusOK, progress)
}

