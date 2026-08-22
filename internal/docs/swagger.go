package docs

import (
	_ "embed"
	"net/http"
	"strings"
)

//go:embed openapi.json
var openAPISpec []byte

// ServeOpenAPIHandler serves the OpenAPI specification as JSON.
func ServeOpenAPIHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(openAPISpec)
}

// SwaggerUIHandler serves a simple Swagger UI page that loads the OpenAPI spec.
func SwaggerUIHandler() http.Handler {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Needly API - Swagger UI</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = function() {
      window.ui = SwaggerUIBundle({
        url: "/docs/openapi.json",
        dom_id: "#swagger-ui",
        deepLinking: true,
        presets: [
          SwaggerUIBundle.presets.apis,
          SwaggerUIBundle.SwaggerUIStandalonePreset
        ],
        layout: "StandaloneLayout"
      });
    };
  </script>
</body>
</html>`

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(html))
	})
}

// RedocUIHandler serves a Redoc HTML page that loads the OpenAPI spec.
func RedocUIHandler() http.Handler {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Needly API Documentation</title>
  <style>body{margin:0;padding:0}</style>
</head>
<body>
  <redoc spec-url='/docs/openapi.json'></redoc>
  <script src="https://cdn.redoc.ly/redoc/v2.1/bundle.js"></script>
  <p style="text-align:center;margin:8px;font-family:sans-serif">
    <a href="/docs">Swagger UI</a> | <a href="/docs/redoc">Redoc</a>
  </p>
</body>
</html>`

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(html))
	})
}

// OpenAPISpec returns the raw OpenAPI specification bytes.
func OpenAPISpec() []byte {
	return openAPISpec
}

// OpenAPISpecString returns the OpenAPI specification as a string.
func OpenAPISpecString() string {
	return string(openAPISpec)
}

// IsOpenAPIPath returns true if the path is part of the API documentation.
func IsOpenAPIPath(path string) bool {
	return strings.HasPrefix(path, "/docs") || strings.HasPrefix(path, "/swagger")
}
