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
    const h1 = document.querySelector('h1');
    if (h1) {
      const t = cleanText(h1.textContent);
      if (t && t.length < 200) return t;
    }
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
    const img = document.querySelector('article img, main img, .content img, [class*="gallery"] img');
    if (img?.src) return makeAbsoluteUrl(img.src);
    return '';
  }

  // 判断文本是否像 footer / 导航 / 模型 tab 等噪音
  function looksLikeNoise(text) {
    const lower = text.toLowerCase();
    const noiseKw = ['©', 'copyright', 'mind motor', '隐私政策', '服务条款', '联系我们', '公司', '博客', '更新', '定价', '应用', '技能', '产品', 'browser', 'extension'];
    if (noiseKw.some(k => lower.includes(k))) return true;

    // 集中出现多个模型名 → 模型 tab / footer 链接
    const modelNames = ['nano banana', 'gpt image', 'seedance', 'seedream', 'sora', 'veo', 'kling', 'runway', 'pika', 'luma', 'midjourney', 'stable diffusion', 'sdxl', 'flux', 'claude', 'gemini', 'grok'];
    let hits = 0;
    for (const m of modelNames) if (lower.includes(m)) hits++;
    return hits >= 3;
  }

  // 找提示词容器：从"提示词"标签向上冒泡，找到包含足够正文的容器
  function findPromptContainer() {
    const labels = Array.from(document.querySelectorAll('*')).filter(el => {
      // 只考虑直接文本节点，避免把包含"提示词"的大容器当成标签
      for (const node of el.childNodes) {
        if (node.nodeType === Node.TEXT_NODE) {
          const t = cleanText(node.textContent);
          if (t === '提示词' || t.toLowerCase() === 'prompt') return true;
        }
      }
      return false;
    });

    let bestContainer = null;
    let bestLen = 0;
    for (const label of labels) {
      let cur = label.parentElement;
      for (let depth = 0; depth < 6 && cur; depth++, cur = cur.parentElement) {
        const t = cleanText(cur.textContent);
        if (t.length > 200 && !looksLikeNoise(t)) {
          if (t.length > bestLen) {
            bestLen = t.length;
            bestContainer = cur;
          }
          break; // 找到合适的容器就停止向上
        }
      }
    }
    return bestContainer;
  }

  // 提取提示词文本（YouMind 通常在特定 section/div 中放置）
  function extractYouMindPrompt() {
    console.log('[Content-YouMind] extractYouMindPrompt() 开始');

    // 策略 0：找"提示词"标签所在的容器（最可靠）
    const container = findPromptContainer();
    if (container) {
      const clone = container.cloneNode(true);
      // 移除 header 里的按钮、图标、图片
      clone.querySelectorAll('button, [role="button"], svg, img, [class*="icon"]').forEach(el => el.remove());
      let t = cleanText(clone.textContent);
      // 去掉容器头部可能残留的"提示词 Prompt 免费生成视频 翻译前"等标题
      t = t.replace(/^(提示词|prompt)\s*/i, '');
      if (t.length > 80 && !looksLikeNoise(t)) {
        return { content: t, strategy: 'prompt-container' };
      }
    }

    // 策略 A：白名单 css class / 标签
    const selectors = [
      'pre.whitespace-pre-wrap',
      '[class*="whitespace-pre-wrap"]',
      '[class*="prompt-content"]',
      '[class*="prompt-text"]',
      '[class*="prompt-body"]',
      '[class*="prompt-detail"]',
      '[class*="description"]',
      'pre code',
      'pre'
    ];
    for (const sel of selectors) {
      const el = document.querySelector(sel);
      if (el) {
        const t = cleanText(el.textContent);
        if (t.length > 50 && !looksLikeNoise(t)) return { content: t, strategy: 'selector-' + sel };
      }
    }

    // 策略 B：查找带"提示词"或"prompt"标签的块（取最长有效段落）
    const labelKeywords = ['提示词', 'prompt', '正文', '描述', '咒语', '指令'];
    const allEls = document.querySelectorAll('h1, h2, h3, h4, h5, h6, strong, b, span, div, p, button');
    let bestLabeled = '';
    let bestStrategy = '';

    for (const label of allEls) {
      const labelText = cleanText(label.textContent).toLowerCase();
      if (!labelKeywords.some(kw => labelText === kw || labelText.startsWith(kw))) continue;

      const next = label.nextElementSibling;
      if (next) {
        const t = cleanText(next.textContent);
        if (t.length > bestLabeled.length && t.length > 30 && !looksLikeNoise(t)) {
          bestLabeled = t;
          bestStrategy = 'labeled-sibling';
        }
      }
      const parent = label.parentElement;
      if (parent) {
        const ps = parent.querySelectorAll('p, pre, code, div');
        for (const p of ps) {
          if (p === label) continue;
          const t = cleanText(p.textContent);
          if (t.length > bestLabeled.length && t.length > 30 && !looksLikeNoise(t)) {
            bestLabeled = t;
            bestStrategy = 'labeled-parent-p';
          }
        }
      }
    }
    if (bestLabeled) return { content: bestLabeled, strategy: bestStrategy };

    // 策略 C：article / main 中的最长段落
    const main = document.querySelector('article, main, [role="main"]');
    if (main) {
      const candidates = main.querySelectorAll('p, pre, code, div, section');
      let best = '';
      for (const c of candidates) {
        const t = cleanText(c.textContent);
        if (t.length > best.length && t.length > 80 && !looksLikeNoise(t)) best = t;
      }
      if (best) return { content: best, strategy: 'main-longest' };
    }

    // 策略 D：兜底 — 抓取页面主要内容区域的所有文字
    const bodyText = document.body?.innerText || '';
    if (bodyText.length > 200) {
      const blocks = bodyText.split(/\n\s*\n/).map(s => cleanText(s)).filter(s => s.length > 50);
      const skipKw = ['登录', '注册', '首页', '导航', 'footer', 'copyright', '©', 'cookie', '隐私', '条款', 'mind motor'];
      let bestBlock = '';
      for (const b of blocks) {
        const lower = b.toLowerCase();
        if (skipKw.some(k => lower.includes(k))) continue;
        if (looksLikeNoise(b)) continue;
        if (b.length > bestBlock.length) bestBlock = b;
      }
      if (bestBlock.length > 80) return { content: bestBlock, strategy: 'body-blocks' };
    }

    return { content: '', strategy: 'none' };
  }

  // 检测模型（YouMind 涉及多种 AI 模型）
  function detectYouMindModel() {
    const models = [
      { name: 'Nano Banana Pro', patterns: ['nano banana pro', 'nano-banana-pro'] },
      { name: 'Nano Banana 2', patterns: ['nano banana 2', 'nano-banana-2'] },
      { name: 'Nano Banana', patterns: ['nano banana', 'nano-banana'] },
      { name: 'GPT Image 2', patterns: ['gpt image 2', 'gpt-image-2'] },
      { name: 'GPT Image 1.5', patterns: ['gpt image 1.5', 'gpt-image-1.5'] },
      { name: 'GPT-4o', patterns: ['gpt-4o', 'gpt 4o'] },
      { name: 'Seedance 2.0', patterns: ['seedance 2.0', 'seedance-2.0', 'seedance2.0'] },
      { name: 'Seedance', patterns: ['seedance'] },
      { name: 'Seedream 4.5', patterns: ['seedream 4.5', 'seedream-4.5', 'seedream4.5'] },
      { name: 'Seedream', patterns: ['seedream'] },
      { name: 'Sora', patterns: ['sora'] },
      { name: 'Veo 3', patterns: ['veo 3', 'veo-3', 'veo3'] },
      { name: 'Veo', patterns: ['veo'] },
      { name: 'Kling', patterns: ['kling'] },
      { name: 'Runway Gen-3', patterns: ['runway gen-3', 'runway-gen-3', 'runway gen3'] },
      { name: 'Runway', patterns: ['runway'] },
      { name: 'Pika', patterns: ['pika'] },
      { name: 'Luma Dream Machine', patterns: ['luma dream machine', 'luma-dream-machine'] },
      { name: 'Luma', patterns: ['luma'] },
      { name: 'Midjourney', patterns: ['midjourney'] },
      { name: 'Stable Diffusion', patterns: ['stable diffusion', 'sdxl', 'sd 1.5'] },
      { name: 'Flux', patterns: ['flux'] },
      { name: 'Claude 3', patterns: ['claude 3', 'claude-3'] },
      { name: 'Claude', patterns: ['claude'] },
      { name: 'Gemini 3 Pro', patterns: ['gemini 3 pro', 'gemini-3-pro'] },
      { name: 'Gemini', patterns: ['gemini'] },
      { name: 'Grok', patterns: ['grok'] }
    ];

    function matchModel(text) {
      const lower = (text || '').toLowerCase();
      for (const m of models) {
        if (m.patterns.some(p => lower.includes(p))) return m.name;
      }
      return '';
    }

    // 1) 优先：h1 附近（包括 h1 上方/同层的小徽章），扩大到 h1 向上 4 层或 article/main
    const h1 = document.querySelector('h1');
    if (h1) {
      // 先扫描 h1 前面紧邻的兄弟/父级兄弟（模型徽章常在标题上方）
      let probe = h1.previousElementSibling;
      while (probe) {
        const t = cleanText(probe.textContent);
        const hit = matchModel(t);
        if (hit) return hit;
        probe = probe.previousElementSibling;
      }
      // 扫描 h1 父级向上 4 层
      let cur = h1.parentElement;
      for (let i = 0; i < 4 && cur; i++, cur = cur.parentElement) {
        const t = cleanText(cur.textContent);
        const hit = matchModel(t);
        if (hit) return hit;
      }
      // 扫描 h1 所在 article/main 容器
      const container = h1.closest('article, main, [class*="detail"], [class*="content"], [class*="header"]');
      if (container) {
        const t = cleanText(container.textContent);
        const hit = matchModel(t);
        if (hit) return hit;
      }
    }

    // 2) 扫描页面中所有小文本元素（< 40 字符），很可能是徽章/标签
    for (const el of document.querySelectorAll('span, div, a, button, li')) {
      const t = cleanText(el.textContent);
      if (!t || t.length < 3 || t.length > 40) continue;
      const hit = matchModel(t);
      if (hit) return hit;
    }

    // 3) 全 body 兜底（但排除 footer）
    const footerLike = document.querySelector('footer, [class*="footer"], [class*="bottom-nav"]');
    const bodyText = (document.body?.textContent || '').toLowerCase();
    const footerText = footerLike ? cleanText(footerLike.textContent).toLowerCase() : '';
    for (const m of models) {
      if (m.patterns.some(p => bodyText.includes(p) && !footerText.includes(p))) return m.name;
    }
    for (const m of models) {
      if (m.patterns.some(p => bodyText.includes(p))) return m.name;
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

  // 从 Cloudflare Stream 封面图 URL 推导视频地址
  function deriveCloudflareVideo(coverUrl) {
    if (!coverUrl) return '';
    // 封面图：https://customer-xxx.cloudflarestream.com/{uid}/thumbnails/thumbnail.jpg?...
    // 视频地址：https://customer-xxx.cloudflarestream.com/{uid}/downloads/default.mp4
    const m = coverUrl.match(/(https:\/\/customer-[^/]+\.cloudflarestream\.com\/[^/]+)/i);
    if (m) {
      const base = m[1];
      return `${base}/downloads/default.mp4`;
    }
    return '';
  }

  // 扫描页面上所有图片，找出 Cloudflare Stream 缩略图并推导视频地址
  function scanCloudflareStreamInAllImages() {
    const imgs = Array.from(document.querySelectorAll('img'));
    // 优先带 cloudflarestream 的 src/srcset/data-src
    for (const img of imgs) {
      const src = img.src || img.getAttribute('data-src') || img.getAttribute('srcset') || '';
      if (src.includes('cloudflarestream')) {
        const derived = deriveCloudflareVideo(src);
        if (derived) return { url: derived, poster: makeAbsoluteUrl(src.split(' ')[0]) };
      }
    }
    // 其次 background-image
    for (const el of document.querySelectorAll('*')) {
      const bg = getComputedStyle(el).backgroundImage || '';
      const m = bg.match(/https:\/\/customer-[^/"')]+\.cloudflarestream\.com\/[^/"')]+/i);
      if (m) {
        const derived = deriveCloudflareVideo(m[0]);
        if (derived) return { url: derived, poster: makeAbsoluteUrl(m[0]) };
      }
    }
    return null;
  }

  // 扫描 iframe 中的 video / 图片
  function scanIframes() {
    for (const iframe of document.querySelectorAll('iframe')) {
      try {
        const doc = iframe.contentDocument || iframe.contentWindow?.document;
        if (!doc) continue;
        const video = doc.querySelector('video');
        if (video) {
          const url = video.src || video.currentSrc || video.querySelector('source')?.src || '';
          if (url) return { url: makeAbsoluteUrl(url), poster: makeAbsoluteUrl(video.poster) };
        }
        const img = doc.querySelector('img');
        if (img?.src) {
          const derived = deriveCloudflareVideo(img.src);
          if (derived) return { url: derived, poster: makeAbsoluteUrl(img.src) };
        }
      } catch (e) {}
    }
    return null;
  }

  // 等待 video 元素出现（YouMind 页面常动态加载）
  function waitForVideo(maxMs = 4000) {
    return new Promise((resolve) => {
      const video = document.querySelector('video');
      if (video) return resolve(video);
      const observer = new MutationObserver(() => {
        const v = document.querySelector('video');
        if (v) {
          observer.disconnect();
          resolve(v);
        }
      });
      observer.observe(document.body || document.documentElement, { childList: true, subtree: true });
      setTimeout(() => {
        observer.disconnect();
        resolve(document.querySelector('video'));
      }, maxMs);
    });
  }

  // 提取视频 URL（video prompt 详情页）
  async function extractYouMindVideo() {
    const logs = [];

    // 0) 先等一等动态加载的 video 标签
    const dynamicVideo = await waitForVideo(3000);
    if (dynamicVideo) {
      const url = dynamicVideo.src || dynamicVideo.currentSrc || dynamicVideo.querySelector('source')?.src || '';
      if (url) {
        logs.push(`命中动态 video 标签: ${url}`);
        return { url: makeAbsoluteUrl(url), poster: makeAbsoluteUrl(dynamicVideo.poster), logs };
      }
    }

    // 1) 页面 <video> 标签
    const video = document.querySelector('video');
    if (video) {
      if (video.src) return { url: makeAbsoluteUrl(video.src), poster: makeAbsoluteUrl(video.poster), logs };
      const source = video.querySelector('source');
      if (source?.src) return { url: makeAbsoluteUrl(source.src), poster: makeAbsoluteUrl(video.poster), logs };
    }

    // 2) og:video / twitter:video
    const ogVideo = document.querySelector('meta[property="og:video"], meta[property="og:video:url"]')?.content;
    if (ogVideo) return { url: makeAbsoluteUrl(ogVideo), poster: '', logs };

    // 3) 页面可见的媒体链接
    for (const a of document.querySelectorAll('a[href*=".mp4"], a[href*=".mov"], a[href*=".webm"], a[href*=".m3u8"]')) {
      return { url: makeAbsoluteUrl(a.href), poster: '', logs };
    }

    for (const el of document.querySelectorAll('[data-src*=".mp4"], [data-src*=".mov"], [data-src*=".webm"], [data-src*=".m3u8"], [src*=".mp4"], [src*=".mov"], [src*=".webm"], [src*=".m3u8"]')) {
      const url = el.getAttribute('data-src') || el.src;
      if (url) return { url: makeAbsoluteUrl(url), poster: '', logs };
    }

    // 4) JSON-LD 中找视频
    for (const script of document.querySelectorAll('script[type="application/ld+json"]')) {
      try {
        const json = JSON.parse(script.textContent);
        const candidates = [json.video, json.contentUrl, json.embedUrl, json.url];
        for (const c of candidates) {
          if (typeof c === 'string' && /\.(mp4|mov|webm|m3u8)(\?|$)/i.test(c)) {
            return { url: makeAbsoluteUrl(c), poster: json.thumbnailUrl || '', logs };
          }
        }
        if (json['@graph']) {
          for (const item of json['@graph']) {
            const u = item.contentUrl || item.embedUrl || item.url;
            if (typeof u === 'string' && /\.(mp4|mov|webm|m3u8)(\?|$)/i.test(u)) {
              return { url: makeAbsoluteUrl(u), poster: item.thumbnailUrl || '', logs };
            }
          }
        }
      } catch (e) {}
    }

    // 5) Cloudflare Stream 专用：从所有图片推导
    const fromImages = scanCloudflareStreamInAllImages();
    if (fromImages) {
      logs.push(`从图片推导出 Cloudflare Stream: ${fromImages.url}`);
      return { ...fromImages, logs };
    }

    // 6) 扫描所有文本中的 cloudflarestream 链接（包括 iframe src、脚本变量）
    const allText = document.documentElement.innerHTML;
    const streamMatches = allText.match(/https:\/\/customer-[^"'\s<>]+\.cloudflarestream\.com\/[^"'\s<>]+/gi);
    if (streamMatches) {
      // 优先找 /downloads/default.mp4
      for (const u of streamMatches) {
        if (/\/downloads\/default\.mp4(\?|$)/i.test(u)) return { url: u, poster: '', logs };
      }
      for (const u of streamMatches) {
        if (/\.(mp4|mov|webm|m3u8)(\?|$)/i.test(u)) return { url: u, poster: '', logs };
      }
      // 如果只有 uid 没有扩展名，补成 /downloads/default.mp4
      const first = streamMatches[0].replace(/\/$/, '');
      if (!/\.[a-z0-9]+(\?|$)/i.test(first)) {
        return { url: `${first}/downloads/default.mp4`, poster: '', logs };
      }
    }

    // 7) iframe 扫描
    const fromIframe = scanIframes();
    if (fromIframe) return { ...fromIframe, logs };

    // 8) 全局 JS 变量（YouMind 可能把资源放在 __INITIAL_STATE__ / __DATA__ / __NUXT__ 等）
    const globals = ['__INITIAL_STATE__', '__DATA__', '__NUXT__', '__APP__', '__CONFIG__'];
    for (const key of globals) {
      try {
        const val = window[key];
        if (!val) continue;
        const str = JSON.stringify(val);
        const m = str.match(/(https?:\/\/[^"'\s]+\.(mp4|mov|webm|m3u8)(\?[^"'\s]*)?)/i);
        if (m) return { url: m[1], poster: '', logs };
        // Cloudflare Stream uid
        const uidMatch = str.match(/([a-f0-9]{32}|[a-zA-Z0-9]{32})/);
        const coverUrl = extractYouMindCover();
        if (uidMatch && coverUrl.includes('cloudflarestream')) {
          const baseMatch = coverUrl.match(/(https:\/\/customer-[^/]+\.cloudflarestream\.com)/i);
          if (baseMatch) return { url: `${baseMatch[1]}/${uidMatch[1]}/downloads/default.mp4`, poster: coverUrl, logs };
        }
      } catch (e) {}
    }

    // 9) 所有 <script> 标签文本中挖视频 URL（兜底）
    for (const script of document.querySelectorAll('script')) {
      const text = script.textContent || '';
      const m = text.match(/(https?:\/\/[^"'\s]+\.(mp4|mov|webm|m3u8)(\?[^"'\s]*)?)/i);
      if (m) return { url: m[1], poster: '', logs };
    }

    logs.push('未找到视频 URL');
    return { url: '', poster: '', logs };
  }

  // 提取作者
  function extractAuthor() {
    const authorMeta = document.querySelector('meta[name="author"]')?.content;
    if (authorMeta) return authorMeta.trim();

    // 1) 优先：明确作者链接（YouMind 常见 /u/xxx /user/xxx /creator/xxx）
    const authorSels = [
      'a[href*="/u/"]',
      'a[href*="/user/"]',
      'a[href*="/creator/"]',
      'a[href*="/author/"]',
      '[class*="author"] a',
      '[class*="creator"] a',
      '[class*="user"] a'
    ];
    for (const sel of authorSels) {
      const el = document.querySelector(sel);
      if (el) {
        const t = cleanText(el.textContent);
        if (t && t.length > 1 && t.length < 60 && !t.toLowerCase().includes('@-webkit') && !t.toLowerCase().includes('@keyframes')) {
          return t;
        }
      }
    }

    // 2) 从 h1 右侧/标题周围的信息区提取 @用户名（允许空格，如 @AI Arainz）
    const h1 = document.querySelector('h1');
    if (h1) {
      const container = h1.closest('article, main, [class*="detail"], [class*="content"]') || document.body;
      const atEls = container.querySelectorAll('*');
      for (const el of atEls) {
        const t = cleanText(el.textContent);
        // 只考虑短小文本节点，避免大段正文
        if (!t || t.length > 80) continue;
        const m = t.match(/@[A-Za-z0-9_\-. ]{2,40}/);
        if (m) {
          const name = cleanText(m[0]);
          const lower = name.toLowerCase();
          if (lower.includes('@-webkit') || lower.includes('@keyframes') || lower.includes('@media') || lower.includes('@supports')) continue;
          return name;
        }
      }
    }

    // 3) 全 body 兜底匹配 @用户名
    const bodyText = document.body?.textContent || '';
    const atMatches = bodyText.match(/@[A-Za-z0-9_\-. ]{2,40}/g);
    if (atMatches) {
      for (const m of atMatches) {
        const name = cleanText(m);
        const lower = name.toLowerCase();
        if (lower.includes('@-webkit') || lower.includes('@keyframes') || lower.includes('@media') || lower.includes('@supports')) continue;
        return name;
      }
    }
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

    // 噪音词：CSS 类名、schema.org 类型、无意义词
    const junk = new Set([
      'nprogress', 'organization', 'website', 'breadcrumb', 'creativew', 'creativework', 'creativeworks',
      'webpage', 'webpageelement', 'videoobject', 'imageobject', 'article', 'person', 'thing', 'product',
      'navbar', 'header', 'footer', 'container', 'wrapper', 'content', 'main', 'section', 'page', 'home',
      'index', 'active', 'selected', 'disabled', 'loading', 'hidden', 'visible', 'stylesheet', 'script'
    ]);

    const add = (text) => {
      const t = text.trim().replace(/^#/, '');
      if (!t || t.length < 2 || t.length > 30) return;
      const lower = t.toLowerCase();
      if (junk.has(lower)) return;
      // 过滤颜色代码：FF6B6B_0 / FF6B6B_100 / #FF6B6B
      if (/^[a-f0-9]{3,8}(_\d+)?$/i.test(t)) return;
      if (/^#[a-f0-9]{3,8}$/i.test(t)) return;
      // 过滤纯数字/下划线组合
      if (/^[0-9_]+$/.test(t)) return;
      if (!seen.has(lower)) {
        seen.add(lower);
        tags.push(t);
      }
    };

    // 策略 1：优先从"分类"区域附近的胶囊标签提取（YouMind 详情页常见）
    const catSection = findCategorySection();
    if (catSection) {
      catSection.querySelectorAll('a, button, span, div').forEach(el => add(el.textContent));
      if (tags.length > 0) return tags.slice(0, 20);
    }

    // 策略 2：显式 tag 元素
    document.querySelectorAll('a[href*="/tag/"], [class*="tag"]:not([class*="tagline"])').forEach(el => add(el.textContent));

    // 策略 3：hashtag
    const bodyText = document.body?.textContent || '';
    const hashes = bodyText.match(/#[A-Za-z0-9_\u4e00-\u9fa5]+/g);
    if (hashes) hashes.forEach(h => add(h));

    return tags.slice(0, 20);
  }

  // 找 YouMind 的"分类"区块（含黄色胶囊标签）
  function findCategorySection() {
    const labels = Array.from(document.querySelectorAll('*')).filter(el => {
      for (const node of el.childNodes) {
        if (node.nodeType === Node.TEXT_NODE) {
          const t = cleanText(node.textContent);
          if (t === '分类' || t.toLowerCase() === 'category') return true;
        }
      }
      return false;
    });

    for (const label of labels) {
      let cur = label.parentElement;
      for (let depth = 0; depth < 5 && cur; depth++, cur = cur.parentElement) {
        // 找包含多个胶囊状子元素的容器
        const children = cur.querySelectorAll('a, button, span, div');
        let tagLikeCount = 0;
        for (const c of children) {
          const t = cleanText(c.textContent);
          if (t && t.length >= 2 && t.length <= 15 && !/^[a-f0-9]{6,8}$/i.test(t)) {
            tagLikeCount++;
          }
        }
        if (tagLikeCount >= 3) return cur;
      }
    }
    return null;
  }

  async function extractPageData() {
    console.log('[Content-YouMind] extractPageData() URL:', location.href);
    const url = location.href;
    const domain = getDomain(url);
    const promptResult = extractYouMindPrompt();
    const model = detectYouMindModel();
    const cat = extractYouMindCategory();
    console.log('[Content-YouMind] 提取结果:', { strategy: promptResult.strategy, contentLen: promptResult.content?.length, model, cat });

    const videoInfo = await extractYouMindVideo();
    console.log('[Content-YouMind] 视频提取:', {
      videoUrl: videoInfo.url,
      poster: videoInfo.poster,
      cover: extractYouMindCover(),
      logs: videoInfo.logs
    });

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
      video_url: videoInfo.url,
      video_poster: videoInfo.poster,
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
