// content-wechat.js -- WeChat article collector
// Shows a floating button on wechat article pages to collect into article management
(function () {
  'use strict';

  console.log('[WeChat Collector] Script injected');

  // ==================== HTML -> Markdown ====================
  function htmlToMarkdown(html) {
    let md = html;
    md = md.replace(/<script[\s\S]*?<\/script>/gi, '');
    md = md.replace(/<style[\s\S]*?<\/style>/gi, '');
    md = md.replace(/<!--[\s\S]*?-->/g, '');

    md = md.replace(/<pre><code[^>]*>([\s\S]*?)<\/code><\/pre>/gi, (_, code) => {
      return '\n```\n' + decodeHtml(code.trim()) + '\n```\n\n';
    });
    // Images: handle src and data-src (wechat lazy loading)
    md = md.replace(/<(?:img|amp-img)[^>]+(?:data-src|src)=["']([^"']+(?:png|jpg|jpeg|gif|webp)[^"']*)["'][^>]*>/gi, function(match, src) {
      var altMatch = match.match(/alt=["']([^"']*)["']/i);
      var cleanSrc = src.split('?')[0];
      return '\n![' + (altMatch ? altMatch[1] : 'image') + '](' + cleanSrc + ')\n';
    });
    // Video: handle multiple patterns (src, data-src, source children)
    md = md.replace(/<video[^>]*>[\s\S]*?<\/video>/gi, function(match) {
      // Try src first, then data-src (wechat lazy loading)
      var src = match.match(/src=["']([^"']+)["']/);
      if (!src) src = match.match(/data-src=["']([^"']+)["']/);
      if (!src) {
        // Try source child element
        var source = match.match(/<source[^>]+src=["']([^"']+)["']/);
        if (source) src = source;
      }
      if (src) return '\n<video controls src="' + src[1] + '"></video>\n';
      return '';
    });
    md = md.replace(/<iframe[^>]+src=["']([^"']+)["'][^>]*>/gi, (_, src) => '\n<iframe src="' + src + '"></iframe>\n');
    // Wechat data-src video (inline style player)
    md = md.replace(/<(div|span)[^>]+data-src=["']([^"']+\.(mp4|webm))["'][^>]*>/gi, (_, tag, videoSrc) => '\n<video controls src="' + videoSrc + '"></video>\n');
    md = md.replace(/<a[^>]+href=["']([^"']+)["'][^>]*>([\s\S]*?)<\/a>/gi, (_, href, text) => {
      const t = text.replace(/<[^>]+>/g, '').trim();
      return t ? '[' + t + '](' + href + ')' : href;
    });
    md = md.replace(/<h1[^>]*>([\s\S]*?)<\/h1>/gi, (_, t) => '\n# ' + stripTags(t) + '\n');
    md = md.replace(/<h2[^>]*>([\s\S]*?)<\/h2>/gi, (_, t) => '\n## ' + stripTags(t) + '\n');
    md = md.replace(/<h3[^>]*>([\s\S]*?)<\/h3>/gi, (_, t) => '\n### ' + stripTags(t) + '\n');
    md = md.replace(/<h4[^>]*>([\s\S]*?)<\/h4>/gi, (_, t) => '\n#### ' + stripTags(t) + '\n');
    md = md.replace(/<strong>([\s\S]*?)<\/strong>/gi, '**$1**');
    md = md.replace(/<b>([\s\S]*?)<\/b>/gi, '**$1**');
    md = md.replace(/<em>([\s\S]*?)<\/em>/gi, '*$1*');
    md = md.replace(/<i>([\s\S]*?)<\/i>/gi, '*$1*');
    md = md.replace(/<blockquote[^>]*>([\s\S]*?)<\/blockquote>/gi, (_, q) => {
      return q.replace(/<[^>]+>/g, '').trim().split('\n').map(function(l) { return '> ' + l; }).join('\n') + '\n';
    });
    md = md.replace(/<ul[^>]*>([\s\S]*?)<\/ul>/gi, (_, list) => {
      return list.replace(/<li[^>]*>([\s\S]*?)<\/li>/gi, function(__, item) { return '- ' + stripTags(item) + '\n'; }) + '\n';
    });
    md = md.replace(/<ol[^>]*>([\s\S]*?)<\/ol>/gi, (_, list) => {
      var i = 0;
      return list.replace(/<li[^>]*>([\s\S]*?)<\/li>/gi, function(__, item) { i++; return i + '. ' + stripTags(item) + '\n'; }) + '\n';
    });
    md = md.replace(/<br\s*\/?>/gi, '\n');
    md = md.replace(/<\/p>/gi, '\n\n');
    md = md.replace(/<\/div>/gi, '\n');
    md = md.replace(/<[^>]+>/g, '');
    md = md.replace(/&nbsp;/g, ' ').replace(/&amp;/g, '&').replace(/&lt;/g, '<').replace(/&gt;/g, '>').replace(/&quot;/g, '"');
    md = md.replace(/\n{4,}/g, '\n\n\n');
    return md.trim();
  }

  function stripTags(html) {
    return html.replace(/<[^>]+>/g, '').trim();
  }

  function decodeHtml(text) {
    var el = document.createElement('textarea');
    el.innerHTML = text;
    return el.value;
  }

  // ==================== Extract article ====================
  function extractWechatArticle() {
    var title = document.querySelector('.rich_media_title');
    title = title ? title.textContent.trim() : (document.querySelector('meta[property="og:title"]') ? document.querySelector('meta[property="og:title"]').getAttribute('content') : document.title.replace(/^(.+?)(?:\s*[-–|]\s*.*)?$/, '$1').trim());

    var author = document.querySelector('#js_name, .rich_media_meta_nickname, .profile_nickname');
    author = author ? author.textContent.trim() : (document.querySelector('meta[name="author"]') ? document.querySelector('meta[name="author"]').getAttribute('content') : '');

    var desc = document.querySelector('#js_description');
    desc = desc ? desc.textContent.trim() : (document.querySelector('meta[property="og:description"]') ? document.querySelector('meta[property="og:description"]').getAttribute('content') : '');

    var ogImage = document.querySelector('meta[property="og:image"]');
    var coverImageUrl = ogImage ? ogImage.getAttribute('content') : '';

    var contentEl = document.querySelector('#js_content, .rich_media_content');
    var content = '';
    if (contentEl) {
      var clone = contentEl.cloneNode(true);
      clone.querySelectorAll('script, style').forEach(function(el) { el.remove(); });
      content = htmlToMarkdown(clone.innerHTML);
    } else {
      console.warn('[WeChat] #js_content not found, trying fallback');
      var article = document.querySelector('article, .article_content, .rich_media');
      if (article) content = htmlToMarkdown(article.innerHTML);
    }

    return { title: title, author: author, summary: desc || title, cover_image_url: coverImageUrl, content: content, source_url: window.location.href };
  }

  // ==================== Floating button ====================
  function createFloatingButton() {
    console.log('[WeChat] Creating button');
    var el = document.createElement('div');
    el.id = 'wc-collector-btn';
    el.textContent = '[+]';
    Object.assign(el.style, {
      position: 'fixed', bottom: '40px', right: '40px', zIndex: '999999',
      background: '#07C160', color: 'white', border: 'none', borderRadius: '50%',
      width: '56px', height: '56px', cursor: 'pointer',
      boxShadow: '0 4px 12px rgba(0,0,0,0.3)',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      fontSize: '22px', fontWeight: 'bold',
      fontFamily: 'Arial, sans-serif', lineHeight: '56px',
    });

    el.addEventListener('mouseenter', function() { el.style.transform = 'scale(1.1)'; });
    el.addEventListener('mouseleave', function() { el.style.transform = 'scale(1)'; });

    el.addEventListener('click', async function() {
      el.style.transform = 'scale(0.9)';
      el.style.background = '#999';
      el.textContent = '...';

      try {
        var article = extractWechatArticle();
        console.log('[WeChat] Extracted:', article);

        if (!article.content) {
          alert('No article content found');
          el.style.background = '#07C160'; el.textContent = '[+]'; el.style.transform = 'scale(1)';
          return;
        }

        // Read config from chrome.storage directly
        var configResult = await chrome.storage.local.get('promptCollectorConfig');
        var cfg = configResult.promptCollectorConfig || {};
        var baseUrl = (cfg.apiBaseUrl || '').replace(/\/$/, '');
        var token = cfg.apiToken || '';
        var userId = cfg.userId || '';

        if (!baseUrl) {
          alert('Please configure API Base URL in extension settings first');
          el.style.background = '#07C160'; el.textContent = '[+]'; el.style.transform = 'scale(1)';
          return;
        }

        var apiBase = baseUrl + (baseUrl.indexOf('/api') >= 0 ? '' : '/api');
        var headers = { 'Content-Type': 'application/json' };
        if (token) headers['Authorization'] = token.indexOf('Bearer ') === 0 ? token : 'Bearer ' + token;
        if (userId) headers['New-API-User'] = userId;

        var resp = await fetch(apiBase + '/admin/articles', {
          method: 'POST',
          headers: headers,
          body: JSON.stringify({
            title: article.title,
            author: article.author,
            content: article.content,
            summary: article.summary,
            cover_image_url: article.cover_image_url,
            tags: JSON.stringify(['WeChat']),
            status: 1,
            is_featured: false,
          }),
        });

        var json = await resp.json();
        if (resp.ok && json.success) {
          el.textContent = '[OK]';
          setTimeout(function() { el.textContent = '[+]'; el.style.transform = 'scale(1)'; }, 2000);
        } else {
          alert('Submit failed: ' + (json.message || ('HTTP ' + resp.status)));
          el.textContent = '[+]'; el.style.transform = 'scale(1)';
        }
      } catch (err) {
        console.error('[WeChat] Error:', err);
        var msg = err.message;
        if (msg.indexOf('Failed to fetch') >= 0 || msg.indexOf('NetworkError') >= 0) {
          msg = 'Network error, please check API URL config';
        } else if (msg.indexOf('storage') >= 0) {
          msg = 'Extension storage read failed, please reopen extension settings';
        }
        alert('Failed: ' + msg);
        el.textContent = '[+]'; el.style.transform = 'scale(1)';
      }
    });

    document.body.appendChild(el);
  }

  // ==================== Wait for page ready ====================
  function waitForContent() {
    function check() {
      var found = document.querySelector('#js_content, .rich_media_content, article');
      if (found && found.innerHTML.trim().length > 50) {
        console.log('[WeChat] Content ready');
        createFloatingButton();
        return true;
      }
      return false;
    }

    if (check()) return;

    var attempts = 0;
    var observer = new MutationObserver(function() {
      attempts++;
      if (check() || attempts > 30) observer.disconnect();
    });
    if (document.body) observer.observe(document.body, { childList: true, subtree: true });

    setTimeout(function() {
      observer.disconnect();
      if (!document.getElementById('wc-collector-btn')) {
        if (document.body.innerHTML.length > 500) createFloatingButton();
      }
    }, 15000);
  }

  // ==================== Start ====================
  var isArticlePage = window.location.pathname.indexOf('/s/') === 0 || window.location.search.indexOf('__biz=') >= 0;
  console.log('[WeChat] Path:', window.location.pathname, 'isArticle:', isArticlePage);

  if (isArticlePage) {
    if (document.readyState === 'complete') {
      waitForContent();
    } else {
      window.addEventListener('load', function() { waitForContent(); });
      if (document.readyState === 'interactive') waitForContent();
    }
  }
})();
