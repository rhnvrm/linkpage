package main

import (
	"fmt"
	"log"
	"os"
	"strings"

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

// applySeed reads a TOML seed file and replaces database links with the seed links.
func (app *App) applySeed(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading seed file: %w", err)
	}

	var seed SeedConfig
	if err := toml.Unmarshal(data, &seed); err != nil {
		return fmt.Errorf("parsing seed file: %w", err)
	}

	if err := app.DB.ReplaceLinks(seed.Links); err != nil {
		return fmt.Errorf("applying seed file: %w", err)
	}

	log.Printf("seeded %d links from %s", len(seed.Links), path)

	// Refresh the in-memory link cache
	if err := app.UpdateLinks(); err != nil {
		return fmt.Errorf("updating links after seed: %w", err)
	}

	return nil
}

// ReplaceLinks replaces all existing links with the provided declarative link set.
func (l *LinkDB) ReplaceLinks(links []SeedLink) error {
	for i, link := range links {
		if err := validateSeedLink(link); err != nil {
			return fmt.Errorf("validating seed link %d: %w", i+1, err)
		}
	}

	tx, err := l.db.Beginx()
	if err != nil {
		return fmt.Errorf("starting seed transaction: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if _, err := tx.Exec("DELETE FROM links;"); err != nil {
		return fmt.Errorf("clearing links table: %w", err)
	}

	for _, link := range links {
		_, err := tx.Exec(
			"INSERT INTO links (url, message, description, image_url, weight) VALUES (?, ?, ?, ?, ?);",
			link.URL, link.Message, link.Description, link.ImageURL, link.Weight,
		)
		if err != nil {
			return fmt.Errorf("inserting seed link %q: %w", link.URL, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing seed transaction: %w", err)
	}
	committed = true

	return nil
}

func validateSeedLink(link SeedLink) error {
	if strings.TrimSpace(link.URL) == "" {
		return fmt.Errorf("url is required")
	}

	if strings.TrimSpace(link.Message) == "" {
		return fmt.Errorf("message is required")
	}

	return nil
}
