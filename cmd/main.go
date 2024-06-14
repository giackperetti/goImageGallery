package main

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"path/filepath"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"
)

type Template struct {
	Templates *template.Template
}

type ImageGallery struct {
	ImagePaths  []string
	ImageNumber int
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.Templates.ExecuteTemplate(w, name, data)
}

func (i *ImageGallery) addImage(path string) {
	i.ImagePaths = append(i.ImagePaths, path)
	i.ImageNumber++
}

func (i *ImageGallery) addMultipleImages(directory string) error {
	var images []string

	walkDirFunc := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			images = append(images, path)
		}
		return nil
	}

	err := filepath.WalkDir(directory, walkDirFunc)
	if err != nil {
		return err
	}

	for _, img := range images {
		i.addImage(img)
	}

	return err
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

func loggingMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		start := time.Now()
		err := next(c)

		logMessage := fmt.Sprintf("[%s] %s %s %v",
			c.Request().Method,
			c.Path(),
			c.QueryString(),
			time.Since(start),
		)
		fmt.Println(logMessage)

		return err
	}
}

func main() {
	e := echo.New()

	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.Recover())
	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(
		rate.Limit(20),
	)))
	e.Use(loggingMiddleware)

	gallery := ImageGallery{}
	gallery.addMultipleImages("static/gallery/")

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
