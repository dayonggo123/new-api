package service

import "github.com/QuantumNous/new-api/model"

// ==================== Model Pose ====================

func GetAllModelPoses(startIdx, num int) (poses []*model.EcommerceModelPose, total int64, err error) {
	return model.GetAllModelPoses(startIdx, num)
}

func GetEnabledModelPoses() ([]*model.EcommerceModelPose, error) {
	return model.GetEnabledModelPoses()
}

func GetModelPoseById(id int) (*model.EcommerceModelPose, error) {
	return model.GetModelPoseById(id)
}

func CreateModelPose(pose *model.EcommerceModelPose) error {
	return pose.Insert()
}

func UpdateModelPose(pose *model.EcommerceModelPose) error {
	return pose.Update()
}

func DeleteModelPoseById(id int) error {
	return model.DeleteModelPoseById(id)
}

// ==================== Case Category ====================

func GetAllCaseCategories(startIdx, num int) (categories []*model.EcommerceCaseCategory, total int64, err error) {
	return model.GetAllCaseCategories(startIdx, num)
}

func GetEnabledCaseCategories() ([]*model.EcommerceCaseCategory, error) {
	return model.GetEnabledCaseCategories()
}

func GetCaseCategoryById(id int) (*model.EcommerceCaseCategory, error) {
	return model.GetCaseCategoryById(id)
}

func CreateCaseCategory(category *model.EcommerceCaseCategory) error {
	return category.Insert()
}

func UpdateCaseCategory(category *model.EcommerceCaseCategory) error {
	return category.Update()
}

func DeleteCaseCategoryById(id int) error {
	return model.DeleteCaseCategoryById(id)
}

// ==================== Case Detail ====================

func GetCaseDetails(startIdx, num int, categoryId, platformId string) (details []*model.EcommerceCaseDetail, total int64, err error) {
	return model.GetCaseDetails(startIdx, num, categoryId, platformId)
}

func GetCaseDetailById(id int) (*model.EcommerceCaseDetail, error) {
	return model.GetCaseDetailById(id)
}

func GetCaseDetailByCategoryAndPlatform(categoryId, platformId string) (*model.EcommerceCaseDetail, error) {
	return model.GetCaseDetailByCategoryAndPlatform(categoryId, platformId)
}

func CreateCaseDetail(detail *model.EcommerceCaseDetail) error {
	return detail.Insert()
}

func UpdateCaseDetail(detail *model.EcommerceCaseDetail) error {
	return detail.Update()
}

func DeleteCaseDetailById(id int) error {
	return model.DeleteCaseDetailById(id)
}
