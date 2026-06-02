// content-list.js - 列表页脚本（适配 opennana.com 弹窗模式）
// v1.2 - 新增：服务端去重 + 批量采集
(function () {
  'use strict';

  // URL 安全检查：仅列表页（无子路径）才运行，避免与 content-detail.js 冲突
  if (window.location.pathname.replace(/\/$/, '') !== '/awesome-prompt-gallery') {
    return;
  }

  const BUTTON_CLASS = 'prompt-collector-btn';
  const INJECTED_ATTR = 'data-pc-injected';
  const COLLECTED_ATTR = 'data-pc-collected';
  const BATCH_CHECKBOX_CLASS = 'pc-batch-checkbox';
  const BATCH_UI_ATTR = 'data-pc-batch-ui';

  let config = {};
  let collectedTitles = new Set();
  let collectedUrls = new Set();  // 基于 source_url 去重（更可靠）
  let batchMode = false;
  let batchUI = null; // { toggleBtn, actionBar, progress }

  // 安全发送消息到 background（处理扩展重载 + service worker 休眠后的连接错误）
  async function safeSendMessage(msg, retries = 2) {
    for (let i = 0; i <= retries; i++) {
      try {
        return await chrome.runtime.sendMessage(msg);
      } catch (e) {
        const msgText = e.message || '';
        const isRetriable = msgText.includes('Could not establish connection') ||
                            msgText.includes('context invalidated') ||
                            msgText.includes('Receiving end does not exist');
        if (isRetriable && i < retries) {
          console.debug('[PC] 消息发送失败，等待重试 (' + (i + 1) + '/' + retries + '):', msgText);
          await new Promise(r => setTimeout(r, 600));
          continue;
        }
        if (msgText.includes('Extension context invalidated') || msgText.includes('context invalidated')) {
          showPageToast('⚠️ 扩展已更新，请刷新页面后重新采集', 10000);
        }
        throw e;
      }
    }
  }

  // 页面内 Toast 提示
  function showPageToast(text, duration = 4000) {
    const existing = document.getElementById('pc-page-toast');
    if (existing) existing.remove();

    const toast = document.createElement('div');
    toast.id = 'pc-page-toast';
    toast.textContent = text;
    Object.assign(toast.style, {
      position: 'fixed',
      top: '16px',
      left: '50%',
      transform: 'translateX(-50%)',
      zIndex: '999999',
      padding: '12px 24px',
      background: '#1e293b',
      color: '#fff',
      borderRadius: '12px',
      fontSize: '14px',
      fontWeight: '600',
      boxShadow: '0 8px 32px rgba(0,0,0,0.2)',
      maxWidth: '90%',
      textAlign: 'center'
    });
    document.body.appendChild(toast);
    setTimeout(() => {
      toast.style.opacity = '0';
      toast.style.transition = 'opacity 0.3s';
      setTimeout(() => toast.remove(), 300);
    }, duration);
  }

  // ==================== 工具函数 ====================

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

  // 从 URL 提取来源平台名
  function extractSource(hostname) {
    const match = hostname.replace(/^www\./, '').match(/^([^.]+)/);
    return match ? match[1] : hostname;
  }

  // 获取卡片详情页 URL
  function getCardDetailUrl(card) {
    const linkEl = card.tagName === 'A' ? card : card.querySelector('a');
    const href = linkEl?.getAttribute('href');
    return href && href.startsWith('/') ? new URL(href, window.location.origin).href : null;
  }

  // 获取卡片标题
  function getCardTitle(card) {
    return (card.querySelector('h3, h4, .title, [class*="title"]')?.innerText || card.innerText?.split('\n')[0] || '').trim();
  }

  // ==================== 去重 ====================

  // 从服务端加载已采集的 URL 列表（支持重试）
  async function loadServerCollectedUrls(retries = 3) {
    if (!config.apiBaseUrl || !config.apiToken) return;

    const tryCheck = async (attempt) => {
      // 收集当前页面上所有卡片的 URL
      const cards = document.querySelectorAll('.prompt-card');
      const urls = [];
      cards.forEach(c => {
        const url = getCardDetailUrl(c);
        if (url) urls.push(url);
      });

      // 没有卡片且还有重试次数 → 等 SPA 渲染
      if (urls.length === 0 && attempt < retries) {
        await new Promise(r => setTimeout(r, 1500));
        return tryCheck(attempt + 1);
      }

      if (urls.length === 0) return;

      try {
        const res = await safeSendMessage({
          action: 'apiRequest',
          method: 'POST',
          path: '/prompt/check-exists',
          body: { urls },
          userId: config.userId
        });
        if (res.success && res.data && res.data.success && res.data.data) {
          const existing = res.data.data.existing || [];
          existing.forEach(u => collectedUrls.add(u));
          console.log('[PC] 服务端已采集 URLs:', existing.length);
          // 有新增去重结果 → 刷新卡片按钮状态
          if (existing.length > 0) scanCards();
        }
      } catch (e) {
        console.debug('[PC] 服务端去重检查失败:', e.message);
      }
    };

    await tryCheck(0);
  }

  // ==================== 提取逻辑 ====================

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
    let videoUrl = '';

    const video = modal.querySelector('video');
    if (video) {
      videoUrl = video.src || video.currentSrc || video.querySelector('source')?.src || '';
      // 视频封面图单独提取
      const poster = video.getAttribute('poster') || '';
      if (poster) coverImageUrl = poster;
    }

    if (!coverImageUrl && !videoUrl) {
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

    // 找背景图（仅在当前页面模式下使用 getComputedStyle）
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

    // 从传入的卡片找缩略图
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
      cover_image_url: videoUrl ? '' : coverImageUrl,
      video_url: videoUrl,
      source: extractSource(new URL(sourceUrl).hostname),
      model,
      media_type: sourceUrl.includes('video') ? 'video' : 'image',
      tags: JSON.stringify(tags.slice(0, 10)),
      source_url: sourceUrl,
      category_id: getCategoryId(model),
      status: 1
    };

    return data;
  }

  // 兼容旧调用
  async function extractFromModal() {
    return extractFromDocument(document, window.location.href, currentCard);
  }

  // ==================== 历史导航冻结（防止 React 路由冲突） ====================

  function freezeHistory(durationMs = 1000) {
    const originalBack = history.back.bind(history);
    const originalGo = history.go.bind(history);
    const originalForward = history.forward.bind(history);
    const nav = window.navigation;
    const originalNavBack = nav && nav.back ? nav.back.bind(nav) : null;

    history.back = () => {};
    history.go = () => {};
    history.forward = () => {};
    if (originalNavBack) nav.back = () => {};

    setTimeout(() => {
      history.back = originalBack;
      history.go = originalGo;
      history.forward = originalForward;
      if (originalNavBack) nav.back = originalNavBack;
    }, durationMs);
  }

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
    } catch (e) {
      console.warn('[Prompt Collector] closeModal error:', e);
    }
  }

  // ==================== 单卡采集 ====================

  let currentCard = null;

  // 提交单条数据到 API
  async function submitDataToApi(data) {
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
      video_url: data.video_url || '',
      source: data.source || '',
      status: 1,
      sort_order: 0,
      is_premium: false,
      unlock_cost: 0,
      author: '采集导入',
      i18n: '{}',
      seo_i18n: '{}'
    };

    const res = await safeSendMessage({
      action: 'apiRequest',
      method: 'POST',
      path: '/prompt/',
      body: payload,
      userId: config.userId
    });

    return res;
  }

  // 采集单张卡片（提取 + 入库）
  async function collectSingleCard(card, onProgress) {
    let data = null;
    const detailUrl = getCardDetailUrl(card);

    // 优先 fetch 详情页 HTML
    if (detailUrl) {
      try {
        const res = await fetch(detailUrl, { credentials: 'same-origin' });
        const html = await res.text();
        const doc = new DOMParser().parseFromString(html, 'text/html');
        data = await extractFromDocument(doc, detailUrl, card);
      } catch (fetchErr) {
        // 回退到弹窗
      }
    }

    // fetch 失败或没内容 → 点击打开弹窗
    if (!data || (!data.content && !data.content_en)) {
      card.click();
      await new Promise(r => setTimeout(r, 800));
      await new Promise(r => setTimeout(r, 600));
      data = await extractFromModal();
    }

    // 再等一下重试
    if (!data || (!data.content && !data.content_en)) {
      await new Promise(r => setTimeout(r, 500));
      const retryData = await extractFromModal();
      if (!retryData || (!retryData.content && !retryData.content_en)) {
        return { success: false, error: '提取失败' };
      }
      Object.assign(data, retryData);
    }

    // 追加到批量列表
    const appendRes = await safeSendMessage({
      action: 'appendBatchData',
      data
    });

    // 标记已采集
    collectedTitles.add(data.title);
    collectedUrls.add(data.source_url);
    card.setAttribute(COLLECTED_ATTR, 'true');

    // 自动提交（如果开启了开关）
    if (config.autoSubmit) {
      if (onProgress) onProgress('提交中...');
      const submitRes = await submitDataToApi(data);
      if (submitRes.success && submitRes.data && submitRes.data.success) {
        if (appendRes.success && appendRes.item) {
          await safeSendMessage({
            action: 'updateBatchItem',
            id: appendRes.item.id,
            updates: { submitted: true, error: '' }
          });
        }
        return { success: true, submitted: true, title: data.title };
      } else {
        return { success: true, submitted: false, title: data.title, warn: submitRes.data?.message || '提交失败' };
      }
    }

    return { success: true, submitted: false, title: data.title };
  }

  // ==================== 批量采集 ====================

  // 创建批量模式 UI
  function createBatchUI() {
    if (document.querySelector(`[${BATCH_UI_ATTR}]`)) return;

    // 浮动切换按钮
    const toggleBtn = document.createElement('button');
    toggleBtn.setAttribute(BATCH_UI_ATTR, '');
    toggleBtn.textContent = '📦 批量';
    Object.assign(toggleBtn.style, {
      position: 'fixed',
      top: '80px',
      right: '16px',
      zIndex: '99999',
      padding: '8px 14px',
      fontSize: '13px',
      fontWeight: '700',
      color: '#fff',
      background: 'linear-gradient(135deg, #8b5cf6, #6366f1)',
      border: 'none',
      borderRadius: '20px',
      cursor: 'pointer',
      boxShadow: '0 4px 12px rgba(99,102,241,0.4)',
      transition: 'all 0.2s ease'
    });
    toggleBtn.addEventListener('mouseenter', () => {
      toggleBtn.style.transform = 'scale(1.05)';
    });
    toggleBtn.addEventListener('mouseleave', () => {
      toggleBtn.style.transform = 'scale(1)';
    });
    toggleBtn.addEventListener('click', toggleBatchMode);
    document.body.appendChild(toggleBtn);

    // 底部操作栏
    const actionBar = document.createElement('div');
    actionBar.setAttribute(BATCH_UI_ATTR, '');
    actionBar.id = 'pcBatchBar';
    Object.assign(actionBar.style, {
      display: 'none',
      position: 'fixed',
      bottom: '0',
      left: '0',
      right: '0',
      zIndex: '99999',
      padding: '12px 16px',
      background: '#fff',
      borderTop: '2px solid #8b5cf6',
      boxShadow: '0 -4px 16px rgba(0,0,0,0.12)',
      display: 'none',
      alignItems: 'center',
      justifyContent: 'space-between',
      fontSize: '14px',
      fontWeight: '600'
    });
    actionBar.innerHTML = `
      <span id="pcBatchCount" style="color:#475569;">已选 <strong style="color:#8b5cf6;">0</strong> 个</span>
      <div style="display:flex;gap:8px;align-items:center;">
        <span id="pcBatchProgress" style="font-size:12px;color:#94a3b8;display:none;"></span>
        <button id="pcBatchGoBtn" style="padding:8px 18px;font-size:13px;font-weight:700;color:#fff;background:linear-gradient(135deg,#8b5cf6,#6366f1);border:none;border-radius:8px;cursor:pointer;box-shadow:0 2px 8px rgba(99,102,241,0.3);">批量采集</button>
        <button id="pcBatchCancelBtn" style="padding:8px 14px;font-size:13px;color:#64748b;background:#f1f5f9;border:1px solid #cbd5e1;border-radius:8px;cursor:pointer;">取消</button>
      </div>
    `;
    document.body.appendChild(actionBar);

    batchUI = {
      toggleBtn,
      actionBar,
      countEl: document.getElementById('pcBatchCount'),
      progressEl: document.getElementById('pcBatchProgress'),
      goBtn: document.getElementById('pcBatchGoBtn'),
      cancelBtn: document.getElementById('pcBatchCancelBtn')
    };

    batchUI.goBtn.addEventListener('click', () => doBatchCollect());
    batchUI.cancelBtn.addEventListener('click', () => {
      // 退出批量模式并刷新卡片
      if (batchMode) toggleBatchMode();
      scanCards();
    });
  }

  // 切换批量模式
  function toggleBatchMode() {
    batchMode = !batchMode;
    const isActive = batchMode;

    // 更新切换按钮样式
    if (batchUI) {
      batchUI.toggleBtn.textContent = isActive ? '📦 退出批量' : '📦 批量';
      batchUI.toggleBtn.style.background = isActive
        ? 'linear-gradient(135deg, #ef4444, #dc2626)'
        : 'linear-gradient(135deg, #8b5cf6, #6366f1)';
      batchUI.actionBar.style.display = isActive ? 'flex' : 'none';
    }

    // 重新扫卡，根据模式展示按钮或复选框
    scanCards();
  }

  // 获取选中的卡片
  function getSelectedCards() {
    const cards = [];
    document.querySelectorAll(`.${BATCH_CHECKBOX_CLASS}:checked`).forEach(cb => {
      const card = document.querySelector(`[data-pc-card-id="${cb.dataset.cardId}"]`);
      if (card) cards.push(card);
    });
    return cards;
  }

  // 更新批量计数
  function updateBatchCount() {
    const selected = document.querySelectorAll(`.${BATCH_CHECKBOX_CLASS}:checked`).length;
    if (batchUI) {
      batchUI.countEl.innerHTML = `已选 <strong style="color:#8b5cf6;">${selected}</strong> 个`;
      batchUI.goBtn.disabled = selected === 0;
      batchUI.goBtn.style.opacity = selected === 0 ? '0.5' : '1';
    }
  }

  // 执行批量采集
  async function doBatchCollect() {
    const cards = getSelectedCards();
    if (cards.length === 0) return;
    if (!config.apiBaseUrl) {
      alert('请先在侧边栏配置 API');
      return;
    }

    const total = cards.length;
    let success = 0, fail = 0;

    batchUI.goBtn.disabled = true;
    batchUI.goBtn.textContent = '采集中...';
    batchUI.progressEl.style.display = 'inline';
    batchUI.progressEl.textContent = '0/' + total;

    for (let i = 0; i < total; i++) {
      const card = cards[i];
      const title = getCardTitle(card);

      // 如果已被采集则跳过
      if (card.hasAttribute(COLLECTED_ATTR)) {
        success++;
        batchUI.progressEl.textContent = `${i + 1}/${total}`;
        continue;
      }

      batchUI.progressEl.textContent = `采集中 ${i + 1}/${total}...`;
      currentCard = card;

      try {
        const result = await collectSingleCard(card, (msg) => {
          batchUI.progressEl.textContent = `${msg} ${i + 1}/${total}...`;
        });
        if (result.success) {
          success++;
        } else {
          fail++;
        }
      } catch (err) {
        console.error('[PC] 批量采集卡片失败:', err);
        fail++;
      }

      // 关闭可能打开的弹窗，避免影响下一张
      closeModal();
      await new Promise(r => setTimeout(r, 300));

      batchUI.progressEl.textContent = `${i + 1}/${total}`;
      currentCard = null;
    }

    // 完成
    batchUI.goBtn.disabled = false;
    batchUI.goBtn.textContent = '批量采集';
    batchUI.progressEl.style.display = 'none';

    if (fail === 0) {
      batchUI.progressEl.textContent = `✅ 全部成功（${success}/${total}）`;
    } else {
      batchUI.progressEl.textContent = `⚠️ 成功 ${success}，失败 ${fail}`;
    }
    batchUI.progressEl.style.display = 'inline';
    setTimeout(() => {
      batchUI.progressEl.style.display = 'none';
    }, 4000);

    // 自动打开侧边栏
    try {
      await safeSendMessage({ action: 'openSidePanel' });
    } catch (e) {}
  }

  // 创建采集按钮（非批量模式）
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

      // 立即打开侧边栏（必须在用户手势窗口内，不用 safeSendMessage 的重试）
      chrome.runtime.sendMessage({ action: 'openSidePanel' }).catch(() => {});

      try {
        const result = await collectSingleCard(card);
        if (result.success && result.submitted) {
          btn.textContent = '✅ 已入库';
          btn.style.background = 'linear-gradient(135deg, #10b981, #059669)';
          btn.disabled = true;
          btn.style.pointerEvents = 'none';
        } else if (result.success) {
          btn.textContent = '✓ 已采集';
          setTimeout(() => {
            btn.textContent = '采集';
            btn.style.pointerEvents = 'auto';
          }, 2000);
        } else {
          btn.textContent = '❌ ' + (result.error || '失败');
          btn.style.background = '#f87171';
          setTimeout(() => {
            btn.textContent = '采集';
            btn.style.background = 'linear-gradient(135deg, #06b6d4, #3b82f6)';
            btn.style.pointerEvents = 'auto';
          }, 2000);
        }
      } catch (err) {
        console.error('[Prompt Collector] 采集失败:', err);
        btn.textContent = '采集';
        btn.style.pointerEvents = 'auto';
      } finally {
        currentCard = null;
      }

      // 自动打开侧边栏
      try {
        await safeSendMessage({ action: 'openSidePanel' });
      } catch (e) {}
    });

    return btn;
  }

  // 创建批量复选框
  function createBatchCheckbox(card, cardId) {
    const cb = document.createElement('input');
    cb.type = 'checkbox';
    cb.className = BATCH_CHECKBOX_CLASS;
    cb.dataset.cardId = cardId;
    Object.assign(cb.style, {
      position: 'absolute',
      top: '8px',
      right: '8px',
      zIndex: '9999',
      width: '20px',
      height: '20px',
      cursor: 'pointer',
      accentColor: '#8b5cf6'
    });
    cb.addEventListener('change', updateBatchCount);
    return cb;
  }

  // 为卡片注入按钮或复选框
  function injectButton(card) {
    if (card.hasAttribute(INJECTED_ATTR)) return;
    card.setAttribute(INJECTED_ATTR, 'true');

    const computed = window.getComputedStyle(card);
    if (computed.position === 'static') {
      card.style.position = 'relative';
    }

    const cardTitle = getCardTitle(card);
    const cardUrl = getCardDetailUrl(card);

    // 检查去重：走 URL 精确匹配，没有 URL 则降级到标题匹配
    const isCollected = collectedUrls.has(cardUrl) || collectedTitles.has(cardTitle);

    if (isCollected) {
      card.setAttribute(COLLECTED_ATTR, 'true');
    }

    if (batchMode && !isCollected) {
      // 批量模式：显示复选框（为每张卡生成唯一 ID）
      const cid = 'pc-' + Date.now() + '-' + Math.random().toString(36).slice(2, 6);
      card.dataset.pcCardId = cid;
      const cb = createBatchCheckbox(card, cid);
      card.appendChild(cb);
      // 隐藏已采集标记
      if (!isCollected) {
        card.removeAttribute(COLLECTED_ATTR);
      }
    } else {
      // 普通模式：显示采集按钮
      card.removeAttribute('data-pc-card-id');
      const btn = createCollectButton(card, isCollected, cardTitle);
      // 如果有旧的复选框，移除
      const oldCb = card.querySelector(`.${BATCH_CHECKBOX_CLASS}`);
      if (oldCb) oldCb.remove();
      card.appendChild(btn);
    }
  }

  // 扫描所有卡片
  function scanCards() {
    const cards = document.querySelectorAll('.prompt-card');
    cards.forEach(injectButton);
  }

  // 加载配置和已采集记录
  async function loadConfigAndHistory() {
    try {
      const cfgRes = await safeSendMessage({ action: 'getConfig' });
      if (cfgRes.success) config = cfgRes.data || {};

      const histRes = await safeSendMessage({ action: 'getBatchData' });
      if (histRes.success) {
        const batch = histRes.batch || [];
        // 从本地批量列表恢复去重 Set
        collectedTitles = new Set(batch.map(i => (i.title || '').trim()).filter(Boolean));
        collectedUrls = new Set(batch.map(i => i.source_url).filter(Boolean));
      }

      // 从服务端加载已采集 URL（跨会话去重）
      await loadServerCollectedUrls();

    } catch (e) {
      console.debug('[Prompt Collector] loadConfigAndHistory 暂未就绪:', e.message);
    }
  }

  // ==================== 分类映射 ====================

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

  function getCategoryId(modelName) {
    if (!modelName) return parseInt(config.defaultCategoryId) || 0;
    const normalized = modelName.trim().toLowerCase();
    const mapping = parseCategoryMapping();
    return mapping[normalized] || parseInt(config.defaultCategoryId) || 0;
  }

  // ==================== 初始化 ====================

  async function init() {
    await loadConfigAndHistory();
    createBatchUI();
    scanCards();

    // 防抖：合并短时间内的多次 DOM 变动
    let debounceTimer = null;
    const debouncedScan = () => {
      if (debounceTimer) clearTimeout(debounceTimer);
      debounceTimer = setTimeout(() => {
        debounceTimer = null;
        scanCards();
      }, 300);
    };

    // 监听 DOM 变化（SPA 加载更多卡片）
    const container = document.querySelector('.prompt-card')?.closest('[class*="grid"], [class*="list"], [class*="container"]') || document.body;
    const observer = new MutationObserver((mutations) => {
      for (const m of mutations) {
        if (m.type === 'childList') {
          for (const node of m.addedNodes) {
            if (node.nodeType === Node.ELEMENT_NODE) {
              if (node.classList?.contains('prompt-card') || node.querySelector?.('.prompt-card')) {
                debouncedScan();
                return;
              }
            }
          }
        }
      }
    });

    observer.observe(container, { childList: true, subtree: true });

    window.addEventListener('beforeunload', () => {
      observer.disconnect();
      if (debounceTimer) clearTimeout(debounceTimer);
    });
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
