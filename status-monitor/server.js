const http = require('http');
const fs = require('fs');
const path = require('path');
const { chromium } = require('playwright');

const PORT = 3456;
const DATA_FILE = path.join(__dirname, 'status.json');
const PUBLIC_DIR = path.join(__dirname, 'public');

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
        category: currentCategory,
        name: modelName,
        status: statusText,
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

function serveFile(res, filePath, contentType) {
  fs.readFile(filePath, (err, data) => {
    if (err) {
      res.writeHead(404);
      res.end('Not Found');
      return;
    }
    res.writeHead(200, { 'Content-Type': contentType });
    res.end(data);
  });
}

const server = http.createServer((req, res) => {
  res.setHeader('Access-Control-Allow-Origin', '*');

  if (req.url === '/api/status') {
    res.writeHead(200, { 'Content-Type': 'application/json; charset=utf-8' });
    if (latestData) {
      res.end(JSON.stringify(latestData));
    } else {
      res.end(JSON.stringify({ error: '数据尚未获取，请等待首次抓取完成' }));
    }
    return;
  }

  if (req.url === '/' || req.url === '/index.html') {
    serveFile(res, path.join(PUBLIC_DIR, 'index.html'), 'text/html; charset=utf-8');
    return;
  }

  res.writeHead(404);
  res.end('Not Found');
});

(async () => {
  if (!fs.existsSync(PUBLIC_DIR)) {
    fs.mkdirSync(PUBLIC_DIR, { recursive: true });
  }

  console.log('🚀 启动状态监控服务...');
  console.log('⏳ 执行首次抓取（约需 5-10 秒）...');

  await fetchStatus();

  setInterval(fetchStatus, 60000);
  console.log('⏰ 已设置每 60 秒自动抓取');

  server.listen(PORT, () => {
    console.log(`\n🌐 监控面板: http://localhost:${PORT}`);
    console.log(`📡 API 接口: http://localhost:${PORT}/api/status\n`);
  });
})();
