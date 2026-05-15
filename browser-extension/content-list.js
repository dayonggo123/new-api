// content-list.js - 列表页脚本
// 在 opennana.com/awesome-prompt-gallery 页面注入"采集到库"按钮

(function () {
  'use strict';

  const BUTTON_CLASS = 'prompt-collector-btn';
  const INJECTED_ATTR = 'data-pc-injected';

  // 创建采集按钮
  function createCollectButton(card, title) {
    const btn = document.createElement('button');
    btn.className = BUTTON_CLASS;
    btn.textContent = '采集到库';
    btn.title = '采集此提示词到 new-api 提示词库';
    Object.assign(btn.style, {
      position: 'absolute',
      top: '8px',
      right: '8px',
      zIndex: '9999',
      padding: '4px 10px',
      fontSize: '12px',
      fontWeight: '600',
      color: '#fff',
      background: 'linear-gradient(135deg, #06b6d4, #3b82f6)',
      border: 'none',
      borderRadius: '6px',
      cursor: 'pointer',
      boxShadow: '0 2px 8px rgba(6,182,212,0.35)',
      transition: 'all 0.2s ease',
      opacity: '0.9',
      pointerEvents: 'auto',
      lineHeight: '1.5',
      whiteSpace: 'nowrap'
    });

    btn.addEventListener('mouseenter', () => {
      btn.style.opacity = '1';
      btn.style.transform = 'scale(1.05)';
    });
    btn.addEventListener('mouseleave', () => {
      btn.style.opacity = '0.9';
      btn.style.transform = 'scale(1)';
    });

    btn.addEventListener('click', async (e) => {
      e.preventDefault();
      e.stopPropagation();
      btn.textContent = '采集中...';
      btn.style.opacity = '0.6';
      btn.style.pointerEvents = 'none';

      try {
        // 获取卡片的点击链接
        const detailUrl = findDetailUrl(card);
        if (!detailUrl) {
          alert('无法获取详情页链接，请直接访问详情页采集');
          resetBtn();
          return;
        }

        // 发送消息给 background 打开详情页
        const resp = await chrome.runtime.sendMessage({
          action: 'openDetailTab',
          url: detailUrl,
          title: title
        });

        if (resp.success) {
          btn.textContent = '已打开详情页';
          setTimeout(resetBtn, 2000);
        } else {
          throw new Error(resp.message);
        }
      } catch (err) {
        console.error('采集失败:', err);
        btn.textContent = '采集失败';
        setTimeout(resetBtn, 2000);
      }

      function resetBtn() {
        btn.textContent = '采集到库';
        btn.style.opacity = '0.9';
        btn.style.pointerEvents = 'auto';
      }
    });

    return btn;
  }

  // 查找卡片的详情页链接
  function findDetailUrl(card) {
    // 策略1：查找卡片内的 a 标签
    const link = card.querySelector('a[href]');
    if (link) {
      const href = link.getAttribute('href');
      if (href && href.startsWith('/')) {
        return 'https://opennana.com' + href;
      }
      if (href && href.startsWith('http')) {
        return href;
      }
    }

    // 策略2：卡片本身可点击，查找最近的 a 标签
    const parentLink = card.closest('a[href]');
    if (parentLink) {
      const href = parentLink.getAttribute('href');
      if (href && href.startsWith('/')) {
        return 'https://opennana.com' + href;
      }
      if (href && href.startsWith('http')) {
        return href;
      }
    }

    // 策略3：查找 onclick 或 data 属性
    const onclick = card.getAttribute('onclick');
    if (onclick) {
      const match = onclick.match(/['"](\/[^'"]*)['"]/);
      if (match) return 'https://opennana.com' + match[1];
    }

    // 策略4：查找 data-href
    const dataHref = card.getAttribute('data-href');
    if (dataHref) {
      if (dataHref.startsWith('/')) return 'https://opennana.com' + dataHref;
      return dataHref;
    }

    return null;
  }

  // 为卡片注入按钮
  function injectButton(card) {
    if (card.hasAttribute(INJECTED_ATTR)) return;
    card.setAttribute(INJECTED_ATTR, 'true');

    // 确保卡片有相对定位，以便按钮绝对定位
    const computed = window.getComputedStyle(card);
    if (computed.position === 'static') {
      card.style.position = 'relative';
    }

    const titleEl = card.querySelector('.card-title');
    const title = titleEl ? titleEl.textContent.trim() : '';

    const btn = createCollectButton(card, title);
    card.appendChild(btn);
  }

  // 扫描并注入按钮
  function scanAndInject() {
    // opennana.com 的卡片选择器
    const selectors = [
      '[class*="card"]',
      'a[href*="awesome-prompt-gallery/"]',
      'article',
      '.group'
    ];

    for (const selector of selectors) {
      document.querySelectorAll(selector).forEach(el => {
        // 只给包含标题的元素注入
        if (el.querySelector('.card-title, h3, h2')) {
          injectButton(el);
        }
      });
    }
  }

  // 初始化
  function init() {
    scanAndInject();

    // 监听 DOM 变化（SPA 加载更多内容）
    const observer = new MutationObserver((mutations) => {
      let shouldScan = false;
      for (const mutation of mutations) {
        if (mutation.type === 'childList' && mutation.addedNodes.length > 0) {
          for (const node of mutation.addedNodes) {
            if (node.nodeType === Node.ELEMENT_NODE) {
              shouldScan = true;
              break;
            }
          }
        }
        if (shouldScan) break;
      }
      if (shouldScan) {
        scanAndInject();
      }
    });

    observer.observe(document.body, {
      childList: true,
      subtree: true
    });
  }

  // 页面加载完成后初始化
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
