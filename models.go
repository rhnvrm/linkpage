package main

type Link struct {
	ID          int    `db:"link_id"`
	URL         string `db:"url"`
	Text        string `db:"message"`
	Description string `db:"description"`
	ImageURL    string `db:"image_url"`
	Weight      int    `db:"weight"`
	Hits        int    `db:"hits"`
}

type Page struct {
	LogoURL string
	Title   string
	Intro   string
	Links   []Link

	Error   string
	Success string

	OGPURL         string
	OGPImage       string
	OGPDesc        string
	OGPDescription string

	Social map[string]string
}
