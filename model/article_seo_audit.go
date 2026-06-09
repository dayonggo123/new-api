package model

import (
	"math"

	"github.com/QuantumNous/new-api/common"
)

// ArticleSEOAudit SEO 审计历史记录
type ArticleSEOAudit struct {
	Id             int    `json:"id" gorm:"primaryKey"`
	ArticleId      int    `json:"article_id" gorm:"index"`
	OverallScore   int    `json:"overall_score"`
	Categories     string `json:"categories" gorm:"type:text"`     // JSON
	CriticalIssues string `json:"critical_issues" gorm:"type:text"` // JSON array
	QuickWins      string `json:"quick_wins" gorm:"type:text"`      // JSON array
	CreatedAt      int64  `json:"created_at"`
}

func (ArticleSEOAudit) TableName() string {
	return "article_seo_audit"
}

// CreateArticleSEOAudit 创建审计记录
func CreateArticleSEOAudit(audit *ArticleSEOAudit) error {
	return DB.Create(audit).Error
}

// GetArticleSEOAudits 获取指定文章的审计历史
func GetArticleSEOAudits(articleId int, limit int) ([]ArticleSEOAudit, error) {
	var audits []ArticleSEOAudit
	err := DB.Where("article_id = ?", articleId).Order("created_at DESC").Limit(limit).Find(&audits).Error
	return audits, err
}

// GetLatestArticleSEOAudit 获取指定文章的最新审计记录
func GetLatestArticleSEOAudit(articleId int) (*ArticleSEOAudit, error) {
	var audit ArticleSEOAudit
	err := DB.Where("article_id = ?", articleId).Order("created_at DESC").First(&audit).Error
	if err != nil {
		return nil, err
	}
	return &audit, nil
}

// GetLatestArticleSEOAuditScores 批量获取多个文章的最新审计分数
func GetLatestArticleSEOAuditScores(articleIds []int) (map[int]int, error) {
	result := make(map[int]int)
	if len(articleIds) == 0 {
		return result, nil
	}
	var audits []ArticleSEOAudit
	err := DB.Raw(`
		SELECT a.* FROM article_seo_audit a
		INNER JOIN (
			SELECT article_id, MAX(id) as max_id
			FROM article_seo_audit
			WHERE article_id IN ?
			GROUP BY article_id
		) b ON a.article_id = b.article_id AND a.id = b.max_id
	`, articleIds).Scan(&audits).Error
	if err != nil {
		return nil, err
	}
	for _, a := range audits {
		result[a.ArticleId] = a.OverallScore
	}
	return result, nil
}

// GetArticleSEOAuditStats 获取文章 SEO 审计统计
func GetArticleSEOAuditStats() (map[string]interface{}, error) {
	var totalArticles int64
	if err := DB.Model(&Article{}).Count(&totalArticles).Error; err != nil {
		return nil, err
	}

	var withSEO int64
	if err := DB.Model(&Article{}).Where("seo_keywords != '' OR seo_title != ''").Count(&withSEO).Error; err != nil {
		return nil, err
	}

	var withAudit int64
	if err := DB.Model(&ArticleSEOAudit{}).Select("COUNT(DISTINCT article_id)").Scan(&withAudit).Error; err != nil {
		return nil, err
	}

	var avgScore float64
	DB.Raw("SELECT COALESCE(AVG(overall_score), 0) FROM article_seo_audit WHERE id IN (SELECT MAX(id) FROM article_seo_audit GROUP BY article_id)").Scan(&avgScore)

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
		FROM article_seo_audit
		WHERE id IN (SELECT MAX(id) FROM article_seo_audit GROUP BY article_id)
		GROUP BY `+"`"+`range`+"`"+`
	`).Scan(&scoreDistribution)

	return map[string]interface{}{
		"total_articles":     totalArticles,
		"with_seo":           withSEO,
		"with_audit":         withAudit,
		"seo_coverage":       math.Round(float64(withSEO)/float64(totalArticles)*100*100) / 100,
		"audit_coverage":     math.Round(float64(withAudit)/float64(totalArticles)*100*100) / 100,
		"average_score":      math.Round(avgScore*10) / 10,
		"score_distribution": scoreDistribution,
	}, nil
}

// GetLowScoreArticles 获取最新审计分低于阈值的文章
func GetLowScoreArticles(threshold int, limit int) ([]struct {
	Id         int    `json:"id"`
	Title      string `json:"title"`
	AuditScore int    `json:"audit_score"`
	AuditDate  int64  `json:"audit_date"`
}, error) {
	var rows []struct {
		ArticleId    int
		OverallScore int
		CreatedAt    int64
	}
	err := DB.Raw(`
		SELECT article_id, overall_score, created_at FROM article_seo_audit a
		WHERE id = (
			SELECT MAX(id) FROM article_seo_audit b WHERE b.article_id = a.article_id
		)
		AND overall_score < ?
		ORDER BY overall_score ASC
		LIMIT ?
	`, threshold, limit).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	var result []struct {
		Id         int    `json:"id"`
		Title      string `json:"title"`
		AuditScore int    `json:"audit_score"`
		AuditDate  int64  `json:"audit_date"`
	}
	for _, r := range rows {
		article, err := GetArticleById(r.ArticleId)
		if err != nil || article == nil {
			continue
		}
		result = append(result, struct {
			Id         int    `json:"id"`
			Title      string `json:"title"`
			AuditScore int    `json:"audit_score"`
			AuditDate  int64  `json:"audit_date"`
		}{
			Id:         article.Id,
			Title:      article.Title,
			AuditScore: r.OverallScore,
			AuditDate:  r.CreatedAt,
		})
	}
	return result, nil
}

// DeleteOldArticleSEOAudits 清理旧的审计记录，每篇文章保留最近 N 条
func DeleteOldArticleSEOAudits(keep int) error {
	if common.UsingSQLite {
		var idsToKeep []int
		rows, err := DB.Raw(`
			SELECT id FROM article_seo_audit a
			WHERE (
				SELECT COUNT(*) FROM article_seo_audit b
				WHERE b.article_id = a.article_id AND b.id >= a.id
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
			return DB.Where("id NOT IN ?", idsToKeep).Delete(&ArticleSEOAudit{}).Error
		}
		return nil
	}
	return DB.Exec(`
		DELETE FROM article_seo_audit
		WHERE id NOT IN (
			SELECT id FROM (
				SELECT id FROM article_seo_audit a
				WHERE (
					SELECT COUNT(*) FROM article_seo_audit b
					WHERE b.article_id = a.article_id AND b.id >= a.id
				) <= ?
			) tmp
		)
	`, keep).Error
}
