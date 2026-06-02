// background.js - Service Worker
// 负责跨域请求代理、Tab 管理、消息路由、批量数据存储

const STORAGE_KEY = 'promptCollectorConfig';
const EXTRACTED_KEY = 'promptCollectorExtracted';
const BATCH_KEY = 'promptCollectorBatch';
const FETCH_TIMEOUT_MS = 15000; // API 请求超时 15s

// 带超时的 fetch
async function fetchWithTimeout(url, options, timeoutMs = FETCH_TIMEOUT_MS) {
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), timeoutMs);
  try {
    const resp = await fetch(url, { ...options, signal: controller.signal });
    return resp;
  } finally {
    clearTimeout(timeoutId);
  }
}

// 消息路由
chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  // 参数校验
  if (!message || typeof message !== 'object') {
    sendResponse({ success: false, message: '无效消息' });
    return;
  }

  (async () => {
    try {
      switch (message.action) {
        case 'openDetailTab': {
          const tab = await chrome.tabs.create({
            url: message.url,
            active: false
          });
          sendResponse({ success: true, tabId: tab.id });
          break;
        }

        // 单条兼容（旧接口）
        case 'saveExtractedData': {
          await chrome.storage.local.set({ [EXTRACTED_KEY]: message.data });
          chrome.runtime.sendMessage({ action: 'dataExtracted', data: message.data }).catch(() => {});
          sendResponse({ success: true });
          break;
        }

        case 'getExtractedData': {
          const result = await chrome.storage.local.get(EXTRACTED_KEY);
          sendResponse({ success: true, data: result[EXTRACTED_KEY] || null });
          break;
        }

        case 'clearExtractedData': {
          await chrome.storage.local.remove(EXTRACTED_KEY);
          chrome.runtime.sendMessage({ action: 'dataCleared' }).catch(() => {});
          sendResponse({ success: true });
          break;
        }

        // 批量接口（新）
        case 'appendBatchData': {
          const result = await chrome.storage.local.get(BATCH_KEY);
          const batch = result[BATCH_KEY] || [];
          const item = {
            id: Date.now() + Math.random().toString(36).slice(2, 8),
            ...message.data,
            submitted: false,
            error: '',
            createdAt: Date.now()
          };
          batch.unshift(item);
          await chrome.storage.local.set({ [BATCH_KEY]: batch });
          chrome.runtime.sendMessage({ action: 'batchUpdated', batch }).catch(() => {});
          sendResponse({ success: true, item });
          break;
        }

        case 'getBatchData': {
          const result = await chrome.storage.local.get(BATCH_KEY);
          sendResponse({ success: true, batch: result[BATCH_KEY] || [] });
          break;
        }

        case 'updateBatchItem': {
          const result = await chrome.storage.local.get(BATCH_KEY);
          const batch = result[BATCH_KEY] || [];
          const idx = batch.findIndex(i => i.id === message.id);
          if (idx >= 0) {
            batch[idx] = { ...batch[idx], ...message.updates };
            await chrome.storage.local.set({ [BATCH_KEY]: batch });
          }
          sendResponse({ success: true });
          break;
        }

        case 'removeBatchItem': {
          const result = await chrome.storage.local.get(BATCH_KEY);
          let batch = result[BATCH_KEY] || [];
          batch = batch.filter(i => i.id !== message.id);
          await chrome.storage.local.set({ [BATCH_KEY]: batch });
          chrome.runtime.sendMessage({ action: 'batchUpdated', batch }).catch(() => {});
          sendResponse({ success: true });
          break;
        }

        case 'clearSubmittedBatch': {
          const result = await chrome.storage.local.get(BATCH_KEY);
          let batch = result[BATCH_KEY] || [];
          batch = batch.filter(i => !i.submitted);
          await chrome.storage.local.set({ [BATCH_KEY]: batch });
          chrome.runtime.sendMessage({ action: 'batchUpdated', batch }).catch(() => {});
          sendResponse({ success: true });
          break;
        }

        case 'clearBatchData': {
          await chrome.storage.local.remove(BATCH_KEY);
          chrome.runtime.sendMessage({ action: 'batchUpdated', batch: [] }).catch(() => {});
          sendResponse({ success: true });
          break;
        }

        case 'fetchTokenFromTab': {
          try {
            // 查找管理后台标签页
            const tabs = await chrome.tabs.query({ url: 'https://heharse.cloud/*' });
            if (tabs.length === 0) {
              sendResponse({ success: false, message: '未找到管理后台标签页，请先打开 https://heharse.cloud' });
              return;
            }
            const tab = tabs[0];

            // 注入脚本：在管理后台页面内直接调用 API 获取 access_token
            const results = await chrome.scripting.executeScript({
              target: { tabId: tab.id },
              func: async () => {
                try {
                  const resp = await fetch('/api/user/token', {
                    credentials: 'same-origin'
                  });
                  const json = await resp.json();
                  if (json.success && json.data) {
                    return { success: true, token: json.data };
                  }
                  // 如果没有 token，尝试先登录
                  // 尝试从 localStorage 读取 user 信息
                  const userStr = localStorage.getItem('user');
                  if (userStr) {
                    const user = JSON.parse(userStr);
                    if (user.access_token) {
                      return { success: true, token: user.access_token };
                    }
                  }
                  return { success: false, message: json.message || '请确保已登录管理后台' };
                } catch (e) {
                  return { success: false, message: e.message };
                }
              }
            });

            const result = results?.[0]?.result;
            if (result && result.success && result.token) {
              sendResponse({ success: true, token: result.token, isAccessToken: true });
            } else {
              sendResponse({ success: false, message: result?.message || '获取失败（需先打开 https://heharse.cloud 并登录）' });
            }
          } catch (err) {
            sendResponse({ success: false, message: err.message });
          }
          break;
        }

        case 'getCollectedUrls': {
          const result = await chrome.storage.local.get(BATCH_KEY);
          const batch = result[BATCH_KEY] || [];
          const urls = batch.map(i => i.source_url).filter(Boolean);
          sendResponse({ success: true, urls });
          break;
        }

        case 'saveConfig': {
          await chrome.storage.local.set({ [STORAGE_KEY]: message.data });
          sendResponse({ success: true });
          break;
        }

        case 'getConfig': {
          const result = await chrome.storage.local.get(STORAGE_KEY);
          sendResponse({ success: true, data: result[STORAGE_KEY] || {} });
          break;
        }

        case 'apiRequest': {
          const config = await chrome.storage.local.get(STORAGE_KEY);
          const cfg = config[STORAGE_KEY] || {};
          const baseUrl = (cfg.apiBaseUrl || '').replace(/\/$/, '');
          const token = cfg.apiToken || '';

          if (!baseUrl) {
            sendResponse({ success: false, message: '请先配置 API Base URL' });
            return;
          }

          // 自动补全 /api 前缀（如果没有）
          const apiBase = baseUrl + (baseUrl.includes('/api') ? '' : '/api');
          const url = `${apiBase}${message.path}`;
          const options = {
            method: message.method || 'GET',
            headers: {
              'Content-Type': 'application/json',
              'New-API-User': message.userId || cfg.userId || '',
              ...(token ? { 'Authorization': token.startsWith('Bearer ') ? token : `Bearer ${token}` } : {})
            }
          };

          if (message.body) {
            options.body = JSON.stringify(message.body);
          }

          console.log('[Background] API Request:', options.method, url);
          const resp = await fetchWithTimeout(url, options);
          const text = await resp.text();
          let data;
          try {
            data = JSON.parse(text);
          } catch (e) {
            data = { raw: text };
          }
          console.log('[Background] API Response:', resp.status, data);
          if (!resp.ok) {
            sendResponse({ success: false, message: `HTTP ${resp.status}: ${data.message || text.slice(0, 200)}` });
            return;
          }
          sendResponse({ success: true, data });
          break;
        }

        case 'closeTab': {
          const targetTabId = message.tabId || sender.tab?.id;
          if (targetTabId) {
            try {
              await chrome.tabs.remove(targetTabId);
            } catch (e) {
              // Tab 可能已关闭，属正常情况
            }
          }
          sendResponse({ success: true, tabClosed: !!targetTabId });
          break;
        }

        case 'openSidePanel': {
          try {
            const windowId = sender.tab?.windowId;
            const tabId = sender.tab?.id;
            if (windowId) {
              // 先打开侧边栏（必须在手势窗口内，前面不能有 await）
              await chrome.sidePanel.open({ windowId });
              // 再异步启用（失败不影响打开）
              if (tabId) {
                chrome.sidePanel.setOptions({ tabId, path: 'sidepanel.html', enabled: true }).catch(() => {});
              }
            } else if (tabId) {
              await chrome.sidePanel.open({ tabId });
            }
            sendResponse({ success: true });
          } catch (err) {
            console.warn('[Background] 打开侧边栏失败:', err.message);
            sendResponse({ success: false, message: err.message });
          }
          break;
        }

        default:
          sendResponse({ success: false, message: '未知 action' });
      }
    } catch (err) {
      if (err.name === 'AbortError') {
        console.error('[Background] 请求超时:', message.action);
        sendResponse({ success: false, message: '请求超时，请检查网络或 API 地址' });
      } else {
        console.error('[Background] error:', message.action, err);
        sendResponse({ success: false, message: err.message || '未知错误' });
      }
    }
  })();

  return true;
});

// 安装时初始化
chrome.runtime.onInstalled.addListener((details) => {
  console.log(`Prompt Collector v${chrome.runtime.getManifest().version} installed (reason: ${details.reason})`);

  // 尝试设置侧边栏行为（容错，失败不影响核心功能）
  try {
    if (chrome.sidePanel && chrome.sidePanel.setPanelBehavior) {
      chrome.sidePanel.setPanelBehavior({ openPanelOnActionClick: true });
    }
  } catch (e) {
    console.warn('[Background] setPanelBehavior 跳过:', e?.message || '不可用');
  }
});
