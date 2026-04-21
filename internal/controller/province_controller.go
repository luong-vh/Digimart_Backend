package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/luong-vh/Digimart_Backend/internal/apperror"
	"github.com/luong-vh/Digimart_Backend/internal/dto"
	"github.com/luong-vh/Digimart_Backend/internal/service"
)

type ProvinceController struct {
	service service.ProvinceService
}

func NewProvinceController(service service.ProvinceService) *ProvinceController {
	return &ProvinceController{service: service}
}

// ==================== Province Endpoints ====================

// CreateProvince creates a new province
func (c *ProvinceController) CreateProvince(ctx *gin.Context) {
	var req dto.CreateProvinceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		dto.SendError(ctx, http.StatusBadRequest, "Invalid request payload", apperror.ErrBadRequest.Code)
		return
	}

	province, err := c.service.Create(&req)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusCreated, "Province created successfully", province)
}

// GetAllProvinces retrieves all provinces
func (c *ProvinceController) GetAllProvinces(ctx *gin.Context) {
	provinces, err := c.service.GetAll()
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Provinces retrieved successfully", provinces)
}

// GetProvinceByID retrieves a province by its ID
func (c *ProvinceController) GetProvinceByID(ctx *gin.Context) {
	id := ctx.Param("id")

	province, err := c.service.GetByID(id)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Province retrieved successfully", province)
}

// UpdateProvince updates an existing province
func (c *ProvinceController) UpdateProvince(ctx *gin.Context) {
	id := ctx.Param("id")

	var req dto.UpdateProvinceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		dto.SendError(ctx, http.StatusBadRequest, "Invalid request payload", apperror.ErrBadRequest.Code)
		return
	}

	province, err := c.service.Update(id, &req)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Province updated successfully", province)
}

// DeleteProvince deletes a province
func (c *ProvinceController) DeleteProvince(ctx *gin.Context) {
	id := ctx.Param("id")

	if err := c.service.Delete(id); err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Province deleted successfully", gin.H{"id": id})
}

// ==================== Ward Endpoints ====================

// AddWard adds a new ward to a province
func (c *ProvinceController) AddWard(ctx *gin.Context) {
	provinceID := ctx.Param("id")

	var req dto.CreateWardRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		dto.SendError(ctx, http.StatusBadRequest, "Invalid request payload", apperror.ErrBadRequest.Code)
		return
	}

	ward, err := c.service.AddWard(provinceID, &req)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusCreated, "Ward added successfully", ward)
}

// GetWardsByProvinceID retrieves all wards of a province
func (c *ProvinceController) GetWardsByProvinceID(ctx *gin.Context) {
	provinceID := ctx.Param("id")

	wards, err := c.service.GetWardsByProvinceID(provinceID)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Wards retrieved successfully", wards)
}

// GetWardByID retrieves a specific ward by its ID
func (c *ProvinceController) GetWardByID(ctx *gin.Context) {
	provinceID := ctx.Param("id")
	wardID := ctx.Param("wardId")

	ward, err := c.service.GetWardByID(provinceID, wardID)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Ward retrieved successfully", ward)
}

// UpdateWard updates an existing ward
func (c *ProvinceController) UpdateWard(ctx *gin.Context) {
	provinceID := ctx.Param("id")
	wardID := ctx.Param("wardId")

	var req dto.UpdateWardRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		dto.SendError(ctx, http.StatusBadRequest, "Invalid request payload", apperror.ErrBadRequest.Code)
		return
	}

	ward, err := c.service.UpdateWard(provinceID, wardID, &req)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Ward updated successfully", ward)
}

// DeleteWard deletes a ward from a province
func (c *ProvinceController) DeleteWard(ctx *gin.Context) {
	provinceID := ctx.Param("id")
	wardID := ctx.Param("wardId")

	if err := c.service.DeleteWard(provinceID, wardID); err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Ward deleted successfully", gin.H{
		"province_id": provinceID,
		"ward_id":     wardID,
	})
}
