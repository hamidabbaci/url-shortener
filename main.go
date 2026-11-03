package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"url-shortener/middleware"
	"url-shortener/store"
)

const (
	maxURLLength = 2048
	maxBodyBytes = 4 << 10 // ۴ کیلوبایت
	codeLength   = 6
)

var codeRe = regexp.MustCompile(`^[A-Za-z0-9]{6}$`)

type createRequest struct {
	URL string `json:"url" binding:"required"`
}

func isValidURL(raw string) bool {
	if len(raw) > maxURLLength {
		return false
	}
	u, err := url.ParseRequestURI(raw)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	// آدرس خالی یا لینک با نام‌کاربری/رمز (http://user:pass@evil.com)
	if u.Host == "" || u.User != nil {
		return false
	}
	host := u.Hostname()
	if host == "localhost" {
		return false
	}
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() {
			return false
		}
	}
	return true
}

// تا ۳ بار برای کد تکراری دوباره تلاش می‌کند
func createLink(s *store.PostgresStore, rawURL string) (string, error) {
	now := time.Now().UTC()
	for i := 0; i < 3; i++ {
		code, err := store.GenerateCode(codeLength)
		if err != nil {
			return "", err
		}
		err = s.Save(store.Link{
			Code:        code,
			OriginalURL: rawURL,
			CreatedAt:   now,
			ExpiresAt:   now.Add(24 * time.Hour),
		})
		if err == nil {
			return code, nil
		}
		if !errors.Is(err, store.ErrCodeExists) {
			return "", err
		}
	}
	return "", store.ErrCodeExists
}

func main() {
	// تو Docker فایل .env وجود ندارد، پس نبودنش خطا نیست
	_ = godotenv.Load()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:" + port
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	s, err := store.NewPostgresStore(context.Background(), dsn)
	if err != nil {
		log.Fatal("Failed to connect to database: ", err)
	}
	defer s.Close()

	r := gin.Default()
	// به هدر X-Forwarded-For اعتماد نکن (جلوی جعل IP)
	if err := r.SetTrustedProxies(nil); err != nil {
		log.Fatal(err)
	}
	r.Use(middleware.SecurityHeaders())

	limiter := middleware.NewRateLimiter(1, 5)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	r.POST("/links", limiter.Middleware(), func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)

		var req createRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "url is required"})
			return
		}
		if !isValidURL(req.URL) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid url"})
			return
		}

		code, err := createLink(s, req.URL)
		if err != nil {
			log.Printf("create link failed: %v", err) // جزئیات فقط تو لاگ
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create link"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"short_code": code,
			"short_url":  baseURL + "/" + code,
		})
	})

	r.GET("/:code", func(c *gin.Context) {
		code := c.Param("code")
		if !codeRe.MatchString(code) {
			c.JSON(http.StatusNotFound, gin.H{"error": "link not found"})
			return
		}

		link, err := s.Get(code)
		if err != nil {
			switch {
			case errors.Is(err, store.ErrNotFound):
				c.JSON(http.StatusNotFound, gin.H{"error": "link not found"})
			case errors.Is(err, store.ErrExpired):
				c.JSON(http.StatusGone, gin.H{"error": "link expired"})
			default:
				log.Printf("get link failed: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			}
			return
		}
		c.Redirect(http.StatusFound, link.OriginalURL)
	})

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	// منتظر Ctrl+C یا Stop تو GoLand، بعد خاموشی تمیز
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Println("forced shutdown:", err)
	}
}
