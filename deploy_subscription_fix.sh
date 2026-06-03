#!/bin/bash
# 在服务器上执行此脚本，修改订阅接口权限并部署

cd /home/lighthouse/new-api

# 备份原文件
cp router/api-router.go router/api-router.go.bak

# 修改 /subscription/plans 为公开访问
python3 << 'EOF'
import re

with open('router/api-router.go', 'r', encoding='utf-8') as f:
    content = f.read()

# 替换订阅路由配置
old_code = '''\t\t// Subscription billing (plans, purchase, admin management)
\t\tsubscriptionRoute := apiRouter.Group("/subscription")
\t\tsubscriptionRoute.Use(middleware.UserAuth())
\t\t{
\t\t\tsubscriptionRoute.GET("/plans", controller.GetSubscriptionPlans)
\t\t\tsubscriptionRoute.GET("/self", controller.GetSubscriptionSelf)'''

new_code = '''\t\t// Subscription billing (plans, purchase, admin management)
\t\t// Plans list is publicly accessible (like pricing)
\t\tapiRouter.GET("/subscription/plans", middleware.TryUserAuth(), controller.GetSubscriptionPlans)
\t\tsubscriptionRoute := apiRouter.Group("/subscription")
\t\tsubscriptionRoute.Use(middleware.UserAuth())
\t\t{
\t\t\tsubscriptionRoute.GET("/self", controller.GetSubscriptionSelf)'''

if old_code in content:
    content = content.replace(old_code, new_code)
    with open('router/api-router.go', 'w', encoding='utf-8') as f:
        f.write(content)
    print("✅ router/api-router.go 修改成功")
else:
    print("⚠️ 未找到匹配代码，可能已修改或文件结构不同")
    print("请手动检查 router/api-router.go")
EOF

# 验证修改
if grep -q "Plans list is publicly accessible" router/api-router.go; then
    echo "✅ 修改验证通过"
else
    echo "❌ 修改失败，请检查"
    exit 1
fi

# 部署
echo "🚀 开始部署..."
docker compose down
docker compose up -d --build

echo "✅ 部署完成"
echo ""
echo "测试命令:"
echo "curl -H 'Authorization: Bearer sk-qjzPMq9UuiNmgr5xOs23anPs5CLVkPNt7lSDwGPHgpNAkuxM' https://heharse.cloud/api/subscription/plans"
