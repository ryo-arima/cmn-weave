// Package controller hosts the HTTP handlers for the REST API.
//
// common.go holds CommonController, its constructor, the /healthz handler,
// and all resource-specific handler methods (task, node, ga).
package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ryo-arima/cmn-weave/pkg/entity/request"
	"github.com/ryo-arima/cmn-weave/pkg/server/usecase"
)

// CommonController holds the single CommonUsecase and exposes all gin handler methods.
type CommonController struct {
	uc *usecase.CommonUsecase
}

// New creates a CommonController backed by the given usecase.
func New(uc *usecase.CommonUsecase) *CommonController {
	return &CommonController{uc: uc}
}

// Health handles GET /healthz. It is exempt from mTLS.
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// ---------------------------------------------------------------------------
// Task handlers
// ---------------------------------------------------------------------------

// Dispatch handles POST /api/v1/dispatch.
func (rcvr *CommonController) Dispatch(c *gin.Context) {
	var req request.DispatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := rcvr.uc.Dispatch(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, resp)
}

// Status handles GET /api/v1/status/:task_id (legacy alias).
func (rcvr *CommonController) Status(c *gin.Context) {
	rcvr.ShowTask(c)
}

// ListTasks handles GET /api/v1/tasks.
func (rcvr *CommonController) ListTasks(c *gin.Context) {
	resp, err := rcvr.uc.ListTasks(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ShowTask handles GET /api/v1/tasks/:task_id.
func (rcvr *CommonController) ShowTask(c *gin.Context) {
	resp, err := rcvr.uc.ShowTask(c.Request.Context(), c.Param("task_id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// Cancel handles DELETE /api/v1/tasks/:task_id.
func (rcvr *CommonController) Cancel(c *gin.Context) {
	if err := rcvr.uc.Cancel(c.Request.Context(), c.Param("task_id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "cancel_requested"})
}

// ---------------------------------------------------------------------------
// Node handlers
// ---------------------------------------------------------------------------

// ListNodes handles GET /api/v1/nodes.
func (rcvr *CommonController) ListNodes(c *gin.Context) {
	resp, err := rcvr.uc.ListNodes(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ShowNode handles GET /api/v1/nodes/:node_id.
func (rcvr *CommonController) ShowNode(c *gin.Context) {
	resp, err := rcvr.uc.ShowNode(c.Request.Context(), c.Param("node_id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ---------------------------------------------------------------------------
// GA handlers
// ---------------------------------------------------------------------------

// StartGAFloyd handles POST /api/v1/ga/floyd.
func (rcvr *CommonController) StartGAFloyd(c *gin.Context) {
	var req request.GAFloydRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := rcvr.uc.StartGAFloyd(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, resp)
}

