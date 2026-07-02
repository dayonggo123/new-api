package controller

import (
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// AdminListTKMaterials handles GET /api/admin/tk/materials
func AdminListTKMaterials(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	category := c.Query("category")
	keyword := c.Query("keyword")
	status := -1
	if s := c.Query("status"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			status = v
		}
	}

	materials, total, err := model.TKMaterialListAll(category, keyword, status, page, pageSize)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"data":  materials,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}

// AdminUploadTKMaterial handles POST /api/admin/tk/materials
func AdminUploadTKMaterial(c *gin.Context) {
	category := c.PostForm("category")
	if category == "" {
		common.ApiErrorMsg(c, "category is required")
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	files := form.File["files"]
	if len(files) == 0 {
		common.ApiErrorMsg(c, "no files provided")
		return
	}

	permanent := c.Query("permanent") == "true"
	results, err := service.UploadTKMaterialFiles(c, files, permanent)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	var saved []model.TKMaterial
	var failed []string
	for _, res := range results {
		if res.Error != "" {
			failed = append(failed, res.Filename+": "+res.Error)
			continue
		}
		m, err := service.SaveTKMaterialFromUpload(category, &res, "upload")
		if err != nil {
			failed = append(failed, res.Filename+": "+err.Error())
			continue
		}
		saved = append(saved, *m)
	}

	common.ApiSuccess(c, gin.H{
		"saved":  saved,
		"failed": failed,
	})
}

// AdminDeleteTKMaterial handles DELETE /api/admin/tk/materials/:id
func AdminDeleteTKMaterial(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "invalid id")
		return
	}
	if err := model.TKMaterialDeleteByID(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

// AdminImportTKMaterialsFromNotion handles POST /api/admin/tk/materials/import/notion
func AdminImportTKMaterialsFromNotion(c *gin.Context) {
	var req struct {
		Token      string   `json:"token"`
		DatabaseID string   `json:"database_id"`
		Categories []string `json:"categories"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if req.DatabaseID == "" {
		common.ApiErrorMsg(c, "database_id is required")
		return
	}
	if len(req.Categories) == 0 {
		req.Categories = model.TKMaterialCategories()
	}

	// Try to read token from option if not provided
	token := req.Token
	if token == "" {
		token = getNotionToken()
	}
	if token == "" {
		common.ApiErrorMsg(c, "notion token is required")
		return
	}

	result, err := service.ImportTKMaterialsFromNotion(token, req.DatabaseID, req.Categories)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

// AdminListTKMaterialCategories handles GET /api/admin/tk/materials/categories
func AdminListTKMaterialCategories(c *gin.Context) {
	common.ApiSuccess(c, model.TKMaterialCategories())
}

// AdminTKMaterialCategoryStats handles GET /api/admin/tk/materials/stats
func AdminTKMaterialCategoryStats(c *gin.Context) {
	stats, err := model.TKMaterialCountByCategory()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, stats)
}

// PublicListTKMaterials handles GET /api/public/tk/materials
func PublicListTKMaterials(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	category := c.Query("category")
	keyword := c.Query("keyword")

	materials, total, err := model.TKMaterialList(category, keyword, page, pageSize)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"data":  materials,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}

// PublicGetTKMaterial handles GET /api/public/tk/materials/:id
func PublicGetTKMaterial(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "invalid id")
		return
	}
	m, err := model.TKMaterialGetByID(id)
	if err != nil {
		common.ApiErrorMsg(c, "material not found")
		return
	}
	if m.Status != 1 {
		common.ApiErrorMsg(c, "material not available")
		return
	}
	common.ApiSuccess(c, m)
}

// PublicGetRandomTKMaterials handles GET /api/public/tk/materials/random
func PublicGetRandomTKMaterials(c *gin.Context) {
	category := c.Query("category")
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit <= 0 {
		limit = 1
	}
	if limit > 100 {
		limit = 100
	}

	// Support multiple categories separated by comma
	var materials []model.TKMaterial
	if strings.Contains(category, ",") {
		categories := strings.Split(category, ",")
		for i, cat := range categories {
			categories[i] = strings.TrimSpace(cat)
		}
		perCategory := limit
		if len(categories) > 0 {
			perCategory = (limit + len(categories) - 1) / len(categories)
		}
		result, err := model.TKMaterialRandomByCategories(categories, perCategory)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		for _, items := range result {
			materials = append(materials, items...)
		}
	} else {
		items, err := model.TKMaterialRandom(category, limit)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		materials = items
	}

	common.ApiSuccess(c, gin.H{
		"data": materials,
	})
}

// PublicUploadTKMaterial handles POST /api/public/tk/materials (optional downstream upload)
func PublicUploadTKMaterial(c *gin.Context) {
	category := c.PostForm("category")
	if category == "" {
		common.ApiErrorMsg(c, "category is required")
		return
	}
	form, err := c.MultipartForm()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	files := form.File["files"]
	if len(files) == 0 {
		common.ApiErrorMsg(c, "no files provided")
		return
	}

	results, err := service.UploadTKMaterialFiles(c, files, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	var saved []model.TKMaterial
	var failed []string
	for _, res := range results {
		if res.Error != "" {
			failed = append(failed, res.Filename+": "+res.Error)
			continue
		}
		m, err := service.SaveTKMaterialFromUpload(category, &res, "upload")
		if err != nil {
			failed = append(failed, res.Filename+": "+err.Error())
			continue
		}
		saved = append(saved, *m)
	}
	common.ApiSuccess(c, gin.H{
		"saved":  saved,
		"failed": failed,
	})
}

func getNotionToken() string {
	return os.Getenv("NOTION_INTEGRATION_TOKEN")
}

// HealthCheckTKMaterial is a simple health endpoint used for internal checks.
func HealthCheckTKMaterial(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
