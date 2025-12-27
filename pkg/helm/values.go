// Package helm provides Go structs for Helm chart values with validation support.
// These structs mirror the structure of helm/stats-agent-team/values.yaml and can be
// used for programmatic validation, testing, and generation of Helm values.
package helm

import (
	"fmt"
	"os"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

// Values represents the complete Helm chart values structure.
type Values struct {
	Global             GlobalConfig         `yaml:"global" validate:"required"`
	Namespace          NamespaceConfig      `yaml:"namespace" validate:"required"`
	LLM                LLMConfig            `yaml:"llm" validate:"required"`
	Search             SearchConfig         `yaml:"search" validate:"required"`
	Secrets            SecretsConfig        `yaml:"secrets"`
	Research           AgentConfig          `yaml:"research" validate:"required"`
	Synthesis          AgentConfig          `yaml:"synthesis" validate:"required"`
	Verification       AgentConfig          `yaml:"verification" validate:"required"`
	Orchestration      OrchestrationConfig  `yaml:"orchestration" validate:"required"`
	Direct             AgentConfig          `yaml:"direct"`
	Ingress            IngressConfig        `yaml:"ingress"`
	ServiceAccount     ServiceAccountConfig `yaml:"serviceAccount"`
	PodSecurityContext PodSecurityContext   `yaml:"podSecurityContext"`
	SecurityContext    SecurityContext      `yaml:"securityContext"`
}

// GlobalConfig contains global settings for all agents.
type GlobalConfig struct {
	Image            ImageConfig       `yaml:"image" validate:"required"`
	ImagePullSecrets []ImagePullSecret `yaml:"imagePullSecrets"`
}

// ImageConfig defines container image settings.
type ImageConfig struct {
	Registry   string `yaml:"registry"`
	Repository string `yaml:"repository"`
	PullPolicy string `yaml:"pullPolicy" validate:"omitempty,oneof=Always IfNotPresent Never"`
	Tag        string `yaml:"tag" validate:"required"`
}

// ImagePullSecret references a Kubernetes secret for pulling images.
type ImagePullSecret struct {
	Name string `yaml:"name" validate:"required"`
}

// NamespaceConfig defines Kubernetes namespace settings.
type NamespaceConfig struct {
	Create bool   `yaml:"create"`
	Name   string `yaml:"name" validate:"required,min=1,max=63"`
}

// LLMConfig contains LLM provider configuration.
type LLMConfig struct {
	Provider    string `yaml:"provider" validate:"required,oneof=gemini claude openai ollama"`
	BaseURL     string `yaml:"baseUrl"`
	GeminiModel string `yaml:"geminiModel"`
	ClaudeModel string `yaml:"claudeModel"`
	OpenAIModel string `yaml:"openaiModel"`
	OllamaModel string `yaml:"ollamaModel"`
	OllamaURL   string `yaml:"ollamaUrl" validate:"omitempty,url"`
}

// SearchConfig contains search provider configuration.
type SearchConfig struct {
	Provider string `yaml:"provider" validate:"required,oneof=serper serpapi"`
}

// SecretsConfig defines API key secrets.
type SecretsConfig struct {
	Create          bool   `yaml:"create"`
	GeminiAPIKey    string `yaml:"geminiApiKey"`
	ClaudeAPIKey    string `yaml:"claudeApiKey"`
	OpenAIAPIKey    string `yaml:"openaiApiKey"`
	AnthropicAPIKey string `yaml:"anthropicApiKey"`
	SerperAPIKey    string `yaml:"serperApiKey"`
	SerpAPIKey      string `yaml:"serpApiKey"`
}

// AgentConfig defines configuration for an individual agent.
type AgentConfig struct {
	Enabled      bool              `yaml:"enabled"`
	ReplicaCount int               `yaml:"replicaCount" validate:"omitempty,min=0,max=100"`
	Image        AgentImageConfig  `yaml:"image"`
	Service      ServiceConfig     `yaml:"service"`
	Resources    ResourcesConfig   `yaml:"resources"`
	Autoscaling  AutoscalingConfig `yaml:"autoscaling"`
	PDB          PDBConfig         `yaml:"pdb"`
	NodeSelector map[string]string `yaml:"nodeSelector"`
	Tolerations  []Toleration      `yaml:"tolerations"`
	Affinity     map[string]any    `yaml:"affinity"`
}

// AutoscalingConfig defines Horizontal Pod Autoscaler settings.
type AutoscalingConfig struct {
	Enabled                           bool                 `yaml:"enabled"`
	MinReplicas                       int                  `yaml:"minReplicas" validate:"omitempty,min=1"`
	MaxReplicas                       int                  `yaml:"maxReplicas" validate:"omitempty,min=1,max=1000"`
	TargetCPUUtilizationPercentage    int                  `yaml:"targetCPUUtilizationPercentage" validate:"omitempty,min=1,max=100"`
	TargetMemoryUtilizationPercentage int                  `yaml:"targetMemoryUtilizationPercentage" validate:"omitempty,min=1,max=100"`
	Behavior                          *AutoscalingBehavior `yaml:"behavior,omitempty"`
}

// AutoscalingBehavior defines scaling behavior policies.
type AutoscalingBehavior struct {
	ScaleDown *ScalingRules `yaml:"scaleDown,omitempty"`
	ScaleUp   *ScalingRules `yaml:"scaleUp,omitempty"`
}

// ScalingRules defines scaling stabilization and policies.
type ScalingRules struct {
	StabilizationWindowSeconds int             `yaml:"stabilizationWindowSeconds,omitempty" validate:"omitempty,min=0,max=3600"`
	Policies                   []ScalingPolicy `yaml:"policies,omitempty"`
	SelectPolicy               string          `yaml:"selectPolicy,omitempty" validate:"omitempty,oneof=Max Min Disabled"`
}

// ScalingPolicy defines a single scaling policy.
type ScalingPolicy struct {
	Type          string `yaml:"type" validate:"required,oneof=Pods Percent"`
	Value         int    `yaml:"value" validate:"required,min=1"`
	PeriodSeconds int    `yaml:"periodSeconds" validate:"required,min=1,max=1800"`
}

// PDBConfig defines Pod Disruption Budget settings.
type PDBConfig struct {
	Enabled        bool   `yaml:"enabled"`
	MinAvailable   string `yaml:"minAvailable,omitempty"`   // Can be int or percentage string
	MaxUnavailable string `yaml:"maxUnavailable,omitempty"` // Can be int or percentage string
}

// OrchestrationConfig extends AgentConfig with orchestration-specific settings.
type OrchestrationConfig struct {
	AgentConfig `yaml:",inline"`
	UseEino     bool `yaml:"useEino"`
}

// AgentImageConfig defines agent-specific image settings.
type AgentImageConfig struct {
	Repository string `yaml:"repository"` // Not required in overlays
	Tag        string `yaml:"tag"`
}

// ServiceConfig defines Kubernetes service settings.
type ServiceConfig struct {
	Type    string `yaml:"type" validate:"omitempty,oneof=ClusterIP NodePort LoadBalancer"`
	Port    int    `yaml:"port" validate:"omitempty,min=1,max=65535"`
	A2APort int    `yaml:"a2aPort" validate:"omitempty,min=1,max=65535"`
}

// ResourcesConfig defines Kubernetes resource requests and limits.
type ResourcesConfig struct {
	Requests ResourceSpec `yaml:"requests"`
	Limits   ResourceSpec `yaml:"limits"`
}

// ResourceSpec defines CPU and memory specifications.
type ResourceSpec struct {
	CPU    string `yaml:"cpu" validate:"omitempty,k8s_resource_quantity"`
	Memory string `yaml:"memory" validate:"omitempty,k8s_resource_quantity"`
}

// Toleration defines a Kubernetes pod toleration.
type Toleration struct {
	Key               string `yaml:"key"`
	Operator          string `yaml:"operator" validate:"omitempty,oneof=Exists Equal"`
	Value             string `yaml:"value"`
	Effect            string `yaml:"effect" validate:"omitempty,oneof=NoSchedule PreferNoSchedule NoExecute"`
	TolerationSeconds *int64 `yaml:"tolerationSeconds"`
}

// IngressConfig defines Kubernetes ingress settings.
type IngressConfig struct {
	Enabled     bool              `yaml:"enabled"`
	ClassName   string            `yaml:"className"`
	Annotations map[string]string `yaml:"annotations"`
	Host        string            `yaml:"host" validate:"omitempty,hostname|fqdn"`
	TLS         []IngressTLS      `yaml:"tls"`
}

// IngressTLS defines TLS configuration for ingress.
type IngressTLS struct {
	SecretName string   `yaml:"secretName" validate:"required"`
	Hosts      []string `yaml:"hosts" validate:"required,min=1,dive,hostname|fqdn"`
}

// ServiceAccountConfig defines Kubernetes service account settings.
type ServiceAccountConfig struct {
	Create      bool              `yaml:"create"`
	Annotations map[string]string `yaml:"annotations"`
	Name        string            `yaml:"name"`
}

// PodSecurityContext defines pod-level security settings.
type PodSecurityContext struct {
	RunAsNonRoot bool  `yaml:"runAsNonRoot"`
	RunAsUser    int64 `yaml:"runAsUser" validate:"omitempty,min=0"`
	RunAsGroup   int64 `yaml:"runAsGroup" validate:"omitempty,min=0"`
	FSGroup      int64 `yaml:"fsGroup" validate:"omitempty,min=0"`
}

// SecurityContext defines container-level security settings.
type SecurityContext struct {
	AllowPrivilegeEscalation bool         `yaml:"allowPrivilegeEscalation"`
	ReadOnlyRootFilesystem   bool         `yaml:"readOnlyRootFilesystem"`
	Capabilities             Capabilities `yaml:"capabilities"`
}

// Capabilities defines Linux capabilities settings.
type Capabilities struct {
	Add  []string `yaml:"add"`
	Drop []string `yaml:"drop"`
}

// Validator provides validation for Helm values.
type Validator struct {
	validate *validator.Validate
}

// NewValidator creates a new Validator with custom validation rules.
func NewValidator() *Validator {
	v := validator.New()

	// Register custom validation for Kubernetes resource quantities (e.g., "100m", "256Mi")
	if err := v.RegisterValidation("k8s_resource_quantity", validateK8sResourceQuantity); err != nil {
		panic(fmt.Sprintf("failed to register k8s_resource_quantity validation: %v", err))
	}

	return &Validator{validate: v}
}

// validateK8sResourceQuantity validates Kubernetes resource quantity format.
func validateK8sResourceQuantity(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true
	}

	// Basic validation for common K8s resource formats
	// CPU: 100m, 0.5, 1, 2000m
	// Memory: 128Mi, 1Gi, 256M, 1G
	validSuffixes := []string{"m", "Mi", "Gi", "Ki", "M", "G", "K", "Ti", "Pi", "Ei"}

	// Check if it's a plain number
	if isNumeric(value) {
		return true
	}

	// Check for valid suffix
	for _, suffix := range validSuffixes {
		if len(value) > len(suffix) && value[len(value)-len(suffix):] == suffix {
			prefix := value[:len(value)-len(suffix)]
			return isNumeric(prefix)
		}
	}

	return false
}

// isNumeric checks if a string represents a numeric value.
func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	dotCount := 0
	for i, c := range s {
		if c == '.' {
			dotCount++
			if dotCount > 1 {
				return false
			}
			continue
		}
		if c == '-' && i == 0 {
			continue
		}
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// Validate validates the given Values struct.
func (v *Validator) Validate(values *Values) error {
	if err := v.validate.Struct(values); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	return nil
}

// ValidateWithContext performs validation with additional business logic checks.
func (v *Validator) ValidateWithContext(values *Values) []error {
	var errs []error

	// Run struct validation
	if err := v.validate.Struct(values); err != nil {
		if validationErrs, ok := err.(validator.ValidationErrors); ok {
			for _, e := range validationErrs {
				errs = append(errs, fmt.Errorf("field '%s' failed validation: %s", e.Namespace(), e.Tag()))
			}
		} else {
			errs = append(errs, err)
		}
	}

	// Business logic validations
	errs = append(errs, v.validateBusinessRules(values)...)

	return errs
}

// validateBusinessRules performs additional business logic validations.
func (v *Validator) validateBusinessRules(values *Values) []error {
	var errs []error

	// Check that at least the core agents are enabled for a functional deployment
	if values.Orchestration.Enabled {
		if !values.Research.Enabled {
			errs = append(errs, fmt.Errorf("research agent must be enabled when orchestration is enabled"))
		}
		if !values.Synthesis.Enabled {
			errs = append(errs, fmt.Errorf("synthesis agent must be enabled when orchestration is enabled"))
		}
		if !values.Verification.Enabled {
			errs = append(errs, fmt.Errorf("verification agent must be enabled when orchestration is enabled"))
		}
	}

	// Check that secrets are configured when secret creation is enabled
	if values.Secrets.Create {
		hasLLMKey := false
		switch values.LLM.Provider {
		case "gemini":
			hasLLMKey = values.Secrets.GeminiAPIKey != ""
		case "claude":
			hasLLMKey = values.Secrets.ClaudeAPIKey != "" || values.Secrets.AnthropicAPIKey != ""
		case "openai":
			hasLLMKey = values.Secrets.OpenAIAPIKey != ""
		case "ollama":
			hasLLMKey = true // Ollama doesn't require API key
		}

		if !hasLLMKey && values.LLM.Provider != "ollama" {
			// This is a warning, not an error - keys might be set via --set
			// We don't add to errs here, but could log a warning
		}
	}

	// Validate port conflicts (only for ports that are actually set)
	ports := make(map[int]string)
	agentPorts := []struct {
		name   string
		config AgentConfig
	}{
		{"research", values.Research},
		{"synthesis", values.Synthesis},
		{"verification", values.Verification},
		{"orchestration", values.Orchestration.AgentConfig},
		{"direct", values.Direct},
	}

	for _, agent := range agentPorts {
		if !agent.config.Enabled {
			continue
		}
		// Skip port validation if port is not set (0 means not configured)
		if agent.config.Service.Port == 0 {
			continue
		}
		if existing, ok := ports[agent.config.Service.Port]; ok {
			errs = append(errs, fmt.Errorf("port conflict: %s and %s both use port %d",
				existing, agent.name, agent.config.Service.Port))
		}
		ports[agent.config.Service.Port] = agent.name

		if agent.config.Service.A2APort > 0 {
			if existing, ok := ports[agent.config.Service.A2APort]; ok {
				errs = append(errs, fmt.Errorf("port conflict: %s and %s A2A both use port %d",
					existing, agent.name, agent.config.Service.A2APort))
			}
			ports[agent.config.Service.A2APort] = agent.name + " A2A"
		}
	}

	// Validate ingress host is set when ingress is enabled
	if values.Ingress.Enabled && values.Ingress.Host == "" {
		errs = append(errs, fmt.Errorf("ingress.host must be set when ingress is enabled"))
	}

	return errs
}

// LoadValuesFile loads and parses a values YAML file.
func LoadValuesFile(path string) (*Values, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read values file: %w", err)
	}

	return ParseValues(data)
}

// ParseValues parses YAML data into a Values struct.
func ParseValues(data []byte) (*Values, error) {
	var values Values
	if err := yaml.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf("failed to parse values YAML: %w", err)
	}

	return &values, nil
}

// LoadAndValidate loads a values file and validates it.
func LoadAndValidate(path string) (*Values, []error) {
	values, err := LoadValuesFile(path)
	if err != nil {
		return nil, []error{err}
	}

	v := NewValidator()
	errs := v.ValidateWithContext(values)

	return values, errs
}

// LoadAndMerge loads a base values file and merges it with an overlay file.
// This mimics how Helm merges values files.
func LoadAndMerge(basePath, overlayPath string) (*Values, error) {
	baseData, err := os.ReadFile(basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read base values: %w", err)
	}

	overlayData, err := os.ReadFile(overlayPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read overlay values: %w", err)
	}

	// Parse base values
	var base map[string]any
	if err := yaml.Unmarshal(baseData, &base); err != nil {
		return nil, fmt.Errorf("failed to parse base values: %w", err)
	}

	// Parse overlay values
	var overlay map[string]any
	if err := yaml.Unmarshal(overlayData, &overlay); err != nil {
		return nil, fmt.Errorf("failed to parse overlay values: %w", err)
	}

	// Merge overlay into base
	merged := mergeMaps(base, overlay)

	// Convert merged map back to YAML and parse into Values struct
	mergedData, err := yaml.Marshal(merged)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal merged values: %w", err)
	}

	return ParseValues(mergedData)
}

// mergeMaps recursively merges src into dst.
func mergeMaps(dst, src map[string]any) map[string]any {
	result := make(map[string]any)

	// Copy dst
	for k, v := range dst {
		result[k] = v
	}

	// Merge src
	for k, v := range src {
		if srcMap, ok := v.(map[string]any); ok {
			if dstMap, ok := result[k].(map[string]any); ok {
				result[k] = mergeMaps(dstMap, srcMap)
				continue
			}
		}
		result[k] = v
	}

	return result
}

// LoadMergeAndValidate loads a base values file, merges it with an overlay,
// and validates the result.
func LoadMergeAndValidate(basePath, overlayPath string) (*Values, []error) {
	values, err := LoadAndMerge(basePath, overlayPath)
	if err != nil {
		return nil, []error{err}
	}

	v := NewValidator()
	errs := v.ValidateWithContext(values)

	return values, errs
}
