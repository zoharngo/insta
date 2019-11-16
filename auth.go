package main

import (
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// AuthRequired an authentication middleware. If the JWT token is invalid, the
// user is redirected to /signup.
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {

		log.Debug("AuthRequired()")

		s := sessions.Default(c)
		jwt := s.Get(accessToken)

		if jwt == nil {
			log.Error("Access token not found in session")
			c.Redirect(http.StatusFound, "/signup")
			return
		}

		cog := NewCognito()
		sub, err := cog.ValidateToken(jwt.(string))

		if err != nil {
			log.Error("Error validating token: ", err)
			c.Redirect(http.StatusFound, "/signup")
			return
		}

		if sub == "" {
			log.Error("sub not found: ", err)
			c.Redirect(http.StatusFound, "/signup")
			return
		}

		c.Next()
	}
}
