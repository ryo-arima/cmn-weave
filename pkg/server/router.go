// Package server wires the REST API together: TLS listener, router, and
// handler registration. Business logic lives in usecase; outbound I/O lives
// in repository.
package server

import (
	"github.com/gin-gonic/gin"
	"github.com/ryo-arima/cmn-weave/pkg/server/controller"
	"github.com/ryo-arima/cmn-weave/pkg/server/share"
)

// InitRouter builds and returns the Gin engine with all routes registered.
// ctrl provides the request handler methods.
func InitRouter(ctrl *controller.CommonController) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	r.GET("/healthz", controller.Health)

	api := r.Group("/api/v1")
	api.Use(share.RequireMTLS())
	{
		// task routes
		api.POST("/dispatch", ctrl.Dispatch)
		api.GET("/tasks", ctrl.ListTasks)
		api.GET("/tasks/:task_id", ctrl.ShowTask)
		api.DELETE("/tasks/:task_id", ctrl.Cancel)

		// legacy aliases kept for backwards compatibility
		api.GET("/status/:task_id", ctrl.Status)
		api.DELETE("/cancel/:task_id", ctrl.Cancel)

		// node routes
		api.GET("/nodes", ctrl.ListNodes)
		api.GET("/nodes/:node_id", ctrl.ShowNode)

		// ga routes
		api.POST("/ga/floyd", ctrl.StartGAFloyd)

	}

	return r
}
