package studio

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed dist/*
var distFS embed.FS

// StaticHandler returns an http.Handler that serves the embedded studio UI.
// Returns nil if the dist directory is empty (dev mode).
func StaticHandler() http.Handler {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil
	}

	// Check if dist has any files
	entries, err := fs.ReadDir(sub, ".")
	if err != nil || len(entries) == 0 {
		return nil
	}

	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the file. If it doesn't exist, serve index.html (SPA fallback).
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}

		// Check if file exists
		f, err := sub.Open(path[1:]) // strip leading /
		if err != nil {
			// SPA fallback — serve index.html for client-side routing
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
			return
		}
		f.Close()

		fileServer.ServeHTTP(w, r)
	})
}
