// Package configs contains all project configs
package configs

import (
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func ConfigHTTPServer(r *gin.Engine) *http.Server {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	timeout, err := strconv.Atoi(os.Getenv("SERVER_READ_TIMEOUT"))
	if err != nil || timeout <= 0 {
		timeout = 5
	}

	return &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  time.Duration(timeout) * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}
