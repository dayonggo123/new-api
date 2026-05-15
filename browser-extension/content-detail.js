// content-detail.js - 详情页脚本
// 在 opennana.com 提示词详情页自动提取 prompt 内容

(function () {
  'use strict';

  const PLATFORM = 'opennana';
  const EXTRACT_DELAY = 2000; // 等待页面渲染完成

  // 等待元素出现
  function waitForElement(selectors, timeout = 5000) {
    return new Promise((resolve) => {
      const start = Date.now();
      const check = () => {
        for (const sel of selectors) {
          const el = document.querySelector(sel);
          if (el && el.textContent.trim()) {
            resolve(el);
            return;
          }
        }
        if (Date.now() - start > timeout) {
          resolve(null);
          return;
        }
        setTimeout(check, 300);
      };
      check();
    });
  }

  // 提取标题
  async function extractTitle() {
    const selectors = [
      'h1',
      '.prompt-title',
      '[class*="title"] h1',
      '[class*="title"] h2',
      'article h1',
      'article h2'
    ];
    const el = await waitForElement(selectors, 3000);
    return el ? el.textContent.trim() : document.title.replace(/\s*-\s*OpenNana.*$/i, '').trim();
  }

  // 提取 prompt 正文（多种策略）
  async function extractPromptContent() {
    // 策略1：查找包含 "Prompt" 标签的 pre/code/textarea/p/div
    const allElements = document.querySelectorAll('pre, code, textarea, p, div');
    for (const el of allElements) {
      const text = el.textContent.trim();
      const prev = el.previousElementSibling;
      const prevText = prev ? prev.textContent.trim().toLowerCase() : '';
      const parentPrev = el.parentElement?.previousElementSibling;
      const parentPrevText = parentPrev ? parentPrev.textContent.trim().toLowerCase() : '';

      if (
        (prevText.includes('prompt') || prevText.includes('提示词') ||
         parentPrevText.includes('prompt') || parentPrevText.includes('提示词')) &&
        text.length > 30
      ) {
        return text;
      }
    }

    // 策略2：查找最长的文本段落（假设 prompt 是最长的）
    let longest = '';
    for (const el of allElements) {
      const text = el.textContent.trim();
      if (text.length > longest.length && text.length > 100 && text.length < 5000) {
        // 排除导航、页脚等常见元素
        const tag = el.tagName.toLowerCase();
        const parentTag = el.parentElement?.tagName.toLowerCase() || '';
        if (!['nav', 'footer', 'header', 'script', 'style'].includes(parentTag)) {
          longest = text;
        }
      }
    }
    if (longest.length > 100) return longest;

    // 策略3：查找特定 class
    const contentSelectors = [
      '[class*="prompt-content"]',
      '[class*="prompt-text"]',
      '[class*="content"] pre',
      '[class*="detail"] p',
      '[class*="body"] p',
      'article p'
    ];
    for (const sel of contentSelectors) {
      const el = document.querySelector(sel);
      if (el && el.textContent.trim().length > 50) {
        return el.textContent.trim();
      }
    }

    // 策略4：获取页面主要文本内容
    const article = document.querySelector('article, main, [role="main"]');
    if (article) {
      const text = article.textContent.trim();
      if (text.length > 50) return text.substring(0, 3000);
    }

    return '';
  }

  // 提取封面图
  function extractCoverImage() {
    const selectors = [
      'meta[property="og:image"]',
      'meta[name="twitter:image"]',
      '[class*="cover"] img',
      '[class*="hero"] img',
      '[class*="banner"] img',
      'article img'
    ];
    for (const sel of selectors) {
      const el = document.querySelector(sel);
      if (el) {
        const src = el.getAttribute('content') || el.getAttribute('src');
        if (src) {
          if (src.startsWith('http')) return src;
          if (src.startsWith('/')) return 'https://opennana.com' + src;
        }
      }
    }
    return '';
  }

  // 提取模型类型
  function extractModel() {
    const pageText = document.body.textContent;
    const models = [
      'GPT Image 2', 'Nano Banana Pro', 'Nano Banana 2', 'Nano Banana',
      'Seedance 2.0', 'Seedance',
      'Midjourney', 'DALL-E', 'DALL-E 3',
      'Stable Diffusion', 'SDXL',
      'Flux', 'Ideogram',
      'Kling', 'Runway', 'Pika',
      'Sora', '可灵', '即梦'
    ];
    for (const m of models) {
      if (pageText.includes(m)) return m;
    }
    return '';
  }

  // 提取媒体类型
  function extractMediaType() {
    const url = window.location.href;
    if (url.includes('media_type=video')) return 'video';
    if (url.includes('media_type=image')) return 'image';
    const pageText = document.body.textContent.toLowerCase();
    if (pageText.includes('视频') || pageText.includes('video')) return 'video';
    return 'image';
  }

  // 提取标签
  function extractTags() {
    const tags = [];
    const tagSelectors = [
      '[class*="tag"]',
      'a[href*="tag"]',
      '[class*="label"]'
    ];
    for (const sel of tagSelectors) {
      document.querySelectorAll(sel).forEach(el => {
        const text = el.textContent.trim();
        if (text && text.length < 20 && !tags.includes(text)) {
          tags.push(text);
        }
      });
    }
    return tags.slice(0, 10);
  }

  // 主提取流程
  async function extract() {
    console.log('[Prompt Collector] 开始提取详情页内容...');

    const title = await extractTitle();
    const content = await extractPromptContent();
    const coverImageUrl = extractCoverImage();
    const model = extractModel();
    const mediaType = extractMediaType();
    const tags = extractTags();

    const data = {
      title,
      content,
      description: title,
      cover_image_url: coverImageUrl,
      model,
      media_type: mediaType,
      tags: JSON.stringify(tags),
      source_url: window.location.href,
      platform: PLATFORM,
      status: 1
    };

    console.log('[Prompt Collector] 提取完成:', data);

    // 发送给 background
    try {
      await chrome.runtime.sendMessage({
        action: 'saveExtractedData',
        data
      });

      // 通知 popup 有新数据
      chrome.runtime.sendMessage({ action: 'dataExtracted' });

      // 可选：在页面角落显示提取成功的提示
      showExtractToast();
    } catch (err) {
      console.error('[Prompt Collector] 发送提取数据失败:', err);
    }
  }

  // 在页面显示提取成功提示
  function showExtractToast() {
    const toast = document.createElement('div');
    toast.textContent = '✅ Prompt 已提取，点击扩展图标查看';
    Object.assign(toast.style, {
      position: 'fixed',
      bottom: '20px',
      right: '20px',
      zIndex: '99999',
      padding: '12px 20px',
      background: 'linear-gradient(135deg, #06b6d4, #3b82f6)',
      color: '#fff',
      borderRadius: '10px',
      fontSize: '14px',
      fontWeight: '600',
      boxShadow: '0 4px 16px rgba(6,182,212,0.35)',
      animation: 'fadeInUp 0.3s ease'
    });
    document.body.appendChild(toast);
    setTimeout(() => {
      toast.style.opacity = '0';
      toast.style.transition = 'opacity 0.5s';
      setTimeout(() => toast.remove(), 500);
    }, 4000);
  }

  // 注入 toast 动画样式
  const style = document.createElement('style');
  style.textContent = `
    @keyframes fadeInUp {
      from { opacity: 0; transform: translateY(10px); }
      to { opacity: 1; transform: translateY(0); }
    }
  `;
  document.head.appendChild(style);

  // 初始化
  function init() {
    setTimeout(extract, EXTRACT_DELAY);
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
