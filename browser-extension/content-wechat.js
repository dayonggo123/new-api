// content-wechat.js — 微信公众号文章采集器
// 在公众号文章页面显示浮动按钮，一键采集到文章管理
(function () {
  'use strict';

  console.log('[微信采集] 脚本已注入');

  // ==================== HTML → Markdown 转换 ====================
  function htmlToMarkdown(html) {
    let md = html;
    md = md.replace(/<script[\s\S]*?<\/script>/gi, '');
    md = md.replace(/<style[\s\S]*?<\/style>/gi, '');
    md = md.replace(/<!--[\s\S]*?-->/g, '');

    md = md.replace(/<pre><code[^>]*>([\s\S]*?)<\/code><\/pre>/gi, (_, code) => {
      return '\n```\n' + decodeHtml(code.trim()) + '\n```\n\n';
    });
    md = md.replace(/<img[^>]+src=["']([^"']+)["'][^>]*/gi, (_, src) => {
      const altMatch = _.match(/alt=["']([^"']*)["']/i);
      return `\n![${altMatch ? altMatch[1] : 'image'}](${src.split('?')[0]})\n`;
    });
    md = md.replace(/<video[^>]+src=["']([^"']+)["'][^>]*>/gi, (_, src) => `\n<video controls src="${src}"></video>\n`);
    md = md.replace(/<iframe[^>]+src=["']([^"']+)["'][^>]*>/gi, (_, src) => `\n<iframe src="${src}"></iframe>\n`);
    md = md.replace(/<a[^>]+href=["']([^"']+)["'][^>]*>([\s\S]*?)<\/a>/gi, (_, href, text) => {
      const t = text.replace(/<[^>]+>/g, '').trim();
      return t ? `[${t}](${href})` : href;
    });
    md = md.replace(/<h1[^>]*>([\s\S]*?)<\/h1>/gi, (_, t) => `\n# ${stripTags(t)}\n`);
    md = md.replace(/<h2[^>]*>([\s\S]*?)<\/h2>/gi, (_, t) => `\n## ${stripTags(t)}\n`);
    md = md.replace(/<h3[^>]*>([\s\S]*?)<\/h3>/gi, (_, t) => `\n### ${stripTags(t)}\n`);
    md = md.replace(/<h4[^>]*>([\s\S]*?)<\/h4>/gi, (_, t) => `\n#### ${stripTags(t)}\n`);
    md = md.replace(/<strong>([\s\S]*?)<\/strong>/gi, '**$1**');
    md = md.replace(/<b>([\s\S]*?)<\/b>/gi, '**$1**');
    md = md.replace(/<em>([\s\S]*?)<\/em>/gi, '*$1*');
    md = md.replace(/<i>([\s\S]*?)<\/i>/gi, '*$1*');
    md = md.replace(/<blockquote[^>]*>([\s\S]*?)<\/blockquote>/gi, (_, q) => {
      return q.replace(/<[^>]+>/g, '').trim().split('\n').map(l => `> ${l}`).join('\n') + '\n';
    });
    md = md.replace(/<ul[^>]*>([\s\S]*?)<\/ul>/gi, (_, list) => {
      return list.replace(/<li[^>]*>([\s\S]*?)<\/li>/gi, (__, item) => `- ${stripTags(item)}\n`) + '\n';
    });
    md = md.replace(/<ol[^>]*>([\s\S]*?)<\/ol>/gi, (_, list) => {
      let i = 0;
      return list.replace(/<li[^>]*>([\s\S]*?)<\/li>/gi, (__, item) => { i++; return `${i}. ${stripTags(item)}\n`; }) + '\n';
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
    const el = document.createElement('textarea');
    el.innerHTML = text;
    return el.value;
  }

  // ==================== 提取公众号文章 ====================
  function extractWechatArticle() {
    const title = document.querySelector('.rich_media_title')?.textContent?.trim()
      || document.querySelector('meta[property="og:title"]')?.getAttribute('content')
      || document.title.replace(/^(.+?)(?:\s*[-–|]\s*.*)?$/, '$1').trim();

    const author = document.querySelector('#js_name, .rich_media_meta_nickname, .profile_nickname')?.textContent?.trim()
      || document.querySelector('meta[name="author"]')?.getAttribute('content')
      || '';

    const desc = document.querySelector('#js_description')?.textContent?.trim()
      || document.querySelector('meta[property="og:description"]')?.getAttribute('content')
      || '';

    const ogImage = document.querySelector('meta[property="og:image"]');
    const coverImageUrl = ogImage?.getAttribute('content') || '';

    // 正文内容
    const contentEl = document.querySelector('#js_content, .rich_media_content');
    let content = '';
    if (contentEl) {
      const clone = contentEl.cloneNode(true);
      clone.querySelectorAll('script, style').forEach(el => el.remove());
      content = htmlToMarkdown(clone.innerHTML);
    } else {
      console.warn('[微信采集] 未找到 #js_content，尝试其他选择器');
      // fallback: 查找文章主体区域
      const article = document.querySelector('article, .article_content, .rich_media');
      if (article) {
        content = htmlToMarkdown(article.innerHTML);
      }
    }

    const timeEl = document.querySelector('#js_publish_time, .rich_media_meta_time, em#publish_time');
    const publishTime = timeEl?.textContent?.trim() || '';

    return { title, author, summary: desc || title, cover_image_url: coverImageUrl, content, publish_time: publishTime, source_url: window.location.href };
  }

  // ==================== 浮窗按钮 ====================
  function createFloatingButton() {
    console.log('[微信采集] 创建浮窗按钮');
    const el = document.createElement('div');
    el.id = 'wc-collector-btn';
    el.innerHTML = '📥';
    Object.assign(el.style, {
      position: 'fixed', bottom: '40px', right: '40px', zIndex: '999999',
      background: '#07C160', color: 'white', border: 'none', borderRadius: '50%',
      width: '56px', height: '56px', cursor: 'pointer',
      boxShadow: '0 4px 12px rgba(0,0,0,0.3)',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      fontSize: '26px', transition: 'transform 0.2s, background 0.2s',
      fontFamily: 'Arial, sans-serif',
    });

    el.addEventListener('mouseenter', () => el.style.transform = 'scale(1.1)');
    el.addEventListener('mouseleave', () => el.style.transform = 'scale(1)');

    el.addEventListener('click', async () => {
      el.style.transform = 'scale(0.9)';
      el.style.background = '#999';
      el.textContent = '⏳';

      try {
        const article = extractWechatArticle();
        console.log('[微信采集] 提取结果:', article);

        if (!article.content) {
          alert('未找到文章正文内容，请确认当前是公众号文章详情页');
          el.style.background = '#07C160'; el.textContent = '📥'; el.style.transform = 'scale(1)';
          return;
        }

        // 构建文章内容预览
        const preview = article.content.length > 100 ? article.content.slice(0, 100) + '...' : article.content;
        const confirmMsg = `确认采集该文章？\n\n标题: ${article.title}\n作者: ${article.author}\n字数: ${article.content.length}\n\n正文预览:\n${preview}`;
        if (!confirm(confirmMsg)) {
          el.style.background = '#07C160'; el.textContent = '📥'; el.style.transform = 'scale(1)';
          return;
        }

        const resp = await chrome.runtime.sendMessage({
          action: 'apiRequest',
          method: 'POST',
          path: '/admin/articles',
          body: {
            title: article.title,
            author: article.author,
            content: article.content,
            summary: article.summary,
            cover_image_url: article.cover_image_url,
            tags: JSON.stringify(['微信公众号']),
            status: 1,
            is_featured: false,
          },
        });

        if (resp?.success) {
          el.textContent = '✅';
          el.style.background = '#07C160';
          setTimeout(() => { el.textContent = '📥'; el.style.transform = 'scale(1)'; }, 2000);
        } else {
          alert('采集失败: ' + (resp?.message || '请检查扩展配置中的 API 地址和 Token'));
          el.style.background = '#07C160'; el.textContent = '📥'; el.style.transform = 'scale(1)';
        }
      } catch (err) {
        console.error('[微信采集] 错误:', err);
        alert('采集失败: ' + err.message);
        el.style.background = '#07C160'; el.textContent = '📥'; el.style.transform = 'scale(1)';
      }
    });

    document.body.appendChild(el);
  }

  // ==================== 等待页面就绪 ====================
  function waitForContent(retries = 30) {
    const check = () => {
      const found = document.querySelector('#js_content, .rich_media_content, article');
      if (found && found.innerHTML.trim().length > 50) {
        console.log('[微信采集] 内容已就绪');
        createFloatingButton();
        return true;
      }
      return false;
    };

    // 立即检查
    if (check()) return;

    // MutationObserver 监听 DOM
    let attempts = 0;
    const observer = new MutationObserver(() => {
      attempts++;
      if (check() || attempts > retries) observer.disconnect();
    });
    observer.observe(document.body, { childList: true, subtree: true });

    // 超时 fallback
    setTimeout(() => {
      observer.disconnect();
      if (!document.getElementById('wc-collector-btn')) {
        // 超时后尝试强制创建
        console.log('[微信采集] 超时，尝试强制创建按钮');
        const bodyText = document.body.innerHTML;
        if (bodyText.length > 500) {
          createFloatingButton();
        } else {
          console.warn('[微信采集] 页面内容仍未就绪，跳过');
        }
      }
    }, 15000);
  }

  // ==================== 启动 ====================
  // 只在文章详情页启动
  const isArticlePage = window.location.pathname.startsWith('/s/') || window.location.search.includes('__biz=');
  console.log('[微信采集] 页面路径:', window.location.pathname, '是文章页:', isArticlePage);

  if (isArticlePage) {
    if (document.readyState === 'complete') {
      waitForContent();
    } else {
      window.addEventListener('load', () => waitForContent());
      // 如果 load 事件已过，立即尝试
      if (document.readyState === 'interactive') waitForContent();
    }
  } else {
    console.log('[微信采集] 非文章详情页，跳过');
  }
})();
