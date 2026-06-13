// content-youmind.js - YouMind.com 专用提示词采集
// 适配 https://youmind.com/* 页面，特别是 /prompts 类详情页
// 与 content-universal.js 互不冲突

(function () {
  'use strict';

  const STORAGE_KEY = 'promptCollectorExtracted';

  function cleanText(text) {
    if (!text) return '';
    return text.replace(/\s+/g, ' ').replace(/[\u200B-\u200D\uFEFF]/g, '').trim();
  }

  function detectLang(text) {
    if (!text) return 'unknown';
    if (/[\u4e00-\u9fa5]/.test(text)) return 'zh';
    if (/[\u3040-\u309F\u30A0-\u30FF]/.test(text)) return 'ja';
    if (/[\uAC00-\uD7AF]/.test(text)) return 'ko';
    return 'en';
  }

  function getDomain(url) {
    try { return new URL(url).hostname.replace(/^www\./, ''); }
    catch (e) { return ''; }
  }

  function makeAbsoluteUrl(url) {
    if (!url) return '';
    if (url.startsWith('http')) return url;
    if (url.startsWith('//')) return 'https:' + url;
    try { return new URL(url, location.href).href; } catch (e) { return url; }
  }

  // 提取 YouMind 的标题
  function extractYouMindTitle() {
    // 1) h1
    const h1 = document.querySelector('h1');
    if (h1) {
      const t = cleanText(h1.textContent);
      if (t && t.length < 200) return t;
    }
    // 2) og:title
    const og = document.querySelector('meta[property="og:title"]')?.content;
    if (og) return cleanText(og);
    return document.title.trim();
  }

  // 提取主图（优先第一张作品图）
  function extractYouMindCover() {
    const og = document.querySelector('meta[property="og:image"]')?.content;
    if (og) return makeAbsoluteUrl(og);
    const tw = document.querySelector('meta[name="twitter:image"]')?.content;
    if (tw) return makeAbsoluteUrl(tw);
    // 文章内首图
    const img = document.querySelector('article img, main img, .content img, [class*="gallery"] img');
    if (img?.src) return makeAbsoluteUrl(img.src);
    return '';
  }

  // 提取提示词文本（YouMind 通常在特定 section/div 中放置）
  function extractYouMindPrompt() {
    // 策略 A：查找带"提示词"或"prompt"标签的块
    const labelKeywords = ['提示词', 'Prompt', 'prompt', '正文', '描述', '咒语', '指令'];
    const allEls = document.querySelectorAll('h1, h2, h3, h4, h5, h6, strong, b, span, div, p');

    for (const label of allEls) {
      const labelText = cleanText(label.textContent).toLowerCase();
      if (labelKeywords.some(kw => labelText === kw.toLowerCase() || labelText.startsWith(kw.toLowerCase()))) {
        // 找下一个兄弟/邻接元素
        const next = label.nextElementSibling;
        if (next) {
          const t = cleanText(next.textContent);
          if (t.length > 30) return { content: t, strategy: 'labeled-sibling' };
        }
        // 找父元素下的所有段落
        const parent = label.parentElement;
        if (parent) {
          const ps = parent.querySelectorAll('p, pre, code, div');
          for (const p of ps) {
            if (p === label) continue;
            const t = cleanText(p.textContent);
            if (t.length > 30) return { content: t, strategy: 'labeled-parent-p' };
          }
        }
      }
    }

    // 策略 B：article / main 中的最长段落
    const main = document.querySelector('article, main, [role="main"]');
    if (main) {
      const candidates = main.querySelectorAll('p, pre, code, div, section');
      let best = '';
      for (const c of candidates) {
        const t = cleanText(c.textContent);
        if (t.length > best.length && t.length > 80) best = t;
      }
      if (best) return { content: best, strategy: 'main-longest' };
    }

    // 策略 C：白名单 css class
    const selectors = [
      '[class*="prompt-content"]',
      '[class*="prompt-text"]',
      '[class*="prompt-body"]',
      '[class*="prompt-detail"]',
      '[class*="description"]',
      '.whitespace-pre-wrap',
      'pre code',
      'pre'
    ];
    for (const sel of selectors) {
      const el = document.querySelector(sel);
      if (el) {
        const t = cleanText(el.textContent);
        if (t.length > 50) return { content: t, strategy: 'selector-' + sel };
      }
    }

    return { content: '', strategy: 'none' };
  }

  // 检测模型（YouMind 涉及多种 AI 模型）
  function detectYouMindModel() {
    const text = (document.body?.textContent || '').toLowerCase();

    const models = [
      { name: 'Nano Banana Pro', patterns: ['nano banana pro', 'nano-banana-pro'] },
      { name: 'Nano Banana', patterns: ['nano banana', 'nano-banana'] },
      { name: 'GPT Image 2', patterns: ['gpt image 2', 'gpt-image-2'] },
      { name: 'GPT-4o', patterns: ['gpt-4o', 'gpt 4o'] },
      { name: 'Sora', patterns: ['sora'] },
      { name: 'Veo', patterns: ['veo 3', 'veo-3', 'veo3', 'veo'] },
      { name: 'Kling', patterns: ['kling'] },
      { name: 'Runway', patterns: ['runway gen-3', 'runway gen3', 'runway'] },
      { name: 'Pika', patterns: ['pika'] },
      { name: 'Luma', patterns: ['luma dream machine', 'luma'] },
      { name: 'Midjourney', patterns: ['midjourney', 'mj'] },
      { name: 'Stable Diffusion', patterns: ['stable diffusion', 'sdxl', 'sd 1.5'] },
      { name: 'Flux', patterns: ['flux'] },
      { name: 'Claude', patterns: ['claude 3', 'claude-3', 'claude'] },
      { name: 'Gemini', patterns: ['gemini'] },
      { name: 'Grok', patterns: ['grok'] }
    ];

    for (const m of models) {
      if (m.patterns.some(p => text.includes(p))) return m.name;
    }
    return '';
  }

  // 检测媒体类型
  function detectMediaType() {
    if (document.querySelector('video')) return 'video';
    const text = (document.body?.textContent || '').toLowerCase();
    if (text.includes('video') || text.includes('视频')) return 'video';
    return 'image';
  }

  // 提取作者
  function extractAuthor() {
    // YouMind 通常在 meta 中标注作者
    const authorMeta = document.querySelector('meta[name="author"]')?.content;
    if (authorMeta) return authorMeta.trim();
    // 找 @username 模式
    const bodyText = document.body?.textContent || '';
    const atMatch = bodyText.match(/@[A-Za-z0-9_\-.]{3,30}/);
    if (atMatch) return atMatch[0];
    return 'YouMind';
  }

  // 提取分类（YouMind 在 URL 中通常有 /nano-banana-pro-prompts/ 等）
  function extractYouMindCategory() {
    const path = location.pathname.toLowerCase();
    const match = path.match(/\/([a-z0-9-]+)-prompts/);
    if (match) {
      return match[1]
        .split('-')
        .map(s => s.charAt(0).toUpperCase() + s.slice(1))
        .join(' ');
    }
    return '';
  }

  // 提取标签
  function extractYouMindTags() {
    const tags = [];
    const seen = new Set();
    const add = (text) => {
      const t = text.trim().replace(/^#/, '');
      if (t && t.length < 30 && !seen.has(t.toLowerCase())) {
        seen.add(t.toLowerCase());
        tags.push(t);
      }
    };
    document.querySelectorAll('a[href*="/tag/"], [class*="tag"]').forEach(el => {
      add(el.textContent);
    });
    // hashtag
    const bodyText = document.body?.textContent || '';
    const hashes = bodyText.match(/#[A-Za-z0-9_\u4e00-\u9fa5]+/g);
    if (hashes) hashes.forEach(h => add(h));
    return tags.slice(0, 20);
  }

  async function extractPageData() {
    const url = location.href;
    const domain = getDomain(url);
    const promptResult = extractYouMindPrompt();
    const model = detectYouMindModel();
    const cat = extractYouMindCategory();

    return {
      title: extractYouMindTitle(),
      content: promptResult.content,
      content_en: detectLang(promptResult.content) === 'en' ? promptResult.content : '',
      description: document.querySelector('meta[name="description"], meta[property="og:description"]')?.content?.trim() || '',
      model: model || cat || 'YouMind',
      source: extractAuthor(),
      author: extractAuthor(),
      media_type: detectMediaType(),
      tags: JSON.stringify(extractYouMindTags()),
      categories: cat ? [cat] : [],
      cover_image_url: extractYouMindCover(),
      video_url: '',
      video_poster: '',
      source_url: url,
      platform: 'youmind.com',
      extracted_by: 'content-youmind',
      extracted_at: new Date().toISOString(),
      _meta: {
        strategy: promptResult.strategy,
        lang: detectLang(promptResult.content),
        contentLength: promptResult.content?.length || 0,
        category: cat,
        model: model
      }
    };
  }

  chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
    if (!message || typeof message !== 'object') return false;
    (async () => {
      try {
        switch (message.action) {
          case 'extractPrompt': {
            const data = await extractPageData();
            await chrome.storage.local.set({ [STORAGE_KEY]: data });
            chrome.runtime.sendMessage({ action: 'dataExtracted', data }).catch(() => {});

            if (!data.content || data.content.length < 30) {
              sendResponse({
                success: true,
                data,
                reason: 'NO_PROMPT_DETECTED',
                message: 'YouMind 页面未识别到提示词正文（可能是列表页/营销页/分类聚合页，请进入具体提示词详情页后重试）'
              });
            } else {
              sendResponse({ success: true, data });
            }
            break;
          }
          case 'getExtractedPrompt': {
            const result = await chrome.storage.local.get(STORAGE_KEY);
            sendResponse({ success: true, data: result[STORAGE_KEY] || null });
            break;
          }
          case 'ping': {
            sendResponse({ success: true });
            break;
          }
          default:
            sendResponse({ success: false, message: '未知动作' });
        }
      } catch (e) {
        sendResponse({ success: false, message: e.message });
      }
    })();
    return true;
  });

  // 页面加载后预提取
  if (document.readyState === 'complete') {
    extractPageData().then(data => {
      chrome.storage.local.set({ [STORAGE_KEY]: data }).catch(() => {});
    }).catch(() => {});
  } else {
    window.addEventListener('load', () => {
      extractPageData().then(data => {
        chrome.storage.local.set({ [STORAGE_KEY]: data }).catch(() => {});
      }).catch(() => {});
    });
  }
})();
