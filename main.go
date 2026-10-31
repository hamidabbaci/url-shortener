package main

import (
	"errors"
	"net/http"

	"time"

	"github.com/gin-gonic/gin"

	"url-shortener/store"
)

type createRequest struct {
	URL string `json:"url" binding:"required"`
}

func main() {
	s := store.NewMemoryStore()
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	// ساخت لینک کوتاه
	r.POST("/links", func(c *gin.Context) {
		var req createRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "url is required"})
			return
		}

		code, err := store.GenerateCode(6)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not generate code"})
			return
		}

		link := store.Link{
			Code:        code,
			OriginalURL: req.URL,
			CreatedAt:   time.Now().UTC(),
		}
		if err := s.Save(link); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not save link"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"short_code": code,
			"short_url":  "http://localhost:8080/" + code,
		})
	})

	// ریدایرکت
	r.GET("/:code", func(c *gin.Context) {
		link, err := s.Get(c.Param("code"))
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "link not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		c.Redirect(http.StatusFound, link.OriginalURL)
	})

	r.Run(":8080")
}
