// content-wechat.js — 微信公众号文章采集器
// 在公众号文章页面显示浮动按钮，一键采集到文章管理

(function () {
  'use strict';

  // ==================== HTML → Markdown 转换 ====================
  function htmlToMarkdown(html) {
    let md = html;

    // 移除 script、style、注释
    md = md.replace(/<script[\s\S]*?<\/script>/gi, '');
    md = md.replace(/<style[\s\S]*?<\/style>/gi, '');
    md = md.replace(/<!--[\s\S]*?-->/g, '');

    // 代码块
    md = md.replace(/<pre><code[^>]*>([\s\S]*?)<\/code><\/pre>/gi, (_, code) => {
      return '\n```\n' + decodeHTMLEntities(code.trim()) + '\n```\n\n';
    });

    // 图片
    md = md.replace(/<img[^>]+src=["']([^"']+)["'][^>]*>/gi, (_, src) => {
      const alt = _.match(/alt=["']([^"']*)["']/i);
      return `![${alt ? alt[1] : ''}](${src})\n`;
    });

    // 视频
    md = md.replace(/<video[^>]+src=["']([^"']+)["'][^>]*>/gi, (_, src) => {
      return `\n<video controls src="${src}"></video>\n`;
    });
    md = md.replace(/<iframe[^>]+src=["']([^"']+)["'][^>]*>/gi, (_, src) => {
      return `\n<iframe src="${src}"></iframe>\n`;
    });

    // 链接
    md = md.replace(/<a[^>]+href=["']([^"']+)["'][^>]*>([\s\S]*?)<\/a>/gi, (_, href, text) => {
      text = text.replace(/<[^>]+>/g, '').trim();
      if (!text) return href;
      return `[${text}](${href})`;
    });

    // 标题
    md = md.replace(/<h1[^>]*>([\s\S]*?)<\/h1>/gi, (_, t) => `\n# ${t.replace(/<[^>]+>/g, '').trim()}\n`);
    md = md.replace(/<h2[^>]*>([\s\S]*?)<\/h2>/gi, (_, t) => `\n## ${t.replace(/<[^>]+>/g, '').trim()}\n`);
    md = md.replace(/<h3[^>]*>([\s\S]*?)<\/h3>/gi, (_, t) => `\n### ${t.replace(/<[^>]+>/g, '').trim()}\n`);
    md = md.replace(/<h4[^>]*>([\s\S]*?)<\/h4>/gi, (_, t) => `\n#### ${t.replace(/<[^>]+>/g, '').trim()}\n`);

    // 加粗/斜体
    md = md.replace(/<strong>([\s\S]*?)<\/strong>/gi, '**$1**');
    md = md.replace(/<b>([\s\S]*?)<\/b>/gi, '**$1**');
    md = md.replace(/<em>([\s\S]*?)<\/em>/gi, '*$1*');
    md = md.replace(/<i>([\s\S]*?)<\/i>/gi, '*$1*');

    // 引用
    md = md.replace(/<blockquote[^>]*>([\s\S]*?)<\/blockquote>/gi, (_, q) => {
      const lines = q.replace(/<[^>]+>/g, '').trim().split('\n');
      return lines.map(l => `> ${l}`).join('\n') + '\n';
    });

    // 无序列表
    md = md.replace(/<ul[^>]*>([\s\S]*?)<\/ul>/gi, (_, list) => {
      return list.replace(/<li[^>]*>([\s\S]*?)<\/li>/gi, (__, item) => {
        return `- ${item.replace(/<[^>]+>/g, '').trim()}\n`;
      }) + '\n';
    });

    // 有序列表
    md = md.replace(/<ol[^>]*>([\s\S]*?)<\/ol>/gi, (_, list) => {
      let idx = 0;
      return list.replace(/<li[^>]*>([\s\S]*?)<\/li>/gi, (__, item) => {
        idx++;
        return `${idx}. ${item.replace(/<[^>]+>/g, '').trim()}\n`;
      }) + '\n';
    });

    // 清理剩余 HTML 标签
    md = md.replace(/<br\s*\/?>/gi, '\n');
    md = md.replace(/<\/p>/gi, '\n');
    md = md.replace(/<[^>]+>/g, '');

    // 合并多余空行
    md = md.replace(/\n{4,}/g, '\n\n');
    md = md.replace(/&nbsp;/g, ' ');
    md = md.replace(/&amp;/g, '&');
    md = md.replace(/&lt;/g, '<');
    md = md.replace(/&gt;/g, '>');
    md = md.replace(/&quot;/g, '"');

    return md.trim();
  }

  function decodeHTMLEntities(text) {
    const textarea = document.createElement('textarea');
    textarea.innerHTML = text;
    return textarea.value;
  }

  // ==================== 提取公众号文章 ====================
  function extractWechatArticle() {
    // 标题
    const titleEl = document.querySelector('.rich_media_title');
    const title = titleEl ? titleEl.textContent.trim() : document.title.trim();

    // 作者/公众号名
    const authorEl = document.querySelector('#js_name, .rich_media_meta_nickname');
    const author = authorEl ? authorEl.textContent.trim() : '';

    // 摘要
    const descEl = document.querySelector('#js_description, .rich_media_meta_desc');
    const summary = descEl ? descEl.textContent.trim() : '';

    // 封面图
    const ogImage = document.querySelector('meta[property="og:image"]');
    const coverImageUrl = ogImage ? ogImage.getAttribute('content') : '';

    // 正文 HTML
    const contentEl = document.querySelector('#js_content, .rich_media_content');
    let content = '';
    if (contentEl) {
      // 克隆节点避免修改原页面
      const clone = contentEl.cloneNode(true);
      // 移除内部的隐藏元素、脚本等
      clone.querySelectorAll('script, style, [style*="display:none"], [style*="display: none"]').forEach(el => el.remove());
      content = htmlToMarkdown(clone.innerHTML);
    }

    // 发布时间
    const timeEl = document.querySelector('#js_publish_time, .rich_media_meta_time');
    const publishTime = timeEl ? timeEl.textContent.trim() : '';

    return {
      title,
      author,
      summary: summary || title,
      cover_image_url: coverImageUrl,
      content,
      publish_time: publishTime,
      source_url: window.location.href,
      source: 'wechat',
      tags: JSON.stringify(['微信公众号']),
    };
  }

  // ==================== 浮窗按钮 ====================
  function createFloatingButton() {
    const btn = document.createElement('div');
    btn.innerHTML = `
      <div style="
        position: fixed; bottom: 40px; right: 40px; z-index: 99999;
        background: #07C160; color: white; border: none; border-radius: 50%;
        width: 56px; height: 56px; cursor: pointer; box-shadow: 0 4px 12px rgba(0,0,0,0.3);
        display: flex; align-items: center; justify-content: center;
        font-size: 24px; transition: transform 0.2s;
      " title="采集到文章管理">
        <span style="transform: scale(1.2)">📥</span>
      </div>
    `;
    const el = btn.firstElementChild;

    el.onmouseenter = () => { el.style.transform = 'scale(1.1)'; };
    el.onmouseleave = () => { el.style.transform = 'scale(1)'; };

    el.onclick = async () => {
      el.style.transform = 'scale(0.9)';
      el.innerHTML = '<span style="font-size:14px">⏳</span>';

      try {
        const article = extractWechatArticle();
        if (!article.content) {
          alert('未找到文章内容，请确认当前页面是公众号文章');
          el.innerHTML = '<span>📥</span>';
          el.style.transform = 'scale(1)';
          return;
        }

        // 通过 background 提交到文章管理 API
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
            tags: article.tags,
            status: 1,
            is_featured: false,
            register_channel: 'wechat_article',
          },
        });

        if (resp && resp.success) {
          el.innerHTML = '<span>✅</span>';
          setTimeout(() => {
            el.innerHTML = '<span>📥</span>';
            el.style.transform = 'scale(1)';
          }, 2000);
        } else {
          alert('采集失败: ' + (resp?.message || '未知错误'));
          el.innerHTML = '<span>📥</span>';
          el.style.transform = 'scale(1)';
        }
      } catch (err) {
        alert('采集失败: ' + err.message);
        el.innerHTML = '<span>📥</span>';
        el.style.transform = 'scale(1)';
      }
    };

    document.body.appendChild(el);
  }

  // ==================== 启动 ====================
  // 确保是文章详情页（包含 #js_content）
  if (document.querySelector('#js_content, .rich_media_content')) {
    // 等待页面完全加载
    if (document.readyState === 'complete') {
      createFloatingButton();
    } else {
      window.addEventListener('load', createFloatingButton);
    }
  }
})();
