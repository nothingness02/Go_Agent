package agent

import (
	"log"
	"strings"

	"github.com/spf13/viper"
)

type AgentConfig struct {
	APIKey       string           `mapstructure:"api_key"`
	BaseURL      string           `mapstructure:"base_url"`
	Model        string           `mapstructure:"model"`
	AllowTools   bool             `mapstructure:"allow_tools"`
	SystemPrompt string           `mapstructure:"system_prompt"`
	Temperature  float32          `mapstructure:"temperature"`
	MaxCircle    int              `mapstructure:"max_circle"`
	ReAct        ReActAgentConfig `mapstructure:"react"`
}

type ReActAgentConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

func DefaultAgentConfig() AgentConfig {
	return AgentConfig{
		BaseURL:      "",
		Model:        "",
		AllowTools:   true,
		SystemPrompt: DefaultSystemPrompt,
		Temperature:  DefaultTemperature,
		MaxCircle:    DefaultMaxCircle,
		ReAct:        ReActAgentConfig{Enabled: false},
	}
}

// LoadAgentConfig loads agent config from a directory containing agent.yaml.
func LoadAgentConfig(path string) (*AgentConfig, error) {
	v := viper.New()
	v.AddConfigPath(path)
	v.SetConfigName("agent")
	v.SetConfigType("yaml")

	v.SetEnvPrefix("AGENT")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	cfg := DefaultAgentConfig()
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
		log.Println("agent config not found, relying on env vars")
	}
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
