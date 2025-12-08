package main

import (
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/otiai10/opengraph/v2"
)

func (app *App) HandleHome(w http.ResponseWriter, r *http.Request) {
	if err := app.Templates.Home.Write(w); err != nil {
		log.Printf("error while writing template: %v", err)
		writeInternalServerErr(w)
	}
}

func (app *App) HandleHits(w http.ResponseWriter, r *http.Request) {
	rawID, ok := mux.Vars(r)["id"]
	if !ok {
		writeBadRequest(w, "id missing")
		return
	}

	id, err := strconv.Atoi(rawID)
	if err != nil {
		log.Printf("error while getting links: %v", err)
		writeBadRequest(w, "bad id, got "+rawID)
		return
	}

	if err := app.DB.IncrementHit(id); err != nil {
		log.Printf("error while incrementing hits: %v", err)
		writeInternalServerErr(w)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}"))
	return
}

func (app *App) renderAdminPage(data Page) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := app.Templates.Admin.Execute(w, data); err != nil {
			log.Printf("error while writing template: %v", err)
			writeInternalServerErr(w)
			return
		}
	}
}

func (app *App) renderAdminPageWithErrMessage(msg string, p Page) func(w http.ResponseWriter, r *http.Request) {
	p.Error = msg
	return app.renderAdminPage(p)
}

func (app *App) HandleAdmin(w http.ResponseWriter, r *http.Request) {
	app.UpdateLinks()
	app.renderAdminPage(app.Data)(w, r)
}

func (app *App) HandleAdminUpdateWeight(w http.ResponseWriter, r *http.Request) {
	keys, ok := r.URL.Query()["action"]

	if !ok || len(keys[0]) < 1 {
		app.renderAdminPageWithErrMessage("action is missing", app.Data)(w, r)
		return
	}

	action := keys[0]

	rawID, ok := mux.Vars(r)["id"]
	if !ok {
		app.renderAdminPageWithErrMessage("id is missing", app.Data)(w, r)
		return
	}

	id, err := strconv.Atoi(rawID)
	if err != nil {
		app.renderAdminPageWithErrMessage("bad id, got: "+rawID, app.Data)(w, r)
		return
	}

	if err := app.DB.UpdateWeight(id, action); err != nil {
		app.renderAdminPageWithErrMessage(
			fmt.Sprintf("error while updating link: %v", err),
			app.Data)(w, r)
		return
	}

	if err := app.UpdateLinks(); err != nil {
		app.renderAdminPageWithErrMessage(
			fmt.Sprintf("error while updating link: %v", err),
			app.Data)(w, r)
		return
	}

	app.renderAdminPage(app.Data)(w, r)
}
func (app *App) HandleAdminDelete(w http.ResponseWriter, r *http.Request) {
	rawID, ok := mux.Vars(r)["id"]
	if !ok {
		app.renderAdminPageWithErrMessage("id is missing", app.Data)(w, r)
		return
	}

	id, err := strconv.Atoi(rawID)
	if err != nil {
		app.renderAdminPageWithErrMessage("bad id, got: "+rawID, app.Data)(w, r)
		return
	}

	if err := app.DB.DeleteLink(id); err != nil {
		app.renderAdminPageWithErrMessage(
			fmt.Sprintf("error while deleting link: %v", err),
			app.Data)(w, r)
		return
	}

	if err := app.UpdateLinks(); err != nil {
		app.renderAdminPageWithErrMessage(
			fmt.Sprintf("error while updating links: %v", err),
			app.Data)(w, r)
	}

	app.renderAdminPage(app.Data)(w, r)
}

func (app *App) HandleAdminUpdate(w http.ResponseWriter, r *http.Request) {
	rawID, ok := mux.Vars(r)["id"]
	if !ok {
		app.renderAdminPageWithErrMessage("id is missing", app.Data)(w, r)
		return
	}

	id, err := strconv.Atoi(rawID)
	if err != nil {
		app.renderAdminPageWithErrMessage("bad id, got: "+rawID, app.Data)(w, r)
		return
	}

	r.ParseForm()

	text := r.Form.Get("text")
	url := r.Form.Get("url")
	description := r.Form.Get("description")
	imageURL := r.Form.Get("image_url")

	if url == "" {
		app.renderAdminPageWithErrMessage("url is missing", app.Data)(w, r)
		return
	}
	if text == "" {
		app.renderAdminPageWithErrMessage("text is missing", app.Data)(w, r)
		return
	}

	if err := app.DB.UpdateLink(id, text, description, url, imageURL); err != nil {
		app.renderAdminPageWithErrMessage(
			fmt.Sprintf("error while updating link: %v", err),
			app.Data)(w, r)
		return
	}

	if err := app.UpdateLinks(); err != nil {
		app.renderAdminPageWithErrMessage(
			fmt.Sprintf("error while updating links: %v", err),
			app.Data)(w, r)
		return
	}

	app.renderAdminPage(app.Data)(w, r)
}

func (app *App) HandleAdminNew(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	text := r.Form.Get("text")
	url := r.Form.Get("url")
	description := r.Form.Get("description")
	imageURL := r.Form.Get("image_url")
	submitType := r.Form.Get("submit")

	if url == "" {
		app.renderAdminPageWithErrMessage("url is missing", app.Data)(w, r)
		return
	}

	if submitType == "Fetch Data" {
		ogp, err := opengraph.Fetch(url)
		if err != nil {
			app.renderAdminPageWithErrMessage(
				fmt.Sprintf("error while fetching link: %v", err),
				app.Data)(w, r)
			return
		}

		ogp.ToAbs()
		if len(ogp.Image) > 0 {
			imageURL = ogp.Image[0].URL
		}

		p := app.Data
		p.OGPImage = imageURL
		p.OGPDesc = ogp.Title
		p.OGPDescription = ogp.Description
		p.OGPURL = url
		app.renderAdminPage(p)(w, r)
		return
	}

	if text == "" {
		app.renderAdminPageWithErrMessage("text is missing", app.Data)(w, r)
		return
	}

	if imageURL == "" {
		ogp, err := opengraph.Fetch(url)
		if err != nil {
			app.renderAdminPageWithErrMessage(
				fmt.Sprintf("error while fetching link: %v", err),
				app.Data)(w, r)
			return
		}

		ogp.ToAbs()
		if len(ogp.Image) > 0 {
			imageURL = ogp.Image[0].URL
		}
	}

	if err := app.DB.InsertLink(text, description, url, imageURL); err != nil{
		app.renderAdminPageWithErrMessage(
			fmt.Sprintf("error while inserting link: %v", err),
			app.Data)(w, r)
		return
	}

	if err := app.UpdateLinks(); err != nil {
		log.Printf("error while updating links: %v", err)
		writeInternalServerErr(w)
		return
	}

	p := app.Data
	p.Success = "New link inserted!"
	app.renderAdminPage(p)(w, r)
}

// customFileServer creates a handler that overlays customDir on top of staticFS.
func customFileServer(customDir string, staticFS embed.FS) http.Handler {
	if customDir == "" {
		return http.FileServer(http.FS(staticFS))
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// First, check if the file exists in the custom directory
		customPath := filepath.Join(customDir, r.URL.Path)
		if _, err := os.Stat(customPath); err == nil {
			// Serve file from custom directory
			http.ServeFile(w, r, customPath)
			return
		}

		// Fallback to the embedded staticFS if the file is not in the custom directory
		staticHandler := http.FileServer(http.FS(staticFS))
		staticHandler.ServeHTTP(w, r)
	})
}
