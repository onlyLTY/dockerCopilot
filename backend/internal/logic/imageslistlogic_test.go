package logic

import "testing"

func TestImageNameForDisplay(t *testing.T) {
	tests := []struct {
		name    string
		image   string
		hubURLs []string
		want    string
	}{
		{
			name:    "removes configured accelerator host",
			image:   "docker.1ms.run/library/nginx",
			hubURLs: []string{"docker.1ms.run"},
			want:    "library/nginx",
		},
		{
			name:    "matches host case insensitively",
			image:   "DOCKER.1MS.RUN/library/nginx",
			hubURLs: []string{"docker.1ms.run"},
			want:    "library/nginx",
		},
		{
			name:    "removes the first matching configured host",
			image:   "docker.m.daocloud.io/library/redis",
			hubURLs: []string{"docker.1ms.run", "docker.m.daocloud.io"},
			want:    "library/redis",
		},
		{
			name:    "keeps similar host",
			image:   "docker.1ms.run-extra/library/nginx",
			hubURLs: []string{"docker.1ms.run"},
			want:    "docker.1ms.run-extra/library/nginx",
		},
		{
			name:    "keeps unconfigured registry",
			image:   "ghcr.io/example/image",
			hubURLs: []string{"docker.1ms.run"},
			want:    "ghcr.io/example/image",
		},
		{
			name:    "ignores empty and slash wrapped hosts",
			image:   "docker.1ms.run/library/nginx",
			hubURLs: []string{"", "/docker.1ms.run/"},
			want:    "library/nginx",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := imageNameForDisplay(tt.image, tt.hubURLs); got != tt.want {
				t.Fatalf("imageNameForDisplay(%q, %v) = %q, want %q", tt.image, tt.hubURLs, got, tt.want)
			}
		})
	}
}
