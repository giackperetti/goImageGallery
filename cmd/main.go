package main

import (
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"
)

type Template struct {
	Templates *template.Template
}

type ImageGallery struct {
	ImagesPaths  []string
	ImagesNumber int
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.Templates.ExecuteTemplate(w, name, data)
}

func (i *ImageGallery) addImage(path string) {
	i.ImagesPaths = append(i.ImagesPaths, path)
	i.ImagesNumber++
}

func newTemplate(templates *template.Template) echo.Renderer {
	return &Template{
		Templates: templates,
	}
}

func NewTemplateRenderer(e *echo.Echo, paths ...string) {
	tmpl := template.New("templates")
	for i := range paths {
		template.Must(tmpl.ParseGlob(paths[i]))
	}
	t := newTemplate(tmpl)
	e.Renderer = t
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

func main() {
	e := echo.New()

	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.Recover())
	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(
		rate.Limit(20),
	)))

	gallery := ImageGallery{}

	images, err := loadImagesFromDirectory("static")
	if err != nil {
		e.Logger.Fatal(err)
	}

	for _, img := range images {
		gallery.addImage(img)
	}

	NewTemplateRenderer(e, "views/*.html")
	e.Static("/static", "static")
	e.Static("/css", "css")

	e.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "index", nil)
	})

	e.GET("/get-gallery", func(c echo.Context) error {
		return c.Render(http.StatusOK, "gallery", gallery)
	})

	e.Logger.Fatal(e.Start(":12345"))
}
