package utiles

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

const containerStatsTimeout = 3 * time.Second

// ContainerUsage 保存一个容器的 CPU/内存使用情况（用于列表展示）。
type ContainerUsage struct {
	CPUUsage    string // 例：12.3%
	MemoryUsage string // 例：1.8GB / 8GB
}

// GetContainerUsage 通过 Docker 的非流式 stats 接口获取某个运行中容器的 CPU 与内存使用率。
// 非流式接口会返回两个采样点，第二个采样点包含 precpu_stats，可用于计算瞬时 CPU 使用率。
func GetContainerUsage(taskCtx context.Context, ctx *svc.ServiceContext, id string) (ContainerUsage, error) {
	var usage ContainerUsage
	if err := requireDocker(ctx); err != nil {
		return usage, err
	}

	statsCtx, cancel := context.WithTimeout(taskCtx, containerStatsTimeout)
	defer cancel()
	reader, err := ctx.DockerClient.ContainerStats(statsCtx, id, false)
	if err != nil {
		return usage, err
	}
	defer reader.Body.Close()

	var stats container.StatsResponse
	if err := json.NewDecoder(reader.Body).Decode(&stats); err != nil {
		return usage, err
	}
	usage.CPUUsage = calcCPUPercent(stats, reader.OSType)
	usage.MemoryUsage = calcMemoryUsage(stats, reader.OSType)
	return usage, nil
}

// calcCPUPercent 计算容器占宿主机总 CPU 资源的比例，结果范围通常为 0% 到 100%。
func calcCPUPercent(stats container.StatsResponse, osType string) string {
	var cpuPercent float64
	if osType == "windows" {
		if stats.CPUStats.CPUUsage.TotalUsage < stats.PreCPUStats.CPUUsage.TotalUsage {
			return ""
		}
		intervals := stats.Read.Sub(stats.PreRead).Nanoseconds() / 100
		if intervals > 0 {
			cpuDelta := stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage
			cpuPercent = float64(cpuDelta) / float64(intervals) * 100.0
		}
	} else {
		if stats.CPUStats.CPUUsage.TotalUsage < stats.PreCPUStats.CPUUsage.TotalUsage ||
			stats.CPUStats.SystemUsage < stats.PreCPUStats.SystemUsage {
			return ""
		}
		cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
		systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)
		if systemDelta > 0 && cpuDelta > 0 {
			cpuPercent = (cpuDelta / systemDelta) * 100.0
		}
	}
	if cpuPercent > 100 {
		cpuPercent = 100
	}
	return fmt.Sprintf("%.1f%%", cpuPercent)
}

// calcMemoryUsage 使用 docker stats 的内存口径，Linux 下排除 inactive file/page cache。
func calcMemoryUsage(stats container.StatsResponse, osType string) string {
	var usage uint64
	if osType == "windows" {
		usage = stats.MemoryStats.PrivateWorkingSet
	} else {
		usage = stats.MemoryStats.Usage
		if inactive, ok := stats.MemoryStats.Stats["total_inactive_file"]; ok && inactive < usage {
			usage -= inactive
		} else if inactive := stats.MemoryStats.Stats["inactive_file"]; inactive < usage {
			usage -= inactive
		}
	}
	if usage == 0 {
		return ""
	}
	return formatBytes(usage)
}

// formatBytes 将字节数格式化为易读的容量字符串。
func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
