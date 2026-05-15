// content-list.js - 列表页脚本（适配 opennana.com 弹窗模式）
(function () {
  'use strict';

  const BUTTON_CLASS = 'prompt-collector-btn';
  const INJECTED_ATTR = 'data-pc-injected';
  const COLLECTED_ATTR = 'data-pc-collected';

  let config = {};
  let collectedTitles = new Set();

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

  // 辅助：按行检测语言并分离中英文
  function splitByLanguage(text) {
    const lines = text.split('\n').map(l => l.trim()).filter(l => l);
    const zh = [], en = [];
    for (const line of lines) {
      const chineseChars = (line.match(/[\u4e00-\u9fff]/g) || []).length;
      const totalChars = line.replace(/\s/g, '').length;
      if (totalChars > 0 && chineseChars / totalChars > 0.3) {
        zh.push(line);
      } else if (line.length > 10) {
        en.push(line);
      }
    }
    return { zh: zh.join('\n'), en: en.join('\n') };
  }

  // 从文档对象提取内容（支持当前页面弹窗或 fetch 的详情页 HTML）
  async function extractFromDocument(doc = document, sourceUrl = window.location.href, cardForImage = null) {
    // 优先找 animate-modal-in，再找任意含 modal 的元素，最后回退到 body
    let modal = doc.querySelector('[class*="animate-modal-in"]') ||
                doc.querySelector('[class*="animate-modal"]');

    if (!modal) {
      const allModals = Array.from(doc.querySelectorAll('div')).filter(el => {
        const cls = el.className || '';
        return (cls.includes('modal') || cls.includes('dialog'));
      });
      modal = allModals[allModals.length - 1];
    }

    if (!modal) {
      modal = doc.body;
    }

    console.log('[Prompt Collector] 提取文档，modal class:', modal.className);

    const modalText = modal.innerText || '';
    const lines = modalText.split('\n').map(l => l.trim()).filter(l => l);

    // 提取标题（第一行非空文本）
    const title = lines[0] || '';

    // 提取模型
    let model = '';
    const modelMatch = modalText.match(/模型[：:]\s*([^\n]+)/);
    if (modelMatch) model = modelMatch[1].trim();

    // 提取 prompt 内容
    let content = '';
    let contentEn = '';

    // 策略1：找 "复制" 后面的内容（JSON 或纯文本）
    const copyMatch = modalText.match(/复制[\s\n]+([\s\S]*?)(?:中文[\s\n]*去 AI 生图[\s\n]*复制[\s\n]*[\s\S]*?)?(?:更多推荐|$)/);
    if (copyMatch) {
      const candidate = copyMatch[1].trim();
      if (candidate.startsWith('{')) {
        try {
          const jsonObj = JSON.parse(candidate);
          contentEn = jsonObj.content || jsonObj.data || jsonObj.prompt || candidate;
        } catch (e) {
          contentEn = candidate;
        }
      } else {
        const split = splitByLanguage(candidate);
        content = split.zh;
        contentEn = split.en;
      }
    }

    // 策略2：明确分段提取 ENGLISH / 中文
    if (!contentEn) {
      const englishMatch = modalText.match(/ENGLISH[\s\n]*(?:去 AI 生图)?[\s\n]*复制[\s\n]*([\s\S]*?)(?:中文|去 AI 生图|更多推荐|$)/i);
      if (englishMatch) {
        contentEn = englishMatch[1].trim();
      }
    }
    if (!content) {
      const chineseMatch = modalText.match(/中文[\s\n]*(?:去 AI 生图)?[\s\n]*复制[\s\n]*([\s\S]*?)(?:更多推荐|$)/);
      if (chineseMatch) {
        content = chineseMatch[1].trim();
      }
    }

    // 策略3：从 DOM 中找最长的文本段落
    if (!content && !contentEn) {
      let longest = '';
      modal.querySelectorAll('p, div, span, pre, code').forEach(el => {
        const text = (el.innerText || '').trim();
        if (text.length > longest.length && text.length > 100 && text.length < 5000) {
          if (!el.closest('button, [class*="close"], [class*="header"]')) {
            longest = text;
          }
        }
      });
      if (longest.length > 100) {
        const split = splitByLanguage(longest);
        content = split.zh;
        contentEn = split.en;
      }
    }

    // 提取封面图/视频
    let coverImageUrl = '';

    const video = modal.querySelector('video');
    if (video) {
      coverImageUrl = video.src || video.currentSrc || video.querySelector('source')?.src || '';
    }

    if (!coverImageUrl) {
      const imgs = Array.from(modal.querySelectorAll('img'));
      const imgData = imgs.map(img => {
        const src = img.src || img.dataset?.src || img.dataset?.original || img.dataset?.lazySrc || '';
        let bestSrc = src;
        const srcset = img.srcset || img.dataset?.srcset || '';
        if (srcset) {
          const candidates = srcset.split(',').map(s => {
            const [url, w] = s.trim().split(' ');
            return { url: url || s.trim(), width: parseInt(w) || 0 };
          });
          candidates.sort((a, b) => b.width - a.width);
          if (candidates[0]) bestSrc = candidates[0].url;
        }
        return {
          src: bestSrc,
          isOpennana: bestSrc.includes('opennana.com') || bestSrc.includes('img.opennana'),
          width: img.naturalWidth || img.width || 0
        };
      });

      const opennanaImgs = imgData.filter(d => d.isOpennana);
      if (opennanaImgs.length > 0) {
        coverImageUrl = opennanaImgs[0].src;
      } else if (imgData.length > 0) {
        coverImageUrl = imgData[0].src;
      }
    }

    // 策略3：找背景图（仅在当前页面模式下使用 getComputedStyle）
    if (!coverImageUrl && doc === document) {
      const els = modal.querySelectorAll('*');
      for (const el of els) {
        const style = window.getComputedStyle(el);
        const bg = style.backgroundImage;
        if (bg && bg !== 'none') {
          const match = bg.match(/url\(["']?([^"')]+)["']?\)/);
          if (match) {
            coverImageUrl = match[1];
            break;
          }
        }
      }
    }

    // 策略4：从传入的卡片找缩略图
    if (!coverImageUrl && cardForImage) {
      const cardImgs = Array.from(cardForImage.querySelectorAll('img'));
      for (const img of cardImgs) {
        const src = img.src || img.dataset?.src || '';
        if (src) {
          coverImageUrl = src;
          break;
        }
      }
    }

    // 提取标签
    const tags = [];
    const tagMatch = modalText.match(/收藏[\s\n]+([\s\S]*?)(?:示例图片|提示词|赞助)/);
    if (tagMatch) {
      const tagText = tagMatch[1].trim();
      const tagArr = tagText.split(/[\s\n]+/).filter(t => t && t.length >= 2 && t.length < 15 && !t.includes('：') && !t.includes(':'));
      tags.push(...tagArr.slice(0, 10));
    }

    const data = {
      title,
      content,
      content_en: contentEn,
      description: title,
      cover_image_url: coverImageUrl,
      model,
      media_type: sourceUrl.includes('video') ? 'video' : 'image',
      tags: JSON.stringify(tags.slice(0, 10)),
      source_url: sourceUrl,
      category_id: getCategoryId(model),
      status: 1
    };

    console.log('[Prompt Collector] 提取结果:', data);
    return data;
  }

  // 兼容旧调用
  async function extractFromModal() {
    return extractFromDocument(document, window.location.href, currentCard);
  }

  // 临时冻结 history 导航（防止 React 卸载时触发 router.back）
  function freezeHistory(durationMs = 1000) {
    const originalBack = history.back.bind(history);
    const originalGo = history.go.bind(history);
    const originalForward = history.forward.bind(history);
    const nav = window.navigation;
    const originalNavBack = nav && nav.back ? nav.back.bind(nav) : null;

    history.back = () => console.log('[PC] 拦截 history.back');
    history.go = () => console.log('[PC] 拦截 history.go');
    history.forward = () => console.log('[PC] 拦截 history.forward');
    if (originalNavBack) nav.back = () => console.log('[PC] 拦截 navigation.back');

    setTimeout(() => {
      history.back = originalBack;
      history.go = originalGo;
      history.forward = originalForward;
      if (originalNavBack) nav.back = originalNavBack;
    }, durationMs);
  }

  // 关闭弹窗（隐藏 DOM 而非移除，避免触发 React useEffect 清理中的 router.back）
  function closeModal() {
    freezeHistory(1500);
    try {
      document.querySelectorAll('[class*="animate-modal-in"]').forEach(m => {
        m.style.display = 'none';
        const parent = m.parentElement;
        if (parent && parent !== document.body) {
          const style = window.getComputedStyle(parent);
          if (style.position === 'fixed') {
            parent.style.display = 'none';
          }
        }
      });

      document.querySelectorAll('div.fixed.inset-0').forEach(el => {
        const hasModal = el.querySelector('[class*="modal"]');
        const hiddenModal = el.querySelector('[class*="animate-modal-in"]');
        if (!hasModal || hiddenModal) {
          el.style.display = 'none';
        }
      });

      document.body.style.overflow = '';
      document.documentElement.style.overflow = '';
    } catch (e) {}
  }

  // 当前正在采集的卡片（用于备选图片提取）
  let currentCard = null;

  // 创建采集按钮
  function createCollectButton(card, isCollected = false, cardTitle = '') {
    const btn = document.createElement('button');
    btn.className = BUTTON_CLASS;
    if (isCollected) {
      btn.textContent = '✓ 已采集';
      btn.title = '此提示词已采集过';
      btn.style.background = 'linear-gradient(135deg, #10b981, #059669)';
      btn.disabled = true;
      btn.style.pointerEvents = 'none';
      card.setAttribute(COLLECTED_ATTR, 'true');
      return btn;
    }
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
      currentCard = card;

      try {
        // 优先尝试 fetch 详情页 HTML，不触碰历史记录，不打开弹窗
        let data = null;
        const linkEl = card.tagName === 'A' ? card : card.querySelector('a');
        const href = linkEl?.getAttribute('href');
        const detailUrl = href && href.startsWith('/') ? new URL(href, window.location.origin).href : null;

        if (detailUrl) {
          try {
            const res = await fetch(detailUrl, { credentials: 'same-origin' });
            const html = await res.text();
            const parser = new DOMParser();
            const doc = parser.parseFromString(html, 'text/html');
            data = await extractFromDocument(doc, detailUrl, card);
            console.log('[PC] fetch 详情页成功:', detailUrl);
          } catch (fetchErr) {
            console.log('[PC] fetch 详情页失败，回退到弹窗模式:', fetchErr);
          }
        }

        // 如果 fetch 失败或没提取到内容，回退到点击卡片打开弹窗
        if (!data || (!data.content && !data.content_en)) {
          card.click();
          await new Promise(r => setTimeout(r, 800));
          await new Promise(r => setTimeout(r, 600));
          data = await extractFromModal();
        }

        if (!data || (!data.content && !data.content_en)) {
          // 如果没找到内容，再等一下再试一次
          await new Promise(r => setTimeout(r, 500));
          const retryData = await extractFromModal();
          if (!retryData || (!retryData.content && !retryData.content_en)) {
            btn.textContent = '❌ 提取失败';
            btn.style.background = '#f87171';
            setTimeout(() => {
              btn.textContent = '采集';
              btn.style.background = 'linear-gradient(135deg, #06b6d4, #3b82f6)';
              btn.style.pointerEvents = 'auto';
            }, 2000);
            console.log('[Prompt Collector] 提取失败，请检查弹窗是否包含 prompt 内容');
            return;
          }
          Object.assign(data, retryData);
        }

        // 追加到批量列表
        const appendRes = await chrome.runtime.sendMessage({
          action: 'appendBatchData',
          data
        });

        // 标记已采集（去重用）
        collectedTitles.add(data.title);
        card.setAttribute(COLLECTED_ATTR, 'true');

        // 自动提交（如果开启了开关）
        if (config.autoSubmit) {
          btn.textContent = '提交中...';
          const payload = {
            title: data.title || '',
            content: data.content || data.content_en || '',
            content_en: data.content_en || '',
            description: data.description || data.title || '',
            model: data.model || '',
            media_type: data.media_type || 'image',
            tags: data.tags || '[]',
            category_id: getCategoryId(data.model),
            cover_image_url: data.cover_image_url || '',
            status: 1,
            sort_order: 0,
            is_premium: false,
            unlock_cost: 0,
            author: '采集导入',
            i18n: '{}',
            seo_i18n: '{}'
          };

          try {
            const submitRes = await chrome.runtime.sendMessage({
              action: 'apiRequest',
              method: 'POST',
              path: '/prompt/',
              body: payload,
              userId: config.userId
            });

            if (submitRes.success && submitRes.data && submitRes.data.success) {
              btn.textContent = '✅ 已入库';
              btn.style.background = 'linear-gradient(135deg, #10b981, #059669)';
              btn.disabled = true;
              btn.style.pointerEvents = 'none';
              // 更新批量列表中的状态
              if (appendRes.success && appendRes.item) {
                await chrome.runtime.sendMessage({
                  action: 'updateBatchItem',
                  id: appendRes.item.id,
                  updates: { submitted: true, error: '' }
                });
              }
            } else {
              btn.textContent = '✓ 已采集';
              setTimeout(() => {
                btn.textContent = '采集';
                btn.style.pointerEvents = 'auto';
              }, 2000);
            }
          } catch (submitErr) {
            btn.textContent = '✓ 已采集';
            setTimeout(() => {
              btn.textContent = '采集';
              btn.style.pointerEvents = 'auto';
            }, 2000);
          }
        } else {
          // 不自动提交，只显示已采集
          btn.textContent = '✓ 已采集';
          setTimeout(() => {
            btn.textContent = '采集';
            btn.style.pointerEvents = 'auto';
          }, 2000);
        }

        // 自动打开侧边栏（如果还没打开）
        try {
          await chrome.runtime.sendMessage({ action: 'openSidePanel' });
        } catch (e) {}

        // 注意：不自动关闭弹窗，也不修改 URL。
        // opennana.com 的 Next.js 弹窗路由很复杂，任何自动关闭操作（Escape/点击关闭/移除 DOM）
        // 都可能触发 router.back() 导致页面跳到其他详情页。
        // 让用户手动点击 X 关闭弹窗是最安全的方式。

      } catch (err) {
        console.error('[Prompt Collector] 采集失败:', err);
        alert('采集失败: ' + err.message);
        btn.textContent = '采集';
        btn.style.pointerEvents = 'auto';
      } finally {
        currentCard = null;
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

    // 检查是否已采集（根据标题去重）
    const cardTitle = (card.querySelector('h3, h4, .title, [class*="title"]')?.innerText || card.innerText?.split('\n')[0] || '').trim();
    const isCollected = collectedTitles.has(cardTitle);

    const btn = createCollectButton(card, isCollected, cardTitle);
    card.appendChild(btn);
  }

  // 扫描所有卡片
  function scanCards() {
    // opennana.com 的实际卡片 class: prompt-card
    const cards = document.querySelectorAll('.prompt-card:not([' + INJECTED_ATTR + '])');
    cards.forEach(injectButton);
  }

  // 加载配置和已采集记录
  async function loadConfigAndHistory() {
    try {
      const cfgRes = await chrome.runtime.sendMessage({ action: 'getConfig' });
      if (cfgRes.success) config = cfgRes.data || {};

      const histRes = await chrome.runtime.sendMessage({ action: 'getBatchData' });
      if (histRes.success) {
        const batch = histRes.batch || [];
        collectedTitles = new Set(batch.map(i => (i.title || '').trim()).filter(Boolean));
      }
    } catch (e) {}
  }

  // 解析映射配置
  function parseCategoryMapping() {
    const mapping = {};
    const text = config.categoryMapping || '';
    if (!text) return mapping;
    text.split('\n').forEach(line => {
      const [key, val] = line.split('=');
      if (key && val) {
        mapping[key.trim().toLowerCase()] = parseInt(val.trim()) || 0;
      }
    });
    return mapping;
  }

  // 根据模型名获取分类 ID
  function getCategoryId(modelName) {
    if (!modelName) return parseInt(config.defaultCategoryId) || 0;
    const normalized = modelName.trim().toLowerCase();
    const mapping = parseCategoryMapping();
    return mapping[normalized] || parseInt(config.defaultCategoryId) || 0;
  }

  // 初始化
  async function init() {
    await loadConfigAndHistory();
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
