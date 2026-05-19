const http = require('http');
const fs = require('fs');
const path = require('path');
const { chromium } = require('playwright');

const PORT = process.env.STATUS_PORT || 3456;
const DATA_FILE = path.join(__dirname, 'status.json');
const ALLOWED_ORIGINS = process.env.ALLOWED_ORIGINS
  ? process.env.ALLOWED_ORIGINS.split(',')
  : ['*'];

let latestData = null;
let lastFetchTime = null;
let isFetching = false;

const CHROME_PATH = process.platform === 'win32'
  ? 'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe'
  : undefined;

function parseStatusPage(text) {
  const lines = text.split('\n').map(l => l.trim()).filter(l => l.length > 0);
  const models = [];
  let currentCategory = '';

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    if (line === 'TẠO VIDEO' || line === 'TẠO ẢNH') {
      currentCategory = line;
    }
    const rateMatch = line.match(/Success rate: ([\d.]+)%/);
    if (rateMatch) {
      const modelName = lines[i - 2] || '';
      const statusText = lines[i - 1] || '';
      models.push({
        category: currentCategory === 'TẠO VIDEO' ? 'video' : 'image',
        categoryRaw: currentCategory,
        name: modelName,
        status: statusText,
        statusCode: statusText.includes('Hoạt động') ? 'operational'
          : statusText.includes('Hiệu suất suy giảm') ? 'degraded'
          : statusText.includes('gián đoạn') ? 'partial_outage'
          : 'unknown',
        successRate: parseFloat(rateMatch[1])
      });
    }
  }
  return models;
}

async function fetchStatus() {
  if (isFetching) return;
  isFetching = true;

  const launchOpts = { headless: true };
  if (CHROME_PATH && fs.existsSync(CHROME_PATH)) {
    launchOpts.executablePath = CHROME_PATH;
  }

  let browser;
  try {
    browser = await chromium.launch(launchOpts);
    const page = await browser.newPage();
    await page.goto('https://geminigen.ai/status/', {
      waitUntil: 'networkidle',
      timeout: 30000
    });
    await page.waitForTimeout(3000);

    const text = await page.evaluate(() => document.body.innerText);
    const models = parseStatusPage(text);

    const data = {
      fetchedAt: new Date().toISOString(),
      source: 'https://geminigen.ai/status/',
      total: models.length,
      models
    };

    latestData = data;
    lastFetchTime = Date.now();
    fs.writeFileSync(DATA_FILE, JSON.stringify(data, null, 2));
    console.log(`[${new Date().toLocaleString('zh-CN')}] ✅ 抓取成功，共 ${models.length} 个模型`);
  } catch (e) {
    console.error(`[${new Date().toLocaleString('zh-CN')}] ❌ 抓取失败: ${e.message}`);
  } finally {
    if (browser) await browser.close();
    isFetching = false;
  }
}

function setCORS(res, origin) {
  const allowOrigin = ALLOWED_ORIGINS.includes('*') ? origin || '*'
    : ALLOWED_ORIGINS.includes(origin) ? origin
    : ALLOWED_ORIGINS[0];
  res.setHeader('Access-Control-Allow-Origin', allowOrigin);
  res.setHeader('Access-Control-Allow-Methods', 'GET, OPTIONS');
  res.setHeader('Access-Control-Allow-Headers', 'Content-Type');
  res.setHeader('Access-Control-Max-Age', '86400');
}

const server = http.createServer((req, res) => {
  const origin = req.headers.origin || '';
  setCORS(res, origin);

  if (req.method === 'OPTIONS') {
    res.writeHead(204);
    res.end();
    return;
  }

  if (req.url === '/api/status' || req.url === '/api/status/') {
    res.writeHead(200, { 'Content-Type': 'application/json; charset=utf-8' });
    if (latestData) {
      res.end(JSON.stringify(latestData));
    } else {
      res.end(JSON.stringify({
        error: '数据尚未获取',
        message: '首次抓取需要 5-10 秒，请稍后再试'
      }));
    }
    return;
  }

  if (req.url === '/health') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({
      status: 'ok',
      lastFetch: lastFetchTime ? new Date(lastFetchTime).toISOString() : null,
      modelCount: latestData ? latestData.total : 0
    }));
    return;
  }

  res.writeHead(404, { 'Content-Type': 'application/json' });
  res.end(JSON.stringify({ error: 'Not Found', available: ['/api/status', '/health'] }));
});

(async () => {
  console.log('🚀 GeminiGen Status API 服务启动...');
  await fetchStatus();
  setInterval(fetchStatus, 60000);

  server.listen(PORT, () => {
    console.log(`\n📡 API: http://localhost:${PORT}/api/status`);
    console.log(`💓 Health: http://localhost:${PORT}/health\n`);
  });
})();
