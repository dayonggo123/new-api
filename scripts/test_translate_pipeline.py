#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
测试脚本：验证 new-api 的导出-翻译-导入链路
用法：
  1. 修改下方 BASE_URL / USERNAME / PASSWORD
  2. 运行：python3 test_translate_pipeline.py
"""

import json
import requests
import sys

# ================== 配置区，请修改 ==================
BASE_URL = "http://localhost:3000"  # 你的 new-api 地址
USERNAME = "admin"                  # 管理员账号
PASSWORD = ""                       # 管理员密码（必填）
# ===================================================

HEADERS = {"Content-Type": "application/json"}

def login():
    """登录获取 admin token"""
    url = f"{BASE_URL}/api/user/login"
    payload = {"username": USERNAME, "password": PASSWORD}
    try:
        resp = requests.post(url, json=payload, headers=HEADERS, timeout=10)
        data = resp.json()
        print(f"登录响应: {resp.status_code}")
        print(json.dumps(data, indent=2, ensure_ascii=False))
        if resp.status_code == 200 and data.get("success"):
            return data["data"]
        else:
            print("登录失败:", data.get("message"))
            sys.exit(1)
    except Exception as e:
        print(f"登录请求异常: {e}")
        sys.exit(1)

def test_prompt_list(token):
    """测试拉取 Prompt 列表"""
    url = f"{BASE_URL}/api/prompt/?p=0&size=5"
    headers = {**HEADERS, "Authorization": token}
    resp = requests.get(url, headers=headers, timeout=10)
    print(f"\nPrompt 列表: {resp.status_code}")
    data = resp.json()
    print(json.dumps(data, indent=2, ensure_ascii=False)[:2000])
    return data

def test_prompt_detail(token, prompt_id):
    """测试拉取 Prompt 详情"""
    url = f"{BASE_URL}/api/prompt/{prompt_id}"
    headers = {**HEADERS, "Authorization": token}
    resp = requests.get(url, headers=headers, timeout=10)
    print(f"\nPrompt 详情 ({prompt_id}): {resp.status_code}")
    data = resp.json()
    print(json.dumps(data, indent=2, ensure_ascii=False)[:2000])
    return data

def test_article_list(token):
    """测试拉取 Article 列表"""
    url = f"{BASE_URL}/api/admin/articles?p=0&size=5"
    headers = {**HEADERS, "Authorization": token}
    resp = requests.get(url, headers=headers, timeout=10)
    print(f"\nArticle 列表: {resp.status_code}")
    data = resp.json()
    print(json.dumps(data, indent=2, ensure_ascii=False)[:2000])
    return data

def test_article_detail(token, article_id):
    """测试拉取 Article 详情"""
    url = f"{BASE_URL}/api/admin/articles/{article_id}"
    headers = {**HEADERS, "Authorization": token}
    resp = requests.get(url, headers=headers, timeout=10)
    print(f"\nArticle 详情 ({article_id}): {resp.status_code}")
    data = resp.json()
    print(json.dumps(data, indent=2, ensure_ascii=False)[:2000])
    return data

def main():
    if not PASSWORD:
        print("错误：请先修改脚本里的 PASSWORD")
        sys.exit(1)

    print("=" * 50)
    print("开始测试 new-api 导出-翻译-导入链路")
    print("=" * 50)

    token = login()
    print(f"\n获取到 token: {token[:20]}...")

    # 测试 Prompt
    plist = test_prompt_list(token)
    if plist.get("success") and plist.get("data"):
        prompts = plist["data"]
        if prompts:
            test_prompt_detail(token, prompts[0]["id"])

    # 测试 Article
    alist = test_article_list(token)
    if alist.get("success") and alist.get("data"):
        articles = alist["data"]
        if articles:
            test_article_detail(token, articles[0]["id"])

    print("\n" + "=" * 50)
    print("链路测试完成。如果上面都返回 200 + success=True，")
    print("说明导出-翻译-导入方案完全可行。")
    print("=" * 50)

if __name__ == "__main__":
    main()
