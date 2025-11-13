package manga

import (
	"encoding/json"
	"os"
	"mangahub/pkg/models"
)

func LoadMangaData() ([]models.Manga, error) {
	data, err := os.ReadFile("../../data/manga.json")

	if err != nil {
		return nil, err
	}

	var mangas []models.Manga
	err = json.Unmarshal(data, &mangas)
	return mangas, err
}
