package compose_project

import (
	"testing"

	composeTypes "github.com/compose-spec/compose-go/v2/types"
)

func TestProjectNameValidation(t *testing.T) {
	for _, name := range []string{"app", "my-project_1"} {
		if err := ValidateProjectName(name); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"", "../app", "UPPER", "app space"} {
		if err := ValidateProjectName(name); err == nil {
			t.Fatalf("expected %q to fail", name)
		}
	}
}

func TestComposeFilenameValidation(t *testing.T) {
	for _, name := range []string{
		"compose.yaml",
		"docker-compose.override.yml",
		"compose.override.yaml",
		"COMPOSE.OVERRIDE.YML",
		".env",
	} {
		if err := ValidateComposeFilename(name); err != nil {
			t.Fatalf("expected %q to be valid: %v", name, err)
		}
	}
	for _, name := range []string{
		"../compose.yaml",
		`..\\compose.yaml`,
		"secret.txt",
		"myoverride.txt",
		"override-secret",
		"compose.override.json",
	} {
		if err := ValidateComposeFilename(name); err == nil {
			t.Fatalf("expected %q to fail", name)
		}
	}
}

func TestInspectRisks(t *testing.T) {
	project := &composeTypes.Project{Services: composeTypes.Services{
		"app": {Name: "app", Privileged: true, NetworkMode: "host"},
	}}
	risks := InspectRisks(project, "/compose/app")
	if len(risks) != 2 {
		t.Fatalf("expected two risks, got %d", len(risks))
	}
}

func TestValidateCriticalRisks(t *testing.T) {
	privileged := &composeTypes.Project{Services: composeTypes.Services{
		"app": {Name: "app", Privileged: true},
	}}
	if err := ValidateCriticalRisks(privileged, "/compose/app", false); err == nil {
		t.Fatal("expected privileged to be blocked without AllowHighRisk")
	}
	if err := ValidateCriticalRisks(privileged, "/compose/app", true); err != nil {
		t.Fatalf("AllowHighRisk should permit privileged: %v", err)
	}

	hostNetOnly := &composeTypes.Project{Services: composeTypes.Services{
		"app": {Name: "app", NetworkMode: "host"},
	}}
	// host 网络属于 high 但非 critical，可由 confirmWarnings 覆盖
	if err := ValidateCriticalRisks(hostNetOnly, "/compose/app", false); err != nil {
		t.Fatalf("host network alone should not be critical-blocked: %v", err)
	}
}

func TestIsCriticalRiskByKind(t *testing.T) {
	if !IsCriticalRisk(Risk{Kind: RiskKindPrivileged, Level: "high"}) {
		t.Fatal("privileged kind should be critical")
	}
	if !IsCriticalRisk(Risk{Kind: RiskKindDockerSocket, Level: "high"}) {
		t.Fatal("docker socket kind should be critical")
	}
	if !IsCriticalRisk(Risk{Kind: RiskKindSensitivePath, Level: "high"}) {
		t.Fatal("sensitive path kind should be critical")
	}
	if IsCriticalRisk(Risk{Kind: RiskKindHostNetwork, Level: "high"}) {
		t.Fatal("host network should not be critical")
	}
	// 文案变更不应影响判定
	if IsCriticalRisk(Risk{Kind: RiskKindHostNetwork, Message: "服务启用了 privileged", Level: "high"}) {
		t.Fatal("critical check must use Kind, not Message")
	}
}
