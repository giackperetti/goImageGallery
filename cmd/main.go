package main

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/time/rate"
)

type Template struct {
	Templates *template.Template
}

type ImageGallery struct {
	ImagesPaths  []string
	ImagesNumber int
}

func (t *Template) Render(w io.Writer, name string, data interface{}) error {
	return t.Templates.ExecuteTemplate(w, name, data)
}

func (i *ImageGallery) addImage(path string) {
	i.ImagesPaths = append(i.ImagesPaths, path)
	i.ImagesNumber++
}

func newTemplate(templates *template.Template) *Template {
	return &Template{
		Templates: templates,
	}
}

var templates *Template

func NewTemplateRenderer(paths ...string) *Template {
	if templates == nil {
		tmpl := template.New("templates")
		for _, p := range paths {
			template.Must(tmpl.ParseGlob(p))
		}
		templates = newTemplate(tmpl)
	}
	return templates
}

func loadImagesFromDirectory(directory string) ([]string, error) {
	var images []string
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			images = append(images, "/static/"+info.Name())
		}
		return nil
	})
	return images, err
}

func logRequest(r *http.Request) {
	fmt.Printf("Request: %s %s\n", r.Method, r.URL.Path)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	logRequest(r)
	templates.Render(w, "index.html", nil)
}

func galleryHandler(w http.ResponseWriter, r *http.Request) {
	logRequest(r)
	gallery := ImageGallery{}
	images, err := loadImagesFromDirectory("static/gallery")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, img := range images {
		gallery.addImage(img)
	}
	templates.Render(w, "gallery.html", gallery)
}

func serveStaticFolders(mux *http.ServeMux) {
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("/css/", func(w http.ResponseWriter, r *http.Request) {
		logRequest(r)
		http.ServeFile(w, r, r.URL.Path[1:])
	})
}

func main() {
	mux := http.NewServeMux()

	NewTemplateRenderer("views/*.html")
	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/get-gallery", galleryHandler)
	serveStaticFolders(mux)

	limiter := rate.NewLimiter(50, 1)

	fmt.Println("Server Started!")
	http.ListenAndServe(":12345", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		mux.ServeHTTP(w, r)
	}))
}
