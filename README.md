# dockerCopilot
<a href="https://www.gnu.org/licenses/agpl-3.0.en.html">
    <img alt="License: AGPLv3" src="https://shields.io/badge/License-AGPL%20v3-blue.svg">
  </a>

# 介绍

一个主打便捷的docker容器管理工具，现在已经支持所有平台。
已经实现：
1. 一键更新容器
2. 指定镜像和tag更新
3. 启动、停止、重启容器
4. 重命名容器
5. 删除无TAG镜像
6. 删除未使用镜像
7. 更新进度查看
8. 备份容器设置
9. 恢复容器设置

## 使用

docker compose 安装

```
services:
  dockercopilot:
    container_name: dockercopilot
    restart: always
    network_mode: bridge
    ports:
      - 127.0.0.1:12712:12712
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./data:/data
    environment:
      - TZ=Asia/Shanghai
      - DOCKER_HOST=unix:///var/run/docker.sock
      - secretKey=请设置你自己的登录密码
    image: 0nlylty/dockercopilot:latest
```

`privileged` 不是必需项；请只挂载 Docker Socket，并限制管理端口的网络访问。注意：挂载 Docker Socket 实际上授予了主机级管理权限，因此不要把管理端口直接暴露到公网。生产环境建议通过反向代理启用 HTTPS，或同时设置 `TLS_CERT_FILE` 与 `TLS_KEY_FILE`。`secretKey` 完全由用户决定；短密码和纯数字不会阻止启动，但程序会输出安全建议。为降低口令猜测风险，仍建议使用不少于 32 个字符的随机密码。备份默认使用 `secretKey` 加密，修改加密密钥后旧备份将无法解密。

可选安全配置：

- `CORS_ALLOWED_ORIGINS`：逗号分隔的可信前端 Origin；默认不开放跨域。
- `BACKUP_ENCRYPTION_KEY`：用户可自行设置的独立备份密钥；短密钥不会阻止使用，生产环境仍建议使用不少于 32 个字符的随机值并妥善保存。
- `TLS_CERT_FILE`、`TLS_KEY_FILE`：同时设置时由服务直接启用 TLS。
- `DOCKER_AUTH_CONFIG` 或 `DOCKER_CONFIG`：检查私有 Registry 镜像更新时使用的 Docker 凭据。
- `UPDATE_CHANNEL`：更新频道，默认 `latest`；`githubProxy` 如设置，必须生成 HTTPS 地址。

## 开发环境

Go 版本：1.25.13+

