package middlewares

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func PrettifyResponseJSON(c *gin.Context) {
	// Only replace the ResponseWriter if this request has the
	// "?pretty" query parameter set
	var modifier *BodyModifier
	if _, ok := c.GetQuery("pretty"); ok {
		modifier = NewBodyModifier(c.Writer)
		c.Writer = modifier
		defer modifier.Flush()
	}

	c.Next()

	if modifier != nil && strings.Contains(c.Writer.Header().Get("Content-Type"), "application/json") {
		modifier.Modify(func(buf *bytes.Buffer) (*bytes.Buffer, error) {
			pretty := new(bytes.Buffer)
			jsonErr := json.Indent(pretty, buf.Bytes(), "", "  ")
			if jsonErr != nil {
				logrus.Warnf("could not prettify JSON response: %v", jsonErr)
				return buf, nil
			}

			return pretty, nil
		})
	}
}
