package container

import "testing"

func TestValidateContainerName(t *testing.T) {
	for _, valid := range []string{"postgres", "app-1", "team_service.2"} {
		if _, err := validateContainerName(valid); err != nil {
			t.Fatalf("valid name %q rejected: %v", valid, err)
		}
	}
	for _, invalid := range []string{"", "/postgres", "bad name", "../escape"} {
		if _, err := validateContainerName(invalid); err == nil {
			t.Fatalf("invalid name %q accepted", invalid)
		}
	}
}
