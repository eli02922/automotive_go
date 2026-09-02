package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/embuscado/automotive-catalog/internal/catalog/model"
	"github.com/embuscado/automotive-catalog/internal/catalog/service"
	apperr "github.com/embuscado/automotive-catalog/pkg/errors"
)

type CatalogHandler struct {
	svc *service.ProductService
}

func NewCatalogHandler(svc *service.ProductService) *CatalogHandler {
	return &CatalogHandler{svc: svc}
}

func (h *CatalogHandler) CreateProduct(c *gin.Context) {
	var req model.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	p, err := h.svc.Create(c.Request.Context(), &req)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, p)
}

func (h *CatalogHandler) GetProduct(c *gin.Context) {
	id := c.Param("id")
	p, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *CatalogHandler) ListProducts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))

	filter := model.ProductFilter{
		Brand:      c.Query("brand"),
		CategoryID: c.Query("category_id"),
		Search:     c.Query("q"),
		Status:     model.ProductStatus(c.Query("status")),
		SortBy:     c.DefaultQuery("sort_by", "created_at"),
		SortOrder:  c.DefaultQuery("sort_order", "desc"),
		Page:       page,
		PageSize:   pageSize,
	}

	result, err := h.svc.List(c.Request.Context(), filter)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *CatalogHandler) UpdateProduct(c *gin.Context) {
	id := c.Param("id")
	var req model.UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	p, err := h.svc.Update(c.Request.Context(), id, &req)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *CatalogHandler) DeleteProduct(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *CatalogHandler) UpsertFitment(c *gin.Context) {
	var req model.UpsertFitmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	f, err := h.svc.UpsertFitment(c.Request.Context(), &req)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, f)
}

func (h *CatalogHandler) GetFitments(c *gin.Context) {
	productID := c.Param("id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "100"))
	_ = productID
	_ = page
	_ = pageSize
	// delegate to fitment service or repo directly
	c.JSON(http.StatusOK, gin.H{"message": "not implemented"})
}

func respondError(c *gin.Context, err error) {
	if ae, ok := err.(*apperr.AppError); ok {
		c.JSON(ae.Code, gin.H{"error": ae.Message, "detail": ae.Detail})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}
