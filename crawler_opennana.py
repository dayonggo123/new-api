#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
OpenNana 提示词图库自动化采集脚本
使用方法:
    1. 安装依赖: pip install playwright
    2. 安装浏览器: playwright install chromium
    3. 运行: python crawler_opennana.py

功能:
    - 自动滚动加载全部提示词列表（支持无限滚动）
    - 逐个访问详情页采集完整提示词内容
    - 断点续传: 中途停止后可从中断处继续
    - 增量更新: 跳过已采集的条目
    - 输出: JSON + CSV 双格式
"""

import json
import csv
import time
import os
import re
from datetime import datetime
from pathlib import Path
from urllib.parse import urljoin, urlparse
from typing import List, Dict, Optional, Any

try:
    from playwright.sync_api import sync_playwright, Page, Locator
except ImportError:
    print("[错误] 请先安装 Playwright: pip install playwright")
    print("[错误] 然后安装浏览器: playwright install chromium")
    exit(1)


# ==================== 配置区 ====================
BASE_URL = "https://opennana.com"
GALLERY_URL = "https://opennana.com/awesome-prompt-gallery"
# 是否只采集图片类型 (image/video/all)
MEDIA_TYPE = "all"
# 每采集多少条保存一次进度
SAVE_INTERVAL = 50
# 访问详情页间隔（秒），防止请求过快被封
DETAIL_DELAY = 0.5
# 滚动加载间隔（秒）
SCROLL_DELAY = 1.5
# 每个详情页最大重试次数
MAX_RETRIES = 3
# 是否下载缩略图到本地
DOWNLOAD_THUMBNAILS = False
# 缩略图保存目录
THUMBNAIL_DIR = "./opennana_thumbnails"
# 数据输出目录
OUTPUT_DIR = "./opennana_data"
# ==================== 配置区结束 ====================


class OpenNanaCrawler:
    """OpenNana 提示词采集器"""

    def __init__(self):
        self.output_dir = Path(OUTPUT_DIR)
        self.output_dir.mkdir(exist_ok=True)
        self.thumbnail_dir = Path(THUMBNAIL_DIR)
        if DOWNLOAD_THUMBNAILS:
            self.thumbnail_dir.mkdir(exist_ok=True)

        # 文件路径
        self.list_file = self.output_dir / "prompts_list.json"
        self.detail_file = self.output_dir / "prompts_detail.json"
        self.csv_file = self.output_dir / "prompts.csv"
        self.progress_file = self.output_dir / "progress.json"
        self.log_file = self.output_dir / "crawler.log"

        # 内存数据
        self.prompts_list: List[Dict] = []
        self.prompts_detail: List[Dict] = []
        self.completed_ids: set = set()
        self.failed_ids: set = set()

    def log(self, message: str):
        """记录日志"""
        timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
        line = f"[{timestamp}] {message}"
        print(line)
        with open(self.log_file, "a", encoding="utf-8") as f:
            f.write(line + "\n")

    def load_progress(self):
        """加载之前的进度"""
        if self.progress_file.exists():
            with open(self.progress_file, "r", encoding="utf-8") as f:
                data = json.load(f)
                self.completed_ids = set(data.get("completed_ids", []))
                self.failed_ids = set(data.get("failed_ids", []))
            self.log(f"[进度] 已加载进度: 完成 {len(self.completed_ids)} 条, 失败 {len(self.failed_ids)} 条")

        if self.detail_file.exists():
            with open(self.detail_file, "r", encoding="utf-8") as f:
                self.prompts_detail = json.load(f)
            self.log(f"[进度] 已加载详情数据: {len(self.prompts_detail)} 条")

    def save_progress(self):
        """保存进度"""
        progress = {
            "completed_ids": list(self.completed_ids),
            "failed_ids": list(self.failed_ids),
            "updated_at": datetime.now().isoformat(),
        }
        with open(self.progress_file, "w", encoding="utf-8") as f:
            json.dump(progress, f, ensure_ascii=False, indent=2)

        with open(self.detail_file, "w", encoding="utf-8") as f:
            json.dump(self.prompts_detail, f, ensure_ascii=False, indent=2)

        self.export_to_csv()
        self.log(f"[保存] 进度已保存: 完成 {len(self.completed_ids)} 条")

    def export_to_csv(self):
        """导出为 CSV 格式"""
        if not self.prompts_detail:
            return

        # 收集所有字段
        all_keys = set()
        for item in self.prompts_detail:
            all_keys.update(item.keys())

        # 固定顺序的字段
        priority_fields = [
            "id", "title", "prompt_text", "prompt_text_en",
            "model", "media_type", "tags",
            "thumbnail_url", "image_urls",
            "source_url", "created_at", "crawled_at"
        ]
        fieldnames = [f for f in priority_fields if f in all_keys]
        fieldnames += sorted([f for f in all_keys if f not in priority_fields])

        with open(self.csv_file, "w", encoding="utf-8-sig", newline="") as f:
            writer = csv.DictWriter(f, fieldnames=fieldnames)
            writer.writeheader()
            for item in self.prompts_detail:
                row = {}
                for key in fieldnames:
                    val = item.get(key, "")
                    if isinstance(val, (list, dict)):
                        val = json.dumps(val, ensure_ascii=False)
                    row[key] = val
                writer.writerow(row)

    def scroll_to_load_all(self, page: Page) -> int:
        """
        自动滚动页面加载所有内容
        返回加载到的卡片数量
        """
        self.log("[滚动] 开始自动滚动加载所有提示词...")
        last_count = 0
        same_count_times = 0
        max_same_times = 5  # 连续5次数量不变认为加载完成

        while same_count_times < max_same_times:
            # 滚动到底部
            page.evaluate("window.scrollTo(0, document.body.scrollHeight)")
            time.sleep(SCROLL_DELAY)

            # 统计当前卡片数量
            cards = page.locator("a[href^='/awesome-prompt/']").all()
            current_count = len(cards)

            if current_count == last_count:
                same_count_times += 1
            else:
                same_count_times = 0
                last_count = current_count
                self.log(f"[滚动] 当前已加载 {current_count} 条提示词...")

            # 检查是否显示"没有更多了"
            no_more = page.locator("text=没有更多了").count() > 0
            if no_more:
                self.log("[滚动] 检测到 '没有更多了'，加载完成")
                break

        self.log(f"[滚动] 加载完成，共 {last_count} 条提示词")
        return last_count

    def extract_list_data(self, page: Page) -> List[Dict]:
        """
        从当前页面提取列表数据
        """
        self.log("[提取] 开始提取列表页数据...")
        prompts = []

        # 使用多种选择器尝试定位卡片
        selectors = [
            "a[href^='/awesome-prompt/']",
            ".prompt-card",
            "[class*='card']",
            ".gallery-item",
        ]

        cards = []
        for selector in selectors:
            cards = page.locator(selector).all()
            if cards:
                self.log(f"[提取] 使用选择器 '{selector}' 找到 {len(cards)} 个卡片")
                break

        for idx, card in enumerate(cards):
            try:
                data = self._parse_card(card, idx)
                if data and data.get("id"):
                    prompts.append(data)
            except Exception as e:
                self.log(f"[提取] 解析第 {idx} 个卡片失败: {e}")

        self.log(f"[提取] 成功提取 {len(prompts)} 条列表数据")
        return prompts

    def _parse_card(self, card: Locator, idx: int) -> Optional[Dict]:
        """解析单个卡片元素"""
        # 获取链接
        href = ""
        try:
            href = card.get_attribute("href") or ""
        except:
            # 尝试从内部 a 标签获取
            links = card.locator("a").all()
            for link in links:
                try:
                    h = link.get_attribute("href") or ""
                    if h.startswith("/awesome-prompt/"):
                        href = h
                        break
                except:
                    continue

        if not href or not href.startswith("/awesome-prompt/"):
            return None

        prompt_id = href.split("/awesome-prompt/")[-1].split("?")[0].split("#")[0]
        source_url = urljoin(BASE_URL, href)

        # 获取标题
        title = ""
        try:
            # 尝试多种标题选择器
            for sel in [".title", "h3", "h2", "[class*='title']", "[class*='name']", ".text-base"]:
                el = card.locator(sel).first
                if el.count() > 0:
                    title = el.inner_text(timeout=1000).strip()
                    if title:
                        break
        except:
            pass

        # 如果上面没找到，尝试图片的 alt 属性
        if not title:
            try:
                img = card.locator("img").first
                if img.count() > 0:
                    title = img.get_attribute("alt") or ""
            except:
                pass

        # 获取缩略图
        thumbnail_url = ""
        try:
            img = card.locator("img").first
            if img.count() > 0:
                src = img.get_attribute("src") or ""
                if src:
                    thumbnail_url = urljoin(BASE_URL, src)
        except:
            pass

        # 获取模型类型（从标签或class推断）
        model = ""
        try:
            # 尝试找模型标签
            badges = card.locator("[class*='badge'], [class*='tag'], [class*='model']").all()
            for badge in badges:
                text = badge.inner_text(timeout=500).strip()
                if text in ["Nano Banana Pro", "Nano Banana 2", "ChatGPT", "Grok", "Seedance 2.0"]:
                    model = text
                    break
        except:
            pass

        return {
            "id": prompt_id,
            "title": title,
            "thumbnail_url": thumbnail_url,
            "source_url": source_url,
            "model": model,
            "media_type": "",
            "crawled_at": datetime.now().isoformat(),
        }

    def fetch_detail(self, page: Page, prompt_id: str, source_url: str) -> Optional[Dict]:
        """
        访问详情页获取完整提示词内容
        """
        for attempt in range(MAX_RETRIES):
            try:
                page.goto(source_url, wait_until="domcontentloaded", timeout=30000)
                time.sleep(1.0)  # 等待内容渲染

                detail = {
                    "id": prompt_id,
                    "source_url": source_url,
                    "crawled_at": datetime.now().isoformat(),
                }

                # 提取标题
                try:
                    title = page.locator("h1, [class*='title']").first.inner_text(timeout=3000).strip()
                    detail["title"] = title
                except:
                    pass

                # 提取提示词原文（多种策略）
                prompt_text = self._extract_prompt_text(page)
                if prompt_text:
                    detail["prompt_text"] = prompt_text
                    # 同时保存英文原文（如果提示词是英文的）
                    if self._is_english_prompt(prompt_text):
                        detail["prompt_text_en"] = prompt_text

                # 提取标签
                tags = self._extract_tags(page)
                if tags:
                    detail["tags"] = tags

                # 提取模型信息
                model = self._extract_model(page)
                if model:
                    detail["model"] = model

                # 提取媒体类型
                media_type = self._extract_media_type(page)
                if media_type:
                    detail["media_type"] = media_type

                # 提取图片/视频URL列表
                media_urls = self._extract_media_urls(page)
                if media_urls:
                    detail["image_urls"] = media_urls

                # 提取生成参数（如果有）
                params = self._extract_generation_params(page)
                if params:
                    detail["generation_params"] = params

                # 提取作者信息
                author = self._extract_author(page)
                if author:
                    detail["author"] = author

                # 提取创建时间
                created_at = self._extract_created_time(page)
                if created_at:
                    detail["created_at"] = created_at

                return detail

            except Exception as e:
                self.log(f"[详情] {prompt_id} 第 {attempt + 1} 次尝试失败: {e}")
                time.sleep(2)

        return None

    def _extract_prompt_text(self, page: Page) -> str:
        """提取提示词文本内容"""
        # 策略1: 找包含 "Prompt" 标签的 textarea 或代码块
        try:
            # 常见的提示词展示元素
            selectors = [
                "textarea[readonly]",
                "pre",
                "code",
                "[class*='prompt']",
                "[class*='content']",
                "[class*='description']",
            ]
            for sel in selectors:
                els = page.locator(sel).all()
                for el in els:
                    text = el.inner_text(timeout=1000).strip()
                    # 过滤掉太短的和非提示词内容
                    if len(text) > 20 and not text.startswith("http"):
                        # 检查是否像提示词（包含英文描述性内容）
                        return text
        except:
            pass

        # 策略2: 查找页面中较长的文本段落
        try:
            paragraphs = page.locator("p").all()
            for p in paragraphs:
                text = p.inner_text(timeout=500).strip()
                if len(text) > 50 and self._is_likely_prompt(text):
                    return text
        except:
            pass

        return ""

    def _is_likely_prompt(self, text: str) -> bool:
        """判断文本是否像 AI 提示词"""
        # 提示词通常包含这些特征
        indicators = [
            "portrait", "photorealistic", "detailed", "lighting", "style",
            "background", "camera", "angle", "color", "texture",
            "人物", "写真", "摄影", "风格", "光线", "背景",
        ]
        text_lower = text.lower()
        return any(ind in text_lower for ind in indicators)

    def _is_english_prompt(self, text: str) -> bool:
        """判断是否为英文提示词"""
        ascii_chars = sum(1 for c in text if ord(c) < 128)
        return ascii_chars / max(len(text), 1) > 0.7

    def _extract_tags(self, page: Page) -> List[str]:
        """提取标签列表"""
        tags = []
        try:
            tag_elements = page.locator("[class*='tag'], a[href*='tag=']").all()
            for tag in tag_elements:
                text = tag.inner_text(timeout=500).strip()
                if text and text not in tags:
                    tags.append(text)
        except:
            pass
        return tags

    def _extract_model(self, page: Page) -> str:
        """提取使用的 AI 模型"""
        try:
            # 常见的模型标识
            model_keywords = ["Nano Banana Pro", "Nano Banana 2", "ChatGPT", "Grok", "Seedance 2.0", "GPT Image 2"]
            page_text = page.locator("body").inner_text(timeout=3000)
            for kw in model_keywords:
                if kw in page_text:
                    return kw
        except:
            pass
        return ""

    def _extract_media_type(self, page: Page) -> str:
        """提取媒体类型 (image/video)"""
        try:
            page_text = page.locator("body").inner_text(timeout=2000).lower()
            if "video" in page_text or "视频" in page_text:
                return "video"
            return "image"
        except:
            return "image"

    def _extract_media_urls(self, page: Page) -> List[str]:
        """提取页面中的图片/视频 URL 列表"""
        urls = []
        try:
            images = page.locator("img").all()
            for img in images:
                src = img.get_attribute("src") or ""
                if src and src not in urls:
                    full_url = urljoin(BASE_URL, src)
                    urls.append(full_url)
        except:
            pass
        return urls

    def _extract_generation_params(self, page: Page) -> Dict[str, Any]:
        """提取生成参数（如分辨率、比例等）"""
        params = {}
        try:
            page_text = page.locator("body").inner_text(timeout=2000)
            # 尝试提取比例
            ratio_match = re.search(r'(\d+:\d+)', page_text)
            if ratio_match:
                params["aspect_ratio"] = ratio_match.group(1)
            # 尝试提取分辨率
            res_match = re.search(r'(\d+)[xX](\d+)', page_text)
            if res_match:
                params["resolution"] = f"{res_match.group(1)}x{res_match.group(2)}"
        except:
            pass
        return params

    def _extract_author(self, page: Page) -> str:
        """提取作者信息"""
        try:
            author_el = page.locator("[class*='author'], [class*='user'], [class*='creator']").first
            if author_el.count() > 0:
                return author_el.inner_text(timeout=1000).strip()
        except:
            pass
        return ""

    def _extract_created_time(self, page: Page) -> str:
        """提取创建时间"""
        try:
            time_el = page.locator("time, [class*='date'], [class*='time']").first
            if time_el.count() > 0:
                return time_el.get_attribute("datetime") or time_el.inner_text(timeout=500).strip()
        except:
            pass
        return ""

    def run(self, headless: bool = True, max_items: Optional[int] = None):
        """
        运行采集任务

        Args:
            headless: 是否无头模式运行（False 可看到浏览器窗口，方便调试）
            max_items: 最大采集数量，None 表示全部
        """
        self.log("=" * 60)
        self.log("OpenNana 提示词采集器启动")
        self.log(f"目标: {GALLERY_URL}")
        self.log(f"媒体类型: {MEDIA_TYPE}")
        self.log(f"无头模式: {headless}")
        self.log("=" * 60)

        self.load_progress()

        with sync_playwright() as p:
            browser = p.chromium.launch(headless=headless)
            context = browser.new_context(
                viewport={"width": 1920, "height": 1080},
                user_agent=(
                    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) "
                    "AppleWebKit/537.36 (KHTML, like Gecko) "
                    "Chrome/125.0.0.0 Safari/537.36"
                ),
            )
            page = context.new_page()

            try:
                # ========== 第一步：加载列表页 ==========
                self.log(f"[列表] 正在打开 {GALLERY_URL}")
                page.goto(GALLERY_URL, wait_until="networkidle", timeout=60000)
                time.sleep(2)

                # 如果有筛选参数，处理 URL
                if MEDIA_TYPE != "all":
                    current_url = page.url
                    separator = "&" if "?" in current_url else "?"
                    page.goto(f"{current_url}{separator}media_type={MEDIA_TYPE}", wait_until="networkidle")
                    time.sleep(2)

                # 滚动加载全部
                total_loaded = self.scroll_to_load_all(page)

                # 提取列表数据
                self.prompts_list = self.extract_list_data(page)

                if not self.prompts_list:
                    self.log("[错误] 未提取到任何列表数据，请检查页面结构是否有变化")
                    browser.close()
                    return

                # 保存列表数据
                with open(self.list_file, "w", encoding="utf-8") as f:
                    json.dump(self.prompts_list, f, ensure_ascii=False, indent=2)
                self.log(f"[列表] 已保存到 {self.list_file}")

                # ========== 第二步：采集详情页 ==========
                total = len(self.prompts_list)
                if max_items:
                    total = min(total, max_items)

                self.log(f"[详情] 开始采集详情页，共 {total} 条...")

                for idx, item in enumerate(self.prompts_list[:total]):
                    prompt_id = item.get("id", "")
                    if not prompt_id:
                        continue

                    # 跳过已完成的
                    if prompt_id in self.completed_ids:
                        continue

                    self.log(f"[详情] [{idx + 1}/{total}] 正在采集: {item.get('title', prompt_id)}")

                    detail = self.fetch_detail(page, prompt_id, item.get("source_url", ""))

                    if detail:
                        # 合并列表数据和详情数据
                        merged = {**item, **detail}
                        # 避免重复字段被覆盖为空
                        for key in item:
                            if not merged.get(key) and item.get(key):
                                merged[key] = item[key]

                        self.prompts_detail.append(merged)
                        self.completed_ids.add(prompt_id)

                        # 如果之前标记为失败，现在成功了就移除
                        if prompt_id in self.failed_ids:
                            self.failed_ids.discard(prompt_id)
                    else:
                        self.failed_ids.add(prompt_id)
                        self.log(f"[详情] {prompt_id} 采集失败，已记录")

                    # 定期保存进度
                    if (idx + 1) % SAVE_INTERVAL == 0:
                        self.save_progress()

                    # 延迟，防止请求过快
                    time.sleep(DETAIL_DELAY)

                # 最终保存
                self.save_progress()

                self.log("=" * 60)
                self.log("采集完成!")
                self.log(f"总计列表: {len(self.prompts_list)} 条")
                self.log(f"成功详情: {len(self.completed_ids)} 条")
                self.log(f"失败: {len(self.failed_ids)} 条")
                self.log(f"数据保存在: {self.output_dir.absolute()}")
                self.log("=" * 60)

            except KeyboardInterrupt:
                self.log("[中断] 用户手动停止，正在保存进度...")
                self.save_progress()
            except Exception as e:
                self.log(f"[错误] 发生异常: {e}")
                self.save_progress()
            finally:
                browser.close()


def main():
    """主入口"""
    import argparse

    parser = argparse.ArgumentParser(description="OpenNana 提示词采集器")
    parser.add_argument(
        "--visible", "-v",
        action="store_true",
        help="显示浏览器窗口（用于调试，默认无头模式）"
    )
    parser.add_argument(
        "--max", "-m",
        type=int,
        default=None,
        help="最多采集多少条（默认全部）"
    )
    parser.add_argument(
        "--media",
        choices=["all", "image", "video"],
        default="all",
        help="只采集指定媒体类型"
    )

    args = parser.parse_args()

    # 更新全局配置
    global MEDIA_TYPE
    MEDIA_TYPE = args.media

    crawler = OpenNanaCrawler()
    crawler.run(headless=not args.visible, max_items=args.max)


if __name__ == "__main__":
    main()
