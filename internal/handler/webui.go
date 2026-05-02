package handler

import (
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterWebUI(r *gin.Engine, webFS fs.FS) {
	fileServer := http.FileServer(http.FS(webFS))

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == "/" || path == "/index.html" {
			c.FileFromFS("/", http.FS(webFS))
			return
		}
		fileServer.ServeHTTP(c.Writer, c.Request)
	})
}
