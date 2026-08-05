package base

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Router struct {
	*gin.RouterGroup
	engine *gin.Engine
}
type Controller struct {
	r *Router
	h Handler
}
type HandlerFunc func(c *Context)

func NewRouter(e *gin.Engine) *Router {
	return &Router{
		RouterGroup: &e.RouterGroup,
		engine:      e,
	}
}

func (r *Router) GET(relativePath string, handlers ...HandlerFunc) *Router {
	r.wrapRoute(http.MethodGet, relativePath, handlers...)
	return r
}

func (r *Router) POST(relativePath string, handlers ...HandlerFunc) *Router {
	r.wrapRoute(http.MethodPost, relativePath, handlers...)
	return r
}

func (r *Router) PUT(relativePath string, handlers ...HandlerFunc) *Router {
	r.wrapRoute(http.MethodPut, relativePath, handlers...)
	return r
}

func (r *Router) PATCH(relativePath string, handlers ...HandlerFunc) *Router {
	r.wrapRoute(http.MethodPatch, relativePath, handlers...)
	return r
}

func (r *Router) HEAD(relativePath string, handlers ...HandlerFunc) *Router {
	r.wrapRoute(http.MethodHead, relativePath, handlers...)
	return r
}

func (r *Router) OPTIONS(relativePath string, handlers ...HandlerFunc) *Router {
	r.wrapRoute(http.MethodOptions, relativePath, handlers...)
	return r
}

func (r *Router) DELETE(relativePath string, handlers ...HandlerFunc) *Router {
	r.wrapRoute(http.MethodDelete, relativePath, handlers...)
	return r
}

func (r *Router) CONNECT(relativePath string, handlers ...HandlerFunc) *Router {
	r.wrapRoute(http.MethodConnect, relativePath, handlers...)
	return r
}

func (r *Router) TRACE(relativePath string, handlers ...HandlerFunc) *Router {
	r.wrapRoute(http.MethodTrace, relativePath, handlers...)
	return r
}

func (r *Router) Use(handlers ...HandlerFunc) *Router {
	r.wrapRoute("use", "", handlers...)
	return r
}

func (r *Router) NoRoute(handlers ...HandlerFunc) *Router {
	hds := make([]gin.HandlerFunc, 0, len(handlers))
	for _, hd := range handlers {
		hds = append(hds, wrapHandler(hd))
	}
	if r.engine != nil {
		r.engine.NoRoute(hds...)
	}
	return r
}

func (r *Router) Group(relativePath string, handlers ...HandlerFunc) *Router {
	g := r.wrapRoute("group", relativePath, handlers...).(*gin.RouterGroup)
	return &Router{
		RouterGroup: g,
		engine:      r.engine,
	}
}

func (r *Router) wrapRoute(method string, relativePath string, handlers ...HandlerFunc) gin.IRoutes {
	hds := make([]gin.HandlerFunc, 0, len(handlers))
	for _, hd := range handlers {
		hds = append(hds, wrapHandler(hd))
	}
	switch method {
	case http.MethodGet:
		return r.RouterGroup.GET(relativePath, hds...)
	case http.MethodPost:
		return r.RouterGroup.POST(relativePath, hds...)
	case http.MethodPut:
		return r.RouterGroup.PUT(relativePath, hds...)
	case http.MethodPatch:
		return r.RouterGroup.PATCH(relativePath, hds...)
	case http.MethodHead:
		return r.RouterGroup.HEAD(relativePath, hds...)
	case http.MethodOptions:
		return r.RouterGroup.OPTIONS(relativePath, hds...)
	case http.MethodDelete:
		return r.RouterGroup.DELETE(relativePath, hds...)
	case http.MethodConnect:
		return r.Handle(http.MethodConnect, relativePath, hds...)
	case "use":
		return r.RouterGroup.Use(hds...)
	case "group":
		return r.RouterGroup.Group(relativePath, hds...)
	}
	return r.Handle(http.MethodTrace, relativePath, hds...)
}

func wrapHandler(hd HandlerFunc) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		hd(&Context{Context: ctx})
	}
}

func (r *Router) Controller(h Handler) *Controller {
	return &Controller{r: r, h: h}
}

func (c *Controller) GET(relativePath, action string) *Controller {
	c.r.GET(relativePath, c.cloneHandler(action, c.h))
	return c
}

func (c *Controller) POST(relativePath, action string) *Controller {
	c.r.POST(relativePath, c.cloneHandler(action, c.h))
	return c
}

func (c *Controller) PUT(relativePath, action string) *Controller {
	c.r.PUT(relativePath, c.cloneHandler(action, c.h))
	return c
}

func (c *Controller) PATCH(relativePath, action string) *Controller {
	c.r.PATCH(relativePath, c.cloneHandler(action, c.h))
	return c
}

func (c *Controller) HEAD(relativePath, action string) *Controller {
	c.r.HEAD(relativePath, c.cloneHandler(action, c.h))
	return c
}

func (c *Controller) OPTIONS(relativePath, action string) *Controller {
	c.r.OPTIONS(relativePath, c.cloneHandler(action, c.h))
	return c
}

func (c *Controller) DELETE(relativePath, action string) *Controller {
	c.r.DELETE(relativePath, c.cloneHandler(action, c.h))
	return c
}

func (c *Controller) CONNECT(relativePath, action string) *Controller {
	c.r.CONNECT(relativePath, c.cloneHandler(action, c.h))
	return c
}

func (c *Controller) TRACE(relativePath, action string) *Controller {
	c.r.TRACE(relativePath, c.cloneHandler(action, c.h))
	return c
}

func (c *Controller) Use(action string) *Controller {
	c.r.wrapRoute("use", "", c.cloneHandler(action, c.h))
	return c
}

func (c *Controller) Group(relativePath, action string) *Controller {
	g := c.r.wrapRoute("group", relativePath, c.cloneHandler(action, c.h)).(*gin.RouterGroup)
	r := &Router{
		RouterGroup: g,
	}
	return r.Controller(c.h)
}

func (c *Controller) Resource(relativePath string) *Controller {
	c.GET(relativePath, http.MethodGet)
	c.POST(relativePath, http.MethodPost)
	c.PUT(relativePath, http.MethodPut)
	c.PATCH(relativePath, http.MethodPatch)
	c.DELETE(relativePath, http.MethodDelete)
	return c
}

func (c *Controller) cloneHandler(action string, h Handler) HandlerFunc {
	return func(c *Context) {
		hd := h.Clone().GetHandler(action)
		if hd != nil {
			hd(c)
		} else {
			c.Error(http.StatusInternalServerError, action+" handler not implemented", gin.H{})
		}
	}
}
