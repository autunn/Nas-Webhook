<p align="center">
  <img src="https://raw.githubusercontent.com/autunn/NasWebhook/main/logo.png" width="180" alt="NasWebhook Logo" />
</p>

<p align="center">
  <h2>NasWebhook</h2>
  <p>连接各类品牌 NAS 与 企业微信的通用化消息桥梁</p>
</p>

<p align="center">
  <a href="https://github.com/autunn/NasWebhook">
    <img src="https://img.shields.io/badge/GitHub-Source%20Code-000000?style=flat-square&logo=github" alt="GitHub" />
  </a>
  <a href="https://hub.docker.com/r/autunn/nas-webhook">
    <img src="https://img.shields.io/docker/pulls/autunn/nas-webhook?style=flat-square&logo=docker&color=0db7ed" alt="Docker Pulls" />
  </a>
  <a href="https://hub.docker.com/r/autunn/nas-webhook">
    <img src="https://img.shields.io/docker/image-size/autunn/nas-webhook?style=flat-square&logo=docker" alt="Docker Image Size" />
  </a>
  <img src="https://img.shields.io/badge/License-MIT-2ecc71?style=flat-square" alt="License" />
</p>


## 简介 (Introduction)

[cite_start]NasWebhook 是一个轻量级的通知转发服务 [cite: 21, 23][cite_start]。它能接收来自不同 NAS 系统（如绿联 UGOS Pro、群晖 DSM、极空间）或下载工具（如 qBittorrent）的 Webhook 信号，并将其转化为精美的图文消息推送到你的企业微信 [cite: 22]。

## 特性 (Features)

- [cite_start]**全平台兼容**：支持所有能自定义 JSON Webhook 的系统与软件 [cite: 21, 22]。
- [cite_start]**现代化 UI**：内置简洁的管理后台，支持在线配置微信参数 [cite: 21]。
- [cite_start]**安全加固**：支持 `ADMIN_PASSWORD` 身份验证，并严格遵循企业微信消息加密协议 [cite: 21]。
- **自动化构建**：支持 Docker 多架构（amd64/arm64）自动构建并推送至 Docker Hub。
- **动态封面**：支持接入随机动漫图片 API，让每一条通知都独一无二。

## 快速启动 (Quick Start)

### Docker Compose (推荐)

```yaml
version: '3'
services:
  webhook:
    image: autunn/nas-webhook:latest
    container_name: nas-webhook
    ports:
      - "5080:5080"
    volumes:
      - ./data:/app/data
    environment:
      - ADMIN_PASSWORD=admin # 修改为你自己的后台密码
      - TZ=Asia/Shanghai
    restart: always

```

### Docker CLI

```bash
docker run -d \
  --name nas-webhook \
  -p 5080:5080 \
  -v $(pwd)/data:/app/data \
  -e ADMIN_PASSWORD=admin \
  --restart always \
  autunn/nas-webhook:latest

```

## 配置指南 (Configuration)

1. 
**进入后台**：访问 `http://IP:5080`，使用密码登录 。


2. **企业微信配置**：在后台填写 `CorpID`、`AgentID`、`Secret`、`Token` 和 `EncodingAESKey`。
3. **设置图片 API**：推荐填入 `https://api.vvhan.com/api/wallpaper/views?type=aliyun` 获取随机动漫封面。
4. **测试连接**：
```powershell
curl -X POST -H "Content-Type: application/json" -d "{\"message\":\"NasWebhook 测试通知\"}" http://你的IP:5080/webhook

```



## 挂载卷 (Volumes)

* 
`/app/data`：用于持久化存储 `config.json` 配置文件 。
