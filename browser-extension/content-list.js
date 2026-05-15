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

  // 从弹窗提取内容（opennana.com 弹窗适配）
  async function extractFromModal() {
    // opennana.com 弹窗：优先找 animate-modal-in，再找任意含 modal 的可见元素
    let modal = document.querySelector('[class*="animate-modal-in"]') ||
                document.querySelector('[class*="animate-modal"]');
    
    // 备选：找可见的、尺寸较大的 modal
    if (!modal || modal.offsetParent === null) {
      const allModals = Array.from(document.querySelectorAll('div')).filter(el => {
        const cls = el.className || '';
        const rect = el.getBoundingClientRect();
        return (cls.includes('modal') || cls.includes('dialog')) &&
               rect.width > 300 && rect.height > 200 &&
               el.offsetParent !== null;
      });
      modal = allModals[allModals.length - 1]; // 取最后出现的（最上层）
    }

    if (!modal || modal.offsetParent === null) {
      console.log('[Prompt Collector] 未找到弹窗');
      return null;
    }

    console.log('[Prompt Collector] 找到弹窗，class:', modal.className, '开始提取...');

    const modalText = modal.innerText || '';
    const lines = modalText.split('\n').map(l => l.trim()).filter(l => l);

    // 提取标题（第一行非空文本）
    const title = lines[0] || '';

    // 提取模型
    let model = '';
    const modelMatch = modalText.match(/模型[：:]\s*([^\n]+)/);
    if (modelMatch) model = modelMatch[1].trim();

    // 提取 prompt 内容（支持纯文本和 JSON 两种格式）
    let content = '';      // 中文
    let contentEn = '';    // 英文

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

    // 策略1：找 "复制" 后面的内容（JSON 或纯文本）
    // opennana 格式：复制\n{JSON...} 或 复制\n英文prompt\n中文\n复制\n中文prompt
    const copyMatch = modalText.match(/复制[\s\n]+([\s\S]*?)(?:中文[\s\n]*去 AI 生图[\s\n]*复制[\s\n]*[\s\S]*?)?(?:更多推荐|$)/);
    if (copyMatch) {
      const candidate = copyMatch[1].trim();
      // 如果以 { 开头，是 JSON 格式，通常是英文 prompt
      if (candidate.startsWith('{')) {
        try {
          const jsonObj = JSON.parse(candidate);
          contentEn = jsonObj.content || jsonObj.data || jsonObj.prompt || candidate;
        } catch (e) {
          contentEn = candidate;
        }
      } else {
        // 纯文本格式，按行分离中英文
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

    // 提取封面图/视频（弹窗里的大图或视频）
    let coverImageUrl = '';

    // 策略1：找 video 标签（视频类型）
    const video = modal.querySelector('video');
    if (video) {
      coverImageUrl = video.src || video.currentSrc || video.querySelector('source')?.src || '';
      console.log('[Prompt Collector] 找到 video:', coverImageUrl);
    }

    // 策略2：找 img 标签（图片类型）
    if (!coverImageUrl) {
      const imgs = Array.from(modal.querySelectorAll('img'));
      console.log('[Prompt Collector] 弹窗内 img 数量:', imgs.length);
      imgs.forEach((img, i) => {
        console.log(`  img[${i}] src=`, img.src, 'dataset=', img.dataset ? Object.keys(img.dataset) : 'none');
      });

      // 获取每个 img 的最佳 src（处理懒加载和 Next.js Image）
      const imgData = imgs.map(img => {
        const src = img.src || img.dataset?.src || img.dataset?.original || img.dataset?.lazySrc || '';
        // 解析 srcset 取最大图
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

      // 优先选 opennana 域名的图
      const opennanaImgs = imgData.filter(d => d.isOpennana);
      if (opennanaImgs.length > 0) {
        coverImageUrl = opennanaImgs[0].src;
        console.log('[Prompt Collector] 选中 opennana 图片:', coverImageUrl);
      } else if (imgData.length > 0) {
        coverImageUrl = imgData[0].src;
        console.log('[Prompt Collector] 选中第一张图片:', coverImageUrl);
      }
    }

    // 策略3：找背景图
    if (!coverImageUrl) {
      const els = modal.querySelectorAll('*');
      for (const el of els) {
        const style = window.getComputedStyle(el);
        const bg = style.backgroundImage;
        if (bg && bg !== 'none') {
          const match = bg.match(/url\(["']?([^"')]+)["']?\)/);
          if (match) {
            coverImageUrl = match[1];
            console.log('[Prompt Collector] 找到背景图:', coverImageUrl);
            break;
          }
        }
      }
    }

    // 策略4：如果弹窗里没找到，尝试从对应的卡片找缩略图
    if (!coverImageUrl && currentCard) {
      console.log('[Prompt Collector] 弹窗内未找到图片，尝试从卡片查找...');
      const cardImgs = Array.from(currentCard.querySelectorAll('img'));
      for (const img of cardImgs) {
        const src = img.src || img.dataset?.src || '';
        if (src) {
          coverImageUrl = src;
          console.log('[Prompt Collector] 从卡片获取图片:', coverImageUrl);
          break;
        }
      }
    }

    if (!coverImageUrl) {
      console.log('[Prompt Collector] 警告: 未找到任何图片/视频');
    }

    // 提取标签（在"收藏"和"示例图片"之间的词）
    const tags = [];
    const tagMatch = modalText.match(/收藏[\s\n]+([\s\S]*?)(?:示例图片|提示词|赞助)/);
    if (tagMatch) {
      const tagText = tagMatch[1].trim();
      // 按行或空格分割，过滤掉过长的
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
    // 策略1：发送 Escape 键盘事件
    try {
      document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }));
    } catch (e) {}

    // 策略2：点击遮罩层背景（opennana 弹窗外层通常是 fixed inset-0）
    const backdrops = document.querySelectorAll('div.fixed.inset-0');
    for (const el of backdrops) {
      if (el.offsetParent !== null && el.childElementCount > 0) {
        try {
          el.dispatchEvent(new MouseEvent('click', { bubbles: true }));
          return;
        } catch (e) {}
      }
    }

    // 策略3：对所有可能的关闭按钮触发 click 事件（不用 btn.click()，避免 crash）
    const closeSelectors = [
      'button svg',
      'button',
      '[class*="close"]',
      '[aria-label*="close"]',
      '[aria-label*="关闭"]'
    ];
    for (const sel of closeSelectors) {
      const els = document.querySelectorAll(sel);
      for (const el of els) {
        if (el.offsetParent !== null) {
          try {
            el.dispatchEvent(new MouseEvent('click', { bubbles: true }));
            // 不 return，多点几个确保关闭
          } catch (e) {}
        }
      }
    }

    // 策略4：兜底——直接移除弹窗 DOM（opennana 用 animate-modal-in）
    try {
      const modals = document.querySelectorAll('[class*="animate-modal-in"]');
      for (const m of modals) {
        m.remove();
      }
      // 同时移除遮罩层
      document.querySelectorAll('div.fixed.inset-0').forEach(el => {
        if (el.childElementCount === 0 || !el.querySelector('[class*="modal"]')) {
          el.remove();
        }
      });
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
        // 点击卡片打开弹窗
        card.click();

        // 等待弹窗渲染（给一点动画时间）
        await new Promise(r => setTimeout(r, 800));

        // 等待图片加载（给懒加载一点时间）
        await new Promise(r => setTimeout(r, 600));

        // 提取内容
        const data = await extractFromModal();

        if (!data || (!data.content && !data.content_en)) {
          // 如果没找到内容，再等一下再试一次
          await new Promise(r => setTimeout(r, 500));
          const retryData = await extractFromModal();
          if (!retryData || (!retryData.content && !retryData.content_en)) {
            alert('未能提取到 prompt 内容，请手动复制后粘贴到插件弹窗');
            btn.textContent = '采集';
            btn.style.pointerEvents = 'auto';
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

        // 关闭弹窗
        closeModal();

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
