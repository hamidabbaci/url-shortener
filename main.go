package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"url-shortener/store"
)

type createRequest struct {
	URL string `json:"url" binding:"required"`
}

func isValidURL(raw string) bool {
	u, err := url.ParseRequestURI(raw)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	return u.Host != ""
}

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	port := os.Getenv("PORT")

	if port == "" {
		port = "8080"
	}

	ctx := context.Background()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	s, err := store.NewPostgresStore(ctx, dsn)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	defer s.Close()

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

		if !isValidURL(req.URL) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid url"})
			return
		}

		code, err := store.GenerateCode(6)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not generate code"})
			return
		}

		now := time.Now().UTC()
		link := store.Link{
			Code:        code,
			OriginalURL: req.URL,
			CreatedAt:   now,
			ExpiresAt:   now.Add(24 * time.Hour),
		}
		if err := s.Save(link); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not save link"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"short_code": code,
			"short_url":  "http://localhost:" + port + "/" + code,
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
			if errors.Is(err, store.ErrExpired) {
				c.JSON(http.StatusGone, gin.H{"error": "link expired"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		c.Redirect(http.StatusFound, link.OriginalURL)
	})

	r.Run(":" + port)
}
