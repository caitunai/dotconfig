package handler

import (
	"strconv"

	"github.com/caitunai/dotconfig/api/base"
	"github.com/caitunai/dotconfig/db"
	"github.com/gin-gonic/gin"
)

const (
	KeyModel = "model"
	KeyError = "error"
)

type IDQueryAll struct {
	IDs []uint `form:"id[]" json:"ids" binding:"required"`
}

type CrudController[M db.IDModel, C db.InputConverter[M], U db.InputConverter[M], V db.ViewConverter[M, V], S db.Searcher] struct {
	Service *db.CrudService[M, C, U, V, S]
}

func NewCrudController[M db.IDModel, C db.InputConverter[M], U db.InputConverter[M], V db.ViewConverter[M, V], S db.Searcher](s *db.CrudService[M, C, U, V, S]) *CrudController[M, C, U, V, S] {
	return &CrudController[M, C, U, V, S]{Service: s}
}

func (ctrl *CrudController[M, C, U, V, S]) RegisterRoutes(r *base.Router) {
	r.POST("", ctrl.Create)
	r.GET("/:id", ctrl.Get)
	r.GET("/list", ctrl.GetAll)
	r.PUT("/:id", ctrl.Update)
	r.DELETE("/:id", ctrl.Delete)
	r.GET("", ctrl.List)
}

// Create data to database
func (ctrl *CrudController[M, C, U, V, S]) Create(c *base.Context) {
	var input C
	if err := c.ShouldBindJSON(&input); err != nil {
		c.ErrorForm(err.Error(), gin.H{})
		return
	}
	view, err := ctrl.Service.Create(c, input)
	if err != nil {
		c.ErrorMessage(err.Error())
		return
	}
	c.Success(gin.H{
		KeyModel: view,
	})
}

// Get one item details, with path param: /:id
func (ctrl *CrudController[M, C, U, V, S]) Get(c *base.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	view, err := ctrl.Service.Get(c, uint(id))
	if err != nil {
		c.NotFound(err.Error(), gin.H{KeyError: "not found"})
		return
	}
	c.Success(gin.H{
		KeyModel: view,
	})
}

// GetAll Get details of items in array, with query params: ?ids[]=1&ids[]=2
func (ctrl *CrudController[M, C, U, V, S]) GetAll(c *base.Context) {
	q := &IDQueryAll{}
	if err := c.ShouldBind(q); err != nil {
		c.ErrorForm(err.Error(), gin.H{})
		return
	}
	view, err := ctrl.Service.GetAll(c, q.IDs)
	if err != nil {
		c.NotFound(err.Error(), gin.H{KeyError: "not found"})
		return
	}
	c.Success(gin.H{
		"models": view,
	})
}

// Update item details, with path param: /:id
func (ctrl *CrudController[M, C, U, V, S]) Update(c *base.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var input U
	if err := c.ShouldBindJSON(&input); err != nil {
		c.ErrorForm(err.Error(), gin.H{})
		return
	}
	view, err := ctrl.Service.Update(c, uint(id), input)
	if err != nil {
		c.ErrorMessage(err.Error())
		return
	}
	c.Success(gin.H{
		KeyModel: view,
	})
}

// Delete one item with path param: /:id
func (ctrl *CrudController[M, C, U, V, S]) Delete(c *base.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.ErrorForm("invalid id", gin.H{})
		return
	}

	if err := ctrl.Service.Delete(uint(id)); err != nil {
		// Determine whether it's a “not found” error or a system error
		if err.Error() == "record not found" {
			c.NotFound("record not found", gin.H{})
			return
		}
		c.ErrorMessage(err.Error())
		return
	}

	c.Success(gin.H{
		"id": id,
	})
}

// List : Automatically bind pagination and search parameters
func (ctrl *CrudController[M, C, U, V, S]) List(c *base.Context) {
	// 1. Bind pagination parameters
	var pageReq db.PagingRequest
	if err := c.ShouldBindQuery(&pageReq); err != nil {
		c.ErrorForm("invalid paging params", gin.H{})
		return
	}

	// 2. Bind search parameters (Generic S)
	searchPtr := ctrl.Service.NewSearcher()
	if err := c.ShouldBindQuery(searchPtr); err != nil {
		c.ErrorForm("invalid search params", gin.H{})
		return
	}

	// 3. Call service
	res, err := ctrl.Service.ListCursor(pageReq, searchPtr)
	if err != nil {
		c.ErrorMessage(err.Error())
		return
	}
	c.Success(gin.H{
		"pagination": res,
	})
}
