package handlers

import (
	"net/http"
	"strconv"

	"job_application_analysis/internal/models"
	"job_application_analysis/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SuggestionHandler struct {
	Service *services.SuggestionService
}

func NewSuggestionHandler(db *gorm.DB) *SuggestionHandler {
	return &SuggestionHandler{
		Service: &services.SuggestionService{DB: db},
	}
}

//
// 1️⃣ Create Suggestion
//
func (h *SuggestionHandler) CreateSuggestion(c *gin.Context) {

	var suggestion models.SuggestedUpdate

	if err := c.ShouldBindJSON(&suggestion); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.Service.CreateSuggestion(&suggestion); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, suggestion)
}

//
// 2️⃣ Get Unconfirmed Suggestions
//
func (h *SuggestionHandler) GetSuggestions(c *gin.Context) {

	idParam := c.Query("application_id")

	id, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid application_id"})
		return
	}

	suggestions, err := h.Service.GetUnconfirmed(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, suggestions)
}

//
// 3️⃣ Confirm Suggestion
//
func (h *SuggestionHandler) ConfirmSuggestion(c *gin.Context) {

	idParam := c.Param("id")

	id, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	if err := h.Service.ConfirmSuggestion(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Suggestion confirmed"})
}
