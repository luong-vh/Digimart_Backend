package service

import (
	"strings"

	"github.com/luong-vh/Digimart_Backend/internal/apperror"
	"github.com/luong-vh/Digimart_Backend/internal/dto"
	"github.com/luong-vh/Digimart_Backend/internal/repo"
	"github.com/luong-vh/Digimart_Backend/internal/util"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ProvinceService interface {
	// Province
	Create(req *dto.CreateProvinceRequest) (*dto.ProvinceResponse, error)
	GetByID(id string) (*dto.ProvinceResponse, error)
	GetAll() ([]dto.ProvinceListResponse, error)
	Update(id string, req *dto.UpdateProvinceRequest) (*dto.ProvinceResponse, error)
	Delete(id string) error

	// Ward
	AddWard(provinceID string, req *dto.CreateWardRequest) (*dto.WardResponse, error)
	GetWardByID(provinceID, wardID string) (*dto.WardResponse, error)
	GetWardsByProvinceID(provinceID string) ([]dto.WardResponse, error)
	UpdateWard(provinceID, wardID string, req *dto.UpdateWardRequest) (*dto.WardResponse, error)
	DeleteWard(provinceID, wardID string) error
}

type provinceService struct {
	repo repo.ProvinceRepo
}

func NewProvinceService(repo repo.ProvinceRepo) ProvinceService {
	return &provinceService{repo: repo}
}

func (s *provinceService) isNotFound(err error) bool {
	return err == mongo.ErrNoDocuments || strings.Contains(err.Error(), "no documents")
}

// ==================== Province ====================

func (s *provinceService) Create(req *dto.CreateProvinceRequest) (*dto.ProvinceResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	province := req.ToModel()
	if err := s.repo.Create(ctx, province); err != nil {
		return nil, err
	}

	return dto.FromProvince(province), nil
}

func (s *provinceService) GetByID(id string) (*dto.ProvinceResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, apperror.ErrInvalidID
	}

	province, err := s.repo.GetByID(ctx, objectID)
	if err != nil {
		if s.isNotFound(err) {
			return nil, apperror.ErrProvinceNotFound
		}
		return nil, err
	}

	return dto.FromProvince(province), nil
}

func (s *provinceService) GetAll() ([]dto.ProvinceListResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	provinces, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	return dto.FromProvinces(provinces), nil
}

func (s *provinceService) Update(id string, req *dto.UpdateProvinceRequest) (*dto.ProvinceResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, apperror.ErrInvalidID
	}

	province, err := s.repo.GetByID(ctx, objectID)
	if err != nil {
		if s.isNotFound(err) {
			return nil, apperror.ErrProvinceNotFound
		}
		return nil, err
	}

	province.Name = req.Name
	if err := s.repo.Update(ctx, province); err != nil {
		return nil, err
	}

	return dto.FromProvince(province), nil
}

func (s *provinceService) Delete(id string) error {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return apperror.ErrInvalidID
	}

	province, err := s.repo.GetByID(ctx, objectID)
	if err != nil {
		if s.isNotFound(err) {
			return apperror.ErrProvinceNotFound
		}
		return err
	}

	if len(province.Wards) > 0 {
		return apperror.ErrProvinceHasWards
	}

	return s.repo.Delete(ctx, objectID)
}

// ==================== Ward ====================

func (s *provinceService) AddWard(provinceID string, req *dto.CreateWardRequest) (*dto.WardResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(provinceID)
	if err != nil {
		return nil, apperror.ErrInvalidID
	}

	if _, err := s.repo.GetByID(ctx, objectID); err != nil {
		if s.isNotFound(err) {
			return nil, apperror.ErrProvinceNotFound
		}
		return nil, err
	}

	ward := req.ToModel(objectID)
	if err := s.repo.AddWard(ctx, objectID, ward); err != nil {
		return nil, err
	}

	return dto.FromWard(ward), nil
}

func (s *provinceService) GetWardByID(provinceID, wardID string) (*dto.WardResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(provinceID)
	if err != nil {
		return nil, apperror.ErrInvalidID
	}

	ward, err := s.repo.GetWardByID(ctx, objectID, wardID)
	if err != nil {
		if s.isNotFound(err) {
			return nil, apperror.ErrWardNotFound
		}
		return nil, err
	}

	return dto.FromWard(ward), nil
}

func (s *provinceService) GetWardsByProvinceID(provinceID string) ([]dto.WardResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(provinceID)
	if err != nil {
		return nil, apperror.ErrInvalidID
	}

	wards, err := s.repo.GetWardsByProvinceID(ctx, objectID)
	if err != nil {
		if s.isNotFound(err) {
			return nil, apperror.ErrProvinceNotFound
		}
		return nil, err
	}

	return dto.FromWards(wards), nil
}

func (s *provinceService) UpdateWard(provinceID, wardID string, req *dto.UpdateWardRequest) (*dto.WardResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(provinceID)
	if err != nil {
		return nil, apperror.ErrInvalidID
	}

	ward, err := s.repo.GetWardByID(ctx, objectID, wardID)
	if err != nil {
		if s.isNotFound(err) {
			return nil, apperror.ErrWardNotFound
		}
		return nil, err
	}

	ward.Name = req.Name
	if err := s.repo.UpdateWard(ctx, objectID, ward); err != nil {
		return nil, err
	}

	return dto.FromWard(ward), nil
}

func (s *provinceService) DeleteWard(provinceID, wardID string) error {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(provinceID)
	if err != nil {
		return apperror.ErrInvalidID
	}

	if _, err := s.repo.GetWardByID(ctx, objectID, wardID); err != nil {
		if s.isNotFound(err) {
			return apperror.ErrWardNotFound
		}
		return err
	}

	return s.repo.DeleteWard(ctx, objectID, wardID)
}
