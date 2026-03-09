package v1

import (
	"fmt"
	"testing"

	"github.com/google/cel-go/cel"
	"k8s.io/apiserver/pkg/cel/library"
)

func newCELEnv(t *testing.T) *cel.Env {
	t.Helper()
	env, err := cel.NewEnv(
		cel.Variable("self", cel.StringType),
		cel.OptionalTypes(),
		library.Format(),
	)
	if err != nil {
		t.Fatalf("failed to create CEL env: %v", err)
	}
	return env
}

func compileCEL(t *testing.T, env *cel.Env, rule string) cel.Program {
	t.Helper()
	ast, issues := env.Compile(rule)
	if issues != nil && issues.Err() != nil {
		t.Fatalf("failed to compile CEL rule: %v", issues.Err())
	}
	prg, err := env.Program(ast)
	if err != nil {
		t.Fatalf("failed to create CEL program: %v", err)
	}
	return prg
}

func TestDeviceClassNameCEL(t *testing.T) {
	env := newCELEnv(t)
	prg := compileCEL(t, env, "!format.qualifiedName().validate(self).hasValue()")

	tests := []struct {
		name  string
		input string
		valid bool
	}{
		// Valid DeviceClass names (DNS subdomains)
		{"NVIDIA GPU", "gpu.nvidia.com", true},
		{"NVIDIA MIG", "mig.nvidia.com", true},
		{"NVIDIA compute domain", "compute-domain-daemon.nvidia.com", true},
		{"AMD GPU", "gpu.amd.com", true},
		{"Intel GPU", "gpu.intel.com", true},
		{"Google TPU", "tpu.google.com", true},
		{"custom no dots", "nvidia-mig", true},
		{"simple label", "gpu", true},
		// Valid per IsQualifiedName (with slash)
		{"with slash", "valid.com/device", true},
		// Valid per IsQualifiedName (dots after slash)
		{"dots after slash", "nvidia.com/mig-1g.5gb", true},

		// Invalid DeviceClass names
		{"starts with hyphen", "-gpu.nvidia.com", false},
		{"ends with hyphen", "gpu.nvidia.com-", false},
		{"starts with dot", ".gpu.nvidia.com", false},
		{"double dots", "gpu..nvidia.com", true}, // valid per IsQualifiedName
		{"empty", "", false},
		{"starts with special", "@invalid", false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s (%s)", tt.name, tt.input), func(t *testing.T) {
			out, _, err := prg.Eval(map[string]interface{}{"self": tt.input})
			if err != nil {
				t.Fatalf("CEL eval error: %v", err)
			}
			result := out.Value().(bool)
			if result != tt.valid {
				t.Errorf("input %q: got valid=%v, want valid=%v", tt.input, result, tt.valid)
			}
		})
	}
}

func TestResourceNameCEL(t *testing.T) {
	env := newCELEnv(t)
	prg := compileCEL(t, env, "!format.qualifiedName().validate(self).hasValue()")

	tests := []struct {
		name  string
		input string
		valid bool
	}{
		// Valid resource names
		{"with slash", "nvidia.com/gpu", true},
		{"AMD with slash", "amd.com/gpu", true},
		{"Intel with slash", "intel.com/gpu", true},
		{"MIG profile with dot", "nvidia.com/mig-1g.5gb", true},
		{"MIG profile with hyphen", "nvidia.com/mig-1g-5gb", true},
		{"subdomain only", "nvidia.com", true},
		{"simple label", "gpu", true},
		{"with underscore", "nvidia.com/my_gpu", true},

		// Invalid resource names
		{"double slash", "nvidia.com/gpu/extra", false},
		{"empty after slash", "nvidia.com/", false},
		{"starts with hyphen", "-nvidia.com/gpu", false},
		{"empty", "", false},
		{"starts with special", "@invalid/gpu", false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s (%s)", tt.name, tt.input), func(t *testing.T) {
			out, _, err := prg.Eval(map[string]interface{}{"self": tt.input})
			if err != nil {
				t.Fatalf("CEL eval error: %v", err)
			}
			result := out.Value().(bool)
			if result != tt.valid {
				t.Errorf("input %q: got valid=%v, want valid=%v", tt.input, result, tt.valid)
			}
		})
	}
}
