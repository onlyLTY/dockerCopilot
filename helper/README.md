# Docker Copilot 守护进程辅助程序

此辅助程序用于 Linux 主机上的本地 Docker Engine，可让 Docker Copilot 读取和修改 Docker 代理配置，并按需重启 Docker。

仅支持 Linux 本地 Docker Engine，不支持 Docker Desktop、远程 Docker、Rootless Docker 或 Windows。

## 安装辅助程序

在项目根目录执行：

```bash
cd helper
go build -o /usr/local/libexec/dockercopilot-helper .
cd ..
install -m 0644 helper/dockercopilot-helper.service /etc/systemd/system/
groupadd --system dockercopilot 2>/dev/null || true
systemctl daemon-reload
systemctl enable --now dockercopilot-helper.service
```

## 挂载套接字

编辑 `docker/docker-compose.yml`，在 `dockercopilot.volumes` 下增加一行：

```yaml
- /run/dockercopilot-helper.sock:/run/dockercopilot-helper.sock
```

然后重新创建容器：

```bash
docker compose -f docker/docker-compose.yml up -d
```

辅助程序只允许更新 Docker 的以下代理配置：

- `proxies.http-proxy`
- `proxies.https-proxy`
- `proxies.no-proxy`

它不会接受文件路径、Shell 命令或其他任意 Docker 配置。更新配置前会自动备份到 `/var/lib/dockercopilot/daemon.json.bak`。修改配置和重启 Docker 都需要在页面中确认，重启 Docker 可能会中断主机上所有容器的 Docker 操作。
