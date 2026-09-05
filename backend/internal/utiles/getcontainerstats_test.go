package utiles

import (
	"testing"
	"time"
)

func TestCPUPercentBetween(t *testing.T) {
	linux := func(cpuTotal, systemTotal uint64) statsSample {
		return statsSample{osType: "linux", cpuTotal: cpuTotal, systemTotal: systemTotal, read: time.Now()}
	}

	cases := []struct {
		name string
		prev statsSample
		cur  statsSample
		want string
	}{
		{
			name: "正常增量",
			prev: linux(100, 1000),
			cur:  linux(200, 2000),
			want: "10.0%",
		},
		{
			name: "系统计数无增长",
			prev: linux(100, 1000),
			cur:  linux(200, 1000),
			want: "0.0%",
		},
		{
			name: "容器重启计数回退",
			prev: linux(500, 1000),
			cur:  linux(100, 2000),
			want: "",
		},
		{
			name: "空转容器为 0",
			prev: linux(100, 1000),
			cur:  linux(100, 2000),
			want: "0.0%",
		},
		{
			name: "超过 100% 截断",
			prev: linux(0, 100),
			cur:  linux(500, 300),
			want: "100.0%",
		},
		{
			name: "windows 按采样间隔折算",
			prev: statsSample{osType: "windows", cpuTotal: 0, read: time.Unix(1000, 0)},
			cur:  statsSample{osType: "windows", cpuTotal: 1_000_000, read: time.Unix(1010, 0)},
			// 10s = 1e8 个 100ns 单位；1e6/1e8*100 = 1%
			want: "1.0%",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := cpuPercentBetween(tc.prev, tc.cur); got != tc.want {
				t.Fatalf("cpuPercentBetween() = %q, want %q", got, tc.want)
			}
		})
	}
}
