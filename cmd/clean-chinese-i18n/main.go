package main

import (
	"fmt"
	"os"
	"unicode"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/joho/godotenv"
)

// 目标语言列表（排除中文）
var targetLangs = []string{
	"en", "fr", "ru", "ja", "vi", "ko", "es", "de", "pt", "it", "ar",
}

func containsChinese(text string) bool {
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

// valueContainsChinese 递归检查任意 JSON value 是否含中文
func valueContainsChinese(v interface{}) bool {
	switch val := v.(type) {
	case string:
		return containsChinese(val)
	case map[string]interface{}:
		for _, item := range val {
			if valueContainsChinese(item) {
				return true
			}
		}
	case []interface{}:
		for _, item := range val {
			if valueContainsChinese(item) {
				return true
			}
		}
	}
	return false
}

// cleanLangNodes 清洗 map 中含中文的语言节点，返回被清理的语言列表
func cleanLangNodes(data map[string]interface{}) []string {
	removed := []string{}
	for _, lang := range targetLangs {
		if node, ok := data[lang]; ok {
			if valueContainsChinese(node) {
				delete(data, lang)
				removed = append(removed, lang)
			}
		}
	}
	return removed
}

func main() {
	_ = godotenv.Load(".env")
	common.InitEnv()

	if err := model.InitDB(); err != nil {
		fmt.Println("InitDB failed:", err)
		os.Exit(1)
	}

	fmt.Println("开始扫描并清洗含中文的脏翻译数据...")

	// 1. 清洗 Prompts 的 seo_i18n / title_i18n / i18n
	var prompts []model.Prompt
	if err := model.DB.Select("id", "title_i18n", "i18n", "seo_i18n").Find(&prompts).Error; err != nil {
		fmt.Println("查询 prompts 失败:", err)
		os.Exit(1)
	}

	promptCleaned := 0
	promptTotalRemoved := 0
	for _, p := range prompts {
		updated := false
		updates := map[string]interface{}{}

		if p.SeoI18n != "" {
			var seoMap map[string]interface{}
			if err := common.Unmarshal([]byte(p.SeoI18n), &seoMap); err == nil {
				removed := cleanLangNodes(seoMap)
				if len(removed) > 0 {
					seoJSON, _ := common.Marshal(seoMap)
					updates["seo_i18n"] = string(seoJSON)
					updated = true
					promptTotalRemoved += len(removed)
					fmt.Printf("Prompt %d seo_i18n 清理语言: %v\n", p.Id, removed)
				}
			}
		}

		if p.TitleI18n != "" {
			var titleMap map[string]interface{}
			if err := common.Unmarshal([]byte(p.TitleI18n), &titleMap); err == nil {
				removed := cleanLangNodes(titleMap)
				if len(removed) > 0 {
					titleJSON, _ := common.Marshal(titleMap)
					updates["title_i18n"] = string(titleJSON)
					updated = true
					promptTotalRemoved += len(removed)
					fmt.Printf("Prompt %d title_i18n 清理语言: %v\n", p.Id, removed)
				}
			}
		}

		if p.I18n != "" {
			var contentMap map[string]interface{}
			if err := common.Unmarshal([]byte(p.I18n), &contentMap); err == nil {
				removed := cleanLangNodes(contentMap)
				if len(removed) > 0 {
					contentJSON, _ := common.Marshal(contentMap)
					updates["i18n"] = string(contentJSON)
					updated = true
					promptTotalRemoved += len(removed)
					fmt.Printf("Prompt %d i18n 清理语言: %v\n", p.Id, removed)
				}
			}
		}

		if updated {
			if err := model.DB.Model(&model.Prompt{}).Where("id = ?", p.Id).Updates(updates).Error; err != nil {
				fmt.Printf("Prompt %d 更新失败: %v\n", p.Id, err)
			} else {
				promptCleaned++
			}
		}
	}

	// 2. 清洗 Articles 的 i18n / seo_i18n
	var articles []model.Article
	if err := model.DB.Select("id", "i18n", "seo_i18n").Find(&articles).Error; err != nil {
		fmt.Println("查询 articles 失败:", err)
		os.Exit(1)
	}

	articleCleaned := 0
	articleTotalRemoved := 0
	for _, a := range articles {
		updated := false
		updates := map[string]interface{}{}

		if a.I18n != "" {
			var i18nMap map[string]interface{}
			if err := common.Unmarshal([]byte(a.I18n), &i18nMap); err == nil {
				removed := cleanLangNodes(i18nMap)
				if len(removed) > 0 {
					i18nJSON, _ := common.Marshal(i18nMap)
					updates["i18n"] = string(i18nJSON)
					updated = true
					articleTotalRemoved += len(removed)
					fmt.Printf("Article %d i18n 清理语言: %v\n", a.Id, removed)
				}
			}
		}

		if a.SeoI18n != "" {
			var seoMap map[string]interface{}
			if err := common.Unmarshal([]byte(a.SeoI18n), &seoMap); err == nil {
				removed := cleanLangNodes(seoMap)
				if len(removed) > 0 {
					seoJSON, _ := common.Marshal(seoMap)
					updates["seo_i18n"] = string(seoJSON)
					updated = true
					articleTotalRemoved += len(removed)
					fmt.Printf("Article %d seo_i18n 清理语言: %v\n", a.Id, removed)
				}
			}
		}

		if updated {
			if err := model.DB.Model(&model.Article{}).Where("id = ?", a.Id).Updates(updates).Error; err != nil {
				fmt.Printf("Article %d 更新失败: %v\n", a.Id, err)
			} else {
				articleCleaned++
			}
		}
	}

	fmt.Println("\n清洗完成:")
	fmt.Printf("- Prompts: %d 条记录被清理，共删除 %d 个脏语言节点\n", promptCleaned, promptTotalRemoved)
	fmt.Printf("- Articles: %d 条记录被清理，共删除 %d 个脏语言节点\n", articleCleaned, articleTotalRemoved)
	fmt.Println("\n注意：被清理的语言节点会在后台自动轮询（每 5 分钟）或手动点击重新翻译后重新生成。")
}
