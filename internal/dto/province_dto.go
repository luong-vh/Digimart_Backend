package dto

import (
	"github.com/luong-vh/Digimart_Backend/internal/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ==================== Request ====================

type CreateProvinceRequest struct {
	Name string `json:"name" binding:"required,min=2,max=100"`
}

type UpdateProvinceRequest struct {
	Name string `json:"name" binding:"required,min=2,max=100"`
}

type CreateWardRequest struct {
	Name string `json:"name" binding:"required,min=2,max=100"`
}

type UpdateWardRequest struct {
	Name string `json:"name" binding:"required,min=2,max=100"`
}

// ==================== Response ====================

type ProvinceResponse struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Wards     []WardResponse `json:"wards,omitempty"`
	WardCount int            `json:"ward_count"`
}

type ProvinceListResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	WardCount int    `json:"ward_count"`
}

type WardResponse struct {
	ID         string `json:"id"`
	ProvinceID string `json:"province_id,omitempty"`
	Name       string `json:"name"`
}

// ==================== Converters ====================

func (req *CreateProvinceRequest) ToModel() *model.Province {
	return &model.Province{
		Name:  req.Name,
		Wards: []model.Ward{},
	}
}

func (req *CreateWardRequest) ToModel(provinceID primitive.ObjectID) *model.Ward {
	return &model.Ward{
		ProvinceID: provinceID,
		Name:       req.Name,
	}
}

func FromProvince(p *model.Province) *ProvinceResponse {
	wards := make([]WardResponse, len(p.Wards))
	for i, w := range p.Wards {
		wards[i] = WardResponse{
			ID:         w.ID,
			ProvinceID: w.ProvinceID.Hex(),
			Name:       w.Name,
		}
	}

	return &ProvinceResponse{
		ID:        p.ID.Hex(),
		Name:      p.Name,
		Wards:     wards,
		WardCount: len(p.Wards),
	}
}

func FromProvinces(provinces []model.Province) []ProvinceListResponse {
	result := make([]ProvinceListResponse, len(provinces))
	for i, p := range provinces {
		result[i] = ProvinceListResponse{
			ID:        p.ID.Hex(),
			Name:      p.Name,
			WardCount: len(p.Wards),
		}
	}
	return result
}

func FromWard(w *model.Ward) *WardResponse {
	return &WardResponse{
		ID:         w.ID,
		ProvinceID: w.ProvinceID.Hex(),
		Name:       w.Name,
	}
}

func FromWards(wards []model.Ward) []WardResponse {
	result := make([]WardResponse, len(wards))
	for i, w := range wards {
		result[i] = *FromWard(&w)
	}
	return result
}
