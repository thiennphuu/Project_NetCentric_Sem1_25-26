package progress

import (
	"database/sql"
	"mangahub/pkg/models"
)

type ProgressRepository struct {
	DB *sql.DB
}

func (r *ProgressRepository) UpdateProgress(progress models.UserProgress) error {
	_, err := r.DB.Exec("INSERT OR REPLACE INTO user_progress (id, user_id, manga_id, chapter) VALUES (?, ?, ?, ?)",
		progress.ID, progress.UserID, progress.MangaID, progress.Chapter)
	return err
}

func (r *ProgressRepository) GetUserProgress(userID string) ([]models.UserProgress, error) {
	rows, err := r.DB.Query("SELECT id, user_id, manga_id, chapter, updated_at FROM user_progress WHERE user_id = ?", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var progresses []models.UserProgress
	for rows.Next() {
		var p models.UserProgress
		err := rows.Scan(&p.ID, &p.UserID, &p.MangaID, &p.Chapter, &p.UpdatedAt)
		if err != nil {
			return nil, err
		}
		progresses = append(progresses, p)
	}
	return progresses, nil
}

func (r *ProgressRepository) GetMangaProgress(userID, mangaID string) (models.UserProgress, error) {
	var p models.UserProgress
	err := r.DB.QueryRow("SELECT id, user_id, manga_id, chapter, updated_at FROM user_progress WHERE user_id = ? AND manga_id = ?", userID, mangaID).
		Scan(&p.ID, &p.UserID, &p.MangaID, &p.Chapter, &p.UpdatedAt)
	return p, err
}

