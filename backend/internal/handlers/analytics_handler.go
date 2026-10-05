package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"job_application_analysis/internal/services"
)

type AnalyticsHandler struct {
	service *services.AnalyticsService
}

func NewAnalyticsHandler(service *services.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{service: service}
}

func (h *AnalyticsHandler) GetDashboardStats(c *gin.Context) {
	userID := c.GetUint("user_id")
	
	stats, err := h.service.GetDashboardStats(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch stats"})
		return
	}
	
	c.JSON(http.StatusOK, stats)
}

func (h *AnalyticsHandler) GetStatusDistribution(c *gin.Context) {
	userID := c.GetUint("user_id")
	
	distribution, err := h.service.GetStatusDistribution(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch distribution"})
		return
	}
	
	c.JSON(http.StatusOK, distribution)
}

func (h *AnalyticsHandler) GetTimeToResponse(c *gin.Context) {
	userID := c.GetUint("user_id")
	
	data, err := h.service.GetTimeToResponse(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch time data"})
		return
	}
	
	c.JSON(http.StatusOK, data)
}

func (h *AnalyticsHandler) GetWeeklySummary(c *gin.Context) {
	userID := c.GetUint("user_id")
	
	summary, err := h.service.GetWeeklySummary(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch weekly summary"})
		return
	}
	
	c.JSON(http.StatusOK, summary)
}
