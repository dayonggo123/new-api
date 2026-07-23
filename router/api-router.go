package router

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"

	// Import oauth package to register providers via init()
	_ "github.com/QuantumNous/new-api/oauth"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func SetApiRouter(router *gin.Engine) {
	apiRouter := router.Group("/api")
	apiRouter.Use(middleware.CORS())
	apiRouter.Use(middleware.RouteTag("api"))
	apiRouter.Use(gzip.Gzip(gzip.DefaultCompression))
	apiRouter.Use(middleware.BodyStorageCleanup()) // 清理请求体存储
	apiRouter.Use(middleware.GlobalAPIRateLimit())
	{
		apiRouter.GET("/setup", controller.GetSetup)
		apiRouter.POST("/setup", controller.PostSetup)
		apiRouter.GET("/status", controller.GetStatus)
		apiRouter.GET("/time", controller.GetServerTime)
		apiRouter.GET("/uptime/status", controller.GetUptimeKumaStatus)
		apiRouter.GET("/models", middleware.UserAuth(), controller.DashboardListModels)
		apiRouter.GET("/status/test", middleware.AdminAuth(), controller.TestStatus)
		apiRouter.GET("/notice", controller.GetNotice)
		apiRouter.GET("/user-agreement", controller.GetUserAgreement)
		apiRouter.GET("/privacy-policy", controller.GetPrivacyPolicy)
		apiRouter.GET("/about", controller.GetAbout)
		//apiRouter.GET("/midjourney", controller.GetMidjourney)
		apiRouter.GET("/home_page_content", controller.GetHomePageContent)
		apiRouter.GET("/pricing", middleware.TryUserAuth(), controller.GetPricing)
		apiRouter.GET("/verification", middleware.EmailVerificationRateLimit(), middleware.TurnstileCheck(), controller.SendEmailVerification)
		apiRouter.GET("/reset_password", middleware.CriticalRateLimit(), middleware.TurnstileCheck(), controller.SendPasswordResetEmail)
		apiRouter.POST("/user/reset", middleware.CriticalRateLimit(), controller.ResetPassword)
		// OAuth routes - specific routes must come before :provider wildcard
		apiRouter.GET("/oauth/state", middleware.CriticalRateLimit(), controller.GenerateOAuthCode)
		apiRouter.POST("/oauth/email/bind", middleware.CriticalRateLimit(), controller.EmailBind)
		// Non-standard OAuth (WeChat, Telegram) - keep original routes
		apiRouter.GET("/oauth/wechat", middleware.CriticalRateLimit(), controller.WeChatAuth)
		apiRouter.POST("/oauth/wechat/bind", middleware.CriticalRateLimit(), controller.WeChatBind)
		apiRouter.GET("/oauth/telegram/login", middleware.CriticalRateLimit(), controller.TelegramLogin)
		apiRouter.GET("/oauth/telegram/bind", middleware.CriticalRateLimit(), controller.TelegramBind)
		// Standard OAuth providers (GitHub, Discord, OIDC, LinuxDO) - unified route
		apiRouter.GET("/oauth/:provider", middleware.CriticalRateLimit(), controller.HandleOAuth)
		apiRouter.GET("/ratio_config", middleware.CriticalRateLimit(), controller.GetRatioConfig)
		apiRouter.GET("/skills", middleware.TryUserAuth(), controller.GetSkills)
		skillAdminRoute := apiRouter.Group("/skills")
		skillAdminRoute.Use(middleware.AdminAuth())
		{
			skillAdminRoute.GET("/all", controller.AdminListSkills)
			skillAdminRoute.POST("/", controller.AdminCreateSkill)
			skillAdminRoute.PUT("/:id", controller.AdminUpdateSkill)
			skillAdminRoute.DELETE("/:id", controller.AdminDeleteSkill)
		}

		apiRouter.POST("/stripe/webhook", controller.StripeWebhook)
		apiRouter.POST("/creem/webhook", controller.CreemWebhook)
		apiRouter.POST("/waffo/webhook", controller.WaffoWebhook)

		// Universal secure verification routes
		apiRouter.POST("/verify", middleware.UserAuth(), middleware.CriticalRateLimit(), controller.UniversalVerify)

		userRoute := apiRouter.Group("/user")
		{
			userRoute.POST("/register", middleware.CriticalRateLimit(), middleware.TurnstileCheck(), controller.Register)
			userRoute.POST("/login", middleware.CriticalRateLimit(), middleware.TurnstileCheck(), controller.Login)
			userRoute.POST("/login/2fa", middleware.CriticalRateLimit(), controller.Verify2FALogin)
			userRoute.POST("/passkey/login/begin", middleware.CriticalRateLimit(), controller.PasskeyLoginBegin)
			userRoute.POST("/passkey/login/finish", middleware.CriticalRateLimit(), controller.PasskeyLoginFinish)
			//userRoute.POST("/tokenlog", middleware.CriticalRateLimit(), controller.TokenLog)
			userRoute.GET("/logout", controller.Logout)
			userRoute.POST("/epay/notify", controller.EpayNotify)
			userRoute.GET("/epay/notify", controller.EpayNotify)
			userRoute.GET("/groups", controller.GetUserGroups)

			selfRoute := userRoute.Group("/")
			selfRoute.Use(middleware.UserAuth())
			{
				selfRoute.GET("/self/groups", controller.GetUserGroups)
				selfRoute.GET("/self", controller.GetSelf)
				selfRoute.GET("/models", controller.GetUserModels)
				selfRoute.PUT("/self", controller.UpdateSelf)
				selfRoute.DELETE("/self", controller.DeleteSelf)
				selfRoute.GET("/token", controller.GenerateAccessToken)
				selfRoute.GET("/passkey", controller.PasskeyStatus)
				selfRoute.GET("/tokens", controller.GetUserTokensFull)
				selfRoute.POST("/auto-token", controller.AutoToken)
				selfRoute.POST("/passkey/register/begin", controller.PasskeyRegisterBegin)
				selfRoute.POST("/passkey/register/finish", controller.PasskeyRegisterFinish)
				selfRoute.POST("/passkey/verify/begin", controller.PasskeyVerifyBegin)
				selfRoute.POST("/passkey/verify/finish", controller.PasskeyVerifyFinish)
				selfRoute.DELETE("/passkey", controller.PasskeyDelete)
				selfRoute.GET("/aff", controller.GetAffCode)
				selfRoute.GET("/topup/info", controller.GetTopUpInfo)
				selfRoute.GET("/topup/self", controller.GetUserTopUps)
				selfRoute.POST("/topup", middleware.CriticalRateLimit(), controller.TopUp)
				selfRoute.POST("/pay", middleware.CriticalRateLimit(), controller.RequestEpay)
				selfRoute.POST("/amount", controller.RequestAmount)
				selfRoute.POST("/stripe/pay", middleware.CriticalRateLimit(), controller.RequestStripePay)
				selfRoute.POST("/stripe/amount", controller.RequestStripeAmount)
				selfRoute.POST("/creem/pay", middleware.CriticalRateLimit(), controller.RequestCreemPay)
				selfRoute.POST("/waffo/pay", middleware.CriticalRateLimit(), controller.RequestWaffoPay)
				selfRoute.POST("/aff_transfer", controller.TransferAffQuota)
				selfRoute.PUT("/setting", controller.UpdateUserSetting)

				// 2FA routes
				selfRoute.GET("/2fa/status", controller.Get2FAStatus)
				selfRoute.POST("/2fa/setup", controller.Setup2FA)
				selfRoute.POST("/2fa/enable", controller.Enable2FA)
				selfRoute.POST("/2fa/disable", controller.Disable2FA)
				selfRoute.POST("/2fa/backup_codes", controller.RegenerateBackupCodes)

				// Check-in routes
				selfRoute.GET("/checkin", controller.GetCheckinStatus)
				selfRoute.POST("/checkin", middleware.TurnstileCheck(), controller.DoCheckin)

				// Points & Signin routes (new points system)
				selfRoute.GET("/points", controller.GetUserPoints)
				selfRoute.POST("/signin", controller.DoSignin)
				selfRoute.GET("/signin-history", controller.GetSigninHistory)
				selfRoute.POST("/unlock-prompt", controller.UnlockPrompt)
				selfRoute.GET("/unlocked-prompts", controller.GetUnlockedPrompts)

				// Custom OAuth bindings
				selfRoute.GET("/oauth/bindings", controller.GetUserOAuthBindings)
				selfRoute.DELETE("/oauth/bindings/:provider_id", controller.UnbindCustomOAuth)
			}

			adminRoute := userRoute.Group("/")
			adminRoute.Use(middleware.AdminAuth())
			{
				adminRoute.GET("/", controller.GetAllUsers)
				adminRoute.GET("/topup", controller.GetAllTopUps)
				adminRoute.POST("/topup/complete", controller.AdminCompleteTopUp)
				adminRoute.GET("/search", controller.SearchUsers)
				adminRoute.GET("/:id/oauth/bindings", controller.GetUserOAuthBindingsByAdmin)
				adminRoute.DELETE("/:id/oauth/bindings/:provider_id", controller.UnbindCustomOAuthByAdmin)
				adminRoute.DELETE("/:id/bindings/:binding_type", controller.AdminClearUserBinding)
				adminRoute.GET("/:id", controller.GetUser)
				adminRoute.GET("/:id/devices", controller.GetUserDeviceSessions)
				adminRoute.DELETE("/:id/devices/:device_id", controller.KickUserDevice)
				adminRoute.POST("/", controller.CreateUser)
				adminRoute.POST("/manage", controller.ManageUser)
				adminRoute.PUT("/", controller.UpdateUser)
				adminRoute.DELETE("/:id", controller.DeleteUser)
				adminRoute.DELETE("/:id/reset_passkey", controller.AdminResetPasskey)

				// Admin 2FA routes
				adminRoute.GET("/2fa/stats", controller.Admin2FAStats)
				adminRoute.DELETE("/:id/2fa", controller.AdminDisable2FA)

				// Admin Points Management
				adminRoute.POST("/points/adjust", controller.AdminAdjustUserPoints)
				adminRoute.GET("/points/transactions", controller.AdminGetUserPointsTransactions)
				adminRoute.GET("/signin/stats", controller.AdminGetSigninStats)
			}
		}

		// Popup (daily announcement)
		apiRouter.GET("/popups/daily", middleware.SubscriptionAuth(), controller.GetDailyPopup)
		popupAdminRoute := apiRouter.Group("/admin/popups")
		popupAdminRoute.Use(middleware.AdminAuth())
		{
			popupAdminRoute.GET("/", controller.AdminListPopups)
			popupAdminRoute.POST("/", controller.AdminCreatePopup)
			popupAdminRoute.PUT("/", controller.AdminUpdatePopup)
			popupAdminRoute.DELETE("/:id", controller.AdminDeletePopup)
		}

		// Marketing Banners
		apiRouter.GET("/marketing/banners", middleware.UserAuth(), controller.GetMarketingBanners)
		bannerAdminRoute := apiRouter.Group("/admin/marketing/banners")
		bannerAdminRoute.Use(middleware.AdminAuth())
		{
			bannerAdminRoute.GET("/", controller.AdminListBanners)
			bannerAdminRoute.POST("/", controller.AdminCreateBanner)
			bannerAdminRoute.PUT("/", controller.AdminUpdateBanner)
			bannerAdminRoute.DELETE("/:id", controller.AdminDeleteBanner)
		}

		// Curated workflow templates
		apiRouter.GET("/curated/templates", controller.GetCuratedTemplates)
		apiRouter.GET("/curated/templates/:id", controller.GetCuratedTemplate)
		apiRouter.GET("/curated/templates/:id/execution-plan", controller.GetCuratedTemplateExecutionPlan)
		apiRouter.GET("/curated/categories", controller.GetCuratedCategories)

		curatedTemplateAdminRoute := apiRouter.Group("/admin/curated/templates")
		curatedTemplateAdminRoute.Use(middleware.AdminAuth())
		{
			curatedTemplateAdminRoute.GET("/", controller.AdminListCuratedTemplates)
			curatedTemplateAdminRoute.POST("/", controller.AdminCreateCuratedTemplate)
			curatedTemplateAdminRoute.PUT("/:id", controller.AdminUpdateCuratedTemplate)
			curatedTemplateAdminRoute.DELETE("/:id", controller.AdminDeleteCuratedTemplate)
			curatedTemplateAdminRoute.PATCH("/:id/status", controller.AdminUpdateCuratedTemplateStatus)
		}

		curatedCategoryAdminRoute := apiRouter.Group("/admin/curated/categories")
		curatedCategoryAdminRoute.Use(middleware.AdminAuth())
		{
			curatedCategoryAdminRoute.GET("/", controller.AdminListCuratedCategories)
			curatedCategoryAdminRoute.POST("/", controller.AdminCreateCuratedCategory)
			curatedCategoryAdminRoute.PUT("/:id", controller.AdminUpdateCuratedCategory)
			curatedCategoryAdminRoute.DELETE("/:id", controller.AdminDeleteCuratedCategory)
		}

		// Subscription billing (plans, purchase, admin management)
		// Plans list is publicly accessible (like pricing)
		apiRouter.GET("/subscription/plans", middleware.TryUserAuth(), controller.GetSubscriptionPlans)
		subscriptionRoute := apiRouter.Group("/subscription")
		subscriptionRoute.Use(middleware.SubscriptionAuth())
		{
			subscriptionRoute.GET("/self", controller.GetSubscriptionSelf)
			subscriptionRoute.PUT("/self/preference", controller.UpdateSubscriptionPreference)
			subscriptionRoute.POST("/discount/validate", controller.ValidateSubscriptionDiscount)
			subscriptionRoute.POST("/epay/pay", middleware.CriticalRateLimit(), controller.SubscriptionRequestEpay)
			subscriptionRoute.POST("/yizhifu/pay", middleware.CriticalRateLimit(), controller.SubscriptionRequestYizhifuV1)
			subscriptionRoute.POST("/stripe/pay", middleware.CriticalRateLimit(), controller.SubscriptionRequestStripePay)
			subscriptionRoute.POST("/creem/pay", middleware.CriticalRateLimit(), controller.SubscriptionRequestCreemPay)
		}
		subscriptionAdminRoute := apiRouter.Group("/subscription/admin")
		subscriptionAdminRoute.Use(middleware.AdminAuth())
		{
			subscriptionAdminRoute.GET("/plans", controller.AdminListSubscriptionPlans)
			subscriptionAdminRoute.POST("/plans", controller.AdminCreateSubscriptionPlan)
			subscriptionAdminRoute.PUT("/plans/:id", controller.AdminUpdateSubscriptionPlan)
			subscriptionAdminRoute.PATCH("/plans/:id", controller.AdminUpdateSubscriptionPlanStatus)
			subscriptionAdminRoute.POST("/bind", controller.AdminBindSubscription)

			// User subscription management (admin)
			subscriptionAdminRoute.GET("/users/:id/subscriptions", controller.AdminListUserSubscriptions)
			subscriptionAdminRoute.POST("/users/:id/subscriptions", controller.AdminCreateUserSubscription)
			subscriptionAdminRoute.POST("/user_subscriptions/:id/invalidate", controller.AdminInvalidateUserSubscription)
			subscriptionAdminRoute.DELETE("/user_subscriptions/:id", controller.AdminDeleteUserSubscription)

			// Discount code management (admin)
			subscriptionAdminRoute.GET("/discounts", controller.AdminListSubscriptionDiscounts)
			subscriptionAdminRoute.POST("/discounts", controller.AdminCreateSubscriptionDiscount)
			subscriptionAdminRoute.PUT("/discounts/:id", controller.AdminUpdateSubscriptionDiscount)
			subscriptionAdminRoute.DELETE("/discounts/:id", controller.AdminDeleteSubscriptionDiscount)
			subscriptionAdminRoute.PATCH("/discounts/:id/status", controller.AdminUpdateSubscriptionDiscountStatus)
		}

		// Subscription payment callbacks (no auth)
		apiRouter.POST("/subscription/epay/notify", controller.SubscriptionEpayNotify)
		apiRouter.GET("/subscription/epay/notify", controller.SubscriptionEpayNotify)
		apiRouter.GET("/subscription/epay/return", controller.SubscriptionEpayReturn)
		apiRouter.POST("/subscription/epay/return", controller.SubscriptionEpayReturn)

		apiRouter.POST("/subscription/yizhifu/notify", controller.SubscriptionYizhifuV1Notify)
		apiRouter.GET("/subscription/yizhifu/notify", controller.SubscriptionYizhifuV1Notify)
		apiRouter.GET("/subscription/yizhifu/return", controller.SubscriptionYizhifuV1Return)
		apiRouter.POST("/subscription/yizhifu/return", controller.SubscriptionYizhifuV1Return)
		optionRoute := apiRouter.Group("/option")
		optionRoute.Use(middleware.RootAuth())
		{
			optionRoute.GET("/", controller.GetOptions)
			optionRoute.PUT("/", controller.UpdateOption)
			optionRoute.GET("/channel_affinity_cache", controller.GetChannelAffinityCacheStats)
			optionRoute.DELETE("/channel_affinity_cache", controller.ClearChannelAffinityCache)
			optionRoute.POST("/rest_model_ratio", controller.ResetModelRatio)
			optionRoute.POST("/migrate_console_setting", controller.MigrateConsoleSetting) // 用于迁移检测的旧键，下个版本会删除
		}

		// Custom OAuth provider management (root only)
		customOAuthRoute := apiRouter.Group("/custom-oauth-provider")
		customOAuthRoute.Use(middleware.RootAuth())
		{
			customOAuthRoute.POST("/discovery", controller.FetchCustomOAuthDiscovery)
			customOAuthRoute.GET("/", controller.GetCustomOAuthProviders)
			customOAuthRoute.GET("/:id", controller.GetCustomOAuthProvider)
			customOAuthRoute.POST("/", controller.CreateCustomOAuthProvider)
			customOAuthRoute.PUT("/:id", controller.UpdateCustomOAuthProvider)
			customOAuthRoute.DELETE("/:id", controller.DeleteCustomOAuthProvider)
		}
		performanceRoute := apiRouter.Group("/performance")
		performanceRoute.Use(middleware.RootAuth())
		{
			performanceRoute.GET("/stats", controller.GetPerformanceStats)
			performanceRoute.DELETE("/disk_cache", controller.ClearDiskCache)
			performanceRoute.POST("/reset_stats", controller.ResetPerformanceStats)
			performanceRoute.POST("/gc", controller.ForceGC)
			performanceRoute.GET("/logs", controller.GetLogFiles)
			performanceRoute.DELETE("/logs", controller.CleanupLogFiles)
		}
		ratioSyncRoute := apiRouter.Group("/ratio_sync")
		ratioSyncRoute.Use(middleware.RootAuth())
		{
			ratioSyncRoute.GET("/channels", controller.GetSyncableChannels)
			ratioSyncRoute.POST("/fetch", controller.FetchUpstreamRatios)
		}
		channelRoute := apiRouter.Group("/channel")
		channelRoute.Use(middleware.AdminAuth())
		{
			channelRoute.GET("/", controller.GetAllChannels)
			channelRoute.GET("/search", controller.SearchChannels)
			channelRoute.GET("/models", controller.ChannelListModels)
			channelRoute.GET("/models_enabled", controller.EnabledListModels)
			channelRoute.GET("/:id", controller.GetChannel)
			channelRoute.POST("/:id/key", middleware.RootAuth(), middleware.CriticalRateLimit(), middleware.DisableCache(), middleware.SecureVerificationRequired(), controller.GetChannelKey)
			channelRoute.GET("/test", controller.TestAllChannels)
			channelRoute.GET("/test/:id", controller.TestChannel)
			channelRoute.GET("/update_balance", controller.UpdateAllChannelsBalance)
			channelRoute.GET("/update_balance/:id", controller.UpdateChannelBalance)
			channelRoute.POST("/", controller.AddChannel)
			channelRoute.PUT("/", controller.UpdateChannel)
			channelRoute.DELETE("/disabled", controller.DeleteDisabledChannel)
			channelRoute.POST("/tag/disabled", controller.DisableTagChannels)
			channelRoute.POST("/tag/enabled", controller.EnableTagChannels)
			channelRoute.PUT("/tag", controller.EditTagChannels)
			channelRoute.DELETE("/:id", controller.DeleteChannel)
			channelRoute.POST("/batch", controller.DeleteChannelBatch)
			channelRoute.POST("/fix", controller.FixChannelsAbilities)
			channelRoute.GET("/fetch_models/:id", controller.FetchUpstreamModels)
			channelRoute.POST("/fetch_models", middleware.RootAuth(), controller.FetchModels)
			channelRoute.POST("/codex/oauth/start", controller.StartCodexOAuth)
			channelRoute.POST("/codex/oauth/complete", controller.CompleteCodexOAuth)
			channelRoute.POST("/:id/codex/oauth/start", controller.StartCodexOAuthForChannel)
			channelRoute.POST("/:id/codex/oauth/complete", controller.CompleteCodexOAuthForChannel)
			channelRoute.POST("/:id/codex/refresh", controller.RefreshCodexChannelCredential)
			channelRoute.GET("/:id/codex/usage", controller.GetCodexChannelUsage)
			channelRoute.POST("/ollama/pull", controller.OllamaPullModel)
			channelRoute.POST("/ollama/pull/stream", controller.OllamaPullModelStream)
			channelRoute.DELETE("/ollama/delete", controller.OllamaDeleteModel)
			channelRoute.GET("/ollama/version/:id", controller.OllamaVersion)
			channelRoute.POST("/batch/tag", controller.BatchSetChannelTag)
			channelRoute.GET("/tag/models", controller.GetTagModels)
			channelRoute.POST("/copy/:id", controller.CopyChannel)
			channelRoute.POST("/multi_key/manage", controller.ManageMultiKeys)
			channelRoute.POST("/upstream_updates/apply", controller.ApplyChannelUpstreamModelUpdates)
			channelRoute.POST("/upstream_updates/apply_all", controller.ApplyAllChannelUpstreamModelUpdates)
			channelRoute.POST("/upstream_updates/detect", controller.DetectChannelUpstreamModelUpdates)
			channelRoute.POST("/upstream_updates/detect_all", controller.DetectAllChannelUpstreamModelUpdates)
		}
		tokenRoute := apiRouter.Group("/token")
		tokenRoute.Use(middleware.UserAuth())
		{
			tokenRoute.GET("/", controller.GetAllTokens)
			tokenRoute.GET("/search", middleware.SearchRateLimit(), controller.SearchTokens)
			tokenRoute.GET("/:id", controller.GetToken)
			tokenRoute.POST("/:id/key", middleware.CriticalRateLimit(), middleware.DisableCache(), controller.GetTokenKey)
			tokenRoute.POST("/", controller.AddToken)
			tokenRoute.PUT("/", controller.UpdateToken)
			tokenRoute.DELETE("/:id", controller.DeleteToken)
			tokenRoute.POST("/batch", controller.DeleteTokenBatch)
			tokenRoute.POST("/batch/keys", middleware.CriticalRateLimit(), middleware.DisableCache(), controller.GetTokenKeysBatch)
		}

		usageRoute := apiRouter.Group("/usage")
		usageRoute.Use(middleware.CORS(), middleware.CriticalRateLimit())
		{
			tokenUsageRoute := usageRoute.Group("/token")
			tokenUsageRoute.Use(middleware.TokenAuthReadOnly())
			{
				tokenUsageRoute.GET("/", controller.GetTokenUsage)
			}
		}

		redemptionRoute := apiRouter.Group("/redemption")
		redemptionRoute.Use(middleware.AdminAuth())
		{
			redemptionRoute.GET("/", controller.GetAllRedemptions)
			redemptionRoute.GET("/search", controller.SearchRedemptions)
			redemptionRoute.GET("/:id", controller.GetRedemption)
			redemptionRoute.POST("/", controller.AddRedemption)
			redemptionRoute.PUT("/", controller.UpdateRedemption)
			redemptionRoute.DELETE("/invalid", controller.DeleteInvalidRedemption)
			redemptionRoute.DELETE("/:id", controller.DeleteRedemption)
		}

		// Prompt Library Admin Routes
		promptCategoryRoute := apiRouter.Group("/prompt-category")
		promptCategoryRoute.Use(middleware.AdminAuth())
		{
			promptCategoryRoute.GET("/", controller.GetAllPromptCategories)
			promptCategoryRoute.GET("/all", controller.GetEnabledPromptCategories)
			promptCategoryRoute.GET("/:id", controller.GetPromptCategory)
			promptCategoryRoute.POST("/", controller.AddPromptCategory)
			promptCategoryRoute.PUT("/", controller.UpdatePromptCategory)
			promptCategoryRoute.DELETE("/:id", controller.DeletePromptCategory)
		}

		promptRoute := apiRouter.Group("/prompt")
		promptRoute.Use(middleware.AdminAuth())
		{
			promptRoute.GET("/", controller.GetAllPrompts)
			promptRoute.GET("/:id", controller.GetPrompt)
			promptRoute.POST("/", controller.AddPrompt)
			promptRoute.POST("/check-exists", controller.CheckPromptsExist)
			promptRoute.PUT("/", controller.UpdatePrompt)
			promptRoute.DELETE("/:id", controller.DeletePrompt)
		}
		// Prompt Media Admin Routes
		apiRouter.POST("/prompt-media", middleware.AdminAuth(), controller.UploadPromptMedia)
		apiRouter.POST("/article-media", middleware.AdminAuth(), controller.UploadArticleMedia)
		apiRouter.DELETE("/prompt-media/:id", middleware.AdminAuth(), controller.DeletePromptMedia)
		apiRouter.DELETE("/article-media/:id", middleware.AdminAuth(), controller.DeleteArticleMedia)

		// Preset Prompt Admin Routes
		presetPromptRoute := apiRouter.Group("/preset-prompt")
		presetPromptRoute.Use(middleware.AdminAuth())
		{
			presetPromptRoute.GET("/", controller.GetAllPresetPrompts)
			presetPromptRoute.GET("/:id", controller.GetPresetPrompt)
			presetPromptRoute.POST("/", controller.AddPresetPrompt)
			presetPromptRoute.PUT("/", controller.UpdatePresetPrompt)
			presetPromptRoute.DELETE("/:id", controller.DeletePresetPrompt)
			presetPromptRoute.GET("/categories/all", controller.GetPresetPromptCategories)
		}

		// Auto Translate Toggle (Admin)
		apiRouter.GET("/admin/auto-translate-status", middleware.AdminAuth(), controller.GetAutoTranslateToggleStatus)
		apiRouter.PUT("/admin/auto-translate-status", middleware.AdminAuth(), controller.SetAutoTranslateToggleStatus)

		// SEO Management Routes
		seoRoute := apiRouter.Group("/prompt/seo")
		seoRoute.Use(middleware.AdminAuth())
		{
			seoRoute.GET("/list", controller.GetPromptSEOList)
			seoRoute.GET("/:id", controller.GetPromptSEODetail)
			seoRoute.PUT("/:id", controller.UpdatePromptSEOFields)
			seoRoute.POST("/:id/regenerate", controller.RegeneratePromptSEO)
			seoRoute.POST("/:id/audit", controller.AuditPromptSEOHandler)
			seoRoute.GET("/:id/audits", controller.GetPromptSEOAHistory)
			seoRoute.GET("/:id/report", controller.GetPromptSEOReport)
			seoRoute.GET("/stats", controller.GetPromptSEOStats)
			seoRoute.GET("/translate-stats", controller.GetPromptTranslateStats)
			seoRoute.GET("/all-translate-stats", controller.GetPromptAllTranslateStats)
			seoRoute.GET("/trends", controller.GetPromptSEOTrends)
			seoRoute.GET("/low-score", controller.GetLowScorePrompts)
			seoRoute.GET("/report-all", controller.GetAllSEOReport)
			seoRoute.POST("/audit-batch", controller.BatchAuditPromptSEO)
			seoRoute.POST("/batch-translate", controller.BatchTranslatePromptSEO)
			seoRoute.GET("/batch-translate/:task_id", controller.GetBatchTranslatePromptSEOStatus)
		}

		// Auto FAQ Generation Routes (Admin)
		apiRouter.POST("/admin/articles/:id/auto-faq", middleware.AdminAuth(), controller.AutoGenerateArticleFAQ)
		apiRouter.POST("/admin/prompts/:id/auto-faq", middleware.AdminAuth(), controller.AutoGeneratePromptFAQ)
		apiRouter.POST("/admin/articles/auto-faq/batch", middleware.AdminAuth(), controller.BatchAutoGenerateArticleFAQ)
		apiRouter.POST("/admin/prompts/auto-faq/batch", middleware.AdminAuth(), controller.BatchAutoGeneratePromptFAQ)
		apiRouter.GET("/admin/auto-faq/batch/:task_id", middleware.AdminAuth(), controller.GetAutoFAQBatchStatus)

		// Auto Translate Status (Admin)
		apiRouter.GET("/admin/auto-translate/:task_id", middleware.AdminAuth(), controller.GetAutoTranslateStatus)
		apiRouter.GET("/admin/auto-translate-queue", middleware.AdminAuth(), controller.GetAutoTranslateQueueStatus)

		// GEO Blocks Generation Routes (Admin)
		apiRouter.POST("/admin/articles/:id/geo-blocks", middleware.AdminAuth(), controller.GenerateArticleGeoBlocks)
		apiRouter.POST("/admin/prompts/:id/geo-blocks", middleware.AdminAuth(), controller.GeneratePromptGeoBlocks)
		apiRouter.POST("/admin/articles/geo-blocks/batch", middleware.AdminAuth(), controller.BatchGenerateArticleGeoBlocks)
		apiRouter.POST("/admin/prompts/geo-blocks/batch", middleware.AdminAuth(), controller.BatchGeneratePromptGeoBlocks)
		apiRouter.GET("/admin/geo-blocks/batch/:task_id", middleware.AdminAuth(), controller.GetGeoBlocksBatchStatus)

		// SEO Keyword Research Routes (Admin)
		apiRouter.POST("/admin/seo/research", middleware.AdminAuth(), controller.ResearchSEOKeywords)
		apiRouter.POST("/admin/seo/research/serp", middleware.AdminAuth(), controller.ResearchSERPKeywords)
		apiRouter.POST("/admin/seo/research/site", middleware.AdminAuth(), controller.ResearchSiteOpportunityKeywords)
		apiRouter.POST("/admin/seo/research/competitor", middleware.AdminAuth(), controller.ResearchCompetitorKeywords)
		apiRouter.POST("/admin/seo/research/community", middleware.AdminAuth(), controller.ResearchCommunityKeywords)
		apiRouter.GET("/admin/seo/research/templates", middleware.AdminAuth(), controller.GetSEOQuickTemplates)
		apiRouter.GET("/admin/seo/research/history", middleware.AdminAuth(), controller.ListSEOResearchHistories)
		apiRouter.GET("/admin/seo/research/history/:id", middleware.AdminAuth(), controller.GetSEOResearchHistory)
		apiRouter.DELETE("/admin/seo/research/history/:id", middleware.AdminAuth(), controller.DeleteSEOResearchHistory)

		// SEO Audit & Internal Links Routes (Admin)
		apiRouter.GET("/admin/seo/audit", middleware.AdminAuth(), controller.GetSEOAudit)
		apiRouter.GET("/admin/seo/internal-links", middleware.AdminAuth(), controller.GetInternalLinkSuggestions)

		// Google Indexing Routes (Admin)
		apiRouter.POST("/admin/seo/indexing", middleware.AdminAuth(), controller.SubmitToGoogleIndexing)
		apiRouter.POST("/admin/seo/indexing/batch", middleware.AdminAuth(), controller.BatchSubmitToGoogleIndexing)
		apiRouter.GET("/admin/seo/indexing/status", middleware.AdminAuth(), controller.GetGoogleIndexingStatus)

		// SEO Monitor Routes (Admin)
		apiRouter.GET("/admin/seo/monitor", middleware.AdminAuth(), controller.GetSEOMonitorData)
		apiRouter.GET("/admin/seo/monitor/history", middleware.AdminAuth(), controller.GetSEOMonitorHistory)
		apiRouter.GET("/admin/seo/monitor/summary", middleware.AdminAuth(), controller.GetSEOHealthSummary)
		apiRouter.POST("/admin/seo/monitor/simulate", middleware.AdminAuth(), controller.SimulateSEOMonitorData)
		apiRouter.POST("/admin/seo/monitor/update", middleware.AdminAuth(), controller.UpdateSEOMonitorData)
		apiRouter.POST("/admin/seo/monitor/sync-gsc", middleware.AdminAuth(), controller.SyncSEOMonitorFromGSC)

		// SEO Optimization Queue Routes (Admin)
		apiRouter.GET("/admin/seo/optimization-queue/low-ctr", middleware.AdminAuth(), controller.GetLowCTROpportunities)
		apiRouter.GET("/admin/seo/optimization-queue/ranking-drop", middleware.AdminAuth(), controller.GetRankingDropOpportunities)
		apiRouter.GET("/admin/seo/optimization-queue", middleware.AdminAuth(), controller.ListSEOOptimizationQueue)
		apiRouter.POST("/admin/seo/optimization-queue", middleware.AdminAuth(), controller.AddSEOOptimizationQueueItem)
		apiRouter.PUT("/admin/seo/optimization-queue/:id/status", middleware.AdminAuth(), controller.UpdateSEOOptimizationQueueItem)

		// Content Generator Routes (Admin)
		apiRouter.POST("/admin/content/generate", middleware.AdminAuth(), controller.GenerateContent)
		apiRouter.POST("/admin/content/optimize", middleware.AdminAuth(), controller.OptimizeContent)

		// CRO Analysis Routes (Admin)
		apiRouter.POST("/admin/cro/analyze", middleware.AdminAuth(), controller.AnalyzeCRO)

		// Article Admin Routes
		apiRouter.GET("/admin/articles", middleware.AdminAuth(), controller.GetArticles)
		apiRouter.POST("/admin/articles", middleware.AdminAuth(), controller.CreateArticle)
		apiRouter.POST("/admin/articles/generate", middleware.AdminAuth(), controller.GenerateArticle)
		apiRouter.POST("/admin/articles/generate-images", middleware.AdminAuth(), controller.GenerateArticleImages)
		apiRouter.GET("/admin/articles/:id", middleware.AdminAuth(), controller.GetArticle)
		apiRouter.PUT("/admin/articles/:id", middleware.AdminAuth(), controller.UpdateArticle)
		apiRouter.DELETE("/admin/articles/:id", middleware.AdminAuth(), controller.DeleteArticle)

		// Admin: batch regenerate prompt slugs
		apiRouter.POST("/admin/prompts/regenerate-slugs", middleware.AdminAuth(), controller.AdminRegeneratePromptSlugs)

		// Image Studio
		apiRouter.POST("/image-studio/generate", middleware.UserAuth(), controller.ImageStudioGenerate)

		apiRouter.GET("/admin/article-categories", middleware.AdminAuth(), controller.GetAllArticleCategories)
		apiRouter.POST("/admin/article-categories", middleware.AdminAuth(), controller.AddArticleCategory)
		apiRouter.GET("/admin/article-categories/:id", middleware.AdminAuth(), controller.GetArticleCategory)
		apiRouter.PUT("/admin/article-categories/:id", middleware.AdminAuth(), controller.UpdateArticleCategory)
		apiRouter.DELETE("/admin/article-categories/:id", middleware.AdminAuth(), controller.DeleteArticleCategory)

		// Article SEO Admin Routes
		articleSEORoute := apiRouter.Group("/article/seo")
		articleSEORoute.Use(middleware.AdminAuth())
		{
			articleSEORoute.GET("/list", controller.GetArticleSEOList)
			articleSEORoute.GET("/:id", controller.GetArticleSEO)
			articleSEORoute.PUT("/:id", controller.UpdateArticleSEOFields)
			articleSEORoute.POST("/:id/regenerate", controller.RegenerateArticleSEO)
			articleSEORoute.POST("/:id/audit", controller.AuditArticleSEOHandler)
			articleSEORoute.GET("/:id/audits", controller.GetArticleSEOAHistory)
			articleSEORoute.GET("/:id/report", controller.GetArticleSEOReport)
			articleSEORoute.GET("/stats", controller.GetAllArticleSEOReport)
			articleSEORoute.GET("/translate-stats", controller.GetArticleTranslateStats)
			articleSEORoute.GET("/all-translate-stats", controller.GetArticleAllTranslateStats)
			articleSEORoute.GET("/low-score", controller.GetLowScoreArticlesHandler)
		}

		// Notification Routes
		apiRouter.GET("/notifications", middleware.UserAuth(), controller.GetNotifications)
		apiRouter.GET("/notifications/unread-count", middleware.UserAuth(), controller.GetUnreadNotificationCount)
		apiRouter.POST("/notifications/:id/read", middleware.UserAuth(), controller.MarkNotificationRead)
		apiRouter.POST("/notifications/read-all", middleware.UserAuth(), controller.MarkAllNotificationsRead)

		// Admin Notification Routes
		apiRouter.GET("/admin/notifications", middleware.AdminAuth(), controller.AdminGetNotifications)
		apiRouter.POST("/admin/notifications", middleware.AdminAuth(), controller.AdminSendNotification)
		apiRouter.PUT("/admin/notifications/:id", middleware.AdminAuth(), controller.AdminUpdateNotification)

		// Translate Routes (Admin only)
		apiRouter.POST("/translate/batch", middleware.AdminAuth(), controller.BatchTranslate)
		apiRouter.POST("/translate/queue", middleware.AdminAuth(), controller.StartTranslateQueue)
		apiRouter.GET("/translate/queue/:id", middleware.AdminAuth(), controller.GetTranslateQueue)
		// Admin Tier Routes
		apiRouter.GET("/admin/tiers", middleware.AdminAuth(), controller.AdminGetTiers)
		apiRouter.POST("/admin/tiers", middleware.AdminAuth(), controller.AdminCreateTier)
		apiRouter.PUT("/admin/tiers", middleware.AdminAuth(), controller.AdminUpdateTier)
		apiRouter.DELETE("/admin/tiers/:id", middleware.AdminAuth(), controller.AdminDeleteTier)
		apiRouter.POST("/admin/users/:id/tier", middleware.AdminAuth(), controller.AdminSetUserTier)

		// Admin Tag Routes
		apiRouter.GET("/admin/tags", middleware.AdminAuth(), controller.AdminGetTags)
		apiRouter.POST("/admin/tags", middleware.AdminAuth(), controller.AdminCreateTag)
		apiRouter.DELETE("/admin/tags/:id", middleware.AdminAuth(), controller.AdminDeleteTag)
		apiRouter.GET("/admin/users/:id/tags", middleware.AdminAuth(), controller.AdminGetUserTags)
		apiRouter.POST("/admin/users/:id/tags", middleware.AdminAuth(), controller.AdminSetUserTags)
		apiRouter.POST("/admin/users/:id/tags/:tag_id", middleware.AdminAuth(), controller.AdminAddUserTag)
		apiRouter.DELETE("/admin/users/:id/tags/:tag_id", middleware.AdminAuth(), controller.AdminRemoveUserTag)
		// Sitemap Routes (no auth required, IP rate limited)
		apiRouter.GET("/articles", middleware.SitemapRateLimit(), controller.GetSitemapArticles)
		apiRouter.GET("/prompts", middleware.SitemapRateLimit(), controller.GetSitemapPrompts)

		// Prompt Library Public Routes (no auth required)
		apiRouter.GET("/public/prompts", controller.GetPublicPrompts)
		apiRouter.GET("/public/prompts/sitemap", controller.GetPublicPromptsSitemap)
		apiRouter.GET("/public/prompts/updates", controller.GetPublicPromptUpdates)
		apiRouter.GET("/public/prompts/:id", controller.GetPublicPrompt)
		apiRouter.GET("/public/prompts/slug/:slug", controller.GetPublicPromptBySlug)
		apiRouter.GET("/public/prompts/:id/geo-blocks", controller.GetPublicPromptGeoBlocks)
		apiRouter.GET("/public/prompts/slug/:slug/geo-blocks", controller.GetPublicPromptGeoBlocksBySlug)
		apiRouter.GET("/public/prompts/geo-blocks/list", controller.GetPublicPromptGeoBlocksList)
		apiRouter.GET("/public/prompt-categories", controller.GetPublicPromptCategories)
		apiRouter.GET("/public/prompt-media/:id", controller.GetPromptMedia)
		apiRouter.GET("/public/article-media/:id", controller.GetArticleMedia)

		// Article Public Routes (no auth required)
		apiRouter.GET("/public/articles", controller.GetPublicArticles)
		apiRouter.GET("/public/articles/sitemap", controller.GetPublicArticlesSitemap)
		apiRouter.GET("/public/articles/:id", controller.GetPublicArticle)
		apiRouter.GET("/public/articles/slug/:slug", controller.GetPublicArticleBySlug)
		apiRouter.GET("/public/articles/:id/geo-blocks", controller.GetPublicArticleGeoBlocks)
		apiRouter.GET("/public/articles/slug/:slug/geo-blocks", controller.GetPublicArticleGeoBlocksBySlug)
		apiRouter.GET("/public/articles/geo-blocks/list", controller.GetPublicArticleGeoBlocksList)
		apiRouter.GET("/public/article-categories", controller.GetPublicArticleCategories)

		// Preset Prompt Public Routes (optional auth for auto language detection)
		apiRouter.GET("/public/preset-prompts", middleware.TryUserAuth(), controller.GetPublicPresetPrompts)
		apiRouter.GET("/public/preset-prompts/updates", middleware.TryUserAuth(), controller.GetPublicPresetPromptUpdates)

		// App Release Public Routes (no auth required)
		apiRouter.GET("/public/releases/latest", controller.GetLatestAppRelease)
		apiRouter.GET("/public/releases/latest.json", controller.GetLatestReleaseJSON)
		apiRouter.GET("/public/releases/download/:platform/:arch", controller.DownloadAppRelease)

		// EchoTik proxy routes (token auth)
		apiRouter.GET("/public/echotik/video/ranklist", middleware.TokenAuthReadOnly(), controller.EchotikVideoRanklist)
		apiRouter.GET("/admin/echotik/status", middleware.AdminAuth(), controller.EchotikSettingStatus)

		// TikHub proxy routes (token auth)
		apiRouter.GET("/public/tikhub/tiktok/video", middleware.TokenAuthReadOnly(), controller.TikHubSingleVideo)
		apiRouter.GET("/public/tikhub/tiktok/video-by-share-url", middleware.TokenAuthReadOnly(), controller.TikHubSingleVideoByShareURL)
		apiRouter.GET("/public/tikhub/tiktok/user-country-by-username", middleware.TokenAuthReadOnly(), controller.TikHubUserCountryByUsername)
		apiRouter.GET("/public/tikhub/tiktok/general-search-result", middleware.TokenAuthReadOnly(), controller.TikHubGeneralSearchResult)
		apiRouter.GET("/public/tikhub/tiktok/video-search-result", middleware.TokenAuthReadOnly(), controller.TikHubVideoSearchResult)
		apiRouter.GET("/public/tikhub/tiktok/user-search-result", middleware.TokenAuthReadOnly(), controller.TikHubUserSearchResult)
		apiRouter.GET("/public/tikhub/tiktok/music-search-result", middleware.TokenAuthReadOnly(), controller.TikHubMusicSearchResult)
		apiRouter.GET("/public/tikhub/tiktok/hashtag-search-result", middleware.TokenAuthReadOnly(), controller.TikHubHashtagSearchResult)
		apiRouter.GET("/public/tikhub/tiktok/music-detail", middleware.TokenAuthReadOnly(), controller.TikHubMusicDetail)
		apiRouter.GET("/public/tikhub/tiktok/music-video-list", middleware.TokenAuthReadOnly(), controller.TikHubMusicVideoList)
		apiRouter.GET("/public/tikhub/tiktok/hashtag-detail", middleware.TokenAuthReadOnly(), controller.TikHubHashtagDetail)
		apiRouter.GET("/public/tikhub/tiktok/hashtag-video-list", middleware.TokenAuthReadOnly(), controller.TikHubHashtagVideoList)
		apiRouter.GET("/public/tikhub/tiktok/creator-search-insights", middleware.TokenAuthReadOnly(), controller.TikHubCreatorSearchInsights)
		apiRouter.GET("/public/tikhub/tiktok/creator-search-insights-detail", middleware.TokenAuthReadOnly(), controller.TikHubCreatorSearchInsightsDetail)
		apiRouter.GET("/public/tikhub/tiktok/creator-search-insights-videos", middleware.TokenAuthReadOnly(), controller.TikHubCreatorSearchInsightsVideos)
		apiRouter.GET("/public/tikhub/tiktok/comment-keywords", middleware.TokenAuthReadOnly(), controller.TikHubCommentKeywords)
		apiRouter.GET("/public/tikhub/tiktok/music-chart-list", middleware.TokenAuthReadOnly(), controller.TikHubMusicChartList)
		apiRouter.GET("/public/tikhub/tiktok/trending-search-words", middleware.TokenAuthReadOnly(), controller.TikHubTrendingSearchWords)
		apiRouter.GET("/public/tikhub/tiktok/product", middleware.TokenAuthReadOnly(), controller.TikHubProductDetail)
		apiRouter.GET("/public/tikhub/tiktok/shop-product-detail", middleware.TokenAuthReadOnly(), controller.TikHubProductDetailV1)
		apiRouter.GET("/public/tikhub/tiktok/product-reviews", middleware.TokenAuthReadOnly(), controller.TikHubProductReviewsV1)
		apiRouter.GET("/public/tikhub/tiktok/product-reviews-v2", middleware.TokenAuthReadOnly(), controller.TikHubProductReviewsV2)
		apiRouter.GET("/public/tikhub/tiktok/seller-products-list", middleware.TokenAuthReadOnly(), controller.TikHubSellerProductsList)
		apiRouter.GET("/public/tikhub/tiktok/search-products-list", middleware.TokenAuthReadOnly(), controller.TikHubSearchProductsList)
		apiRouter.GET("/public/tikhub/tiktok/hot-selling-products-list-v1", middleware.TokenAuthReadOnly(), controller.TikHubHotSellingProductsListV1)
		apiRouter.POST("/public/tikhub/tiktok/account-health-status", middleware.TokenAuthReadOnly(), controller.TikHubAccountHealthStatus)
		apiRouter.POST("/public/tikhub/tiktok/account-insights-overview", middleware.TokenAuthReadOnly(), controller.TikHubAccountInsightsOverview)
		apiRouter.POST("/public/tikhub/tiktok/video-analytics-summary", middleware.TokenAuthReadOnly(), controller.TikHubVideoAnalyticsSummary)
		apiRouter.POST("/public/tikhub/tiktok/video-list-analytics", middleware.TokenAuthReadOnly(), controller.TikHubVideoListAnalytics)
		apiRouter.POST("/public/tikhub/tiktok/product-analytics-list", middleware.TokenAuthReadOnly(), controller.TikHubProductAnalyticsList)
		apiRouter.POST("/public/tikhub/tiktok/product-related-videos", middleware.TokenAuthReadOnly(), controller.TikHubProductRelatedVideos)
		apiRouter.GET("/public/tikhub/tiktok/trends-hashtag-list", middleware.TokenAuthReadOnly(), controller.TikHubTrendsHashtagList)
		apiRouter.GET("/public/tikhub/tiktok/hot-selling-products-list", middleware.TokenAuthReadOnly(), controller.TikHubHotSellingProductsList)
		apiRouter.GET("/public/tikhub/tiktok/video-comments", middleware.TokenAuthReadOnly(), controller.TikHubVideoComments)
		apiRouter.GET("/public/tikhub/tiktok/video-metrics", middleware.TokenAuthReadOnly(), controller.TikHubVideoMetrics)
		apiRouter.GET("/public/tikhub/tiktok/detect-fake-views", middleware.TokenAuthReadOnly(), controller.TikHubDetectFakeViews)
		apiRouter.GET("/public/tikhub/tiktok/creator-info-milestones", middleware.TokenAuthReadOnly(), controller.TikHubCreatorInfoAndMilestones)
		apiRouter.POST("/public/tikhub/tiktok/account-violation-list", middleware.TokenAuthReadOnly(), controller.TikHubAccountViolationList)
		apiRouter.POST("/public/tikhub/tiktok/video-audience-stats", middleware.TokenAuthReadOnly(), controller.TikHubVideoAudienceStats)
		apiRouter.GET("/public/tikhub/tiktok/post-comment", middleware.TokenAuthReadOnly(), controller.TikHubPostComment)

		// TikTok Ads API
		apiRouter.GET("/public/tikhub/tiktok/ads/search-ads", middleware.TokenAuthReadOnly(), controller.TikHubSearchAds)
		apiRouter.GET("/public/tikhub/tiktok/ads/top-ads-spotlight", middleware.TokenAuthReadOnly(), controller.TikHubTopAdsSpotlight)
		apiRouter.GET("/public/tikhub/tiktok/ads/ad-keyframe-analysis", middleware.TokenAuthReadOnly(), controller.TikHubAdKeyframeAnalysis)
		apiRouter.GET("/public/tikhub/tiktok/ads/ad-percentile", middleware.TokenAuthReadOnly(), controller.TikHubAdPercentile)
		apiRouter.GET("/public/tikhub/tiktok/ads/ad-interactive-analysis", middleware.TokenAuthReadOnly(), controller.TikHubAdInteractiveAnalysis)
		apiRouter.GET("/public/tikhub/tiktok/ads/trends-hashtag-detail", middleware.TokenAuthReadOnly(), controller.TikHubTrendsHashtagDetail)

		// TikHub 整合报告 API
		apiRouter.GET("/public/tikhub/report/product-analysis", middleware.TokenAuthReadOnly(), controller.TikHubProductAnalysisReport)
		apiRouter.POST("/public/tikhub/report/creator-diagnosis", middleware.TokenAuthReadOnly(), controller.TikHubCreatorDiagnosisReport)
		apiRouter.GET("/public/tikhub/report/ad-creative-analysis", middleware.TokenAuthReadOnly(), controller.TikHubAdCreativeAnalysisReport)
		apiRouter.GET("/public/tikhub/report/content-trends", middleware.TokenAuthReadOnly(), controller.TikHubContentTrendsReport)
		apiRouter.GET("/public/tikhub/report/video-analysis", middleware.TokenAuthReadOnly(), controller.TikHubVideoAnalysisReport)
		apiRouter.GET("/public/tikhub/report/competitor-monitor", middleware.TokenAuthReadOnly(), controller.TikHubCompetitorMonitorReport)

		// TikHub 价格公开接口
		apiRouter.GET("/public/tikhub/prices", controller.GetTikHubPrices)

		apiRouter.GET("/admin/tikhub/status", middleware.AdminAuth(), controller.TikHubSettingStatus)
		apiRouter.GET("/admin/tikhub/prices", middleware.AdminAuth(), controller.GetTikHubPriceConfigs)
		apiRouter.POST("/admin/tikhub/prices", middleware.AdminAuth(), controller.CreateTikHubPriceConfig)
		apiRouter.POST("/admin/tikhub/prices/init", middleware.AdminAuth(), controller.InitTikHubPriceConfigs)
		apiRouter.POST("/admin/tikhub/prices/test", middleware.AdminAuth(), controller.TestTikHubEndpoint)
		apiRouter.PUT("/admin/tikhub/prices/:id", middleware.AdminAuth(), controller.UpdateTikHubPriceConfig)
		apiRouter.DELETE("/admin/tikhub/prices/:id", middleware.AdminAuth(), controller.DeleteTikHubPriceConfig)

		// Prompt Video URL Repair (admin only)
		apiRouter.GET("/admin/prompt/broken-video-urls", middleware.AdminAuth(), controller.GetBrokenPromptVideoUrls)
		apiRouter.POST("/admin/prompt/repair-video-urls", middleware.AdminAuth(), controller.RepairPromptVideoUrls)

		// TK Material Library Routes
		tkMaterialAdminRoute := apiRouter.Group("/admin/tk/materials")
		tkMaterialAdminRoute.Use(middleware.AdminAuth())
		{
			tkMaterialAdminRoute.GET("/", controller.AdminListTKMaterials)
			tkMaterialAdminRoute.POST("/", controller.AdminUploadTKMaterial)
			tkMaterialAdminRoute.GET("/categories", controller.AdminListTKMaterialCategories)
			tkMaterialAdminRoute.GET("/stats", controller.AdminTKMaterialCategoryStats)
			tkMaterialAdminRoute.DELETE("/:id", controller.AdminDeleteTKMaterial)
			tkMaterialAdminRoute.POST("/import/notion", controller.AdminImportTKMaterialsFromNotion)
		}
		tkMaterialPublicRoute := apiRouter.Group("/public/tk/materials")
		{
			tkMaterialPublicRoute.GET("/", controller.PublicListTKMaterials)
			tkMaterialPublicRoute.GET("/random", controller.PublicGetRandomTKMaterials)
			tkMaterialPublicRoute.GET("/:id", controller.PublicGetTKMaterial)
			tkMaterialPublicRoute.POST("/", controller.PublicUploadTKMaterial)
		}

		// App Release Admin Routes
		apiRouter.GET("/admin/releases", middleware.AdminAuth(), controller.GetAllAppReleases)
		apiRouter.POST("/admin/releases", middleware.AdminAuth(), controller.UploadAppRelease)
		apiRouter.DELETE("/admin/releases/:id", middleware.AdminAuth(), controller.DeleteAppRelease)
		apiRouter.PUT("/admin/releases/:id", middleware.AdminAuth(), controller.UpdateAppRelease)
		apiRouter.PUT("/admin/releases/:id/latest", middleware.AdminAuth(), controller.MarkAppReleaseAsLatest)

		logRoute := apiRouter.Group("/log")
		logRoute.GET("/", middleware.AdminAuth(), controller.GetAllLogs)
		logRoute.DELETE("/", middleware.AdminAuth(), controller.DeleteHistoryLogs)
		logRoute.GET("/stat", middleware.AdminAuth(), controller.GetLogsStat)
		logRoute.GET("/self/stat", middleware.UserAuth(), controller.GetLogsSelfStat)
		logRoute.GET("/channel_affinity_usage_cache", middleware.AdminAuth(), controller.GetChannelAffinityUsageCacheStats)
		logRoute.GET("/search", middleware.AdminAuth(), controller.SearchAllLogs)
		logRoute.GET("/self", middleware.UserAuth(), controller.GetUserLogs)
		logRoute.GET("/self/search", middleware.UserAuth(), middleware.SearchRateLimit(), controller.SearchUserLogs)

		dataRoute := apiRouter.Group("/data")
		dataRoute.GET("/", middleware.AdminAuth(), controller.GetAllQuotaDates)
		dataRoute.GET("/users", middleware.AdminAuth(), controller.GetQuotaDatesByUser)
		dataRoute.GET("/self", middleware.UserAuth(), controller.GetUserQuotaDates)

		logRoute.Use(middleware.CORS(), middleware.CriticalRateLimit())
		{
			logRoute.GET("/token", middleware.TokenAuthReadOnly(), controller.GetLogByKey)
		}
		groupRoute := apiRouter.Group("/group")
		groupRoute.Use(middleware.AdminAuth())
		{
			groupRoute.GET("/", controller.GetGroups)
		}

		prefillGroupRoute := apiRouter.Group("/prefill_group")
		prefillGroupRoute.Use(middleware.AdminAuth())
		{
			prefillGroupRoute.GET("/", controller.GetPrefillGroups)
			prefillGroupRoute.POST("/", controller.CreatePrefillGroup)
			prefillGroupRoute.PUT("/", controller.UpdatePrefillGroup)
			prefillGroupRoute.DELETE("/:id", controller.DeletePrefillGroup)
		}

		mjRoute := apiRouter.Group("/mj")
		mjRoute.GET("/self", middleware.UserAuth(), controller.GetUserMidjourney)
		mjRoute.GET("/", middleware.AdminAuth(), controller.GetAllMidjourney)

		taskRoute := apiRouter.Group("/task")
		{
			taskRoute.GET("/self", middleware.UserAuth(), controller.GetUserTask)
			taskRoute.GET("/", middleware.AdminAuth(), controller.GetAllTask)
		}

		vendorRoute := apiRouter.Group("/vendors")
		vendorRoute.Use(middleware.AdminAuth())
		{
			vendorRoute.GET("/", controller.GetAllVendors)
			vendorRoute.GET("/search", controller.SearchVendors)
			vendorRoute.GET("/:id", controller.GetVendorMeta)
			vendorRoute.POST("/", controller.CreateVendorMeta)
			vendorRoute.PUT("/", controller.UpdateVendorMeta)
			vendorRoute.DELETE("/:id", controller.DeleteVendorMeta)
		}

		modelsRoute := apiRouter.Group("/models")
		modelsRoute.Use(middleware.AdminAuth())
		{
			modelsRoute.GET("/sync_upstream/preview", controller.SyncUpstreamPreview)
			modelsRoute.POST("/sync_upstream", controller.SyncUpstreamModels)
			modelsRoute.GET("/missing", controller.GetMissingModels)
			modelsRoute.GET("/", controller.GetAllModelsMeta)
			modelsRoute.GET("/search", controller.SearchModelsMeta)
			modelsRoute.GET("/:id", controller.GetModelMeta)
			modelsRoute.POST("/", controller.CreateModelMeta)
			modelsRoute.PUT("/", controller.UpdateModelMeta)
			modelsRoute.DELETE("/:id", controller.DeleteModelMeta)
		}

		// Deployments (model deployment management)
		deploymentsRoute := apiRouter.Group("/deployments")
		deploymentsRoute.Use(middleware.AdminAuth())
		{
			deploymentsRoute.GET("/settings", controller.GetModelDeploymentSettings)
			deploymentsRoute.POST("/settings/test-connection", controller.TestIoNetConnection)
			deploymentsRoute.GET("/", controller.GetAllDeployments)
			deploymentsRoute.GET("/search", controller.SearchDeployments)
			deploymentsRoute.POST("/test-connection", controller.TestIoNetConnection)
			deploymentsRoute.GET("/hardware-types", controller.GetHardwareTypes)
			deploymentsRoute.GET("/locations", controller.GetLocations)
			deploymentsRoute.GET("/available-replicas", controller.GetAvailableReplicas)
			deploymentsRoute.POST("/price-estimation", controller.GetPriceEstimation)
			deploymentsRoute.GET("/check-name", controller.CheckClusterNameAvailability)
			deploymentsRoute.POST("/", controller.CreateDeployment)

			deploymentsRoute.GET("/:id", controller.GetDeployment)
			deploymentsRoute.GET("/:id/logs", controller.GetDeploymentLogs)
			deploymentsRoute.GET("/:id/containers", controller.ListDeploymentContainers)
			deploymentsRoute.GET("/:id/containers/:container_id", controller.GetContainerDetails)
			deploymentsRoute.PUT("/:id", controller.UpdateDeployment)
			deploymentsRoute.PUT("/:id/name", controller.UpdateDeploymentName)
			deploymentsRoute.POST("/:id/extend", controller.ExtendDeployment)
			deploymentsRoute.DELETE("/:id", controller.DeleteDeployment)
		}

		// Ecommerce Image Wizard Routes
		// Public routes
		apiRouter.GET("/ecommerce/model-poses", middleware.TryUserAuth(), controller.GetEnabledModelPoses)
		apiRouter.GET("/ecommerce/case-categories", middleware.TryUserAuth(), controller.GetEnabledCaseCategories)
		apiRouter.GET("/ecommerce/case-detail", middleware.TryUserAuth(), controller.GetCaseDetail)

		// Admin routes
		apiRouter.GET("/admin/ecommerce/model-poses", middleware.AdminAuth(), controller.GetAllModelPoses)
		apiRouter.POST("/admin/ecommerce/model-poses", middleware.AdminAuth(), controller.CreateModelPose)
		apiRouter.PUT("/admin/ecommerce/model-poses/:id", middleware.AdminAuth(), controller.UpdateModelPose)
		apiRouter.DELETE("/admin/ecommerce/model-poses/:id", middleware.AdminAuth(), controller.DeleteModelPose)

		apiRouter.GET("/admin/ecommerce/case-categories", middleware.AdminAuth(), controller.GetAllCaseCategories)
		apiRouter.POST("/admin/ecommerce/case-categories", middleware.AdminAuth(), controller.CreateCaseCategory)
		apiRouter.PUT("/admin/ecommerce/case-categories/:id", middleware.AdminAuth(), controller.UpdateCaseCategory)
		apiRouter.DELETE("/admin/ecommerce/case-categories/:id", middleware.AdminAuth(), controller.DeleteCaseCategory)

		apiRouter.GET("/admin/ecommerce/case-details", middleware.AdminAuth(), controller.GetCaseDetails)
		apiRouter.POST("/admin/ecommerce/case-details", middleware.AdminAuth(), controller.CreateCaseDetail)
		apiRouter.PUT("/admin/ecommerce/case-details/:id", middleware.AdminAuth(), controller.UpdateCaseDetail)
		apiRouter.DELETE("/admin/ecommerce/case-details/:id", middleware.AdminAuth(), controller.DeleteCaseDetail)
	}
}
