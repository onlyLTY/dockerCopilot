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
	for _, name := range []string{"compose.yaml", "docker-compose.override.yml", ".env"} {
		if err := ValidateComposeFilename(name); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"../compose.yaml", `..\compose.yaml`, "secret.txt"} {
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
