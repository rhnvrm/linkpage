package main

type Link struct {
	ID       int    `db:"link_id"`
	URL      string `db:"url"`
	Text     string `db:"message"`
	ImageURL string `db:"image_url"`
	Weight   int    `db:"weight"`
	Hits     int    `db:"hits"`
}

type Page struct {
	LogoURL string
	Title   string
	Intro   string
	Links   []Link

	Error   string
	Success string

	OGPURL   string
	OGPImage string
	OGPDesc  string

	Social map[string]string
}
