package swagger

import (
	"embed"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
)

// =============================================================================
// Swagger UI Integration
// =============================================================================
// This package provides Swagger UI for API documentation in development.
//
// Usage:
//
//	app.Use("/docs", swagger.Handler(swagger.Config{
//	    SpecURL: "/api/openapi.yaml",
//	    Title:   "EquiShare API",
//	}))
//
// Or serve embedded specs:
//
//	//go:embed openapi/*.yaml
//	var specs embed.FS
//
//	app.Use("/docs", swagger.Handler(swagger.Config{
//	    SpecFS:  specs,
//	    SpecDir: "openapi",
//	    Title:   "EquiShare API",
//	}))
// =============================================================================

// Config holds Swagger UI configuration
type Config struct {
	// SpecURL is the URL to the OpenAPI spec (if serving from URL)
	SpecURL string

	// SpecFS is an embedded filesystem containing spec files
	SpecFS embed.FS

	// SpecDir is the directory within SpecFS containing specs
	SpecDir string

	// Title shown in Swagger UI
	Title string

	// BasePath is the base path where Swagger UI is served
	BasePath string
}

// Handler returns a Fiber handler that serves Swagger UI
func Handler(config Config) fiber.Handler {
	if config.Title == "" {
		config.Title = "API Documentation"
	}
	if config.BasePath == "" {
		config.BasePath = "/docs"
	}

	return func(c *fiber.Ctx) error {
		path := c.Path()

		// Serve OpenAPI specs from embedded FS
		if config.SpecFS != (embed.FS{}) {
			if strings.HasPrefix(path, config.BasePath+"/spec/") {
				specPath := strings.TrimPrefix(path, config.BasePath+"/spec/")
				return serveSpec(c, config.SpecFS, config.SpecDir, specPath)
			}
		}

		// Serve Swagger UI HTML
		if path == config.BasePath || path == config.BasePath+"/" {
			return serveSwaggerUI(c, config)
		}

		return c.Next()
	}
}

func serveSwaggerUI(c *fiber.Ctx, config Config) error {
	specURL := config.SpecURL
	if specURL == "" && config.SpecFS != (embed.FS{}) {
		specURL = config.BasePath + "/spec/gateway.yaml"
	}

	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>` + config.Title + `</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
    <style>
        html { box-sizing: border-box; overflow-y: scroll; }
        *, *:before, *:after { box-sizing: inherit; }
        body { margin: 0; background: #fafafa; }
        .swagger-ui .topbar { display: none; }
        .swagger-ui .info { margin: 20px 0; }
        .swagger-ui .info .title { font-size: 2em; }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            SwaggerUIBundle({
                url: "` + specURL + `",
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout",
                persistAuthorization: true,
                displayRequestDuration: true,
                filter: true,
                tryItOutEnabled: true
            });
        };
    </script>
</body>
</html>`

	c.Set("Content-Type", "text/html")
	return c.SendString(html)
}

func serveSpec(c *fiber.Ctx, fsys embed.FS, dir, path string) error {
	fullPath := filepath.Join(dir, path)
	data, err := fsys.ReadFile(fullPath)
	if err != nil {
		return c.Status(fiber.StatusNotFound).SendString("Spec not found")
	}

	// Set content type based on extension
	if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
		c.Set("Content-Type", "application/x-yaml")
	} else if strings.HasSuffix(path, ".json") {
		c.Set("Content-Type", "application/json")
	}

	return c.Send(data)
}

// ServeSpecs returns a middleware that serves OpenAPI specs from an embedded FS
func ServeSpecs(fsys embed.FS, dir string) fiber.Handler {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic("failed to create sub filesystem: " + err.Error())
	}

	return filesystem.New(filesystem.Config{
		Root:       http.FS(sub),
		PathPrefix: dir,
		Browse:     false,
	})
}

// RedocHandler returns a handler that serves Redoc documentation
func RedocHandler(config Config) fiber.Handler {
	if config.Title == "" {
		config.Title = "API Documentation"
	}

	return func(c *fiber.Ctx) error {
		specURL := config.SpecURL
		if specURL == "" && config.SpecFS != (embed.FS{}) {
			specURL = config.BasePath + "/spec/gateway.yaml"
		}

		html := `<!DOCTYPE html>
<html>
<head>
    <title>` + config.Title + `</title>
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <link href="https://fonts.googleapis.com/css?family=Montserrat:300,400,700|Roboto:300,400,700" rel="stylesheet">
    <style>
        body { margin: 0; padding: 0; }
    </style>
</head>
<body>
    <redoc spec-url='` + specURL + `'></redoc>
    <script src="https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js"></script>
</body>
</html>`

		c.Set("Content-Type", "text/html")
		return c.SendString(html)
	}
}
