package utiles

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

const containerStatsTimeout = 3 * time.Second

// statsCollectInterval 活跃期间的采样周期。略小于前端列表刷新间隔（20s），保证页面读到足够新的样本。
const statsCollectInterval = 15 * time.Second

// statsIdleAfter 最后一次读取超过该时长即视为无人访问，采集器暂停采样（对 Docker 零调用）。
// 需容忍两类活跃间隙：切去任务/镜像等其他页面后返回（stats 只有容器页展示），
// 以及容器页批量操作期间软刷新被跳过（busy/checking 可持续数分钟）。
// 空闲成本随阈值线性增长但总量极小（罕见访问场景下每天多几次毫秒级调用），取宽不取窄。
const statsIdleAfter = 2 * time.Minute

// statsSampleMaxAge 缓存样本的最大可信时长，需大于采样周期。
// 超过则读取方视为过期：回退为同步 one-shot 采集并唤醒采集器。
const statsSampleMaxAge = 2 * statsCollectInterval

// statsCollectWorkers 单轮采样并发数
const statsCollectWorkers = 8

// ContainerUsage 保存一个容器的 CPU/内存使用情况（用于列表展示）。
type ContainerUsage struct {
	CPUUsage    string // 例：12.3%
	MemoryUsage string // 例：1.8GB / 8GB
}

// statsSample 一次 one-shot stats 采样的原始数据。
// CPU 使用率需要相邻两个采样点的计数器差值，因此缓存原始计数器而非仅结果字符串。
type statsSample struct {
	read        time.Time
	osType      string
	cpuTotal    uint64
	systemTotal uint64
	usage       ContainerUsage
}

var (
	statsMu         sync.RWMutex
	statsSamples    = map[string]statsSample{} // containerID -> 最近一次采样
	statsWake       = make(chan struct{}, 1)   // 有读取时唤醒暂停中的采集器（最多积压 1 个）
	lastStatsReadAt atomic.Int64               // 最近一次 RefreshContainerUsage 的时间（unix nano）
)

// StartStatsCollector 阻塞式后台任务：对运行中容器做 one-shot stats 采样，
// 列表接口读缓存即可返回，避免同步等待非流式 stats 的 1-2 秒/容器延迟。
// 按需工作：超过 statsIdleAfter 无人读取时暂停采样、对 Docker 零调用；
// 任一读取会唤醒采集器立即补一轮，之后保持活跃直至再次空闲。
func StartStatsCollector(ctx *svc.ServiceContext) {
	ticker := time.NewTicker(statsCollectInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
		case <-statsWake:
		}
		if time.Since(time.Unix(0, lastStatsReadAt.Load())) > statsIdleAfter {
			continue // 无人访问，保持暂停
		}
		collectAllStats(context.Background(), ctx)
	}
}

// collectAllStats 采集一轮：列出运行中容器 → 并发 one-shot 采样 → 与上一轮样本算 CPU% 后整体替换缓存。
func collectAllStats(taskCtx context.Context, ctx *svc.ServiceContext) {
	if ctx == nil || ctx.DockerClient == nil {
		return
	}
	listCtx, cancel := context.WithTimeout(taskCtx, containerStatsTimeout)
	dockerList, err := ctx.DockerClient.ContainerList(listCtx, container.ListOptions{All: false})
	cancel()
	if err != nil {
		logx.Debugf("stats 采集获取容器列表失败: %v", err)
		return
	}
	ids := make([]string, 0, len(dockerList))
	for _, c := range dockerList {
		if c.State == "running" {
			ids = append(ids, c.ID)
		}
	}

	type sampleResult struct {
		id     string
		sample statsSample
	}
	jobs := make(chan string, len(ids))
	results := make(chan sampleResult, len(ids))
	workers := statsCollectWorkers
	if workers > len(ids) {
		workers = len(ids)
	}
	if workers < 1 {
		workers = 1
	}
	for w := 0; w < workers; w++ {
		go func() {
			for id := range jobs {
				sample, sampleErr := oneShotSample(taskCtx, ctx, id)
				if sampleErr != nil {
					// 容器可能刚好退出等瞬时情况；读为空值即丢弃，不刷日志
					results <- sampleResult{id: id}
					continue
				}
				results <- sampleResult{id: id, sample: sample}
			}
		}()
	}
	for _, id := range ids {
		jobs <- id
	}
	close(jobs)

	// 只保留本轮仍在运行的容器样本，容器停止/删除后其条目自然清除
	fresh := make(map[string]statsSample, len(ids))
	statsMu.RLock()
	for range ids {
		r := <-results
		if r.sample.read.IsZero() {
			continue
		}
		if prev, ok := statsSamples[r.id]; ok && cpuGapUsable(prev, r.sample) {
			r.sample.usage.CPUUsage = cpuPercentBetween(prev, r.sample)
		}
		fresh[r.id] = r.sample
	}
	statsMu.RUnlock()
	statsMu.Lock()
	statsSamples = fresh
	statsMu.Unlock()
}

// oneShotSample 获取单个容器的一次性采样（不等 Docker 采样，毫秒级）。
// 内存占用单样本即完整；CPU 需相邻样本差值，此处先留空。
func oneShotSample(taskCtx context.Context, ctx *svc.ServiceContext, id string) (statsSample, error) {
	var sample statsSample
	if err := requireDocker(ctx); err != nil {
		return sample, err
	}
	statsCtx, cancel := context.WithTimeout(taskCtx, containerStatsTimeout)
	defer cancel()
	reader, err := ctx.DockerClient.ContainerStatsOneShot(statsCtx, id)
	if err != nil {
		return sample, err
	}
	defer reader.Body.Close()

	var stats container.StatsResponse
	if err := json.NewDecoder(reader.Body).Decode(&stats); err != nil {
		return sample, err
	}
	sample = statsSample{
		read:        stats.Read,
		osType:      reader.OSType,
		cpuTotal:    stats.CPUStats.CPUUsage.TotalUsage,
		systemTotal: stats.CPUStats.SystemUsage,
		usage:       ContainerUsage{MemoryUsage: calcMemoryUsage(stats, reader.OSType)},
	}
	if sample.read.IsZero() {
		sample.read = time.Now()
	}
	return sample, nil
}

// RefreshContainerUsage 供列表接口读取容器 CPU/内存。
// 命中未过期的采样缓存直接返回；无样本或样本过期（采集器已暂停）时同步做一次
// one-shot 采集（毫秒级，内存立即可用、CPU 留空），覆盖旧样本并唤醒采集器恢复采样。
func RefreshContainerUsage(taskCtx context.Context, ctx *svc.ServiceContext, id string) ContainerUsage {
	lastStatsReadAt.Store(time.Now().UnixNano())
	statsMu.RLock()
	sample, ok := statsSamples[id]
	statsMu.RUnlock()
	if ok && time.Since(sample.read) <= statsSampleMaxAge {
		return sample.usage
	}
	fresh, err := oneShotSample(taskCtx, ctx, id)
	if err != nil {
		logx.Debugf("one-shot stats 采集失败 id=%s: %v", id, err)
		return ContainerUsage{}
	}
	statsMu.Lock()
	statsSamples[id] = fresh
	statsMu.Unlock()
	select {
	case statsWake <- struct{}{}:
	default:
	}
	return fresh.usage
}

// cpuGapUsable 判断两个样本的时间间隔是否适合计算 CPU 使用率：
// 过短（<1s）噪声大；过长说明中间长期无人访问或主机休眠等异常，差值已无意义，
// 本次留空，等下一对相邻样本。
func cpuGapUsable(prev, cur statsSample) bool {
	gap := cur.read.Sub(prev.read)
	return gap >= time.Second && gap <= 3*statsCollectInterval
}

// cpuPercentBetween 用相邻两个采样点的计数器差值计算容器 CPU 使用率（0-100%），采样不足时返回空串。
func cpuPercentBetween(prev, cur statsSample) string {
	var cpuPercent float64
	if cur.osType == "windows" {
		// Windows 无 SystemUsage 口径，按两次采样的时间间隔折算
		if cur.cpuTotal < prev.cpuTotal {
			return ""
		}
		intervals := cur.read.Sub(prev.read).Nanoseconds() / 100
		if intervals > 0 {
			cpuPercent = float64(cur.cpuTotal-prev.cpuTotal) / float64(intervals) * 100.0
		}
	} else {
		if cur.cpuTotal < prev.cpuTotal || cur.systemTotal < prev.systemTotal {
			return ""
		}
		cpuDelta := float64(cur.cpuTotal - prev.cpuTotal)
		systemDelta := float64(cur.systemTotal - prev.systemTotal)
		if systemDelta > 0 && cpuDelta > 0 {
			cpuPercent = cpuDelta / systemDelta * 100.0
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
