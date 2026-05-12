package model

import (
	"math"

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
			END as range,
			COUNT(*) as count
		FROM prompt_seo_audit
		WHERE id IN (SELECT MAX(id) FROM prompt_seo_audit GROUP BY prompt_id)
		GROUP BY range
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
