package route

import (
	"github.com/caitunai/dotconfig/api/base"
	"github.com/caitunai/dotconfig/api/handler"
)

func InitRoute(r *base.Router) {
	InitMiddleware()
	r.Use(AttemptAuth())

	initPackageHandler(r)
	initConfigCenterHandler(r)
	initAPIHandler(r)
}

func initPackageHandler(r *base.Router) {
	r.GET("/", handler.HomePage)
	r.GET("/assets/*filepath", handler.ServeAssetFile)
	r.HEAD("/assets/*filepath", handler.ServeAssetFile)
	r.NoRoute(handler.ServeRootStaticFiles)
}

func initConfigCenterHandler(r *base.Router) {
	configCenter := r.Group("/config-center", configCenterEnabled, configCenterNoStore)
	if configCenterAuth != nil {
		configCenter.RouterGroup.Use(configCenterAuth)
	}
	configCenter.GET("", handler.ConfigCenterPage)
	handler.ConfigCenterControl(configCenter.Group("/api"))

	configCenterRuntime := r.Group("/config-center/api", configCenterEnabled)
	handler.ConfigCenterRuntimeControl(configCenterRuntime)
}

func initAPIHandler(r *base.Router) {
	api := r.Group("/api", apiAuthorized)
	api.GET("/", handler.APIHomePage)

	// Add CRUD services for db.User
	handler.UserControl(api.Group("/users"))
}
