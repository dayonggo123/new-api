// content-universal.js - 通用页面提示词提取（兼容 Prompt Collector）
// 作为 content-detail.js / content-list.js / content-wechat.js 的补充，
// 当页面没有特定脚本覆盖时，提供通用的 prompt 提取能力。

(function () {
  'use strict';

  const STORAGE_KEY = 'promptCollectorExtracted';

  // 工具函数
  function getDomain(url) {
    try {
      return new URL(url).hostname.replace(/^www\./, '');
    } catch (e) {
      return '';
    }
  }

  function cleanText(text) {
    if (!text) return '';
    return text
      .replace(/\s+/g, ' ')
      .replace(/[\u200B-\u200D\uFEFF]/g, '')
      .trim();
  }

  function isPromptLike(text) {
    if (!text || text.length < 50) return false;
    const keywords = [
      'prompt', '提示词', 'prompts', '提示', 'instruction',
      'generate', 'create', 'produce', 'render', 'imagine',
      'midjourney', 'stable diffusion', 'dall-e', 'gpt', 'claude',
      'sora', 'veo', 'kling', 'runway', 'pika', 'luma',
      'flux', 'sdxl', 'comfyui', 'fooocus',
      'cinematic', 'hyper-realistic', 'highly detailed',
      '4k', '8k', 'uhd', 'masterpiece', 'best quality',
      'negative prompt', '反向提示词', '负面提示词'
    ];
    const lower = text.toLowerCase();
    return keywords.some(kw => lower.includes(kw.toLowerCase()));
  }

  function detectLang(text) {
    if (/[\u4e00-\u9fa5]/.test(text)) return 'zh';
    if (/[\u3040-\u309F\u30A0-\u30FF]/.test(text)) return 'ja';
    if (/[\uAC00-\uD7AF]/.test(text)) return 'ko';
    return 'en';
  }

  function makeAbsoluteUrl(url) {
    if (!url) return '';
    if (url.startsWith('http')) return url;
    if (url.startsWith('//')) return 'https:' + url;
    try {
      return new URL(url, location.href).href;
    } catch (e) {
      return url;
    }
  }

  function extractAuthor() {
    const pageText = document.body?.textContent || '';
    const patterns = [
      /来源[\s:：]+(@[A-Za-z0-9_\-.]+)/,
      /作者[\s:：]+(@[A-Za-z0-9_\-.]+)/,
      /作者[\s:：]+([A-Za-z0-9_\-.]{3,30})/,
      /By[\s:：]+(@[A-Za-z0-9_\-.]+)/i,
      /Created by[\s:：]+(@[A-Za-z0-9_\-.]+)/i,
      /@[A-Za-z0-9_\-.]{3,30}/
    ];
    for (const pattern of patterns) {
      const match = pageText.match(pattern);
      if (match) return match[1].trim();
    }
    return '';
  }

  function extractCategories() {
    const categories = [];
    const seen = new Set();
    const add = (text) => {
      const t = text.trim();
      if (t && t.length < 30 && !seen.has(t.toLowerCase())) {
        seen.add(t.toLowerCase());
        categories.push(t);
      }
    };

    const selectors = [
      '[class*="category"] a',
      '[class*="tag"] a',
      '[class*="tag"]',
      '.breadcrumb a',
      '[class*="breadcrumb"] a',
      'a[href*="/category/"]',
      'a[href*="/tag/"]'
    ];
    for (const sel of selectors) {
      document.querySelectorAll(sel).forEach(el => add(el.textContent));
    }

    return categories.slice(0, 10);
  }

  function extractTags() {
    const tags = [];
    const seen = new Set();
    const add = (text) => {
      const t = text.trim().toLowerCase().replace(/^#/, '');
      if (t && t.length < 30 && !seen.has(t)) {
        seen.add(t);
        tags.push(text.trim().replace(/^#/, ''));
      }
    };

    document.querySelectorAll('a[href*="/tag/"], a[href*="/tags/"], [class*="tag"]').forEach(el => {
      add(el.textContent);
    });

    const pageText = document.body?.textContent || '';
    const hashtags = pageText.match(/#[A-Za-z0-9_\u4e00-\u9fa5]+/g);
    if (hashtags) hashtags.forEach(tag => add(tag));

    return tags.slice(0, 20);
  }

  function extractTitle() {
    const h1 = document.querySelector('h1');
    if (h1) {
      const text = h1.textContent.trim();
      if (text && text.length < 200) return text;
    }
    const ogTitle = document.querySelector('meta[property="og:title"]')?.content;
    if (ogTitle) return ogTitle.trim();
    return document.title.trim();
  }

  function extractDescription() {
    const desc = document.querySelector('meta[name="description"], meta[property="og:description"]')?.content;
    return desc ? desc.trim() : '';
  }

  function detectModel() {
    const pageText = document.body?.textContent || '';
    const lower = pageText.toLowerCase();

    const models = [
      { name: 'Sora', patterns: ['sora'] },
      { name: 'Veo', patterns: ['veo 3', 'veo-3', 'veo3', 'veo'] },
      { name: 'Kling', patterns: ['kling'] },
      { name: 'Runway', patterns: ['runway'] },
      { name: 'Pika', patterns: ['pika'] },
      { name: 'Luma Dream Machine', patterns: ['luma', 'dream machine'] },
      { name: 'Stable Video Diffusion', patterns: ['svd', 'stable video'] },
      { name: 'Midjourney', patterns: ['midjourney', 'mj'] },
      { name: 'Stable Diffusion', patterns: ['stable diffusion', 'sdxl', 'sd 1.5'] },
      { name: 'DALL-E', patterns: ['dall-e', 'dalle'] },
      { name: 'Flux', patterns: ['flux'] },
      { name: 'GPT-4o', patterns: ['gpt-4o', 'gpt 4o'] },
      { name: 'GPT-4', patterns: ['gpt-4', 'gpt 4'] },
      { name: 'Claude', patterns: ['claude 3', 'claude-3', 'claude'] },
      { name: 'Gemini', patterns: ['gemini'] },
      { name: 'Seedance', patterns: ['seedance'] },
      { name: 'Grok', patterns: ['grok'] }
    ];

    for (const model of models) {
      if (model.patterns.some(p => lower.includes(p))) return model.name;
    }

    return detectMediaType() === 'video' ? 'Video Model' : 'Image Model';
  }

  function detectMediaType() {
    const pageText = document.body?.textContent.toLowerCase() || '';
    if (pageText.includes('video') || document.querySelector('video')) return 'video';
    return 'image';
  }

  function extractCoverImage() {
    const video = document.querySelector('video');
    if (video?.poster) return makeAbsoluteUrl(video.poster);

    const ogImage = document.querySelector('meta[property="og:image"]')?.content;
    if (ogImage) return makeAbsoluteUrl(ogImage);

    const twitterImage = document.querySelector('meta[name="twitter:image"]')?.content;
    if (twitterImage) return makeAbsoluteUrl(twitterImage);

    const firstImg = document.querySelector('article img, main img, .content img');
    if (firstImg?.src) return makeAbsoluteUrl(firstImg.src);

    return '';
  }

  function extractVideoUrl() {
    const video = document.querySelector('video');
    if (video) {
      if (video.src) return makeAbsoluteUrl(video.src);
      const source = video.querySelector('source');
      if (source?.src) return makeAbsoluteUrl(source.src);
    }

    const iframes = document.querySelectorAll('iframe');
    for (const iframe of iframes) {
      const src = iframe.src;
      if (src && /(cloudflarestream|youtube|vimeo|bilibili)/i.test(src)) {
        return src;
      }
    }

    const html = document.documentElement.innerHTML;
    const streamMatch = html.match(/(customer-[a-z0-9]+\.cloudflarestream\.com\/[a-f0-9]+)/);
    if (streamMatch) {
      return `https://${streamMatch[1]}/downloads/default.mp4`;
    }

    return '';
  }

  function extractVideoPoster() {
    const video = document.querySelector('video');
    if (video?.poster) return makeAbsoluteUrl(video.poster);

    const html = document.documentElement.innerHTML;
    const streamMatch = html.match(/(customer-[a-z0-9]+\.cloudflarestream\.com\/[a-f0-9]+)/);
    if (streamMatch) {
      return `https://${streamMatch[1]}/thumbnails/thumbnail.jpg`;
    }

    return '';
  }

  function extractPromptText() {
    // 策略1：带标签的 prompt
    const labeled = extractLabeledPrompt();
    if (labeled) return { content: labeled, lang: detectLang(labeled) };

    // 策略2：CSS 选择器匹配
    const selectorPrompt = extractSelectorPrompt();
    if (selectorPrompt) return { content: selectorPrompt, lang: detectLang(selectorPrompt) };

    // 策略3：最长且像提示词的文本
    const longest = extractLongestPromptLike();
    if (longest) return { content: longest, lang: detectLang(longest) };

    // 策略4：兜底 main/article 内容
    const main = document.querySelector('article, main, [role="main"]');
    if (main) {
      const text = cleanText(main.textContent);
      if (text.length > 50) {
        return { content: text.substring(0, 3000), lang: detectLang(text) };
      }
    }

    return { content: '', lang: 'unknown' };
  }

  function extractLabeledPrompt() {
    const allElements = document.querySelectorAll('pre, code, textarea, p, div, section');
    for (const el of allElements) {
      const text = el.textContent.trim();
      if (text.length < 50) continue;

      const prev = el.previousElementSibling;
      const prevText = prev ? prev.textContent.trim().toLowerCase() : '';
      const parentPrev = el.parentElement?.previousElementSibling;
      const parentPrevText = parentPrev ? parentPrev.textContent.trim().toLowerCase() : '';
      const labelText = prevText + ' ' + parentPrevText;

      if (labelText.includes('prompt') || labelText.includes('提示词') ||
          labelText.includes('提示') || labelText.includes('咒语') ||
          labelText.includes('positive prompt')) {
        return cleanText(text);
      }
    }
    return '';
  }

  function extractSelectorPrompt() {
    const selectors = [
      '[class*="prompt-content"]',
      '[class*="prompt-text"]',
      '[class*="prompt-body"]',
      '[class*="prompt-detail"]',
      '[data-testid*="prompt"]',
      '.whitespace-pre-wrap',
      '[class*="pre-wrap"]',
      '[class*="prompt"]:not(button):not(a)'
    ];

    for (const sel of selectors) {
      const el = document.querySelector(sel);
      if (el) {
        const text = el.textContent.trim();
        if (text.length > 80 && isPromptLike(text)) {
          return cleanText(text);
        }
      }
    }
    return '';
  }

  function extractLongestPromptLike() {
    let best = '';
    const candidates = document.querySelectorAll('pre, code, div, p, section, article');

    for (const el of candidates) {
      const text = el.textContent.trim();
      if (text.length < 100 || text.length > 8000) continue;
      if (text.length <= best.length) continue;

      const parentTag = el.parentElement?.tagName.toLowerCase() || '';
      if (['nav', 'footer', 'header', 'script', 'style'].includes(parentTag)) continue;

      if (isPromptLike(text)) best = text;
    }

    return best ? cleanText(best) : '';
  }

  // 主提取函数，返回 Prompt Collector 兼容格式
  async function extractPageData() {
    const url = location.href;
    const domain = getDomain(url);
    const promptResult = extractPromptText();
    const author = extractAuthor();

    const data = {
      title: extractTitle(),
      content: promptResult.content,
      content_en: promptResult.lang === 'en' ? promptResult.content : '',
      description: extractDescription(),
      model: detectModel(),
      source: author || domain,
      author: author || domain,
      media_type: detectMediaType(),
      tags: JSON.stringify(extractTags()),
      categories: extractCategories(),
      cover_image_url: extractCoverImage(),
      video_url: extractVideoUrl(),
      video_poster: extractVideoPoster(),
      source_url: url,
      platform: domain,
      extracted_by: 'content-universal',
      extracted_at: new Date().toISOString()
    };

    return data;
  }

  // 监听来自 background/popup 的消息
  chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
    if (!message || typeof message !== 'object') return false;

    (async () => {
      try {
        switch (message.action) {
          case 'extractPrompt': {
            const data = await extractPageData();
            await chrome.storage.local.set({ [STORAGE_KEY]: data });
            chrome.runtime.sendMessage({ action: 'dataExtracted', data }).catch(() => {});
            sendResponse({ success: true, data });
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

  // 页面加载后自动预提取（静默）
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
