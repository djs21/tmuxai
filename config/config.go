package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
	"github.com/spf13/viper"
)

// Config holds the application configuration
type Config struct {
	Debug                 bool               `mapstructure:"debug"`
	MaxCaptureLines       int                `mapstructure:"max_capture_lines"`
	MaxContextSize        int                `mapstructure:"max_context_size"`
	WaitInterval          int                `mapstructure:"wait_interval"`
	SendKeysConfirm       bool               `mapstructure:"send_keys_confirm"`
	PasteMultilineConfirm bool               `mapstructure:"paste_multiline_confirm"`
	ExecConfirm           bool               `mapstructure:"exec_confirm"`
	WhitelistPatterns     []string           `mapstructure:"whitelist_patterns"`
	BlacklistPatterns     []string           `mapstructure:"blacklist_patterns"`
	OpenRouter            OpenRouterConfig   `mapstructure:"openrouter"`
	AzureOpenAI           AzureOpenAIConfig  `mapstructure:"azure_openai"`
	Browserless           BrowserlessConfig  `mapstructure:"browserless"`
	Prompts               PromptsConfig      `mapstructure:"prompts"`
	Personas              map[string]*Persona `mapstructure:"personas"`
	PersonaRules          []PersonaRule      `mapstructure:"persona_rules"`
	DefaultPersona        string             `mapstructure:"default_persona"`
}

// OpenRouterConfig holds OpenRouter API configuration
type OpenRouterConfig struct {
	APIKey  string `mapstructure:"api_key"`
	Model   string `mapstructure:"model"`
	BaseURL string `mapstructure:"base_url"`
}

// AzureOpenAIConfig holds Azure OpenAI API configuration
type AzureOpenAIConfig struct {
	APIKey         string `mapstructure:"api_key"`
	APIBase        string `mapstructure:"api_base"`
	APIVersion     string `mapstructure:"api_version"`
	DeploymentName string `mapstructure:"deployment_name"`
}

// BrowserlessConfig holds Browserless API configuration
type BrowserlessConfig struct {
	Token   string `mapstructure:"token"`
	BaseURL string `mapstructure:"base_url"`
}

// PromptsConfig holds customizable prompt templates
type PromptsConfig struct {
	BaseSystem            string `mapstructure:"base_system"`
	ChatAssistant         string `mapstructure:"chat_assistant"`
	ChatAssistantPrepared string `mapstructure:"chat_assistant_prepared"`
	Watch                 string `mapstructure:"watch"`
}

// Persona represents a single persona configuration
type Persona struct {
	Prompt     string `yaml:"prompt"`
	Description string `yaml:"description"`
}

// PersonaRule defines rules for auto-selecting personas
type PersonaRule struct {
	Match   string `mapstructure:"match"`
	Persona string `mapstructure:"persona"`
}

// DefaultConfig returns a configuration with default values
func DefaultConfig() *Config {
	defaultPersonas := map[string]*Persona{
		"pair_programmer": {
			Prompt: `You are TmuxAI assistant. You are AI agent and live inside user's Tmux's window and can see all panes in that window.
Think of TmuxAI as a pair programmer that sits beside user, watching users terminal window exactly as user see it.
TmuxAI's design philosophy mirrors the way humans collaborate at the terminal. Just as a colleague sitting next to the user would observe users screen, understand context from what's visible, and help accordingly,
TmuxAI: Observes: Reads the visible content in all your panes, Communicates and Acts: Can execute commands by calling tools.
You and user both are able to control and interact with tmux ai exec pane.

You have perfect understanding of human common sense.
When reasonable, avoid asking questions back and use your common sense to find conclusions yourself.
Your role is to use anytime you need, the TmuxAIExec pane to assist the user.
You are expert in all kinds of shell scripting, shell usage diffence between bash, zsh, fish, powershell, cmd, batch, etc and different OS-es.
You always strive for simple, elegant, clean and effective solutions.
Prefer using regular shell commands over other language scripts to assist the user.

Address the root cause instead of the symptoms.
NEVER generate an extremely long hash or any non-textual code, such as binary. These are not helpful to the USER and are very expensive.
Always address user directly as 'you' in a conversational tone, avoiding third-person phrases like 'the user' or 'one should.'

IMPORTANT: BE CONCISE AND AVOID VERBOSITY. BREVITY IS CRITICAL. Minimize output tokens as much as possible while maintaining helpfulness, quality, and accuracy. Only address the specific query or task at hand.

Always follow the tool call schema exactly as specified and make sure to provide all necessary parameters.
The conversation may reference tools that are no longer available. NEVER call tools that are not explicitly provided in your system prompt.
Before calling each tool, first explain why you are calling it.

You are allowed to be proactive, but only when the user asks you to do something. You should strive to strike a balance between: (a) doing the right thing when asked, including taking actions and follow-up actions, and (b) not surprising the user by taking actions without asking. For example, if the user asks you how to approach something, you should do your best to answer their question first, and not immediately jump into calling a tool.

DO NOT WRITE MORE TEXT AFTER THE TOOL CALLS IN A RESPONSE. You can wait until the next response to summarize the actions you've done.`,
			Description: "Assists with coding and development tasks as a pair programmer.",
		},
	}

	return &Config{
		Debug:                 false,
		MaxCaptureLines:       200,
		MaxContextSize:        100000,
		WaitInterval:          5,
		SendKeysConfirm:       true,
		PasteMultilineConfirm: true,
		ExecConfirm:           true,
		WhitelistPatterns:     []string{},
		BlacklistPatterns:     []string{},
		OpenRouter: OpenRouterConfig{
			BaseURL: "https://openrouter.ai/api/v1",
			Model:   "google/gemini-2.5-flash-preview",
		},
		AzureOpenAI: AzureOpenAIConfig{},
		Browserless: BrowserlessConfig{
			BaseURL: "wss://chrome.browserless.io",
			Token:   "",
		},
		Prompts: PromptsConfig{
			BaseSystem:    ``,
			ChatAssistant: ``,
		},
		Personas:       defaultPersonas,
		DefaultPersona: "pair_programmer",
		PersonaRules:   []PersonaRule{},
	}
}

// Load loads the configuration from file or environment variables
func Load() (*Config, error) {
	config := DefaultConfig()

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	viper.AddConfigPath(".")

	configDir, err := GetConfigDir()
	if err == nil {
		viper.AddConfigPath(configDir)
	} else {
		viper.AddConfigPath(filepath.Join(homeDir, ".config", "tmuxai"))
	}

	// Environment variables
	viper.SetEnvPrefix("TMUXAI")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Automatically bind all config keys to environment variables
	configType := reflect.TypeOf(*config)
	for _, key := range EnumerateConfigKeys(configType, "") {
		_ = viper.BindEnv(key)
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	ResolveEnvKeyInConfig(config)

	// Load personas from directory
	configDir, _ = GetConfigDir()
	personasDir := filepath.Join(configDir, "personas")
	if err := LoadPersonasFromDir(&config.Personas, personasDir); err != nil {
		fmt.Printf("Warning: Failed to load personas from directory: %v\n", err)
	}

	// Override with inline personas if present (inline takes precedence)
	if len(config.Personas) == 0 || config.DefaultPersona == "" {
		defaultPersonas := map[string]*Persona{
			"pair_programmer": {
				Prompt: `You are TmuxAI assistant. You are AI agent and live inside user's Tmux's window and can see all panes in that window.
Think of TmuxAI as a pair programmer that sits beside user, watching users terminal window exactly as user see it.
TmuxAI's design philosophy mirrors the way humans collaborate at the terminal. Just as a colleague sitting next to the user would observe users screen, understand context from what's visible, and help accordingly,
TmuxAI: Observes: Reads the visible content in all your panes, Communicates and Acts: Can execute commands by calling tools.
You and user both are able to control and interact with tmux ai exec pane.

You have perfect understanding of human common sense.
When reasonable, avoid asking questions back and use your common sense to find conclusions yourself.
Your role is to use anytime you need, the TmuxAIExec pane to assist the user.
You are expert in all kinds of shell scripting, shell usage diffence between bash, zsh, fish, powershell, cmd, batch, etc and different OS-es.
You always strive for simple, elegant, clean and effective solutions.
Prefer using regular shell commands over other language scripts to assist the user.

Address the root cause instead of the symptoms.
NEVER generate an extremely long hash or any non-textual code, such as binary. These are not helpful to the USER and are very expensive.
Always address user directly as 'you' in a conversational tone, avoiding third-person phrases like 'the user' or 'one should.'

IMPORTANT: BE CONCISE AND AVOID VERBOSITY. BREVITY IS CRITICAL. Minimize output tokens as much as possible while maintaining helpfulness, quality, and accuracy. Only address the specific query or task at hand.

Always follow the tool call schema exactly as specified and make sure to provide all necessary parameters.
The conversation may reference tools that are no longer available. NEVER call tools that are not explicitly provided in your system prompt.
Before calling each tool, first explain why you are calling it.

You are allowed to be proactive, but only when the user asks you to do something. You should strive to strike a balance between: (a) doing the right thing when asked, including taking actions and follow-up actions, and (b) not surprising the user by taking actions without asking. For example, if the user asks you how to approach something, you should do your best to answer their question first, and not immediately jump into calling a tool.

DO NOT WRITE MORE TEXT AFTER THE TOOL CALLS IN A RESPONSE. You can wait until the next response to summarize the actions you've done.`,
				Description: "Assists with coding and development tasks as a pair programmer.",
			},
		}
		config.Personas = defaultPersonas // Fallback to defaults if nothing loaded
		config.DefaultPersona = "pair_programmer"
	}

	return config, nil
}

// EnumerateConfigKeys returns all config keys (dot notation) for the given struct type.
func EnumerateConfigKeys(cfgType reflect.Type, prefix string) []string {
	var keys []string
	for i := 0; i < cfgType.NumField(); i++ {
		field := cfgType.Field(i)
		tag := field.Tag.Get("mapstructure")
		if tag == "" {
			tag = strings.ToLower(field.Name)
		}
		key := tag
		if prefix != "" {
			key = prefix + "." + tag
		}
		if field.Type.Kind() == reflect.Struct {
			keys = append(keys, EnumerateConfigKeys(field.Type, key)...)
		} else {
			keys = append(keys, key)
		}
	}
	return keys
}

// GetConfigDir returns the path to the tmuxai config directory (~/.config/tmuxai)
func GetConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "tmuxai")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create personas subdirectory
	personasDir := filepath.Join(configDir, "personas")
	if err := os.MkdirAll(personasDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create personas directory: %w", err)
	}

	return configDir, nil
}

// LoadPersonasFromDir loads persona files from the specified directory
func LoadPersonasFromDir(personas *map[string]*Persona, dir string) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Dir doesn't exist yet, no error
		}
		return fmt.Errorf("failed to read personas directory: %w", err)
	}

	loaded := 0
	for _, file := range files {
		if file.IsDir() || (!strings.HasSuffix(file.Name(), ".yaml") && !strings.HasSuffix(file.Name(), ".yml")) {
			continue
		}

		filename := strings.TrimSuffix(file.Name(), ".yaml")
		if strings.HasSuffix(file.Name(), ".yml") {
			filename = strings.TrimSuffix(filename, ".yml")
		}

		if len(*personas) >= 50 { // Limit to prevent overload
			fmt.Printf("Warning: Skipping %s - max 50 personas loaded\\n", file.Name())
			continue
		}

		filePath := filepath.Join(dir, file.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Printf("Warning: Failed to read %s: %v\\n", file.Name(), err)
			continue
		}

		var persona Persona
		if err := yaml.Unmarshal(data, &persona); err != nil {
			fmt.Printf("Warning: Failed to parse %s: %v\\n", file.Name(), err)
			continue
		}

		if persona.Prompt == "" {
			fmt.Printf("Warning: %s has no prompt, skipping\\n", file.Name())
			continue
		}

		(*personas)[filename] = &persona
		loaded++
	}

	if loaded > 0 {
		fmt.Printf("Loaded %d personas from %s\\n", loaded, dir)
	}

	return nil
}

func GetConfigFilePath(filename string) string {
	configDir, _ := GetConfigDir()
	return filepath.Join(configDir, filename)
}

func TryInferType(key, value string) any {
	var typedValue any = value
	// Only basic type inference for bool/int/string
	for i := 0; i < reflect.TypeOf(Config{}).NumField(); i++ {
		field := reflect.TypeOf(Config{}).Field(i)
		tag := field.Tag.Get("mapstructure")
		if tag == "" {
			tag = strings.ToLower(field.Name)
		}
		// Support dot notation for nested fields
		fullKey := tag
		if key == fullKey {
			switch field.Type.Kind() {
			case reflect.Bool:
				switch value {
				case "true":
					typedValue = true
				case "false":
					typedValue = false
				}
			case reflect.Int, reflect.Int64, reflect.Int32:
				var intVal int
				_, err := fmt.Sscanf(value, "%d", &intVal)
				if err == nil {
					typedValue = intVal
				}
			}
		}
		// Nested struct support
		if field.Type.Kind() == reflect.Struct {
			nestedType := field.Type
			prefix := tag + "."
			if strings.HasPrefix(key, prefix) {
				nestedKey := key[len(prefix):]
				for j := 0; j < nestedType.NumField(); j++ {
					nf := nestedType.Field(j)
					ntag := nf.Tag.Get("mapstructure")
					if ntag == "" {
						ntag = strings.ToLower(nf.Name)
					}
					if ntag == nestedKey {
						switch nf.Type.Kind() {
						case reflect.Bool:
							switch value {
							case "true":
								typedValue = true
							case "false":
								typedValue = false
							}
						case reflect.Int, reflect.Int64, reflect.Int32:
							var intVal int
							_, err := fmt.Sscanf(value, "%d", &intVal)
							if err == nil {
								typedValue = intVal
							}
						}
					}
				}
			}
		}
	}
	return typedValue
}

// ResolveEnvKeyInConfig recursively expands environment variables in all string fields of the config struct.
func ResolveEnvKeyInConfig(cfg *Config) {
	val := reflect.ValueOf(cfg).Elem()
	resolveEnvKeyReferenceInValue(val)
}

func resolveEnvKeyReferenceInValue(val reflect.Value) {
	switch val.Kind() {
	case reflect.String:
		val.SetString(os.ExpandEnv(val.String()))
	case reflect.Struct:
		for i := 0; i < val.NumField(); i++ {
			resolveEnvKeyReferenceInValue(val.Field(i))
		}
	case reflect.Ptr:
		if !val.IsNil() {
			resolveEnvKeyReferenceInValue(val.Elem())
		}
	}
}
