package model

import (
	"encoding/json"
	"math"
	"time"

	"github.com/QuantumNous/new-api/common"
)

// PromptSEOAudit SEO 审计历史记录
type PromptSEOAudit struct {
	Id             int    `json:"id" gorm:"primaryKey"`
	PromptId       int    `json:"prompt_id" gorm:"index"`
	OverallScore   int    `json:"overall_score"`
	Categories     string `json:"categories" gorm:"type:text"`     // JSON
	CriticalIssues string `json:"critical_issues" gorm:"type:text"` // JSON array
	QuickWins      string `json:"quick_wins" gorm:"type:text"`      // JSON array
	CreatedAt      int64  `json:"created_at"`
}

func (PromptSEOAudit) TableName() string {
	return "prompt_seo_audit"
}

// CreatePromptSEOAudit 创建审计记录
func CreatePromptSEOAudit(audit *PromptSEOAudit) error {
	return DB.Create(audit).Error
}

// GetPromptSEOAudits 获取指定 prompt 的审计历史
func GetPromptSEOAudits(promptId int, limit int) ([]PromptSEOAudit, error) {
	var audits []PromptSEOAudit
	err := DB.Where("prompt_id = ?", promptId).Order("created_at DESC").Limit(limit).Find(&audits).Error
	return audits, err
}

// GetLatestPromptSEOAudit 获取指定 prompt 的最新审计记录
func GetLatestPromptSEOAudit(promptId int) (*PromptSEOAudit, error) {
	var audit PromptSEOAudit
	err := DB.Where("prompt_id = ?", promptId).Order("created_at DESC").First(&audit).Error
	if err != nil {
		return nil, err
	}
	return &audit, nil
}

// GetPromptSEOAuditStats 获取所有 prompt 的 SEO 审计统计
func GetPromptSEOAuditStats() (map[string]interface{}, error) {
	var totalPrompts int64
	if err := DB.Model(&Prompt{}).Count(&totalPrompts).Error; err != nil {
		return nil, err
	}

	var withSEO int64
	if err := DB.Model(&Prompt{}).Where("seo_keywords != '' OR intro != ''").Count(&withSEO).Error; err != nil {
		return nil, err
	}

	var withAudit int64
	if err := DB.Model(&PromptSEOAudit{}).Select("COUNT(DISTINCT prompt_id)").Scan(&withAudit).Error; err != nil {
		return nil, err
	}

	var avgScore float64
	DB.Raw("SELECT COALESCE(AVG(overall_score), 0) FROM prompt_seo_audit WHERE id IN (SELECT MAX(id) FROM prompt_seo_audit GROUP BY prompt_id)").Scan(&avgScore)

	var scoreDistribution []struct {
		Range string `json:"range"`
		Count int64  `json:"count"`
	}
	DB.Raw(`
		SELECT 
			CASE 
				WHEN overall_score >= 80 THEN 'excellent'
				WHEN overall_score >= 60 THEN 'good'
				WHEN overall_score >= 40 THEN 'average'
				ELSE 'poor'
			END as `+"`"+`range`+"`"+`,
			COUNT(*) as count
		FROM prompt_seo_audit
		WHERE id IN (SELECT MAX(id) FROM prompt_seo_audit GROUP BY prompt_id)
		GROUP BY `+"`"+`range`+"`"+`
	`).Scan(&scoreDistribution)

	return map[string]interface{}{
		"total_prompts":      totalPrompts,
		"with_seo":           withSEO,
		"with_audit":         withAudit,
		"seo_coverage":       math.Round(float64(withSEO)/float64(totalPrompts)*100*100) / 100,
		"audit_coverage":     math.Round(float64(withAudit)/float64(totalPrompts)*100*100) / 100,
		"average_score":      math.Round(avgScore*10) / 10,
		"score_distribution": scoreDistribution,
	}, nil
}

// GetLatestPromptSEOAuditScores 批量获取多个 prompt 的最新审计分数
func GetLatestPromptSEOAuditScores(promptIds []int) (map[int]int, error) {
	result := make(map[int]int)
	if len(promptIds) == 0 {
		return result, nil
	}
	var audits []PromptSEOAudit
	if common.UsingSQLite {
		// SQLite 下用子查询获取每个 prompt 最新记录
		err := DB.Raw(`
			SELECT a.* FROM prompt_seo_audit a
			INNER JOIN (
				SELECT prompt_id, MAX(id) as max_id
				FROM prompt_seo_audit
				WHERE prompt_id IN ?
				GROUP BY prompt_id
			) b ON a.prompt_id = b.prompt_id AND a.id = b.max_id
		`, promptIds).Scan(&audits).Error
		if err != nil {
			return nil, err
		}
	} else {
		err := DB.Raw(`
			SELECT a.* FROM prompt_seo_audit a
			INNER JOIN (
				SELECT prompt_id, MAX(id) as max_id
				FROM prompt_seo_audit
				WHERE prompt_id IN ?
				GROUP BY prompt_id
			) b ON a.prompt_id = b.prompt_id AND a.id = b.max_id
		`, promptIds).Scan(&audits).Error
		if err != nil {
			return nil, err
		}
	}
	for _, a := range audits {
		result[a.PromptId] = a.OverallScore
	}
	return result, nil
}

// DeleteOldPromptSEOAudits 清理旧的审计记录，每个 prompt 保留最近 N 条
func DeleteOldPromptSEOAudits(keep int) error {
	if common.UsingSQLite {
		// SQLite 不支持 DELETE with subquery 直接使用，需要分步
		var idsToKeep []int
		rows, err := DB.Raw(`
			SELECT id FROM prompt_seo_audit a
			WHERE (
				SELECT COUNT(*) FROM prompt_seo_audit b
				WHERE b.prompt_id = a.prompt_id AND b.id >= a.id
			) <= ?
		`, keep).Rows()
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id int
			if err := rows.Scan(&id); err == nil {
				idsToKeep = append(idsToKeep, id)
			}
		}
		if len(idsToKeep) > 0 {
			return DB.Where("id NOT IN ?", idsToKeep).Delete(&PromptSEOAudit{}).Error
		}
		return nil
	}
	return DB.Exec(`
		DELETE FROM prompt_seo_audit
		WHERE id NOT IN (
			SELECT id FROM (
				SELECT id FROM prompt_seo_audit a
				WHERE (
					SELECT COUNT(*) FROM prompt_seo_audit b
					WHERE b.prompt_id = a.prompt_id AND b.id >= a.id
				) <= ?
			) tmp
		)
	`, keep).Error
}


// SEOTrendPoint 单天的趋势数据
type SEOTrendPoint struct {
	Date        string  `json:"date"`
	AvgScore    float64 `json:"avg_score"`
	AuditCount  int     `json:"audit_count"`
	PromptCount int     `json:"prompt_count"`
}

// GetSEOTrends 获取最近 N 天的 SEO 审计趋势（按天聚合）
func GetSEOTrends(days int) ([]SEOTrendPoint, error) {
	var audits []PromptSEOAudit
	startTime := time.Now().AddDate(0, 0, -days).Unix()
	if err := DB.Where("created_at >= ?", startTime).Order("created_at asc").Find(&audits).Error; err != nil {
		return nil, err
	}

	type dayAgg struct {
		totalScore int
		count      int
		promptIds  map[int]struct{}
	}
	aggMap := make(map[string]*dayAgg)

	for _, a := range audits {
		day := time.Unix(a.CreatedAt, 0).Format("2006-01-02")
		if aggMap[day] == nil {
			aggMap[day] = &dayAgg{promptIds: make(map[int]struct{})}
		}
		aggMap[day].totalScore += a.OverallScore
		aggMap[day].count++
		aggMap[day].promptIds[a.PromptId] = struct{}{}
	}

	var result []SEOTrendPoint
	for i := days; i >= 0; i-- {
		day := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		if agg, ok := aggMap[day]; ok {
			result = append(result, SEOTrendPoint{
				Date:        day,
				AvgScore:    math.Round(float64(agg.totalScore)/float64(agg.count)*10) / 10,
				AuditCount:  agg.count,
				PromptCount: len(agg.promptIds),
			})
		} else {
			result = append(result, SEOTrendPoint{Date: day, AvgScore: 0, AuditCount: 0, PromptCount: 0})
		}
	}
	return result, nil
}

// LowScorePrompt 低分提示词
type LowScorePrompt struct {
	Id           int    `json:"id"`
	Title        string `json:"title"`
	AuditScore   int    `json:"audit_score"`
	AuditDate    int64  `json:"audit_date"`
	CategoryName string `json:"category_name"`
}

// TranslateStats 翻译统计
type TranslateStats struct {
	Total            int64 `json:"total"`
	WithTranslation  int64 `json:"with_translation"`  // 有 seo_i18n 的记录数
	FullyTranslated  int64 `json:"fully_translated"`  // 全部 11 种语言都翻译完成的记录数
	CoveragePercent  float64 `json:"coverage_percent"`
	FullPercent      float64 `json:"full_percent"`
}

// GetPromptTranslateStats 获取 Prompt SEO 翻译统计
func GetPromptTranslateStats() (*TranslateStats, error) {
	var total int64
	if err := DB.Model(&Prompt{}).Count(&total).Error; err != nil {
		return nil, err
	}

	var withTranslation int64
	if err := DB.Model(&Prompt{}).Where("seo_i18n IS NOT NULL AND seo_i18n != '' AND seo_i18n != '{}' AND seo_i18n != 'null'").Count(&withTranslation).Error; err != nil {
		return nil, err
	}

	// 查询所有有 seo_i18n 的记录，统计有多少条包含全部 11 种语言
	var fullyTranslated int64
	if withTranslation > 0 {
		var prompts []Prompt
		if err := DB.Select("id", "seo_i18n").Where("seo_i18n IS NOT NULL AND seo_i18n != '' AND seo_i18n != '{}' AND seo_i18n != 'null'").Find(&prompts).Error; err != nil {
			return nil, err
		}
		for _, p := range prompts {
			var parsed map[string]interface{}
			if err := json.Unmarshal([]byte(p.SeoI18n), &parsed); err == nil {
				if len(parsed) >= 11 {
					fullyTranslated++
				}
			}
		}
	}

	return &TranslateStats{
		Total:           total,
		WithTranslation: withTranslation,
		FullyTranslated: fullyTranslated,
		CoveragePercent: math.Round(float64(withTranslation)/float64(total)*100*100) / 100,
		FullPercent:     math.Round(float64(fullyTranslated)/float64(total)*100*100) / 100,
	}, nil
}

// GetArticleTranslateStats 获取 Article SEO 翻译统计
func GetArticleTranslateStats() (*TranslateStats, error) {
	var total int64
	if err := DB.Model(&Article{}).Count(&total).Error; err != nil {
		return nil, err
	}

	var withTranslation int64
	if err := DB.Model(&Article{}).Where("seo_i18n IS NOT NULL AND seo_i18n != '' AND seo_i18n != '{}' AND seo_i18n != 'null'").Count(&withTranslation).Error; err != nil {
		return nil, err
	}

	var fullyTranslated int64
	if withTranslation > 0 {
		var articles []Article
		if err := DB.Select("id", "seo_i18n").Where("seo_i18n IS NOT NULL AND seo_i18n != '' AND seo_i18n != '{}' AND seo_i18n != 'null'").Find(&articles).Error; err != nil {
			return nil, err
		}
		for _, a := range articles {
			var parsed map[string]interface{}
			if err := json.Unmarshal([]byte(a.SeoI18n), &parsed); err == nil {
				if len(parsed) >= 11 {
					fullyTranslated++
				}
			}
		}
	}

	return &TranslateStats{
		Total:           total,
		WithTranslation: withTranslation,
		FullyTranslated: fullyTranslated,
		CoveragePercent: math.Round(float64(withTranslation)/float64(total)*100*100) / 100,
		FullPercent:     math.Round(float64(fullyTranslated)/float64(total)*100*100) / 100,
	}, nil
}

// GetLowScorePrompts 获取最新审计分低于阈值的提示词
func GetLowScorePrompts(threshold int, limit int) ([]LowScorePrompt, error) {
	var rows []struct {
		PromptId     int
		OverallScore int
		CreatedAt    int64
	}
	err := DB.Raw(`
		SELECT prompt_id, overall_score, created_at FROM prompt_seo_audit a
		WHERE id = (
			SELECT MAX(id) FROM prompt_seo_audit b WHERE b.prompt_id = a.prompt_id
		)
		AND overall_score < ?
		ORDER BY overall_score ASC
		LIMIT ?
	`, threshold, limit).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	var result []LowScorePrompt
	for _, r := range rows {
		prompt, err := GetPromptById(r.PromptId)
		if err != nil || prompt == nil {
			continue
		}
		result = append(result, LowScorePrompt{
			Id:           prompt.Id,
			Title:        prompt.Title,
			AuditScore:   r.OverallScore,
			AuditDate:    r.CreatedAt,
			CategoryName: prompt.CategoryName,
		})
	}
	return result, nil
}
