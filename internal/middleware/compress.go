package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

var gzipWriterPool = sync.Pool{
	New: func() interface{} {
		w, _ := gzip.NewWriterLevel(nil, gzip.BestSpeed)
		return w
	},
}

type gzipResponseWriter struct {
	gin.ResponseWriter
	io.Writer
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func (w *gzipResponseWriter) WriteHeader(code int) {
	w.ResponseWriter.Header().Del("Content-Length")
	w.ResponseWriter.WriteHeader(code)
}

func (w *gzipResponseWriter) Flush() {
	if fw, ok := w.Writer.(*gzip.Writer); ok {
		fw.Flush()
	}
	if fw, ok := w.ResponseWriter.(http.Flusher); ok {
		fw.Flush()
	}
}

// CompressionMiddleware compresses every response with gzip when the client
// supports it. It uses a sync.Pool of gzip writers for efficiency and skips
// WebSocket upgrades.
func CompressionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") {
			c.Next()
			return
		}
		if c.GetHeader("Upgrade") != "" {
			c.Next()
			return
		}

		w := gzipWriterPool.Get().(*gzip.Writer)
		w.Reset(c.Writer)

		c.Writer.Header().Set("Content-Encoding", "gzip")
		c.Writer.Header().Del("Content-Length")

		c.Writer = &gzipResponseWriter{ResponseWriter: c.Writer, Writer: w}

		c.Next()

		w.Close()
		gzipWriterPool.Put(w)
	}
}
