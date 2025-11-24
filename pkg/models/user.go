package models

type User struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	PasswordHash string `json:"-"`
	CreatedAt    string `json:"created_at"`
}

// UserProgress tracks reading progress for a user
type UserProgress struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	MangaID   string `json:"manga_id"`
	Chapter   int    `json:"chapter"`
	UpdatedAt string `json:"updated_at"`
}

// UserLibrary represents a manga in user's library
type UserLibrary struct {
	ID      string `json:"id"`
	UserID  string `json:"user_id"`
	MangaID string `json:"manga_id"`
	Status  string `json:"status"` // reading, completed, plan_to_read, dropped
	AddedAt string `json:"added_at"`
}
