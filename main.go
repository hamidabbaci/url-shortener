package main

import (
	"fmt"
	"time"
	"url-shortener/store"

	"github.com/gin-gonic/gin"
)

func main() {

	s := store.NewMemoryStore()

	err := s.Save(store.Link{
		Code:        "abc123",
		OriginalURL: "https://example.com",
		CreatedAt:   time.Now(),
	})
	fmt.Println("save error:", err)

	link, err := s.Get("abc123")
	fmt.Println("found:", link, err)

	_, err = s.Get("nope")
	fmt.Println("not found error:", err)

	err = s.Save(store.Link{Code: "abc123"})
	fmt.Println("duplicate error:", err)

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "OK",
		})
	})

	//ping
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	r.Run(":8080")
}
