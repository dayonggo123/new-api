// background.js - Service Worker
// 负责跨域请求代理、Tab 管理、消息路由、批量数据存储
// v1.4.0 - 增加错误分类、SW 心跳保活、详细日志

const STORAGE_KEY = 'promptCollectorConfig';
const EXTRACTED_KEY = 'promptCollectorExtracted';
const BATCH_KEY = 'promptCollectorBatch';
const FETCH_TIMEOUT_MS = 10000; // API 请求超时 10s（从 15s 降低，更快反馈）
const FETCH_RETRY_DELAY_MS = 2000; // 网络错误重试间隔 2s
const FETCH_MAX_RETRIES = 1; // 网络错误最多重试 1 次
const SW_VERSION = '1.4.1';
const HEARTBEAT_ALARM = 'promptCollectorHeartbeat';
const STATS_KEY = 'promptCollectorStats';

// 启动日志（每次 SW 启动都打一条，便于排查失效问题）
console.log(`[PromptCollector] SW v${SW_VERSION} starting at ${new Date().toISOString()}`);

// ===== 全局 unhandledrejection 处理（防止 Extension context invalidated 漏出）=====
self.addEventListener('unhandledrejection', (event) => {
  const msg = event.reason?.message || String(event.reason);
  if (msg.includes('Extension context') || msg.includes('context invalidated')) {
    event.preventDefault(); // 吞掉，不显示在扩展错误页
    console.debug('[Background] swallowed context-invalidated rejection');
  }
});

// ===== 统计与诊断 =====
async function bumpStat(key, val = 1) {
  try {
    const r = await chrome.storage.local.get(STATS_KEY);
    const stats = r[STATS_KEY] || { startedAt: Date.now(), apiCalls: 0, apiErrors: 0, errorByType: {} };
    stats[key] = (stats[key] || 0) + val;
    await chrome.storage.local.set({ [STATS_KEY]: stats });
  } catch (e) { /* 统计失败不影响主流程 */ }
}

// ===== 带超时的 fetch，错误分类 =====
async function fetchWithTimeout(url, options, timeoutMs = FETCH_TIMEOUT_MS) {
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), timeoutMs);
  try {
    const resp = await fetch(url, { ...options, signal: controller.signal });
    return resp;
  } catch (e) {
    // 分类错误，便于排查
    if (e.name === 'AbortError') {
      const err = new Error(`请求超时（${timeoutMs / 1000}s）`);
      err.errorType = 'TIMEOUT';
      err.cause = e;
      throw err;
    }
    // Failed to fetch 可能是：网络断开、DNS 失败、CORS 拒绝、SSL 错误
    const err = new Error(e.message || '网络请求失败');
    err.errorType = e.message?.includes('CORS') ? 'CORS'
      : e.message?.includes('SSL') ? 'SSL'
      : e.message?.includes('DNS') ? 'DNS'
      : 'NETWORK';
    err.cause = e;
    throw err;
  } finally {
    clearTimeout(timeoutId);
  }
}

// ===== 友好错误信息生成器 =====
function friendlyApiError(err, url) {
  const tip = '（请检查 heharse.cloud 是否可访问，或在「API 配置」核对 Base URL）';
  switch (err.errorType) {
    case 'TIMEOUT':
      return `API 请求超时${tip}`;
    case 'CORS':
      return `API 跨域被拒绝（CORS）${tip}`;
    case 'SSL':
      return `SSL 证书错误${tip}`;
    case 'DNS':
      return `域名解析失败（DNS）${tip}`;
    case 'NETWORK':
      return `网络请求失败：${err.message}${tip}`;
    default:
      return `API 调用失败：${err.message}`;
  }
}

// ===== SW 心跳保活（防止 30s 闲置被 Chrome 杀掉）=====
// 重要：MV3 SW 不应做长期心跳，但调用 apiRequest 等动作会自动唤醒
// 这里只在 SW 启动时设置一次 alarm，用于快速验证存活
if (chrome.alarms) {
  chrome.alarms.create(HEARTBEAT_ALARM, { periodInMinutes: 0.4 }); // ~24s
  chrome.alarms.onAlarm.addListener((alarm) => {
    if (alarm.name === HEARTBEAT_ALARM) {
      // 仅做轻量记录，不发起网络请求（避免被误判为活跃）
      bumpStat('heartbeatTicks', 1);
    }
  });
}

// ===== 安全广播（防止 Extension context invalidated 漏出 unhandled rejection）=====
function safeBroadcast(msg) {
  try {
    chrome.runtime.sendMessage(msg).catch(() => {});
  } catch (e) {
    // SW context 已失效时 sendMessage 本身会 throw
  }
}

// ===== 消息路由 =====
chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  // 参数校验
  if (!message || typeof message !== 'object') {
    sendResponse({ success: false, message: '无效消息', errorType: 'INVALID' });
    return;
  }

  (async () => {
    const start = Date.now();
    try {
      switch (message.action) {
        case 'openDetailTab': {
          const tab = await chrome.tabs.create({ url: message.url, active: false });
          sendResponse({ success: true, tabId: tab.id });
          break;
        }

        case 'saveExtractedData': {
          await chrome.storage.local.set({ [EXTRACTED_KEY]: message.data });
          safeBroadcast({ action: 'dataExtracted', data: message.data });
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
          safeBroadcast({ action: 'dataCleared' });
          sendResponse({ success: true });
          break;
        }

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
          safeBroadcast({ action: 'batchUpdated', batch });
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
          safeBroadcast({ action: 'batchUpdated', batch });
          sendResponse({ success: true });
          break;
        }

        case 'clearSubmittedBatch': {
          const result = await chrome.storage.local.get(BATCH_KEY);
          let batch = result[BATCH_KEY] || [];
          batch = batch.filter(i => !i.submitted);
          await chrome.storage.local.set({ [BATCH_KEY]: batch });
          safeBroadcast({ action: 'batchUpdated', batch });
          sendResponse({ success: true });
          break;
        }

        case 'clearBatchData': {
          await chrome.storage.local.remove(BATCH_KEY);
          safeBroadcast({ action: 'batchUpdated', batch: [] });
          sendResponse({ success: true });
          break;
        }

        case 'fetchTokenFromTab': {
          try {
            const tabs = await chrome.tabs.query({ url: 'https://heharse.cloud/*' });
            if (tabs.length === 0) {
              sendResponse({ success: false, message: '未找到管理后台标签页，请先打开 https://heharse.cloud', errorType: 'NO_TAB' });
              return;
            }
            const tab = tabs[0];

            const results = await chrome.scripting.executeScript({
              target: { tabId: tab.id },
              func: async () => {
                try {
                  const resp = await fetch('/api/user/token', { credentials: 'same-origin' });
                  const json = await resp.json();
                  if (json.success && json.data) return { success: true, token: json.data };
                  const userStr = localStorage.getItem('user');
                  if (userStr) {
                    const user = JSON.parse(userStr);
                    if (user.access_token) return { success: true, token: user.access_token };
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
              sendResponse({ success: false, message: result?.message || '获取失败（需先打开 https://heharse.cloud 并登录）', errorType: 'AUTH' });
            }
          } catch (err) {
            sendResponse({ success: false, message: err.message, errorType: 'TAB_SCRIPT' });
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

        case 'getStats': {
          const result = await chrome.storage.local.get(STATS_KEY);
          sendResponse({ success: true, data: result[STATS_KEY] || {}, swVersion: SW_VERSION });
          break;
        }

        case 'resetStats': {
          await chrome.storage.local.remove(STATS_KEY);
          sendResponse({ success: true });
          break;
        }

        case 'testConnection': {
          // 诊断接口：测试 API 连通性，返回详细时序
          const config = await chrome.storage.local.get(STORAGE_KEY);
          const cfg = config[STORAGE_KEY] || {};
          const baseUrl = (cfg.apiBaseUrl || '').replace(/\/$/, '');
          if (!baseUrl) {
            sendResponse({ success: false, message: '请先配置 API Base URL', errorType: 'NO_CONFIG' });
            return;
          }
          const apiBase = baseUrl + (baseUrl.includes('/api') ? '' : '/api');
          const testUrl = `${apiBase}/prompt-category/all`;
          const t0 = Date.now();
          try {
            const resp = await fetchWithTimeout(testUrl, {
              method: 'GET',
              credentials: 'include',
              headers: { 'Content-Type': 'application/json' }
            }, 8000);
            const elapsed = Date.now() - t0;
            const text = await resp.text();
            sendResponse({
              success: true,
              url: testUrl,
              status: resp.status,
              ok: resp.ok,
              latencyMs: elapsed,
              bodyPreview: text.slice(0, 300),
              tip: resp.ok ? '✅ API 连通正常' : (resp.status === 401 ? '⚠️ API 可达但需认证（Token 可能过期）' : `⚠️ HTTP ${resp.status}`)
            });
          } catch (e) {
            sendResponse({
              success: false,
              url: testUrl,
              latencyMs: Date.now() - t0,
              errorType: e.errorType || 'NETWORK',
              message: friendlyApiError(e, testUrl),
              tip: e.errorType === 'TIMEOUT'
                ? '❌ 连接超时（heharse.cloud 不通或被墙，建议开全局代理）'
                : e.errorType === 'NETWORK'
                ? '❌ 网络不可达（DNS 解析失败或服务器无响应）'
                : `❌ ${e.errorType}: ${e.message}`
            });
          }
          break;
        }

        case 'apiRequest': {
          const config = await chrome.storage.local.get(STORAGE_KEY);
          const cfg = config[STORAGE_KEY] || {};
          const baseUrl = (cfg.apiBaseUrl || '').replace(/\/$/, '');
          const token = cfg.apiToken || '';

          if (!baseUrl) {
            sendResponse({ success: false, message: '请先配置 API Base URL', errorType: 'NO_CONFIG' });
            return;
          }

          const apiBase = baseUrl + (baseUrl.includes('/api') ? '' : '/api');
          const url = `${apiBase}${message.path}`;
          // 优先用 Token 鉴权（不带 cookie，避免 session 干扰）
          // 用户未填 Token 时才允许带 cookie（session 鉴权降级）
          const rawToken = (token || '').trim();
          const useToken = rawToken && rawToken !== '';
          const bearerToken = rawToken.startsWith('Bearer ') ? rawToken : `Bearer ${rawToken}`;
          const options = {
            method: message.method || 'GET',
            credentials: useToken ? 'omit' : 'include',
            headers: {
              'Content-Type': 'application/json',
              'New-API-User': message.userId || cfg.userId || '',
              ...(useToken ? { 'Authorization': bearerToken } : {})
            }
          };

          if (message.body) {
            options.body = JSON.stringify(message.body);
          }

          bumpStat('apiCalls', 1);
          console.log('[Background] API Request:', options.method, url);

          // ===== 带重试的 fetch =====
          let resp;
          let lastError;
          for (let attempt = 0; attempt <= FETCH_MAX_RETRIES; attempt++) {
            try {
              resp = await fetchWithTimeout(url, options);
              lastError = null;
              break; // 成功，跳出重试
            } catch (e) {
              lastError = e;
              const canRetry = (e.errorType === 'TIMEOUT' || e.errorType === 'NETWORK') && attempt < FETCH_MAX_RETRIES;
              console.warn(`[Background] API attempt ${attempt + 1} failed:`, e.errorType, e.message, canRetry ? '→ 重试中...' : '→ 不再重试');
              if (canRetry) {
                await new Promise(r => setTimeout(r, FETCH_RETRY_DELAY_MS));
                continue;
              }
            }
          }

          if (lastError) {
            bumpStat('apiErrors', 1);
            bumpStat('errorByType.' + (lastError.errorType || 'UNKNOWN'), 1);
            const msg = friendlyApiError(lastError, url);
            console.error('[Background] API 最终失败:', lastError.errorType, lastError.message, '| URL:', url);
            sendResponse({ success: false, message: msg, errorType: lastError.errorType || 'NETWORK', url });
            return;
          }

          const text = await resp.text();
          let data;
          try {
            data = JSON.parse(text);
          } catch (e) {
            data = { raw: text };
          }
          console.log('[Background] API Response:', resp.status, `(${Date.now() - start}ms)`, data);

          if (!resp.ok) {
            bumpStat('apiErrors', 1);
            const isAuth = resp.status === 401 || resp.status === 403;
            let friendlyMsg = `HTTP ${resp.status}`;
            if (isAuth) {
              friendlyMsg = `⚠️ 鉴权失败（HTTP ${resp.status}）— Token 可能已过期，请点击侧边栏「从后台获取 Token」重新获取`;
            } else {
              friendlyMsg = `HTTP ${resp.status}: ${data.message || text.slice(0, 200)}`;
            }
            sendResponse({
              success: false,
              message: friendlyMsg,
              errorType: isAuth ? 'AUTH'
                       : resp.status === 404 ? 'NOT_FOUND'
                       : resp.status >= 500 ? 'SERVER'
                       : 'HTTP_ERROR',
              status: resp.status,
              url
            });
            return;
          }
          sendResponse({ success: true, data, latencyMs: Date.now() - start, url });
          break;
        }

        case 'closeTab': {
          const targetTabId = message.tabId || sender.tab?.id;
          if (targetTabId) {
            try { await chrome.tabs.remove(targetTabId); } catch (e) { /* Tab 可能已关闭 */ }
          }
          sendResponse({ success: true, tabClosed: !!targetTabId });
          break;
        }

        case 'openSidePanel': {
          try {
            const windowId = sender.tab?.windowId;
            const tabId = sender.tab?.id;
            if (windowId) {
              await chrome.sidePanel.open({ windowId });
              if (tabId) {
                chrome.sidePanel.setOptions({ tabId, path: 'sidepanel.html', enabled: true }).catch(() => {});
              }
            } else if (tabId) {
              await chrome.sidePanel.open({ tabId });
            }
            sendResponse({ success: true });
          } catch (err) {
            console.warn('[Background] 打开侧边栏失败:', err.message);
            sendResponse({ success: false, message: err.message, errorType: 'SIDEPANEL' });
          }
          break;
        }

        case 'extractFromActiveTab': {
          try {
            const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
            if (!tab || !tab.id) {
              sendResponse({ success: false, message: '未找到活动标签页', errorType: 'NO_TAB' });
              return;
            }
            const res = await extractFromTab(tab.id);
            sendResponse(res);
          } catch (err) {
            sendResponse({ success: false, message: err.message, errorType: 'EXTRACT' });
          }
          break;
        }

        default:
          sendResponse({ success: false, message: '未知 action', errorType: 'UNKNOWN_ACTION' });
      }
    } catch (err) {
      // 顶层兜底
      const isContextInvalid = err.message?.includes('Extension context') || err.message?.includes('context invalidated');
      console.error('[Background] error:', message.action, err);
      sendResponse({
        success: false,
        message: isContextInvalid
          ? '扩展已休眠，请刷新页面或重新打开侧边栏'
          : (err.message || '未知错误'),
        errorType: isContextInvalid ? 'CONTEXT_INVALIDATED' : (err.name === 'AbortError' ? 'TIMEOUT' : 'UNKNOWN')
      });
    }
  })();

  return true;
});

// ===== 安装 / 启动 =====
chrome.runtime.onInstalled.addListener((details) => {
  console.log(`[PromptCollector] v${chrome.runtime.getManifest().version} installed (reason: ${details.reason})`);

  try {
    if (chrome.sidePanel && chrome.sidePanel.setPanelBehavior) {
      chrome.sidePanel.setPanelBehavior({ openPanelOnActionClick: true });
    }
  } catch (e) {
    console.warn('[Background] setPanelBehavior 跳过:', e?.message || '不可用');
  }

  try {
    chrome.contextMenus.removeAll(() => {
      chrome.contextMenus.create({
        id: 'collectPrompt',
        title: '🎯 采集此页面提示词',
        contexts: ['page', 'selection']
      });
    });
  } catch (e) {
    console.warn('[Background] 上下文菜单注册失败:', e?.message);
  }
});

chrome.runtime.onStartup.addListener(() => {
  console.log(`[PromptCollector] v${SW_VERSION} browser startup`);
  bumpStat('starts', 1);
});

// ===== 上下文菜单点击 =====
chrome.contextMenus.onClicked.addListener(async (info, tab) => {
  if (info.menuItemId === 'collectPrompt' && tab?.id) {
    const res = await extractFromTab(tab.id);
    if (res.success) {
      try {
        await chrome.sidePanel.open({ windowId: tab.windowId });
      } catch (e) {
        console.warn('[Background] 打开侧边栏失败:', e.message);
      }
    }
  }
});

// ===== 从指定 tab 提取数据 =====
async function extractFromTab(tabId) {
  try {
    console.log(`[Background] extractFromTab(${tabId}) 开始`);
    // 先尝试向页面发送消息（如果已有 content script 注入）
    const result = await chrome.tabs.sendMessage(tabId, { action: 'extractPrompt' }).catch(() => null);
    console.log('[Background] extractFromTab sendMessage 结果:', result);
    if (result && result.success && result.data && result.data.content && result.data.content.length > 30) {
      return result;
    }

    // 动态注入：先检查 tab 的 URL，决定注入哪个脚本
    let scriptFile = 'content-universal.js';
    try {
      const tab = await chrome.tabs.get(tabId);
      if (tab?.url?.includes('youmind.com')) scriptFile = 'content-youmind.js';
      if (tab?.url?.includes('opennana.com')) scriptFile = 'content-detail.js';
      if (tab?.url?.includes('mp.weixin.qq.com')) scriptFile = 'content-wechat.js';
    } catch (e) {}

    console.log(`[Background] 动态注入: ${scriptFile}`);
    await chrome.scripting.executeScript({
      target: { tabId },
      files: [scriptFile]
    });

    await new Promise(r => setTimeout(r, 800));
    const retryResult = await chrome.tabs.sendMessage(tabId, { action: 'extractPrompt' }).catch(() => null);
    console.log('[Background] extractFromTab 重试结果:', retryResult);
    if (retryResult?.success && retryResult?.data?.content?.length > 30) return retryResult;

    // 返回原始结果（即使 content 为空，也把 reason 带给调用方）
    return retryResult || result || { success: false, message: '未能从页面提取到提示词', errorType: 'EMPTY_RESULT' };
  } catch (e) {
    console.error('[Background] extractFromTab 异常:', e);
    return { success: false, message: e.message, errorType: 'EXTRACT' };
  }
}
