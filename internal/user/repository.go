package user

import (
	"database/sql"
	"mangahub/pkg/models"
)

type UserRepository struct {
	DB *sql.DB
}

func (r *UserRepository) CreateUser(user models.User) error {
	_, err := r.DB.Exec("INSERT INTO users (id, username, password_hash) VALUES (?, ?, ?)",
		user.ID, user.Username, user.PasswordHash)
	return err
}

func (r *UserRepository) GetUserByUsername(username string) (models.User, error) {
	row := r.DB.QueryRow("SELECT id, username, password_hash FROM users WHERE username = ?", username)
	var user models.User
	err := row.Scan(&user.ID, &user.Username, &user.PasswordHash)
	return user, err
}
