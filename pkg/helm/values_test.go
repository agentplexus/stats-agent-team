package helm

import (
	"os"
	"path/filepath"
	"testing"
)

// getHelmValuesPath returns the path to the helm chart values directory.
func getHelmValuesPath(t *testing.T) string {
	t.Helper()

	// Try relative path from pkg/helm
	paths := []string{
		"../../helm/stats-agent-team",
		"../../../helm/stats-agent-team",
		"helm/stats-agent-team",
	}

	for _, p := range paths {
		if _, err := os.Stat(filepath.Join(p, "values.yaml")); err == nil {
			return p
		}
	}

	// Try to find from GOPATH or working directory
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	// Walk up to find the helm directory
	for dir := wd; dir != "/"; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, "helm", "stats-agent-team")
		if _, err := os.Stat(filepath.Join(candidate, "values.yaml")); err == nil {
			return candidate
		}
	}

	t.Skip("helm/stats-agent-team directory not found")
	return ""
}

func TestLoadDefaultValues(t *testing.T) {
	helmPath := getHelmValuesPath(t)
	valuesPath := filepath.Join(helmPath, "values.yaml")

	values, err := LoadValuesFile(valuesPath)
	if err != nil {
		t.Fatalf("failed to load values.yaml: %v", err)
	}

	// Verify key fields are parsed correctly
	if values.Global.Image.Tag != "latest" {
		t.Errorf("expected global.image.tag='latest', got '%s'", values.Global.Image.Tag)
	}

	if values.LLM.Provider != "gemini" {
		t.Errorf("expected llm.provider='gemini', got '%s'", values.LLM.Provider)
	}

	if values.Namespace.Name != "stats-agent" {
		t.Errorf("expected namespace.name='stats-agent', got '%s'", values.Namespace.Name)
	}

	if !values.Research.Enabled {
		t.Error("expected research.enabled=true")
	}

	if values.Research.Service.Port != 8001 {
		t.Errorf("expected research.service.port=8001, got %d", values.Research.Service.Port)
	}
}

func TestValidateDefaultValues(t *testing.T) {
	helmPath := getHelmValuesPath(t)
	valuesPath := filepath.Join(helmPath, "values.yaml")

	values, errs := LoadAndValidate(valuesPath)
	if len(errs) > 0 {
		for _, err := range errs {
			t.Errorf("validation error: %v", err)
		}
		t.FailNow()
	}

	if values == nil {
		t.Fatal("values should not be nil after successful validation")
	}
}

func TestValidateMinikubeValues(t *testing.T) {
	helmPath := getHelmValuesPath(t)
	basePath := filepath.Join(helmPath, "values.yaml")
	overlayPath := filepath.Join(helmPath, "values-minikube.yaml")

	// Load merged values (base + overlay) like Helm does
	values, errs := LoadMergeAndValidate(basePath, overlayPath)
	if len(errs) > 0 {
		for _, err := range errs {
			t.Errorf("validation error: %v", err)
		}
		t.FailNow()
	}

	// Verify Minikube-specific settings
	if values.Global.Image.PullPolicy != "Never" {
		t.Errorf("expected minikube pullPolicy='Never', got '%s'", values.Global.Image.PullPolicy)
	}

	// Verify base values are still present
	if values.Research.Service.Port != 8001 {
		t.Errorf("expected research port=8001 from base, got %d", values.Research.Service.Port)
	}
}

func TestValidateEKSValues(t *testing.T) {
	helmPath := getHelmValuesPath(t)
	basePath := filepath.Join(helmPath, "values.yaml")
	overlayPath := filepath.Join(helmPath, "values-eks.yaml")

	// Load merged values (base + overlay) like Helm does
	values, errs := LoadMergeAndValidate(basePath, overlayPath)

	// EKS values should load successfully
	// Note: Some business rule validations may fail (e.g., missing ingress host)
	// which is expected for a template file
	if values == nil {
		t.Fatal("failed to load EKS values")
	}

	// Check for loading errors (not validation warnings)
	for _, err := range errs {
		// Ingress host warning is expected (it's meant to be set at deploy time)
		if err.Error() == "ingress.host must be set when ingress is enabled" {
			continue
		}
		t.Errorf("unexpected validation error: %v", err)
	}

	// Verify EKS-specific settings
	if values.Global.Image.PullPolicy != "Always" {
		t.Errorf("expected EKS pullPolicy='Always', got '%s'", values.Global.Image.PullPolicy)
	}

	if !values.Ingress.Enabled {
		t.Error("expected ingress.enabled=true for EKS")
	}

	if values.Ingress.ClassName != "alb" {
		t.Errorf("expected ingress.className='alb', got '%s'", values.Ingress.ClassName)
	}

	// Verify base values are preserved
	if values.LLM.Provider != "gemini" {
		t.Errorf("expected llm.provider='gemini' from base, got '%s'", values.LLM.Provider)
	}
}

func TestValidateAllValuesFiles(t *testing.T) {
	helmPath := getHelmValuesPath(t)
	basePath := filepath.Join(helmPath, "values.yaml")

	// Test base values file standalone
	t.Run("values.yaml", func(t *testing.T) {
		values, err := LoadValuesFile(basePath)
		if err != nil {
			t.Fatalf("failed to load values.yaml: %v", err)
		}

		v := NewValidator()
		if err := v.Validate(values); err != nil {
			t.Errorf("struct validation failed for values.yaml: %v", err)
		}
	})

	// Test overlay files merged with base
	overlays := []string{
		"values-minikube.yaml",
		"values-eks.yaml",
	}

	for _, overlay := range overlays {
		t.Run(overlay, func(t *testing.T) {
			overlayPath := filepath.Join(helmPath, overlay)

			values, err := LoadAndMerge(basePath, overlayPath)
			if err != nil {
				t.Fatalf("failed to load and merge %s: %v", overlay, err)
			}

			v := NewValidator()
			if err := v.Validate(values); err != nil {
				t.Errorf("struct validation failed for %s: %v", overlay, err)
			}
		})
	}
}

func TestAgentPortsUnique(t *testing.T) {
	helmPath := getHelmValuesPath(t)
	valuesPath := filepath.Join(helmPath, "values.yaml")

	values, err := LoadValuesFile(valuesPath)
	if err != nil {
		t.Fatalf("failed to load values: %v", err)
	}

	ports := make(map[int]string)

	agents := []struct {
		name    string
		enabled bool
		port    int
		a2aPort int
	}{
		{"research", values.Research.Enabled, values.Research.Service.Port, values.Research.Service.A2APort},
		{"synthesis", values.Synthesis.Enabled, values.Synthesis.Service.Port, values.Synthesis.Service.A2APort},
		{"verification", values.Verification.Enabled, values.Verification.Service.Port, values.Verification.Service.A2APort},
		{"orchestration", values.Orchestration.Enabled, values.Orchestration.Service.Port, values.Orchestration.Service.A2APort},
		{"direct", values.Direct.Enabled, values.Direct.Service.Port, values.Direct.Service.A2APort},
	}

	for _, agent := range agents {
		if !agent.enabled {
			continue
		}

		if existing, ok := ports[agent.port]; ok {
			t.Errorf("port conflict: %s and %s both use port %d", existing, agent.name, agent.port)
		}
		ports[agent.port] = agent.name

		if agent.a2aPort > 0 {
			if existing, ok := ports[agent.a2aPort]; ok {
				t.Errorf("A2A port conflict: %s and %s both use port %d", existing, agent.name, agent.a2aPort)
			}
			ports[agent.a2aPort] = agent.name + " (A2A)"
		}
	}
}

func TestLLMProviderValidation(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		wantErr  bool
	}{
		{"valid gemini", "gemini", false},
		{"valid claude", "claude", false},
		{"valid openai", "openai", false},
		{"valid ollama", "ollama", false},
		{"invalid provider", "invalid", true},
		{"empty provider", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := createTestValues()
			values.LLM.Provider = tt.provider

			v := NewValidator()
			err := v.Validate(values)

			if tt.wantErr && err == nil {
				t.Errorf("expected validation error for provider '%s'", tt.provider)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected validation error for provider '%s': %v", tt.provider, err)
			}
		})
	}
}

// createTestValues returns a valid Values struct for testing.
func createTestValues() *Values {
	return &Values{
		Global: GlobalConfig{
			Image: ImageConfig{Tag: "latest", PullPolicy: "Always"},
		},
		Namespace: NamespaceConfig{Name: "test"},
		LLM:       LLMConfig{Provider: "gemini"},
		Search:    SearchConfig{Provider: "serper"},
		Research: AgentConfig{
			Enabled: true,
			Image:   AgentImageConfig{Repository: "test-research"},
			Service: ServiceConfig{Port: 8001},
		},
		Synthesis: AgentConfig{
			Enabled: true,
			Image:   AgentImageConfig{Repository: "test-synthesis"},
			Service: ServiceConfig{Port: 8002},
		},
		Verification: AgentConfig{
			Enabled: true,
			Image:   AgentImageConfig{Repository: "test-verification"},
			Service: ServiceConfig{Port: 8003},
		},
		Orchestration: OrchestrationConfig{
			AgentConfig: AgentConfig{
				Enabled: true,
				Image:   AgentImageConfig{Repository: "test-orchestration"},
				Service: ServiceConfig{Port: 8004},
			},
		},
		Direct: AgentConfig{
			Enabled: false,
			Image:   AgentImageConfig{Repository: "test-direct"},
			Service: ServiceConfig{Port: 8005},
		},
	}
}

func TestSearchProviderValidation(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		wantErr  bool
	}{
		{"valid serper", "serper", false},
		{"valid serpapi", "serpapi", false},
		{"invalid provider", "google", true},
		{"empty provider", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := createTestValues()
			values.Search.Provider = tt.provider

			v := NewValidator()
			err := v.Validate(values)

			if tt.wantErr && err == nil {
				t.Errorf("expected validation error for provider '%s'", tt.provider)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected validation error for provider '%s': %v", tt.provider, err)
			}
		})
	}
}

func TestK8sResourceQuantityValidation(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		isValid bool
	}{
		{"cpu millicores", "100m", true},
		{"cpu cores decimal", "0.5", true},
		{"cpu cores whole", "2", true},
		{"memory mebibytes", "256Mi", true},
		{"memory gibibytes", "1Gi", true},
		{"memory megabytes", "256M", true},
		{"memory gigabytes", "1G", true},
		{"empty", "", true},
		{"invalid suffix", "100x", false},
		{"invalid format", "abc", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.value == "" || isNumeric(tt.value) || validateResourceQuantityString(tt.value)

			if tt.isValid && !isValid {
				t.Errorf("expected '%s' to be valid", tt.value)
			}
			if !tt.isValid && isValid {
				t.Errorf("expected '%s' to be invalid", tt.value)
			}
		})
	}
}

// validateResourceQuantityString is a test helper to check resource format.
func validateResourceQuantityString(value string) bool {
	if value == "" {
		return true
	}

	validSuffixes := []string{"m", "Mi", "Gi", "Ki", "M", "G", "K", "Ti", "Pi", "Ei"}

	for _, suffix := range validSuffixes {
		if len(value) > len(suffix) && value[len(value)-len(suffix):] == suffix {
			prefix := value[:len(value)-len(suffix)]
			return isNumeric(prefix)
		}
	}

	return false
}

func TestBusinessRuleValidation(t *testing.T) {
	t.Run("orchestration requires other agents", func(t *testing.T) {
		values := &Values{
			Global: GlobalConfig{
				Image: ImageConfig{Tag: "latest"},
			},
			Namespace: NamespaceConfig{Name: "test"},
			LLM:       LLMConfig{Provider: "gemini"},
			Search:    SearchConfig{Provider: "serper"},
			Research: AgentConfig{
				Enabled: false, // Disabled!
				Image:   AgentImageConfig{Repository: "test"},
				Service: ServiceConfig{Port: 8001},
			},
			Synthesis: AgentConfig{
				Enabled: true,
				Image:   AgentImageConfig{Repository: "test"},
				Service: ServiceConfig{Port: 8002},
			},
			Verification: AgentConfig{
				Enabled: true,
				Image:   AgentImageConfig{Repository: "test"},
				Service: ServiceConfig{Port: 8003},
			},
			Orchestration: OrchestrationConfig{
				AgentConfig: AgentConfig{
					Enabled: true, // Enabled but research is disabled
					Image:   AgentImageConfig{Repository: "test"},
					Service: ServiceConfig{Port: 8004},
				},
			},
		}

		v := NewValidator()
		errs := v.ValidateWithContext(values)

		foundResearchError := false
		for _, err := range errs {
			if err.Error() == "research agent must be enabled when orchestration is enabled" {
				foundResearchError = true
				break
			}
		}

		if !foundResearchError {
			t.Error("expected validation error for disabled research agent with enabled orchestration")
		}
	})

	t.Run("ingress requires host", func(t *testing.T) {
		values := &Values{
			Global: GlobalConfig{
				Image: ImageConfig{Tag: "latest"},
			},
			Namespace: NamespaceConfig{Name: "test"},
			LLM:       LLMConfig{Provider: "gemini"},
			Search:    SearchConfig{Provider: "serper"},
			Research: AgentConfig{
				Enabled: true,
				Image:   AgentImageConfig{Repository: "test"},
				Service: ServiceConfig{Port: 8001},
			},
			Synthesis: AgentConfig{
				Enabled: true,
				Image:   AgentImageConfig{Repository: "test"},
				Service: ServiceConfig{Port: 8002},
			},
			Verification: AgentConfig{
				Enabled: true,
				Image:   AgentImageConfig{Repository: "test"},
				Service: ServiceConfig{Port: 8003},
			},
			Orchestration: OrchestrationConfig{
				AgentConfig: AgentConfig{
					Enabled: true,
					Image:   AgentImageConfig{Repository: "test"},
					Service: ServiceConfig{Port: 8004},
				},
			},
			Ingress: IngressConfig{
				Enabled: true,
				Host:    "", // Empty host!
			},
		}

		v := NewValidator()
		errs := v.ValidateWithContext(values)

		foundHostError := false
		for _, err := range errs {
			if err.Error() == "ingress.host must be set when ingress is enabled" {
				foundHostError = true
				break
			}
		}

		if !foundHostError {
			t.Error("expected validation error for enabled ingress without host")
		}
	})
}
