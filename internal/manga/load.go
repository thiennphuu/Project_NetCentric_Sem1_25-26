package manga

import (
	"encoding/json"
	"mangahub/pkg/models"
	"os"
)

func LoadMangaData() ([]models.Manga, error) {
	data, err := os.ReadFile("data/manga.json")

	if err != nil {
		return nil, err
	}

	var mangas []models.Manga
	err = json.Unmarshal(data, &mangas)
	return mangas, err
}
