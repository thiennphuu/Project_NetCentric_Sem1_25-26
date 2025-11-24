package library

import (
	"database/sql"
	"mangahub/pkg/models"
)

type LibraryRepository struct {
	DB *sql.DB
}

func (r *LibraryRepository) AddToLibrary(library models.UserLibrary) error {
	_, err := r.DB.Exec("INSERT OR REPLACE INTO user_library (id, user_id, manga_id, status) VALUES (?, ?, ?, ?)",
		library.ID, library.UserID, library.MangaID, library.Status)
	return err
}

func (r *LibraryRepository) GetUserLibrary(userID string) ([]models.UserLibrary, error) {
	rows, err := r.DB.Query("SELECT id, user_id, manga_id, status, added_at FROM user_library WHERE user_id = ?", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var libraries []models.UserLibrary
	for rows.Next() {
		var l models.UserLibrary
		err := rows.Scan(&l.ID, &l.UserID, &l.MangaID, &l.Status, &l.AddedAt)
		if err != nil {
			return nil, err
		}
		libraries = append(libraries, l)
	}
	return libraries, nil
}

func (r *LibraryRepository) UpdateLibraryStatus(userID, mangaID, status string) error {
	_, err := r.DB.Exec("UPDATE user_library SET status = ? WHERE user_id = ? AND manga_id = ?", status, userID, mangaID)
	return err
}

func (r *LibraryRepository) RemoveFromLibrary(userID, mangaID string) error {
	_, err := r.DB.Exec("DELETE FROM user_library WHERE user_id = ? AND manga_id = ?", userID, mangaID)
	return err
}

