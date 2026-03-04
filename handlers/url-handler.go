package handlers

import (
	"database/sql"
	"net/http"
	"url-shortener/config"
	"url-shortener/utils"

	"github.com/gin-gonic/gin"
	"github.com/skip2/go-qrcode"
)

type ShortenRequest struct {
	URL   string `json:"url"`
	Alias string `json:"alias"`
}

func ShortenURL(c *gin.Context) {
	var body ShortenRequest

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if !utils.IsValidURL(body.URL) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid URL format"})
		return
	}

	shortCode := body.Alias

	if shortCode == "" {
		shortCode = utils.GenerateShortCode(6)
	} else {
		var exists string
		err := config.DB.QueryRow(
			"SELECT short_code FROM urls WHERE short_code=$1",
			shortCode,
		).Scan(&exists)

		if err != sql.ErrNoRows {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Custom already taken"})
			return
		}
	}

	

	_, err := config.DB.Exec(
		"INSERT INTO urls ( original_url, short_code) VALUES ($1,$2)",
		 body.URL, shortCode,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save URL"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"short_url": shortCode,
	})
}

func RedirectURL(c *gin.Context) {
	code := c.Param("code")

	var original string

	err := config.DB.QueryRow(
		"SELECT original_url FROM urls WHERE short_code=$1",
		code,
	).Scan(&original)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "URL not found"})
		return
	}

	config.DB.Exec(
		"UPDATE urls SET hit_count = hit_count + 1 WHERE short_code=$1",
		code,
	)

	c.Redirect(http.StatusFound, original)
}

func GetStats(c *gin.Context) {
	code := c.Param("code")

	var hitCount int

	err := config.DB.QueryRow(
		"SELECT hit_count FROM urls WHERE short_code=$1",
		code,
	).Scan(&hitCount)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "URL not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"hit_count": hitCount,
	})
}

func GenerateQR(c *gin.Context) {
	code := c.Param("code")

	fullURL := "https://" + c.Request.Host + "/" + code

	png, err := qrcode.Encode(fullURL, qrcode.Medium, 256)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "QR generation failed"})
		return
	}

	c.Data(http.StatusOK, "image/png", png)
}