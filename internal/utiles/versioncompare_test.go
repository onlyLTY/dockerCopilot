package utiles

import "testing"

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		current string
		remote  string
		want    bool
	}{
		{"v2.1.3", "v2.1.4", true},
		{"v2.1.3-FNOS", "v2.2.0-FNOS", true},
		{"v2.1.3", "v2.1.3", false},
		{"v2.2.0", "v2.1.9", false},
	}
	for _, test := range tests {
		got, err := isNewerVersion(test.current, test.remote)
		if err != nil {
			t.Fatalf("compare %s and %s: %v", test.current, test.remote, err)
		}
		if got != test.want {
			t.Fatalf("compare %s and %s: got %v want %v", test.current, test.remote, got, test.want)
		}
	}
}

func TestBinarySelfUpdateDisabledFailsClosed(t *testing.T) {
	t.Setenv("DISABLE_BINARY_SELF_UPDATE", "not-a-boolean")
	if !BinarySelfUpdateDisabled() {
		t.Fatal("invalid container update configuration must fail closed")
	}
}
