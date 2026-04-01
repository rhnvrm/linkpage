package main

import (
	"fmt"
	"log"
	"os"

	"github.com/pelletier/go-toml/v2"
)

// SeedConfig represents the structure of a seed TOML file.
type SeedConfig struct {
	Links []SeedLink `toml:"links"`
}

// SeedLink represents a single link entry in the seed file.
type SeedLink struct {
	URL         string `toml:"url"`
	Message     string `toml:"message"`
	Description string `toml:"description"`
	ImageURL    string `toml:"image_url"`
	Weight      int    `toml:"weight"`
}

// applySeed reads a TOML seed file, clears existing links, and inserts the seed links.
func (app *App) applySeed(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading seed file: %w", err)
	}

	var seed SeedConfig
	if err := toml.Unmarshal(data, &seed); err != nil {
		return fmt.Errorf("parsing seed file: %w", err)
	}

	// Clear all existing links
	if _, err := app.DB.db.Exec("DELETE FROM links;"); err != nil {
		return fmt.Errorf("clearing links table: %w", err)
	}

	// Insert seed links
	for _, link := range seed.Links {
		_, err := app.DB.db.Exec(
			"INSERT INTO links (url, message, description, image_url, weight) VALUES (?, ?, ?, ?, ?);",
			link.URL, link.Message, link.Description, link.ImageURL, link.Weight,
		)
		if err != nil {
			return fmt.Errorf("inserting seed link %q: %w", link.URL, err)
		}
	}

	log.Printf("seeded %d links from %s", len(seed.Links), path)

	// Refresh the in-memory link cache
	if err := app.UpdateLinks(); err != nil {
		return fmt.Errorf("updating links after seed: %w", err)
	}

	return nil
}
