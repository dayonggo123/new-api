package controller

import (
	"encoding/json"
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
	err := json.NewDecoder(c.Request.Body).Decode(&req)
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
	c.String(http.StatusOK, sb.String())
}

// GetPromptSEOPage 为每个提示词生成独立的 SEO HTML 页面
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

	serverAddr := strings.TrimSuffix(system_setting.ServerAddress, "/")
	if serverAddr == "" {
		serverAddr = "https://example.com"
	}

	pageURL := fmt.Sprintf("%s/prompt/%d", serverAddr, prompt.Id)
	title := prompt.Title
	if title == "" {
		title = "Prompt Gallery"
	}

	// Build description from content (truncated)
	description := prompt.Description
	if description == "" {
		description = prompt.Content
	}
	if len(description) > 200 {
		description = description[:200] + "..."
	}

	// Build keywords from tags
	var keywords string
	if prompt.Tags != "" {
		var tagArr []string
		_ = json.Unmarshal([]byte(prompt.Tags), &tagArr)
		keywords = strings.Join(tagArr, ", ")
	}
	if prompt.Model != "" {
		if keywords != "" {
			keywords += ", "
		}
		keywords += prompt.Model
	}
	if keywords == "" {
		keywords = "AI提示词, Prompt工程, AI创作"
	}

	// Clean content for display (escape HTML)
	contentDisplay := strings.ReplaceAll(prompt.Content, "<", "&lt;")
	contentDisplay = strings.ReplaceAll(contentDisplay, ">", "&gt;")
	contentEnDisplay := strings.ReplaceAll(prompt.ContentEn, "<", "&lt;")
	contentEnDisplay = strings.ReplaceAll(contentEnDisplay, ">", "&gt;")

	// Schema.org JSON-LD
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
	schemaJSON, _ := json.Marshal(schema)

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Header("Cache-Control", "public, max-age=3600")

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>%s - OpenNana Prompt Gallery</title>
  <meta name="description" content="%s">
  <meta name="keywords" content="%s">
  <link rel="canonical" href="%s">
  <meta property="og:title" content="%s">
  <meta property="og:description" content="%s">
  <meta property="og:url" content="%s">
  <meta property="og:type" content="article">
  <meta property="og:site_name" content="OpenNana Prompt Gallery">
  %s
  <meta name="twitter:card" content="summary_large_image">
  <meta name="twitter:title" content="%s">
  <meta name="twitter:description" content="%s">
  %s
  <script type="application/ld+json">%s</script>
  <style>
    body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,"Helvetica Neue",Arial,sans-serif;line-height:1.6;color:#333;max-width:800px;margin:0 auto;padding:20px}
    h1{color:#4f46e5;font-size:28px;margin-bottom:12px}
    .meta{color:#666;font-size:14px;margin-bottom:20px}
    .cover{width:100%%;max-width:640px;border-radius:12px;margin-bottom:20px}
    .tag{display:inline-block;background:#f0f0f0;padding:4px 12px;border-radius:16px;font-size:13px;margin-right:8px;margin-bottom:8px}
    .content{background:#f8f9fa;padding:16px;border-radius:8px;white-space:pre-wrap;word-break:break-word;margin-bottom:20px}
    .btn{display:inline-block;background:#4f46e5;color:#fff;padding:12px 24px;border-radius:8px;text-decoration:none;font-weight:500}
    .btn:hover{background:#4338ca}
    footer{margin-top:40px;padding-top:20px;border-top:1px solid #eee;color:#999;font-size:13px}
  </style>
</head>
<body>
  <h1>%s</h1>
  <div class="meta">
    %s
    %s
  </div>
  %s
  %s
  <div class="content">%s</div>
  %s
  <a class="btn" href="/#/prompt-gallery">在完整网站中查看</a>
  <footer>
    <p>© OpenNana Prompt Gallery · <a href="%s">%s</a></p>
  </footer>
</body>
</html>`,
		title, description, keywords, pageURL,
		title, description, pageURL,
		func() string {
			if prompt.CoverImageUrl != "" {
				return fmt.Sprintf(`<meta property="og:image" content="%s">`, prompt.CoverImageUrl)
			}
			return ""
		}(),
		title, description,
		func() string {
			if prompt.CoverImageUrl != "" {
				return fmt.Sprintf(`<meta name="twitter:image" content="%s">`, prompt.CoverImageUrl)
			}
			return ""
		}(),
		string(schemaJSON),
		title,
		func() string {
			if prompt.Author != "" {
				return fmt.Sprintf(`<span>来源: %s</span> · `, prompt.Author)
			}
			return ""
		}(),
		func() string {
			if prompt.Model != "" {
				return fmt.Sprintf(`<span>模型: %s</span>`, prompt.Model)
			}
			return ""
		}(),
		func() string {
			if prompt.CoverImageUrl != "" {
				return fmt.Sprintf(`<img class="cover" src="%s" alt="%s">`, prompt.CoverImageUrl, title)
			}
			return ""
		}(),
		func() string {
			if keywords != "" {
				var tags []string
				_ = json.Unmarshal([]byte(prompt.Tags), &tags)
				var tagHTML strings.Builder
				for _, tag := range tags {
					tagHTML.WriteString(fmt.Sprintf(`<span class="tag">%s</span>`, tag))
				}
				return tagHTML.String()
			}
			return ""
		}(),
		contentDisplay,
		func() string {
			if contentEnDisplay != "" {
				return fmt.Sprintf(`<h3>English</h3><div class="content">%s</div>`, contentEnDisplay)
			}
			return ""
		}(),
		pageURL, pageURL,
	)

	c.String(http.StatusOK, html)
}
