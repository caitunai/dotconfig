package handler

import (
	"fmt"

	"github.com/caitunai/dotconfig/api/base"
	"github.com/caitunai/dotconfig/db"
	"github.com/gin-gonic/gin"
)

func HomePage(c *base.Context) {
	var user *db.User
	if c.IsDatabaseEnabled() {
		user = c.LoginUser()
	}
	if user == nil {
		c.Forbidden("you are not login", gin.H{
			"db_enabled": c.IsDatabaseEnabled(),
		})
	} else {
		c.Success(gin.H{
			"data": fmt.Sprintf("your user id is: %d", user.ID),
		})
	}
}

func APIHomePage(c *base.Context) {
	user := c.GetAPIUser()
	if user == nil {
		c.Forbidden("you are not login", gin.H{})
	}
	c.Success(gin.H{
		"user": user,
	})
}
