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
    privileged: true
    network_mode: bridge
    ports:
      - 12712:12712
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./data:/data
    environment:
      - TZ=Asia/Shanghai
      - DOCKER_HOST=unix:///var/run/docker.sock
      - secretKey=密码，不少于八位且非纯数字
```

## Compose 项目管理

Compose 项目扫描、编辑和部署需要将宿主机的 Compose 根目录挂载到容器内 `/compose`，并在配置中设置 `Compose.ScanPaths`。例如：

```yaml
volumes:
  - /宿主机/docker目录:/compose
  - ./data:/data
```

Compose 文件管理会限制在配置的扫描根目录内。部署功能依赖宿主机可用的 `docker compose` 插件；上传或编辑的文件会先经过 YAML/Compose 校验和高风险配置检查。

当前服务通过 Docker socket 管理宿主机容器，等同于较高的 Docker 管理权限。生产环境建议限制管理端口访问来源，优先使用 Docker socket 代理，并评估移除 `privileged: true`、启用 `no-new-privileges` 和最小 capabilities。

## 开发环境

go版本：1.21+

前端源码位于 `frontend/`。构建前端：

```bash
cd frontend
npm ci
npm run build
```

构建产物直接写入仓库根目录 `dist/browser/`，Go 后端会从该目录嵌入管理界面。后端本地编译前需要先生成该目录；`dist/` 和 `build/` 均为构建产物目录，不应提交。

