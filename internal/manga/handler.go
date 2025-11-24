package manga

import (
	"net/http"
	"mangahub/internal/udp"
	"mangahub/pkg/models"

	"github.com/gin-gonic/gin"
)

type MangaHandler struct {
	Repo      *MangaRepository
	UDPServer *udp.Server
}

func (h *MangaHandler) GetAllManga(c *gin.Context) {
	mangas, err := h.Repo.GetAllManga()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch manga"})
		return
	}
	c.JSON(http.StatusOK, mangas)
}

func (h *MangaHandler) GetMangaByID(c *gin.Context) {
	id := c.Param("id")
	manga, err := h.Repo.GetMangaByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Manga not found"})
		return
	}
	c.JSON(http.StatusOK, manga)
}

func (h *MangaHandler) CreateManga(c *gin.Context) {
	var newManga models.Manga
	if err := c.BindJSON(&newManga); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.Repo.CreateManga(newManga); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create manga"})
		return
	}

	// Broadcast new manga notification via UDP
	if h.UDPServer != nil {
		h.UDPServer.BroadcastNewManga(newManga.ID, newManga.Title)
	}

	c.JSON(http.StatusCreated, newManga)
}

func (h *MangaHandler) SearchManga(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query parameter 'q' is required"})
		return
	}

	mangas, err := h.Repo.SearchManga(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search manga"})
		return
	}

	c.JSON(http.StatusOK, mangas)
}
