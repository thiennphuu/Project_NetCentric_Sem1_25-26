package manga

import (
	"database/sql"
	"encoding/json"
	"mangahub/pkg/models"
)

type MangaRepository struct {
	DB *sql.DB
}

func (r *MangaRepository) GetAllManga() ([]models.Manga, error) {
	rows, err := r.DB.Query("SELECT id, title, author, genres, status, total_chapters, description, cover_url FROM manga")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mangas []models.Manga
	for rows.Next() {
		var m models.Manga
		var genresJSON sql.NullString
		err := rows.Scan(&m.ID, &m.Title, &m.Author, &genresJSON, &m.Status, &m.TotalChapters, &m.Description, &m.CoverURL)
		if err != nil {
			return nil, err
		}
		if genresJSON.Valid {
			json.Unmarshal([]byte(genresJSON.String), &m.Genres)
		}
		mangas = append(mangas, m)
	}
	return mangas, nil
}

func (r *MangaRepository) GetMangaByID(id string) (models.Manga, error) {
	var m models.Manga
	var genresJSON sql.NullString
	err := r.DB.QueryRow("SELECT id, title, author, genres, status, total_chapters, description, cover_url FROM manga WHERE id = ?", id).
		Scan(&m.ID, &m.Title, &m.Author, &genresJSON, &m.Status, &m.TotalChapters, &m.Description, &m.CoverURL)
	if err != nil {
		return m, err
	}
	if genresJSON.Valid {
		json.Unmarshal([]byte(genresJSON.String), &m.Genres)
	}
	return m, nil
}

func (r *MangaRepository) CreateManga(manga models.Manga) error {
	genresJSON, _ := json.Marshal(manga.Genres)
	_, err := r.DB.Exec("INSERT INTO manga (id, title, author, genres, status, total_chapters, description, cover_url) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		manga.ID, manga.Title, manga.Author, string(genresJSON), manga.Status, manga.TotalChapters, manga.Description, manga.CoverURL)
	return err
}

func (r *MangaRepository) SearchManga(query string) ([]models.Manga, error) {
	rows, err := r.DB.Query("SELECT id, title, author, genres, status, total_chapters, description, cover_url FROM manga WHERE title LIKE ? OR author LIKE ? OR description LIKE ?",
		"%"+query+"%", "%"+query+"%", "%"+query+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mangas []models.Manga
	for rows.Next() {
		var m models.Manga
		var genresJSON sql.NullString
		err := rows.Scan(&m.ID, &m.Title, &m.Author, &genresJSON, &m.Status, &m.TotalChapters, &m.Description, &m.CoverURL)
		if err != nil {
			return nil, err
		}
		if genresJSON.Valid {
			json.Unmarshal([]byte(genresJSON.String), &m.Genres)
		}
		mangas = append(mangas, m)
	}
	return mangas, nil
}

