// content-list.js - 列表页脚本（适配 opennana.com 弹窗模式）
(function () {
  'use strict';

  const BUTTON_CLASS = 'prompt-collector-btn';
  const INJECTED_ATTR = 'data-pc-injected';

  // 等待元素出现
  function waitForElement(selector, timeout = 5000, parent = document) {
    return new Promise((resolve) => {
      const start = Date.now();
      const check = () => {
        const el = parent.querySelector(selector);
        if (el && el.offsetParent !== null) {
          resolve(el);
          return;
        }
        if (Date.now() - start > timeout) {
          resolve(null);
          return;
        }
        setTimeout(check, 200);
      };
      check();
    });
  }

  // 从弹窗提取内容
  async function extractFromModal() {
    // opennana.com 的弹窗通常是一个固定定位的 overlay
    const modalSelectors = [
      '[role="dialog"]',
      '[class*="modal"]',
      '[class*="overlay"]:not([class*="hidden"])',
      '[class*="dialog"]',
      'div.fixed.inset-0'
    ];

    let modal = null;
    for (const sel of modalSelectors) {
      modal = document.querySelector(sel);
      if (modal && modal.offsetParent !== null) break;
    }

    if (!modal) {
      console.log('[Prompt Collector] 未找到弹窗');
      return null;
    }

    console.log('[Prompt Collector] 找到弹窗，开始提取...');

    // 提取标题（弹窗里的 h2/h3 或第一个大标题）
    let title = '';
    const titleSelectors = ['h2', 'h3', '[class*="title"]', '[class*="heading"]'];
    for (const sel of titleSelectors) {
      const el = modal.querySelector(sel);
      if (el && el.textContent.trim()) {
        title = el.textContent.trim();
        break;
      }
    }

    // 提取 prompt 正文
    let content = '';
    const modalText = modal.innerText || '';

    // 策略1：查找包含 "Prompt" 标签附近的文本块
    const allBlocks = modal.querySelectorAll('pre, code, textarea, p, div, span');
    for (const el of allBlocks) {
      const text = el.innerText?.trim() || '';
      const prev = el.previousElementSibling;
      const prevText = prev ? (prev.innerText?.trim().toLowerCase() || '') : '';
      const parentPrev = el.parentElement?.previousElementSibling;
      const parentPrevText = parentPrev ? (parentPrev.innerText?.trim().toLowerCase() || '') : '';

      if (
        (prevText.includes('prompt') || prevText.includes('提示词') || prevText.includes('咒语') ||
         parentPrevText.includes('prompt') || parentPrevText.includes('提示词')) &&
        text.length > 30 && text.length < 5000
      ) {
        content = text;
        break;
      }
    }

    // 策略2：如果策略1没找到，找弹窗里最长的文本块
    if (!content) {
      let longest = '';
      for (const el of allBlocks) {
        const text = el.innerText?.trim() || '';
        if (text.length > longest.length && text.length > 100 && text.length < 5000) {
          // 排除导航、按钮文字等
          if (!el.closest('button, nav, header, [role="tablist"]')) {
            longest = text;
          }
        }
      }
      if (longest.length > 100) content = longest;
    }

    // 策略3：从 innerText 中按关键词截取
    if (!content) {
      const lines = modalText.split('\n').map(l => l.trim()).filter(l => l.length > 30);
      if (lines.length > 0) {
        content = lines[0];
      }
    }

    // 提取模型
    let model = '';
    const modelPatterns = [
      /模型[：:]\s*([^\n]+)/i,
      /模型\s*([^\n]+)/i,
      /Model[：:]\s*([^\n]+)/i
    ];
    for (const p of modelPatterns) {
      const m = modalText.match(p);
      if (m) { model = m[1].trim(); break; }
    }
    if (!model) {
      const knownModels = ['GPT Image 2', 'Nano Banana Pro', 'Nano Banana 2', 'Nano Banana', 'Seedance 2.0', 'Seedance', 'Midjourney', 'DALL-E', 'Stable Diffusion', 'Flux', 'Kling', 'Runway', 'Pika', 'Sora'];
      for (const km of knownModels) {
        if (modalText.includes(km)) { model = km; break; }
      }
    }

    // 提取封面图
    let coverImageUrl = '';
    const img = modal.querySelector('img[src*="opennana.com"], img[src*="img.opennana"]');
    if (img) coverImageUrl = img.src;

    // 提取标签
    const tags = [];
    modal.querySelectorAll('[class*="tag"], [class*="label"], [class*="chip"]').forEach(el => {
      const text = el.innerText?.trim() || '';
      if (text && text.length < 20 && !tags.includes(text)) tags.push(text);
    });

    const data = {
      title,
      content,
      description: title,
      cover_image_url: coverImageUrl,
      model,
      media_type: window.location.href.includes('video') ? 'video' : 'image',
      tags: JSON.stringify(tags.slice(0, 10)),
      source_url: window.location.href,
      status: 1
    };

    console.log('[Prompt Collector] 提取结果:', data);
    return data;
  }

  // 关闭弹窗
  function closeModal() {
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }));
    // 备选：点击关闭按钮
    const closeBtns = document.querySelectorAll('[class*="close"], button svg[class*="x"], [aria-label*="close"], [aria-label*="关闭"]');
    closeBtns.forEach(btn => {
      if (btn.offsetParent !== null) btn.click();
    });
  }

  // 创建采集按钮
  function createCollectButton(card) {
    const btn = document.createElement('button');
    btn.className = BUTTON_CLASS;
    btn.textContent = '采集';
    btn.title = '采集此提示词到 new-api';
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
      opacity: '0.85',
      pointerEvents: 'auto',
      lineHeight: '1.5',
      whiteSpace: 'nowrap'
    });

    btn.addEventListener('mouseenter', () => {
      btn.style.opacity = '1';
      btn.style.transform = 'scale(1.05)';
    });
    btn.addEventListener('mouseleave', () => {
      btn.style.opacity = '0.85';
      btn.style.transform = 'scale(1)';
    });

    btn.addEventListener('click', async (e) => {
      e.preventDefault();
      e.stopPropagation();
      btn.textContent = '采集中...';
      btn.style.pointerEvents = 'none';

      try {
        // 点击卡片打开弹窗
        card.click();

        // 等待弹窗渲染（给一点动画时间）
        await new Promise(r => setTimeout(r, 800));

        // 提取内容
        const data = await extractFromModal();

        if (!data || !data.content) {
          // 如果没找到内容，再等一下再试一次
          await new Promise(r => setTimeout(r, 500));
          const retryData = await extractFromModal();
          if (!retryData || !retryData.content) {
            alert('未能提取到 prompt 内容，请手动复制后粘贴到插件弹窗');
            btn.textContent = '采集';
            btn.style.pointerEvents = 'auto';
            return;
          }
          Object.assign(data, retryData);
        }

        // 发送给 background
        await chrome.runtime.sendMessage({
          action: 'saveExtractedData',
          data
        });

        // 关闭弹窗
        closeModal();

        btn.textContent = '✓ 已采集';
        setTimeout(() => {
          btn.textContent = '采集';
          btn.style.pointerEvents = 'auto';
        }, 2000);

        // 打开插件弹窗提示用户
        chrome.action.openPopup?.();

      } catch (err) {
        console.error('[Prompt Collector] 采集失败:', err);
        alert('采集失败: ' + err.message);
        btn.textContent = '采集';
        btn.style.pointerEvents = 'auto';
      }
    });

    return btn;
  }

  // 为卡片注入按钮
  function injectButton(card) {
    if (card.hasAttribute(INJECTED_ATTR)) return;
    card.setAttribute(INJECTED_ATTR, 'true');

    const computed = window.getComputedStyle(card);
    if (computed.position === 'static') {
      card.style.position = 'relative';
    }

    const btn = createCollectButton(card);
    card.appendChild(btn);
  }

  // 扫描所有卡片
  function scanCards() {
    // opennana.com 的实际卡片 class: prompt-card
    const cards = document.querySelectorAll('.prompt-card:not([' + INJECTED_ATTR + '])');
    cards.forEach(injectButton);
  }

  // 初始化
  function init() {
    scanCards();

    // 监听 DOM 变化（SPA 加载更多卡片）
    const observer = new MutationObserver((mutations) => {
      let shouldScan = false;
      for (const m of mutations) {
        if (m.type === 'childList') {
          for (const node of m.addedNodes) {
            if (node.nodeType === Node.ELEMENT_NODE) {
              if (node.classList?.contains('prompt-card') || node.querySelector?.('.prompt-card')) {
                shouldScan = true;
                break;
              }
            }
          }
        }
        if (shouldScan) break;
      }
      if (shouldScan) scanCards();
    });

    observer.observe(document.body, { childList: true, subtree: true });
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
