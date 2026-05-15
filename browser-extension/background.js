// background.js - Service Worker
// 负责跨域请求代理、Tab 管理、消息路由、批量数据存储

const STORAGE_KEY = 'promptCollectorConfig';
const EXTRACTED_KEY = 'promptCollectorExtracted';
const BATCH_KEY = 'promptCollectorBatch';

// 消息路由
chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
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

          const url = `${baseUrl}${message.path}`;
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
          const resp = await fetch(url, options);
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
              // Tab 可能已关闭
            }
          }
          sendResponse({ success: true });
          break;
        }

        case 'openSidePanel': {
          try {
            const windowId = sender.tab?.windowId;
            if (windowId) {
              await chrome.sidePanel.open({ windowId });
            }
            sendResponse({ success: true });
          } catch (err) {
            sendResponse({ success: false, message: err.message });
          }
          break;
        }

        default:
          sendResponse({ success: false, message: '未知 action' });
      }
    } catch (err) {
      console.error('Background error:', err);
      sendResponse({ success: false, message: err.message });
    }
  })();

  return true;
});

// 安装时初始化
chrome.runtime.onInstalled.addListener(() => {
  console.log('Prompt Collector installed');
});
