package main

// Result represents the validation result for JSON output
type Result struct {
	BasicValid      bool     `json:"basic_valid"`
	ProductionValid bool     `json:"production_valid"`
	Tips            []string `json:"tips,omitempty"`
	Messages        []string `json:"messages,omitempty"`
	Errors          []string `json:"errors,omitempty"`
	Configuration   *Config  `json:"configuration,omitempty"`
}

// Config represents the configuration details for JSON output
type Config struct {
	ServiceName  string `json:"service_name"`
	TrustDomain  string `json:"trust_domain"`
	AgentSocket  string `json:"agent_socket,omitempty"`
}