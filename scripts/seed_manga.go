package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"mangahub/pkg/database"
	"mangahub/pkg/models"
)

var (
	titles = []string{
		"Dragon", "Naruto", "Bleach", "One Piece", "Hunter", "Slayer", "Titan", "Note", "Alchemist", "Tail",
		"Clover", "Hero", "Academia", "Ghoul", "Punch", "Man", "Ball", "Z", "Super", "Kaisen",
		"Revengers", "Family", "Spy", "Chainsaw", "Blue", "Lock", "Stone", "Fire", "Force", "Exorcist",
	}
	authors = []string{
		"Akira Toriyama", "Eiichiro Oda", "Masashi Kishimoto", "Tite Kubo", "Yoshihiro Togashi",
		"Hajime Isayama", "Tsugumi Ohba", "Hiromu Arakawa", "Hiro Mashima", "Yuki Tabata",
		"Kohei Horikoshi", "Sui Ishida", "ONE", "Yusuke Murata", "Gege Akutami",
	}
	genres = []string{
		"Action", "Adventure", "Comedy", "Drama", "Fantasy", "Horror", "Mystery", "Romance", "Sci-Fi", "Slice of Life",
		"Sports", "Supernatural", "Thriller", "Psychological", "Seinen", "Shounen", "Shoujo", "Josei",
	}
	statuses = []string{"ongoing", "completed", "hiatus"}
)

func main() {
	db := database.ConnectDB()
	defer db.Close()

	rand.Seed(time.Now().UnixNano())

	fmt.Println("Seeding 100 manga entries...")

	for i := 0; i < 100; i++ {
		manga := generateRandomManga()
		if err := createManga(db, manga); err != nil {
			log.Printf("Failed to create manga %s: %v", manga.Title, err)
		} else {
			// fmt.Printf("Created: %s\n", manga.Title)
		}
	}

	fmt.Println("Done!")
}

func generateRandomManga() models.Manga {
	title := fmt.Sprintf("%s %s", titles[rand.Intn(len(titles))], titles[rand.Intn(len(titles))])
	// Ensure unique-ish titles by appending a number if needed, but for 100 items, collisions are rare enough or acceptable for mock data.
	// Let's add a random suffix to be safe.
	title = fmt.Sprintf("%s %d", title, rand.Intn(1000))

	author := authors[rand.Intn(len(authors))]
	status := statuses[rand.Intn(len(statuses))]
	totalChapters := rand.Intn(500) + 1

	// Random genres (1 to 3)
	numGenres := rand.Intn(3) + 1
	mangaGenres := make([]string, 0, numGenres)
	seenGenres := make(map[string]bool)
	for j := 0; j < numGenres; j++ {
		g := genres[rand.Intn(len(genres))]
		if !seenGenres[g] {
			mangaGenres = append(mangaGenres, g)
			seenGenres[g] = true
		}
	}

	// Generate slug-like ID
	id := strings.ToLower(title)
	id = strings.ReplaceAll(id, " ", "-")
	// Remove non-alphanumeric chars (except hyphens) for cleaner IDs
	reg, _ := regexp.Compile("[^a-z0-9-]+")
	id = reg.ReplaceAllString(id, "")

	return models.Manga{
		ID:            id,
		Title:         title,
		Author:        author,
		Genres:        mangaGenres,
		Status:        status,
		TotalChapters: totalChapters,
		Description:   fmt.Sprintf("A randomly generated manga about %s.", title),
		CoverURL:      fmt.Sprintf("https://example.com/covers/%s.jpg", id),
	}
}

func createManga(db *sql.DB, manga models.Manga) error {
	genresJSON, _ := json.Marshal(manga.Genres)
	_, err := db.Exec("INSERT INTO manga (id, title, author, genres, status, total_chapters, description, cover_url) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		manga.ID, manga.Title, manga.Author, string(genresJSON), manga.Status, manga.TotalChapters, manga.Description, manga.CoverURL)
	return err
}
