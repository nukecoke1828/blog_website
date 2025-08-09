package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func ProfileHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "profile.html", gin.H{
		"title":        "关于我",
		"name":         "???",
		"school":       "???",
		"hometown":     "???",
		"hobbies":      []string{"???", "???", "???"},
		"introduction": "???",
	})
}
