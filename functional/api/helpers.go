package api

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetQueryInt extracts an integer query parameter with a default value
func GetQueryInt(c *gin.Context, key string, defaultValue int) int {
	value := c.DefaultQuery(key, strconv.Itoa(defaultValue))
	intVal, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intVal
}