# 应用端（Rust）对接自托管更新接口指南

> 适配从 GitHub Releases 迁移到 New-API 自托管更新服务

---

## 一、升级检测接口变更

### 旧接口（GitHub Releases）
```
GET https://api.github.com/repos/{owner}/{repo}/releases/latest
```

### 新接口（New-API 自托管）
```
GET https://heharse.cloud/api/public/releases/latest
```

**无需鉴权**，应用端直接请求即可。

---

## 二、响应格式对比

两个接口返回的 JSON 结构基本一致，字段名兼容 GitHub Releases API：

```json
{
  "tag_name": "v1.2.3",
  "name": "v1.2.3",
  "body": "## 更新内容\n- 修复了xxx\n- 优化了yyy",
  "published_at": "2025-05-15T12:00:00Z",
  "assets": [
    {
      "id": 1,
      "name": "harsetv_1.2.3_windows_x86_64.exe",
      "size": 104857600,
      "browser_download_url": "https://heharse.cloud/api/public/releases/download/windows/x86_64"
    }
  ]
}
```

**应用端改动点：**
1. 把请求 URL 从 GitHub API 改成 `https://heharse.cloud/api/public/releases/latest`
2. 其余解析逻辑不变，`tag_name`、`body`、`assets`、`browser_download_url` 字段名完全一样

---

## 三、Rust 代码参考

### 3.1 升级检测

```rust
use reqwest;
use serde::Deserialize;

const UPDATE_API_URL: &str = "https://heharse.cloud/api/public/releases/latest";

#[derive(Debug, Deserialize)]
pub struct ReleaseAsset {
    pub id: u64,
    pub name: String,
    pub size: u64,
    pub browser_download_url: String,
}

#[derive(Debug, Deserialize)]
pub struct LatestRelease {
    pub tag_name: String,
    pub name: String,
    pub body: String,
    pub published_at: String,
    pub assets: Vec<ReleaseAsset>,
}

/// 检测是否有新版本
pub async fn check_update(current_version: &str) -> Result<Option<LatestRelease>, Box<dyn std::error::Error>> {
    let client = reqwest::Client::new();
    let release: LatestRelease = client
        .get(UPDATE_API_URL)
        .timeout(std::time::Duration::from_secs(10))
        .send()
        .await?
        .json()
        .await?;

    // 版本号对比（去掉 v 前缀）
    let latest = release.tag_name.trim_start_matches('v');
    let current = current_version.trim_start_matches('v');

    if latest > current {
        Ok(Some(release))
    } else {
        Ok(None)
    }
}
```

### 3.2 根据平台选择对应安装包

```rust
/// 根据当前平台选择对应的 asset
pub fn select_asset(release: &LatestRelease) -> Option<&ReleaseAsset> {
    let (platform, arch) = get_current_platform_arch();

    release.assets.iter().find(|asset| {
        asset.name.contains(&platform) && asset.name.contains(&arch)
    })
}

fn get_current_platform_arch() -> (String, String) {
    let platform = if cfg!(target_os = "windows") {
        "windows"
    } else if cfg!(target_os = "macos") {
        "darwin"
    } else {
        "linux"
    };

    let arch = if cfg!(target_arch = "x86_64") {
        "x86_64"
    } else if cfg!(target_arch = "aarch64") {
        "aarch64"
    } else {
        "x86_64"
    };

    (platform.to_string(), arch.to_string())
}
```

### 3.3 下载安装包

```rust
use std::fs::File;
use std::io::copy;

/// 下载安装包到本地临时目录
pub async fn download_update(asset: &ReleaseAsset, temp_dir: &std::path::Path) -> Result<std::path::PathBuf, Box<dyn std::error::Error>> {
    let client = reqwest::Client::new();
    let response = client
        .get(&asset.browser_download_url)
        .timeout(std::time::Duration::from_secs(300))
        .send()
        .await?;

    let file_path = temp_dir.join(&asset.name);
    let mut file = File::create(&file_path)?;
    let content = response.bytes().await?;
    copy(&mut content.as_ref(), &mut file)?;

    Ok(file_path)
}
```

### 3.4 强制升级判断

```rust
/// 调用接口时带上请求头，后端可返回强制升级标记
/// 当前接口 body 字段不含 is_force，如需支持需扩展接口或走管理配置
/// 建议：在应用本地配置中维护一个最小版本号，后端可后续扩展
```

> **关于强制升级：** 目前 `GET /api/public/releases/latest` 返回的是 GitHub 兼容格式，不含 `is_force` 字段。如果需要强制升级功能，建议：
> 1. 方案 A：扩展接口返回 `is_force: true`（需后端改造）
> 2. 方案 B：在应用配置文件/远程配置中维护一个 `min_required_version`，应用启动时对比

---

## 四、打包构建时注意事项

### 4.1 文件名规范

上传安装包时，文件名必须包含 **平台标识** 和 **架构标识**，以便应用端识别：

```
harsetv_1.2.3_windows_x86_64.exe
harsetv_1.2.3_windows_aarch64.exe
harsetv_1.2.3_darwin_x86_64.dmg
harsetv_1.2.3_darwin_aarch64.dmg
harsetv_1.2.3_linux_x86_64.AppImage
harsetv_1.2.3_linux_aarch64.AppImage
```

### 4.2 构建脚本示例（GitHub Actions）

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  build-windows:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v4
      - name: Build
        run: cargo build --release
      - name: Package
        run: |
          # NSIS 打包或其他打包方式
          mv target/release/harsetv.exe harsetv_${{ github.ref_name }}_windows_x86_64.exe
      - name: Upload to New-API
        run: |
          curl -X POST "https://heharse.cloud/api/admin/releases" \
            -H "Authorization: Bearer ${{ secrets.ADMIN_TOKEN }}" \
            -F "version=${{ github.ref_name }}" \
            -F "tag=${{ github.ref_name }}" \
            -F "platform=windows" \
            -F "arch=x86_64" \
            -F "file=@harsetv_${{ github.ref_name }}_windows_x86_64.exe"

  build-macos:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v4
      - name: Build
        run: cargo build --release --target aarch64-apple-darwin
      - name: Package
        run: |
          # 打包 dmg
          mv target/aarch64-apple-darwin/release/harsetv harsetv_${{ github.ref_name }}_darwin_aarch64.dmg
      - name: Upload to New-API
        run: |
          curl -X POST "https://heharse.cloud/api/admin/releases" \
            -H "Authorization: Bearer ${{ secrets.ADMIN_TOKEN }}" \
            -F "version=${{ github.ref_name }}" \
            -F "tag=${{ github.ref_name }}" \
            -F "platform=darwin" \
            -F "arch=aarch64" \
            -F "file=@harsetv_${{ github.ref_name }}_darwin_aarch64.dmg"
```

---

## 五、对接验收清单

给应用端开发者的 checklist：

- [ ] 把升级检测 URL 改为 `https://heharse.cloud/api/public/releases/latest`
- [ ] 保持 `tag_name`、`body`、`assets` 解析逻辑不变
- [ ] 根据当前系统平台（`target_os`）和架构（`target_arch`）选择对应 asset
- [ ] 下载地址使用 `browser_download_url`（会自动重定向到正确的安装包）
- [ ] 构建产物文件名包含平台和架构标识
- [ ] CI/CD 中添加上传到 New-API 的脚本

---

## 六、常见问题

**Q: 如果同版本有多平台包，`latest` 接口会返回多个 asset 吗？**
A: 会。`assets` 数组会包含所有标记为 `is_latest = 1` 的安装包（Windows + macOS + Linux）。应用端根据平台过滤即可。

**Q: 下载接口支持断点续传吗？**
A: `GET /api/public/releases/download/:platform/:arch` 使用 Gin 的 `c.File()` 返回文件，支持 HTTP Range 请求（取决于浏览器/客户端实现）。

**Q: 如何测试？**
A: 在后台「安装包管理」页面上传一个测试版本，然后访问 `https://heharse.cloud/api/public/releases/latest` 查看返回结果。
