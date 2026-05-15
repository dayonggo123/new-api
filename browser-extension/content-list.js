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
    let content = '';

    // 策略1：找 "复制" 后面的内容（JSON 或纯文本）
    // opennana 格式：复制\n{JSON...} 或 复制\n英文prompt\n中文\n复制\n中文prompt
    const copyMatch = modalText.match(/复制[\s\n]+([\s\S]*?)(?:中文[\s\n]*去 AI 生图[\s\n]*复制[\s\n]*[\s\S]*?)?(?:更多推荐|$)/);
    if (copyMatch) {
      const candidate = copyMatch[1].trim();
      // 如果以 { 开头，是 JSON 格式
      if (candidate.startsWith('{')) {
        content = candidate;
      } else {
        // 纯文本格式，取最长的连续段落
        const paragraphs = candidate.split('\n').map(l => l.trim()).filter(l => l.length > 20);
        content = paragraphs.join('\n');
      }
    }

    // 策略2：如果没找到，尝试 ENGLISH / 中文 分段提取
    if (!content) {
      const englishMatch = modalText.match(/ENGLISH[\s\n]*(?:去 AI 生图)?[\s\n]*复制[\s\n]*([\s\S]*?)(?:中文|去 AI 生图|更多推荐|$)/);
      if (englishMatch) {
        content = englishMatch[1].trim();
      }
    }
    if (!content) {
      const chineseMatch = modalText.match(/中文[\s\n]*(?:去 AI 生图)?[\s\n]*复制[\s\n]*([\s\S]*?)(?:更多推荐|$)/);
      if (chineseMatch) {
        content = chineseMatch[1].trim();
      }
    }

    // 策略3：从 DOM 中找最长的文本段落
    if (!content) {
      let longest = '';
      modal.querySelectorAll('p, div, span, pre, code').forEach(el => {
        const text = (el.innerText || '').trim();
        if (text.length > longest.length && text.length > 100 && text.length < 5000) {
          if (!el.closest('button, [class*="close"], [class*="header"]')) {
            longest = text;
          }
        }
      });
      if (longest.length > 100) content = longest;
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

  // 当前正在采集的卡片（用于备选图片提取）
  let currentCard = null;

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

        // 自动打开侧边栏（如果还没打开）
        try {
          await chrome.runtime.sendMessage({ action: 'openSidePanel' });
        } catch (e) {
          // 侧边栏可能已打开或不被支持，忽略错误
        }

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
