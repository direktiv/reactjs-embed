package main

import (
	"embed"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

//go:embed build/*
var Assets embed.FS

type fsFunc func(name string) (fs.File, error)

func (f fsFunc) Open(name string) (fs.File, error) {
	return f(name)
}

func AssetHandler(prefix, root string) http.Handler {
	handler := fsFunc(func(name string) (fs.File, error) {
		assetPath := path.Join(root, name)

		f, err := Assets.Open(assetPath)
		if os.IsNotExist(err) {
			// if asset does not exist open index.html
			return os.Open("./index.html")
		}

		// check if index.html is needed redirect to different file?
		if strings.Contains(assetPath, "index.html") {
			return os.Open("./index.html")
		}

		return f, err
	})
	return http.StripPrefix(prefix, http.FileServer(http.FS(handler)))
}

func main() {
	// read embed index html to replace values with environment variables
	data, err := Assets.ReadFile("build/index.html")
	if err != nil {
		log.Fatal("unable to read index.html")
	}

	new := strings.ReplaceAll(string(data), "API-URL", os.Getenv("API_URL"))
	newer := strings.ReplaceAll(new, "KEYCLOAK-URL", os.Getenv("KEYCLOAK_URL"))

	err = ioutil.WriteFile("index.html", []byte(newer), 0600)
	if err != nil {
		log.Fatal("unable to write new index html file")
	}

	handler := AssetHandler("/", "build")

	srv := &http.Server{
		Handler: handler,
		Addr:    fmt.Sprintf("%s:%s", os.Getenv("SERVE_ADDRESS"), os.Getenv("SERVE_PORT")),
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}
