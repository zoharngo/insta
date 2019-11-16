package main

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

func registerRoutes() *gin.Engine {

	log.Info("Registering routes")

	r := gin.Default()
	store := cookie.NewStore([]byte("viErkShjgQP59tgelRXsILXNEarwRA6p"))
	r.Use(sessions.Sessions("photos-session", store))

	r.NoRoute(noroute)

	r.LoadHTMLGlob("templates/**/*.html")

	r.Static("/public", "./public")

	r.GET("/", home)

	r.GET("/login", loginForm)
	r.POST("/login", login)
	r.GET("/logout", logout)

	r.GET("/signup", signupForm)
	r.POST("/signup", signup)

	user := r.Group("/user", AuthRequired())
	{
		user.GET("/:username", Profile)
		user.POST("/:id/follow", Follow)
		user.POST("/:id/unfollow", Unfollow)
	}

	photos := r.Group("/photos", AuthRequired())
	{
		photos.POST("/", CreatePhoto)
		photos.GET("/", FetchAllPhotos)
		photos.GET("/:id", FetchSinglePhoto)
		photos.DELETE("/:id", DeletePhoto)
		photos.POST("/:id/like", LikePhoto)
		photos.POST("/:id/comment", CommentPhoto)
	}

	return r
}

func home(c *gin.Context) {
	session := sessions.Default(c)
	u := session.Get(userKey)

	if u != nil {
		log.Debugf("user: %v", u)
		user, err := findUserByID(u.(string))

		if err != nil {
			log.Error("Error getting user:", err.Error())
			c.Redirect(302, "/signup")
			return
		}

		log.Debugf("Found session user: %v", user)
		c.Redirect(302, "/photos")
	} else {
		c.Redirect(302, "/signup")
	}
}

func noroute(c *gin.Context) {
	c.HTML(http.StatusNotFound, "404.html", nil)
}
