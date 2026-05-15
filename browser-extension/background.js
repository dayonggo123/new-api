// background.js - Service Worker
// 负责跨域请求代理、Tab 管理、消息路由

const STORAGE_KEY = 'promptCollectorConfig';
const EXTRACTED_KEY = 'promptCollectorExtracted';

// 消息路由
chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  (async () => {
    try {
      switch (message.action) {
        case 'openDetailTab': {
          // 在后台打开详情页
          const tab = await chrome.tabs.create({
            url: message.url,
            active: false
          });
          sendResponse({ success: true, tabId: tab.id });
          break;
        }

        case 'saveExtractedData': {
          // 保存提取的数据
          await chrome.storage.local.set({ [EXTRACTED_KEY]: message.data });
          // 通知 popup 有新数据
          chrome.runtime.sendMessage({ action: 'dataExtracted', data: message.data });
          sendResponse({ success: true });
          break;
        }

        case 'getExtractedData': {
          // 获取提取的数据
          const result = await chrome.storage.local.get(EXTRACTED_KEY);
          sendResponse({ success: true, data: result[EXTRACTED_KEY] || null });
          break;
        }

        case 'clearExtractedData': {
          // 清除提取的数据
          await chrome.storage.local.remove(EXTRACTED_KEY);
          sendResponse({ success: true });
          break;
        }

        case 'saveConfig': {
          // 保存配置
          await chrome.storage.local.set({ [STORAGE_KEY]: message.data });
          sendResponse({ success: true });
          break;
        }

        case 'getConfig': {
          // 获取配置
          const result = await chrome.storage.local.get(STORAGE_KEY);
          sendResponse({ success: true, data: result[STORAGE_KEY] || {} });
          break;
        }

        case 'apiRequest': {
          // 代理跨域请求到 new-api
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

          const resp = await fetch(url, options);
          const data = await resp.json();
          sendResponse({ success: true, data });
          break;
        }

        case 'closeTab': {
          // 关闭指定 Tab（不传 tabId 则关闭发送者的 tab）
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

        default:
          sendResponse({ success: false, message: '未知 action' });
      }
    } catch (err) {
      console.error('Background error:', err);
      sendResponse({ success: false, message: err.message });
    }
  })();

  return true; // 保持通道开启以支持异步
});

// 安装时初始化
chrome.runtime.onInstalled.addListener(() => {
  console.log('Prompt Collector installed');
});
