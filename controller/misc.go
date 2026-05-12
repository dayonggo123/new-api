package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/oauth"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/console_setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
)

func TestStatus(c *gin.Context) {
	err := model.PingDB()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"message": "数据库连接失败",
		})
		return
	}
	// 获取HTTP统计信息
	httpStats := middleware.GetStats()
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    "Server is running",
		"http_stats": httpStats,
	})
	return
}

func GetStatus(c *gin.Context) {

	cs := console_setting.GetConsoleSetting()
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()

	passkeySetting := system_setting.GetPasskeySettings()
	legalSetting := system_setting.GetLegalSettings()

	data := gin.H{
		"version":                     common.Version,
		"start_time":                  common.StartTime,
		"email_verification":          common.EmailVerificationEnabled,
		"github_oauth":                common.GitHubOAuthEnabled,
		"github_client_id":            common.GitHubClientId,
		"discord_oauth":               system_setting.GetDiscordSettings().Enabled,
		"discord_client_id":           system_setting.GetDiscordSettings().ClientId,
		"linuxdo_oauth":               common.LinuxDOOAuthEnabled,
		"linuxdo_client_id":           common.LinuxDOClientId,
		"linuxdo_minimum_trust_level": common.LinuxDOMinimumTrustLevel,
		"telegram_oauth":              common.TelegramOAuthEnabled,
		"telegram_bot_name":           common.TelegramBotName,
		"system_name":                 common.SystemName,
		"logo":                        common.Logo,
		"footer_html":                 common.Footer,
		"wechat_qrcode":               common.WeChatAccountQRCodeImageURL,
		"wechat_login":                common.WeChatAuthEnabled,
		"server_address":              system_setting.ServerAddress,
		"turnstile_check":             common.TurnstileCheckEnabled,
		"turnstile_site_key":          common.TurnstileSiteKey,
		"top_up_link":                 common.TopUpLink,
		"docs_link":                   operation_setting.GetGeneralSetting().DocsLink,
		"quota_per_unit":              common.QuotaPerUnit,
		// 兼容旧前端：保留 display_in_currency，同时提供新的 quota_display_type
		"display_in_currency":           operation_setting.IsCurrencyDisplay(),
		"quota_display_type":            operation_setting.GetQuotaDisplayType(),
		"custom_currency_symbol":        operation_setting.GetGeneralSetting().CustomCurrencySymbol,
		"custom_currency_exchange_rate": operation_setting.GetGeneralSetting().CustomCurrencyExchangeRate,
		"enable_batch_update":           common.BatchUpdateEnabled,
		"enable_drawing":                common.DrawingEnabled,
		"enable_task":                   common.TaskEnabled,
		"enable_data_export":            common.DataExportEnabled,
		"data_export_default_time":      common.DataExportDefaultTime,
		"default_collapse_sidebar":      common.DefaultCollapseSidebar,
		"mj_notify_enabled":             setting.MjNotifyEnabled,
		"chats":                         setting.Chats,
		"demo_site_enabled":             operation_setting.DemoSiteEnabled,
		"self_use_mode_enabled":         operation_setting.SelfUseModeEnabled,
		"default_use_auto_group":        setting.DefaultUseAutoGroup,

		"usd_exchange_rate": operation_setting.USDExchangeRate,
		"price":             operation_setting.Price,
		"stripe_unit_price": setting.StripeUnitPrice,

		// 面板启用开关
		"api_info_enabled":      cs.ApiInfoEnabled,
		"uptime_kuma_enabled":   cs.UptimeKumaEnabled,
		"announcements_enabled": cs.AnnouncementsEnabled,
		"faq_enabled":           cs.FAQEnabled,

		// 模块管理配置
		"HeaderNavModules":    common.OptionMap["HeaderNavModules"],
		"SidebarModulesAdmin": common.OptionMap["SidebarModulesAdmin"],

		"oidc_enabled":                system_setting.GetOIDCSettings().Enabled,
		"oidc_client_id":              system_setting.GetOIDCSettings().ClientId,
		"oidc_authorization_endpoint": system_setting.GetOIDCSettings().AuthorizationEndpoint,
		"passkey_login":               passkeySetting.Enabled,
		"passkey_display_name":        passkeySetting.RPDisplayName,
		"passkey_rp_id":               passkeySetting.RPID,
		"passkey_origins":             passkeySetting.Origins,
		"passkey_allow_insecure":      passkeySetting.AllowInsecureOrigin,
		"passkey_user_verification":   passkeySetting.UserVerification,
		"passkey_attachment":          passkeySetting.AttachmentPreference,
		"setup":                       constant.Setup,
		"user_agreement_enabled":      legalSetting.UserAgreement != "",
		"privacy_policy_enabled":      legalSetting.PrivacyPolicy != "",
		"checkin_enabled":             operation_setting.GetCheckinSetting().Enabled,
	}

	// 根据启用状态注入可选内容
	if cs.ApiInfoEnabled {
		data["api_info"] = console_setting.GetApiInfo()
	}
	if cs.AnnouncementsEnabled {
		data["announcements"] = console_setting.GetAnnouncements()
	}
	if cs.FAQEnabled {
		data["faq"] = console_setting.GetFAQ()
	}

	// Add enabled custom OAuth providers
	customProviders := oauth.GetEnabledCustomProviders()
	if len(customProviders) > 0 {
		type CustomOAuthInfo struct {
			Id                    int    `json:"id"`
			Name                  string `json:"name"`
			Slug                  string `json:"slug"`
			Icon                  string `json:"icon"`
			ClientId              string `json:"client_id"`
			AuthorizationEndpoint string `json:"authorization_endpoint"`
			Scopes                string `json:"scopes"`
		}
		providersInfo := make([]CustomOAuthInfo, 0, len(customProviders))
		for _, p := range customProviders {
			config := p.GetConfig()
			providersInfo = append(providersInfo, CustomOAuthInfo{
				Id:                    config.Id,
				Name:                  config.Name,
				Slug:                  config.Slug,
				Icon:                  config.Icon,
				ClientId:              config.ClientId,
				AuthorizationEndpoint: config.AuthorizationEndpoint,
				Scopes:                config.Scopes,
			})
		}
		data["custom_oauth_providers"] = providersInfo
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    data,
	})
	return
}

func GetNotice(c *gin.Context) {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    common.OptionMap["Notice"],
	})
	return
}

func GetAbout(c *gin.Context) {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    common.OptionMap["About"],
	})
	return
}

func GetUserAgreement(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    system_setting.GetLegalSettings().UserAgreement,
	})
	return
}

func GetPrivacyPolicy(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    system_setting.GetLegalSettings().PrivacyPolicy,
	})
	return
}

func GetMidjourney(c *gin.Context) {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    common.OptionMap["Midjourney"],
	})
	return
}

func GetHomePageContent(c *gin.Context) {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    common.OptionMap["HomePageContent"],
	})
	return
}

func SendEmailVerification(c *gin.Context) {
	email := c.Query("email")
	if err := common.Validate.Var(email, "required,email"); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无效的参数",
		})
		return
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无效的邮箱地址",
		})
		return
	}
	localPart := parts[0]
	domainPart := parts[1]
	if common.EmailDomainRestrictionEnabled {
		allowed := false
		for _, domain := range common.EmailDomainWhitelist {
			if domainPart == domain {
				allowed = true
				break
			}
		}
		if !allowed {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "The administrator has enabled the email domain name whitelist, and your email address is not allowed due to special symbols or it's not in the whitelist.",
			})
			return
		}
	}
	if common.EmailAliasRestrictionEnabled {
		containsSpecialSymbols := strings.Contains(localPart, "+") || strings.Contains(localPart, ".")
		if containsSpecialSymbols {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "管理员已启用邮箱地址别名限制，您的邮箱地址由于包含特殊符号而被拒绝。",
			})
			return
		}
	}

	if model.IsEmailAlreadyTaken(email) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "邮箱地址已被占用",
		})
		return
	}
	code := common.GenerateVerificationCode(6)
	common.RegisterVerificationCodeWithKey(email, code, common.EmailVerificationPurpose)
	subject := fmt.Sprintf("%s邮箱验证邮件", common.SystemName)
	content := fmt.Sprintf("<p>您好，你正在进行%s邮箱验证。</p>"+
		"<p>您的验证码为: <strong>%s</strong></p>"+
		"<p>验证码 %d 分钟内有效，如果不是本人操作，请忽略。</p>", common.SystemName, code, common.VerificationValidMinutes)
	err := common.SendEmail(subject, email, content)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

func SendPasswordResetEmail(c *gin.Context) {
	email := c.Query("email")
	if err := common.Validate.Var(email, "required,email"); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无效的参数",
		})
		return
	}
	if model.IsEmailAlreadyTaken(email) {
		code := common.GenerateVerificationCode(0)
		common.RegisterVerificationCodeWithKey(email, code, common.PasswordResetPurpose)
		link := fmt.Sprintf("%s/user/reset?email=%s&token=%s", system_setting.ServerAddress, email, code)
		subject := fmt.Sprintf("%s密码重置", common.SystemName)
		content := fmt.Sprintf("<p>您好，你正在进行%s密码重置。</p>"+
			"<p>点击 <a href='%s'>此处</a> 进行密码重置。</p>"+
			"<p>如果链接无法点击，请尝试点击下面的链接或将其复制到浏览器中打开：<br> %s </p>"+
			"<p>重置链接 %d 分钟内有效，如果不是本人操作，请忽略。</p>", common.SystemName, link, link, common.VerificationValidMinutes)
		err := common.SendEmail(subject, email, content)
		if err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("failed to send password reset email to %s: %s", email, err.Error()))
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

type PasswordResetRequest struct {
	Email string `json:"email"`
	Token string `json:"token"`
}

func ResetPassword(c *gin.Context) {
	var req PasswordResetRequest
	err := common.DecodeJson(c.Request.Body, &req)
	if req.Email == "" || req.Token == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无效的参数",
		})
		return
	}
	if !common.VerifyCodeWithKey(req.Email, req.Token, common.PasswordResetPurpose) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "重置链接非法或已过期",
		})
		return
	}
	password := common.GenerateVerificationCode(12)
	err = model.ResetUserPasswordByEmail(req.Email, password)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.DeleteKey(req.Email, common.PasswordResetPurpose)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    password,
	})
	return
}

// GetSitemap 生成 SEO Sitemap XML
func GetSitemap(c *gin.Context) {
	serverAddr := strings.TrimSuffix(system_setting.ServerAddress, "/")
	if serverAddr == "" {
		serverAddr = "https://example.com"
	}

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.Header("Cache-Control", "public, max-age=3600")

	urls := []string{
		serverAddr + "/",
		serverAddr + "/pricing",
		serverAddr + "/about",
		serverAddr + "/prompt-gallery",
		serverAddr + "/user-agreement",
		serverAddr + "/privacy-policy",
	}

	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	sb.WriteString("\n")
	sb.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	sb.WriteString("\n")

	for _, url := range urls {
		sb.WriteString("  <url>\n")
		sb.WriteString(fmt.Sprintf("    <loc>%s</loc>\n", url))
		sb.WriteString(fmt.Sprintf("    <lastmod>%s</lastmod>\n", time.Now().Format("2006-01-02")))
		sb.WriteString("    <changefreq>weekly</changefreq>\n")
		sb.WriteString("    <priority>0.8</priority>\n")
		sb.WriteString("  </url>\n")
	}

	// Add all public prompts to sitemap
	prompts, _, err := model.GetPublicPrompts(0, "", 0, 10000)
	if err == nil {
		for _, p := range prompts {
			sb.WriteString("  <url>\n")
			sb.WriteString(fmt.Sprintf("    <loc>%s/prompt/%d</loc>\n", serverAddr, p.Id))
			sb.WriteString(fmt.Sprintf("    <lastmod>%s</lastmod>\n", time.Now().Format("2006-01-02")))
			sb.WriteString("    <changefreq>weekly</changefreq>\n")
			sb.WriteString("    <priority>0.6</priority>\n")
			sb.WriteString("  </url>\n")
		}
	}

	sb.WriteString("</urlset>")
	c.Data(http.StatusOK, "application/xml; charset=utf-8", []byte(sb.String()))
}

// buildSEOKeywords 基于提示词内容智能构建丰富的 SEO 关键词
func buildSEOKeywords(prompt *model.Prompt) string {
	// 优先使用 Google Suggest API 生成的实时热门关键词
	if prompt.SeoKeywords != "" {
		return prompt.SeoKeywords
	}

	var kwSet = make(map[string]struct{})
	var result []string

	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		if _, ok := kwSet[s]; !ok {
			kwSet[s] = struct{}{}
			result = append(result, s)
		}
	}

	// 1. 核心标题词
	add(prompt.Title)

	// 2. 模型 + 标题组合（高价值）
	if prompt.Model != "" {
		add(prompt.Model)
		add(prompt.Model + " " + prompt.Title)
		add(prompt.Model + " Prompt")
		add(prompt.Model + " 提示词")
	}

	// 3. 标签
	if prompt.Tags != "" {
		var tags []string
		_ = common.Unmarshal([]byte(prompt.Tags), &tags)
		for _, t := range tags {
			add(t)
			add(t + " Prompt")
			add(t + " 提示词")
			if prompt.Model != "" {
				add(prompt.Model + " " + t)
			}
		}
	}

	// 4. 内容类型智能判断
	contentLower := strings.ToLower(prompt.Content + " " + prompt.Title)
	var contentType string
	if strings.Contains(contentLower, "image") || strings.Contains(contentLower, "图片") || strings.Contains(contentLower, "photo") || strings.Contains(contentLower, "绘画") {
		contentType = "image"
	} else if strings.Contains(contentLower, "video") || strings.Contains(contentLower, "视频") || strings.Contains(contentLower, "seedance") {
		contentType = "video"
	} else if strings.Contains(contentLower, "code") || strings.Contains(contentLower, "代码") || strings.Contains(contentLower, "programming") {
		contentType = "code"
	} else if strings.Contains(contentLower, "text") || strings.Contains(contentLower, "文案") || strings.Contains(contentLower, "writing") || strings.Contains(contentLower, "文章") {
		contentType = "text"
	}

	switch contentType {
	case "image":
		add("AI图片生成")
		add("AI绘画提示词")
		add("AI生图Prompt")
		add("图片生成提示词")
		add("AI图像生成")
	case "video":
		add("AI视频生成")
		add("视频生成提示词")
		add("AI视频Prompt")
		add("视频创作提示词")
	case "code":
		add("AI编程提示词")
		add("代码生成Prompt")
		add("AI代码辅助")
	case "text":
		add("AI文案生成")
		add("文本创作提示词")
		add("AI写作Prompt")
	}

	// 5. 从内容提取核心名词（简单分词，提取 2-4 字以上的词）
	contentWords := extractKeywordsFromText(prompt.Title + " " + prompt.Content)
	for _, w := range contentWords {
		if len([]rune(w)) >= 2 {
			add(w)
		}
	}

	// 6. 通用高价值长尾词
	add("AI提示词")
	add("Prompt工程")
	add("Prompt模板")
	add("AI创作")
	add("提示词分享")
	add("高质量Prompt")
	add("OpenNana提示词")

	// 限制总长度（Google 建议 meta keywords 不要太长）
	var filtered []string
	var totalLen int
	for _, k := range result {
		if totalLen+len(k) > 500 {
			break
		}
		filtered = append(filtered, k)
		totalLen += len(k) + 2 // +2 for ", "
	}

	return strings.Join(filtered, ", ")
}

// extractKeywordsFromText 从文本中提取潜在关键词（中文按字符，英文按空格）
func extractKeywordsFromText(text string) []string {
	var words []string
	// 简单提取：按空格分割英文，按常见标点分割中文短语
	replacer := strings.NewReplacer(
		"，", " ", "、", " ", "。", " ", "！", " ", "？", " ",
		",", " ", ".", " ", "!", " ", "?", " ",
		"\n", " ", "\t", " ",
	)
	parts := strings.Fields(replacer.Replace(text))
	seen := make(map[string]struct{})
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || len(p) < 2 {
			continue
		}
		// 过滤常见停用词
		stopWords := map[string]bool{
			"the": true, "a": true, "an": true, "is": true, "are": true, "was": true, "were": true,
			"be": true, "been": true, "being": true, "have": true, "has": true, "had": true,
			"do": true, "does": true, "did": true, "will": true, "would": true, "could": true,
			"should": true, "may": true, "might": true, "must": true, "shall": true,
			"can": true, "need": true, "dare": true, "ought": true, "used": true,
			"to": true, "of": true, "in": true, "for": true, "on": true, "with": true,
			"at": true, "by": true, "from": true, "as": true, "into": true, "through": true,
			"during": true, "before": true, "after": true, "above": true, "below": true,
			"between": true, "under": true, "again": true, "further": true, "then": true, "once": true,
			"这里": true, "那里": true, "这个": true, "那个": true, "然后": true, "但是": true,
			"因为": true, "所以": true, "如果": true, "虽然": true, "而且": true,
		}
		lower := strings.ToLower(p)
		if stopWords[lower] {
			continue
		}
		if _, ok := seen[lower]; !ok {
			seen[lower] = struct{}{}
			words = append(words, p)
		}
	}
	return words
}

// IndexPage 是嵌入的 SPA index.html，用于 SEO 页面注入
var IndexPage []byte

// GetPromptSEOPage 为每个提示词生成独立的 SEO HTML 页面（基于 SPA index.html 注入）
func GetPromptSEOPage(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Redirect(http.StatusFound, "/")
		return
	}

	prompt, err := model.GetPublicPromptById(id)
	if err != nil {
		c.Redirect(http.StatusFound, "/")
		return
	}

	// 多语言 SEO：优先 ?lang= 参数，其次用户语言偏好
	lang := c.Query("lang")
	if lang == "" {
		userId := c.GetInt("id")
		if userId > 0 {
			lang = model.GetUserLanguage(userId)
		}
	}
	prompt.ApplyLanguage(lang)

	serverAddr := strings.TrimSuffix(system_setting.ServerAddress, "/")
	if serverAddr == "" {
		serverAddr = "https://example.com"
	}

	pageURL := fmt.Sprintf("%s/prompt/%d", serverAddr, prompt.Id)
	title := prompt.Title
	if title == "" {
		title = "AI Prompt Gallery"
	}

	// Build description: prefer AI-generated intro, fallback to content
	description := prompt.Intro
	if description == "" {
		description = prompt.Description
	}
	if description == "" {
		description = prompt.Content
	}
	if len(description) > 200 {
		description = description[:200] + "..."
	}

	// Build rich SEO keywords from prompt data
	keywords := buildSEOKeywords(prompt.Prompt)

	// Clean content for display (escape HTML)
	escapeHTML := func(s string) string {
		s = strings.ReplaceAll(s, "<", "&lt;")
		s = strings.ReplaceAll(s, ">", "&gt;")
		return s
	}
	contentDisplay := escapeHTML(prompt.Content)
	contentEnDisplay := escapeHTML(prompt.ContentEn)
	introDisplay := escapeHTML(prompt.Intro)

	// Build FAQ section and Schema.org FAQPage markup
	faqHTML := ""
	faqSchemaJSON := ""
	if prompt.Faq != "" {
		var faqItems []struct {
			Question string `json:"question"`
			Answer   string `json:"answer"`
		}
		_ = common.Unmarshal([]byte(prompt.Faq), &faqItems)
		if len(faqItems) > 0 {
			var faqBuilder strings.Builder
			faqBuilder.WriteString(`<section class="seo-faq"><h2>常见问题</h2>`)
			for _, item := range faqItems {
				q := escapeHTML(item.Question)
				a := escapeHTML(item.Answer)
				faqBuilder.WriteString(fmt.Sprintf(`<details class="seo-faq-item"><summary>%s</summary><p>%s</p></details>`, q, a))
			}
			faqBuilder.WriteString(`</section>`)
			faqHTML = faqBuilder.String()

			faqSchema := map[string]interface{}{
				"@context": "https://schema.org",
				"@type":    "FAQPage",
				"mainEntity": func() []map[string]interface{} {
					var items []map[string]interface{}
					for _, item := range faqItems {
						items = append(items, map[string]interface{}{
							"@type":          "Question",
							"name":           item.Question,
							"acceptedAnswer": map[string]string{"@type": "Answer", "text": item.Answer},
						})
					}
					return items
				}(),
			}
			faqSchemaBytes, _ := common.Marshal(faqSchema)
			faqSchemaJSON = string(faqSchemaBytes)
		}
	}

	// Schema.org JSON-LD for CreativeWork
	schema := map[string]interface{}{
		"@context":    "https://schema.org",
		"@type":       "CreativeWork",
		"name":        title,
		"description": description,
		"url":         pageURL,
		"author": map[string]string{
			"@type": "Person",
			"name":  prompt.Author,
		},
		"datePublished": time.Unix(prompt.CreatedTime, 0).Format(time.RFC3339),
		"dateModified":  time.Unix(prompt.UpdatedTime, 0).Format(time.RFC3339),
		"keywords":      keywords,
	}
	if prompt.CoverImageUrl != "" {
		schema["image"] = prompt.CoverImageUrl
	}
	schemaJSON, _ := common.Marshal(schema)

	// Build SEO head injection
	var seoHead strings.Builder
	seoHead.WriteString(`<meta name="google-site-verification" content="W_hk0thQjq8IV0KmBQZXFVclXCfRxlhxRcjrhYDSxbg" />`)
	seoHead.WriteString(fmt.Sprintf(`<title>%s</title>`, title))
	seoHead.WriteString(fmt.Sprintf(`<meta name="description" content="%s">`, description))
	seoHead.WriteString(fmt.Sprintf(`<meta name="keywords" content="%s">`, keywords))
	seoHead.WriteString(fmt.Sprintf(`<link rel="canonical" href="%s">`, pageURL))
	seoHead.WriteString(fmt.Sprintf(`<meta property="og:title" content="%s">`, title))
	seoHead.WriteString(fmt.Sprintf(`<meta property="og:description" content="%s">`, description))
	seoHead.WriteString(fmt.Sprintf(`<meta property="og:url" content="%s">`, pageURL))
	seoHead.WriteString(`<meta property="og:type" content="article">`)
	if prompt.CoverImageUrl != "" {
		seoHead.WriteString(fmt.Sprintf(`<meta property="og:image" content="%s">`, prompt.CoverImageUrl))
		seoHead.WriteString(fmt.Sprintf(`<meta name="twitter:image" content="%s">`, prompt.CoverImageUrl))
	}
	seoHead.WriteString(`<meta name="twitter:card" content="summary_large_image">`)
	seoHead.WriteString(fmt.Sprintf(`<meta name="twitter:title" content="%s">`, title))
	seoHead.WriteString(fmt.Sprintf(`<meta name="twitter:description" content="%s">`, description))
	seoHead.WriteString(fmt.Sprintf(`<script type="application/ld+json">%s</script>`, string(schemaJSON)))
	if faqSchemaJSON != "" {
		seoHead.WriteString(fmt.Sprintf(`<script type="application/ld+json">%s</script>`, faqSchemaJSON))
	}
	seoHead.WriteString(`<style>.seo-content{display:none}</style>`)

	// Build noscript visible content for crawlers
	var seoBody strings.Builder
	seoBody.WriteString(`<noscript><div class="seo-noscript" style="max-width:800px;margin:0 auto;padding:20px;font-family:sans-serif;line-height:1.6">`)
	seoBody.WriteString(fmt.Sprintf(`<h1>%s</h1>`, title))
	if prompt.CoverImageUrl != "" {
		seoBody.WriteString(fmt.Sprintf(`<img src="%s" alt="%s" style="max-width:100%%;border-radius:12px;margin-bottom:16px">`, prompt.CoverImageUrl, title))
	}
	if introDisplay != "" {
		seoBody.WriteString(fmt.Sprintf(`<p style="background:#eef2ff;padding:16px;border-radius:8px;border-left:4px solid #4f46e5">%s</p>`, introDisplay))
	}
	if prompt.Author != "" {
		seoBody.WriteString(fmt.Sprintf(`<p>来源: %s</p>`, prompt.Author))
	}
	if prompt.Model != "" {
		seoBody.WriteString(fmt.Sprintf(`<p>模型: %s</p>`, prompt.Model))
	}
	if prompt.MediaType != "" {
		seoBody.WriteString(fmt.Sprintf(`<p>类型: %s</p>`, prompt.MediaType))
	}
	seoBody.WriteString(fmt.Sprintf(`<h2>提示词内容</h2><pre style="background:#f8f9fa;padding:16px;border-radius:8px;white-space:pre-wrap">%s</pre>`, contentDisplay))
	if contentEnDisplay != "" {
		seoBody.WriteString(fmt.Sprintf(`<h2>English Prompt</h2><pre style="background:#f8f9fa;padding:16px;border-radius:8px;white-space:pre-wrap">%s</pre>`, contentEnDisplay))
	}
	seoBody.WriteString(faqHTML)
	seoBody.WriteString(`<p><a href="/prompt-gallery" style="display:inline-block;background:#4f46e5;color:#fff;padding:12px 24px;border-radius:8px;text-decoration:none">浏览更多提示词</a></p>`)
	seoBody.WriteString(`</div></noscript>`)

	// Inject into index.html
	htmlStr := string(IndexPage)
	// Remove default title/meta and inject SEO head before </head>
	htmlStr = strings.Replace(htmlStr, "<title>New API</title>", "", 1)
	htmlStr = strings.Replace(htmlStr, "</head>", seoHead.String()+"</head>", 1)
	// Inject noscript content right after <body>
	htmlStr = strings.Replace(htmlStr, "<body>", "<body>"+seoBody.String(), 1)

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Header("Cache-Control", "public, max-age=3600")
	c.String(http.StatusOK, htmlStr)
}

// GetArticleSEOPage 为每篇文章生成独立的 SEO HTML 页面（基于 SPA index.html 注入）
func GetArticleSEOPage(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Redirect(http.StatusFound, "/")
		return
	}

	article, err := model.GetPublicArticleById(id)
	if err != nil {
		c.Redirect(http.StatusFound, "/")
		return
	}

	// 多语言 SEO：优先 ?lang= 参数，其次用户语言偏好
	lang := c.Query("lang")
	if lang == "" {
		userId := c.GetInt("id")
		if userId > 0 {
			lang = model.GetUserLanguage(userId)
		}
	}
	article.ApplyLanguage(lang)

	serverAddr := strings.TrimSuffix(system_setting.ServerAddress, "/")
	if serverAddr == "" {
		serverAddr = "https://example.com"
	}

	pageURL := fmt.Sprintf("%s/article/%d", serverAddr, article.Id)

	// 优先使用 SEO 标题，其次使用文章标题
	title := article.SeoTitle
	if title == "" {
		title = article.Title
	}
	if title == "" {
		title = "Article"
	}

	// 优先使用 SEO 描述，其次摘要，再次内容
	description := article.SeoDescription
	if description == "" {
		description = article.Summary
	}
	if description == "" {
		description = article.Content
	}
	if len(description) > 200 {
		description = description[:200] + "..."
	}

	keywords := article.SeoKeywords
	if keywords == "" && article.Tags != "" {
		var tags []string
		_ = common.Unmarshal([]byte(article.Tags), &tags)
		keywords = strings.Join(tags, ", ")
	}
	if keywords == "" {
		keywords = article.Title
	}

	// Clean content for display (escape HTML)
	escapeHTML := func(s string) string {
		s = strings.ReplaceAll(s, "<", "&lt;")
		s = strings.ReplaceAll(s, ">", "&gt;")
		return s
	}
	contentDisplay := escapeHTML(article.Content)
	summaryDisplay := escapeHTML(article.Summary)

	// Schema.org JSON-LD for Article
	schema := map[string]interface{}{
		"@context":    "https://schema.org",
		"@type":       "Article",
		"headline":    title,
		"description": description,
		"url":         pageURL,
		"author": map[string]string{
			"@type": "Person",
			"name":  article.Author,
		},
		"datePublished": time.Unix(article.CreatedTime, 0).Format(time.RFC3339),
		"dateModified":  time.Unix(article.UpdatedTime, 0).Format(time.RFC3339),
		"keywords":      keywords,
	}
	if article.CoverImageUrl != "" {
		schema["image"] = article.CoverImageUrl
	}
	schemaJSON, _ := common.Marshal(schema)

	// Build SEO head injection
	var seoHead strings.Builder
	seoHead.WriteString(`<meta name="google-site-verification" content="W_hk0thQjq8IV0KmBQZXFVclXCfRxlhxRcjrhYDSxbg" />`)
	seoHead.WriteString(fmt.Sprintf(`<title>%s</title>`, title))
	seoHead.WriteString(fmt.Sprintf(`<meta name="description" content="%s">`, description))
	seoHead.WriteString(fmt.Sprintf(`<meta name="keywords" content="%s">`, keywords))
	seoHead.WriteString(fmt.Sprintf(`<link rel="canonical" href="%s">`, pageURL))
	seoHead.WriteString(fmt.Sprintf(`<meta property="og:title" content="%s">`, title))
	seoHead.WriteString(fmt.Sprintf(`<meta property="og:description" content="%s">`, description))
	seoHead.WriteString(fmt.Sprintf(`<meta property="og:url" content="%s">`, pageURL))
	seoHead.WriteString(`<meta property="og:type" content="article">`)
	if article.CoverImageUrl != "" {
		seoHead.WriteString(fmt.Sprintf(`<meta property="og:image" content="%s">`, article.CoverImageUrl))
		seoHead.WriteString(fmt.Sprintf(`<meta name="twitter:image" content="%s">`, article.CoverImageUrl))
	}
	seoHead.WriteString(`<meta name="twitter:card" content="summary_large_image">`)
	seoHead.WriteString(fmt.Sprintf(`<meta name="twitter:title" content="%s">`, title))
	seoHead.WriteString(fmt.Sprintf(`<meta name="twitter:description" content="%s">`, description))
	seoHead.WriteString(fmt.Sprintf(`<script type="application/ld+json">%s</script>`, string(schemaJSON)))
	seoHead.WriteString(`<style>.seo-content{display:none}</style>`)

	// Build noscript visible content for crawlers
	var seoBody strings.Builder
	seoBody.WriteString(`<noscript><div class="seo-noscript" style="max-width:800px;margin:0 auto;padding:20px;font-family:sans-serif;line-height:1.6">`)
	seoBody.WriteString(fmt.Sprintf(`<h1>%s</h1>`, title))
	if article.CoverImageUrl != "" {
		seoBody.WriteString(fmt.Sprintf(`<img src="%s" alt="%s" style="max-width:100%%;border-radius:12px;margin-bottom:16px">`, article.CoverImageUrl, title))
	}
	if article.Author != "" {
		seoBody.WriteString(fmt.Sprintf(`<p>作者: %s</p>`, article.Author))
	}
	seoBody.WriteString(fmt.Sprintf(`<p>发布时间: %s</p>`, time.Unix(article.CreatedTime, 0).Format("2006-01-02")))
	if summaryDisplay != "" {
		seoBody.WriteString(fmt.Sprintf(`<p style="background:#eef2ff;padding:16px;border-radius:8px;border-left:4px solid #4f46e5">%s</p>`, summaryDisplay))
	}
	// Article content as markdown text (escaped) — crawlers can read the raw text
	seoBody.WriteString(fmt.Sprintf(`<h2>正文</h2><pre style="background:#f8f9fa;padding:16px;border-radius:8px;white-space:pre-wrap">%s</pre>`, contentDisplay))
	seoBody.WriteString(`<p><a href="/article-gallery" style="display:inline-block;background:#4f46e5;color:#fff;padding:12px 24px;border-radius:8px;text-decoration:none">浏览更多文章</a></p>`)
	seoBody.WriteString(`</div></noscript>`)

	// Inject into index.html
	htmlStr := string(IndexPage)
	htmlStr = strings.Replace(htmlStr, "<title>New API</title>", "", 1)
	htmlStr = strings.Replace(htmlStr, "</head>", seoHead.String()+"</head>", 1)
	htmlStr = strings.Replace(htmlStr, "<body>", "<body>"+seoBody.String(), 1)

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Header("Cache-Control", "public, max-age=3600")
	c.String(http.StatusOK, htmlStr)
}
