package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
)

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		gin.DefaultWriter.Write([]byte("[NovelTTS] "))
		gin.DefaultWriter.Write([]byte(method))
		gin.DefaultWriter.Write([]byte(" "))
		gin.DefaultWriter.Write([]byte(path))
		gin.DefaultWriter.Write([]byte(" "))
		gin.DefaultWriter.Write([]byte(statusCodeStr(status)))
		gin.DefaultWriter.Write([]byte(" "))
		gin.DefaultWriter.Write([]byte(latency.String()))
		gin.DefaultWriter.Write([]byte("\n"))
	}
}

func statusCodeStr(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "\033[32m" + itoa(code) + "\033[0m"
	case code >= 400 && code < 500:
		return "\033[33m" + itoa(code) + "\033[0m"
	case code >= 500:
		return "\033[31m" + itoa(code) + "\033[0m"
	default:
		return itoa(code)
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	s := ""
	for i > 0 {
		s = string(rune('0'+i%10)) + s
		i /= 10
	}
	return s
}
