package service

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

// GetCuratedCategories 获取公开分类列表，并追加 "all" 分类
func GetCuratedCategories() (*dto.CuratedCategoryListResponse, error) {
	categories, err := model.GetCuratedCategories()
	if err != nil {
		return nil, err
	}

	resp := make([]dto.CuratedCategoryResponse, 0, len(categories)+1)
	resp = append(resp, dto.CuratedCategoryResponse{
		Key:       "all",
		Name:      "全部",
		IconUrl:   "",
		SortOrder: -1,
		Enabled:   true,
	})
	for _, c := range categories {
		resp = append(resp, dto.CuratedCategoryResponse{
			Key:       c.Key,
			Name:      c.Name,
			IconUrl:   c.IconUrl,
			SortOrder: c.SortOrder,
			Enabled:   c.Enabled,
		})
	}
	return &dto.CuratedCategoryListResponse{Categories: resp}, nil
}

// AdminListCuratedCategories 管理后台获取全部分类
func AdminListCuratedCategories() ([]*model.CuratedCategory, error) {
	return model.AdminListCuratedCategories()
}

// AdminCreateCategory 管理后台创建分类
func AdminCreateCategory(req *dto.AdminUpsertCategoryRequest) (*model.CuratedCategory, error) {
	if req.Key == "" || req.Name == "" {
		return nil, fmt.Errorf("key and name are required")
	}

	existing, err := model.GetCuratedCategoryByKey(req.Key)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("category key already exists")
	}

	category := &model.CuratedCategory{
		Key:       req.Key,
		Name:      req.Name,
		IconUrl:   req.IconUrl,
		SortOrder: req.SortOrder,
		Enabled:   req.Enabled,
	}
	if err := category.Insert(); err != nil {
		return nil, err
	}
	_ = InvalidateCuratedCategoryCache()
	return category, nil
}

// AdminUpdateCategory 管理后台更新分类
func AdminUpdateCategory(id int, req *dto.AdminUpsertCategoryRequest) error {
	if id <= 0 {
		return fmt.Errorf("invalid category id")
	}

	category, err := model.GetCuratedCategoryById(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("category not found")
		}
		return err
	}

	if req.Key != "" && req.Key != category.Key {
		existing, err := model.GetCuratedCategoryByKey(req.Key)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if existing != nil {
			return fmt.Errorf("category key already exists")
		}
		category.Key = req.Key
	}

	if req.Name != "" {
		category.Name = req.Name
	}
	category.IconUrl = req.IconUrl
	category.SortOrder = req.SortOrder
	category.Enabled = req.Enabled

	if err := category.Update(); err != nil {
		return err
	}
	_ = InvalidateCuratedCategoryCache()
	return nil
}

// AdminDeleteCategory 管理后台删除分类
func AdminDeleteCategory(id int) error {
	category, err := model.GetCuratedCategoryById(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("category not found")
		}
		return err
	}

	exists, err := model.CuratedTemplateExistsByCategory(category.Key)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("category is in use by templates")
	}

	if err := category.Delete(); err != nil {
		return err
	}
	_ = InvalidateCuratedCategoryCache()
	return nil
}
