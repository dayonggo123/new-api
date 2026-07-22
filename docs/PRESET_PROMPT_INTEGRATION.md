# 预设提示词全局缓存集成方案（Tauri Rust）

## 方案概述

在 Tauri Rust 层维护一个全局缓存，前端通过 `invoke` 命令读取缓存数据。

- **缓存策略**：TTL 5 分钟，过期自动刷新
- **降级策略**：网络失败时返回过期缓存，绝不阻断用户操作
- **并发安全**：`tokio::sync::RwLock` 保证多线程安全

---

## 1. Rust 端实现

### Cargo.toml 新增依赖

```toml
[dependencies]
# 已有依赖保持不变，新增：
lazy_static = "1.4"
reqwest = { version = "0.12", features = ["json"] }
serde = { version = "1.0", features = ["derive"] }
serde_json = "1.0"
tokio = { version = "1", features = ["sync", "time"] }
```

### src/preset_prompt.rs（新建文件）

```rust
use std::sync::Arc;
use std::time::{Duration, Instant};
use tokio::sync::RwLock;
use serde::{Deserialize, Serialize};

// ============ 数据模型 ============

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PresetPrompt {
    pub id: i64,
    pub name: String,
    pub system_prompt: String,
    pub user_prompt: String,
    pub description: String,
    pub category: String,
    pub status: i32,
    pub sort_order: i32,
}

// ============ 全局缓存 ============

struct CacheState {
    data: Vec<PresetPrompt>,
    last_updated: Instant,
}

lazy_static::lazy_static! {
    static ref CACHE: Arc<RwLock<CacheState>> = Arc::new(RwLock::new(CacheState {
        data: Vec::new(),
        last_updated: Instant::now() - Duration::from_secs(3600),
    }));
}

const CACHE_TTL: Duration = Duration::from_secs(300); // 5分钟
const API_URL: &str = "https://heharse.cloud/public/preset-prompts";

// ============ 内部方法 ============

async fn fetch_from_remote() -> Result<Vec<PresetPrompt>, String> {
    let client = reqwest::Client::builder()
        .timeout(Duration::from_secs(10))
        .build()
        .map_err(|e| e.to_string())?;

    let res = client
        .get(API_URL)
        .send()
        .await
        .map_err(|e| format!("网络请求失败: {}", e))?;

    let body: serde_json::Value = res
        .json()
        .await
        .map_err(|e| format!("JSON解析失败: {}", e))?;

    if !body["success"].as_bool().unwrap_or(false) {
        return Err(body["message"]
            .as_str()
            .unwrap_or("上游返回错误")
            .to_string());
    }

    let data = serde_json::from_value(body["data"].clone())
        .map_err(|e| format!("数据格式不匹配: {}", e))?;

    Ok(data)
}

// ============ Tauri 命令 ============

#[tauri::command]
pub async fn get_preset_prompts(force_refresh: bool) -> Result<Vec<PresetPrompt>, String> {
    // 1. 读锁检查缓存是否有效
    {
        let cache = CACHE.read().await;
        let is_fresh = cache.last_updated.elapsed() < CACHE_TTL;
        let has_data = !cache.data.is_empty();

        if !force_refresh && is_fresh && has_data {
            return Ok(cache.data.clone());
        }
    } // 读锁在此释放

    // 2. 获取写锁，重新检查（防止多个请求同时刷新）
    let mut cache = CACHE.write().await;
    let is_fresh = cache.last_updated.elapsed() < CACHE_TTL;
    let has_data = !cache.data.is_empty();

    if !force_refresh && is_fresh && has_data {
        return Ok(cache.data.clone());
    }

    // 3. 从远程拉取
    match fetch_from_remote().await {
        Ok(data) => {
            cache.data = data.clone();
            cache.last_updated = Instant::now();
            Ok(data)
        }
        Err(e) => {
            // 降级：网络失败时返回过期缓存（如果有数据）
            if !cache.data.is_empty() {
                Ok(cache.data.clone())
            } else {
                Err(e)
            }
        }
    }
}

#[tauri::command]
pub async fn get_preset_prompt_by_id(id: i64) -> Result<Option<PresetPrompt>, String> {
    let list = get_preset_prompts(false).await?;
    Ok(list.into_iter().find(|p| p.id == id))
}
```

### main.rs 注册命令

```rust
mod preset_prompt;

fn main() {
    tauri::Builder::default()
        .invoke_handler(tauri::generate_handler![
            // 你已有的命令...
            preset_prompt::get_preset_prompts,
            preset_prompt::get_preset_prompt_by_id,
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
```

---

## 2. 前端调用（TypeScript）

### types/preset-prompt.ts

```typescript
export interface PresetPrompt {
  id: number;
  name: string;
  system_prompt: string;
  user_prompt: string;
  description: string;
  category: string;
  status: number;
  sort_order: number;
}
```

### services/preset-prompt.ts

```typescript
import { invoke } from '@tauri-apps/api/core';
import type { PresetPrompt } from '../types/preset-prompt';

/**
 * 获取预设提示词列表（带全局缓存）
 * @param forceRefresh 是否强制刷新缓存
 */
export async function getPresetPrompts(forceRefresh = false): Promise<PresetPrompt[]> {
  return await invoke<PresetPrompt[]>('get_preset_prompts', { forceRefresh });
}

/**
 * 根据 ID 获取单个预设提示词
 */
export async function getPresetPromptById(id: number): Promise<PresetPrompt | null> {
  return await invoke<PresetPrompt | null>('get_preset_prompt_by_id', { id });
}
```

### 使用示例

```typescript
import { getPresetPrompts, getPresetPromptById } from './services/preset-prompt';

// 1. 页面加载时读取缓存（或首次加载）
const presets = await getPresetPrompts();

// 2. 用户下拉选择预设
const selected = presets.find(p => p.id === selectedId);

// 3. 构造 messages 调用聊天接口
const messages = [
  ...(selected.system_prompt ? [{ role: 'system', content: selected.system_prompt }] : []),
  ...(selected.user_prompt ? [{ role: 'user', content: selected.user_prompt }] : []),
];

await generateOpenAIImage({
  model: 'gpt-4o',
  messages,
  // ...
});

// 4. 手动刷新缓存（如用户点击"刷新"按钮）
const fresh = await getPresetPrompts(true);
```

---

## 3. 缓存行为说明

| 场景 | 行为 |
|---|---|
| 首次调用 | 请求远程 API，写入缓存，返回数据 |
| 5 分钟内再次调用 | 直接返回缓存数据，**零网络请求** |
| 缓存过期 + 网络正常 | 自动刷新缓存，返回最新数据 |
| 缓存过期 + 网络失败 | **降级返回过期缓存**，不报错 |
| 首次调用 + 网络失败 | 返回错误，前端需处理空状态 |
| `forceRefresh=true` | 无视 TTL，强制请求远程 API |

---

## 4. 进阶：应用启动时预加载

在 `main.rs` 中应用启动时静默预热缓存：

```rust
use tauri::Manager;

fn main() {
    tauri::Builder::default()
        .setup(|app| {
            // 启动时静默预加载预设提示词
            let handle = app.handle().clone();
            tauri::async_runtime::spawn(async move {
                let _ = preset_prompt::get_preset_prompts(false).await;
            });
            Ok(())
        })
        .invoke_handler(tauri::generate_handler![...])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
```
