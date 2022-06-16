package main

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/vorteil/direktiv/pkg/util"
)

//go:embed build/*
var Assets embed.FS

type fsFunc func(name string) (fs.File, error)

func (f fsFunc) Open(name string) (fs.File, error) {
	return f(name)
}

type IndexFile struct {
	io.ReadCloser
	Contents []byte
}

type IndexFileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modtime time.Time
	isDir   bool

	sys interface{}
}

func (i *IndexFileInfo) Name() string {
	return i.name
}

func (i *IndexFileInfo) Size() int64 {
	return i.size
}

func (i *IndexFileInfo) ModTime() time.Time {
	return i.modtime
}

func (i *IndexFileInfo) IsDir() bool {
	return i.isDir
}

func (i *IndexFileInfo) Mode() os.FileMode {
	return i.mode
}

func (i *IndexFileInfo) Sys() interface{} {
	return i.sys
}

// var indexHTML IndexFile

var indexString string

func (index *IndexFile) Stat() (fs.FileInfo, error) {
	fi := &IndexFileInfo{
		name:    "index.html",
		size:    int64(len(index.Contents)),
		isDir:   false,
		modtime: time.Now(),
		mode:    os.ModePerm,
		sys:     nil,
	}

	return fi, nil
}

func AssetHandler(prefix, root string) http.Handler {
	handler := fsFunc(func(name string) (fs.File, error) {
		assetPath := path.Join(root, name)

		f, err := Assets.Open(assetPath)
		if os.IsNotExist(err) || strings.Contains(assetPath, "index.html") {

			indexHTML := IndexFile{
				Contents: []byte(indexString),
			}

			indexHTML.ReadCloser = io.NopCloser(bytes.NewReader(indexHTML.Contents))

			// if asset does not exist open index.html
			return &indexHTML, nil
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

	indexString = string(data)

	handler := AssetHandler("/", "build")

	srv := &http.Server{
		Handler: handler,
		Addr:    fmt.Sprintf("%s:%s", os.Getenv("SERVE_ADDRESS"), os.Getenv("SERVE_PORT")),
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	k, c, _ := util.CertsForComponent(util.TLSHttpComponent)
	if len(k) > 0 {
		log.Printf("starting ui, tls enabled\n")
		log.Fatal(srv.ListenAndServeTLS(c, k))
	}

	fmt.Println("starting ui\n")
	log.Fatal(srv.ListenAndServe())

}
