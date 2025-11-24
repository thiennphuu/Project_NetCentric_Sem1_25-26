package manga

import (
	"database/sql"
	"mangahub/pkg/models"
	"testing"

	_ "github.com/glebarez/go-sqlite"
	"github.com/stretchr/testify/assert"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory db: %v", err)
	}

	createMangaTable := `
	CREATE TABLE IF NOT EXISTS manga (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		author TEXT,
		genres TEXT,
		status TEXT,
		total_chapters INTEGER DEFAULT 0,
		description TEXT,
		cover_url TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`
	_, err = db.Exec(createMangaTable)
	if err != nil {
		t.Fatalf("Failed to create manga table: %v", err)
	}

	return db
}

func TestMangaRepository_CreateManga(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := &MangaRepository{DB: db}

	manga := models.Manga{
		ID:            "one-piece",
		Title:         "One Piece",
		Author:        "Oda Eiichiro",
		Genres:        []string{"Action", "Adventure"},
		Status:        "Ongoing",
		TotalChapters: 1100,
		Description:   "Pirates...",
		CoverURL:      "http://example.com/op.jpg",
	}

	err := repo.CreateManga(manga)
	assert.NoError(t, err)

	// Verify it exists
	var title string
	err = db.QueryRow("SELECT title FROM manga WHERE id = ?", manga.ID).Scan(&title)
	assert.NoError(t, err)
	assert.Equal(t, "One Piece", title)
}

func TestMangaRepository_GetMangaByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := &MangaRepository{DB: db}

	// Seed data
	manga := models.Manga{
		ID:            "naruto",
		Title:         "Naruto",
		Author:        "Kishimoto Masashi",
		Genres:        []string{"Ninja", "Action"},
		Status:        "Completed",
		TotalChapters: 700,
		Description:   "Ninjas...",
		CoverURL:      "http://example.com/naruto.jpg",
	}
	err := repo.CreateManga(manga)
	assert.NoError(t, err)

	// Test Get
	fetched, err := repo.GetMangaByID("naruto")
	assert.NoError(t, err)
	assert.Equal(t, manga.ID, fetched.ID)
	assert.Equal(t, manga.Title, fetched.Title)
	assert.Equal(t, manga.Genres, fetched.Genres)
}

func TestMangaRepository_SearchManga(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := &MangaRepository{DB: db}

	// Seed data
	m1 := models.Manga{ID: "aot", Title: "Attack on Titan", Author: "Isayama", Description: "Titans"}
	m2 := models.Manga{ID: "aot-jr", Title: "Attack on Titan: Junior High", Author: "Isayama", Description: "School"}
	m3 := models.Manga{ID: "bleach", Title: "Bleach", Author: "Kubo", Description: "Ghosts"}

	repo.CreateManga(m1)
	repo.CreateManga(m2)
	repo.CreateManga(m3)

	// Search "Titan"
	results, err := repo.SearchManga("Titan")
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	
	// Verify results contain expected IDs
	ids := make(map[string]bool)
	for _, m := range results {
		ids[m.ID] = true
	}
	assert.True(t, ids["aot"])
	assert.True(t, ids["aot-jr"])
	assert.False(t, ids["bleach"])
}
