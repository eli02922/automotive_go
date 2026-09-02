package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/embuscado/automotive-catalog/internal/catalog/service"
)

type SearchHandler struct {
	svc *service.ProductService
}

func NewSearchHandler(svc *service.ProductService) *SearchHandler {
	return &SearchHandler{svc: svc}
}

// VehicleSearch finds products by year/make/model (fitment lookup).
func (h *SearchHandler) VehicleSearch(c *gin.Context) {
	yearStr := c.Query("year")
	make_ := c.Query("make")
	model_ := c.Query("model")

	if yearStr == "" || make_ == "" || model_ == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "year, make, and model are required"})
		return
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil || year < 1900 || year > 2100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid year"})
		return
	}

	products, err := h.svc.VehicleSearch(c.Request.Context(), year, make_, model_)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"year":     year,
		"make":     make_,
		"model":    model_,
		"count":    len(products),
		"products": products,
	})
}

func (h *SearchHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
