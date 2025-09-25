package internal

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/alvinunreal/tmuxai/config"
	"github.com/alvinunreal/tmuxai/logger"
	"github.com/alvinunreal/tmuxai/system"
)

const helpMessage = `Available commands:
- /info: Display system information
- /clear: Clear the chat history
- /reset: Reset the chat history
- /prepare: Prepare the pane for TmuxAI automation
- /watch <prompt>: Start watch mode
- /squash: Summarize the chat history
- /exit: Exit the application
- /persona [name]: List available personas or switch to the specified one`

var commands = []string{
	"/help",
	"/clear",
	"/reset",
	"/exit",
	"/info",
	"/watch",
	"/prepare",
	"/config",
	"/squash",
	"/persona",
}

// checks if the given content is a command
func (m *Manager) IsMessageSubcommand(content string) bool {
	content = strings.TrimSpace(strings.ToLower(content)) // Normalize input

	// Any message starting with / is considered a command
	return strings.HasPrefix(content, "/")
}

// processes a command and returns a response
func (m *Manager) ProcessSubCommand(command string) {
	commandLower := strings.ToLower(strings.TrimSpace(command))
	logger.Info("Processing command: %s", command)

	// Get the first word from the command (e.g., "/watch" from "/watch something")
	parts := strings.Fields(commandLower)
	if len(parts) == 0 {
		m.Println("Empty command")
		return
	}

	commandPrefix := parts[0]

	// Process the command using prefix matching
	switch {
	case prefixMatch(commandPrefix, "/help"):
		m.Println(helpMessage)
		return

	case prefixMatch(commandPrefix, "/info"):
		m.formatInfo()
		return

	case prefixMatch(commandPrefix, "/prepare"):
		supportedShells := []string{"bash", "zsh", "fish"}
		m.InitExecPane()

		// Check if exec pane is a subshell
		if m.ExecPane.IsSubShell {
			if len(parts) > 1 {
				shell := parts[1]
				isSupported := false
				for _, supportedShell := range supportedShells {
					if shell == supportedShell {
						isSupported = true
						break
					}
				}
				if !isSupported {
					m.Println(fmt.Sprintf("Shell '%s' is not supported. Supported shells are: %s", shell, strings.Join(supportedShells, ", ")))
					return
				}
				m.PrepareExecPaneWithShell(shell)
			} else {
				m.Println("Shell detection is not supported on subshells.")
				m.Println("Please specify the shell manually: /prepare bash, /prepare zsh, or /prepare fish")
				return
			}
		} else {
			if len(parts) > 1 {
				shell := parts[1]
				isSupported := false
				for _, supportedShell := range supportedShells {
					if shell == supportedShell {
						isSupported = true
						break
					}
				}

				if !isSupported {
					m.Println(fmt.Sprintf("Shell '%s' is not supported. Supported shells are: %s", shell, strings.Join(supportedShells, ", ")))
					return
				}
				m.PrepareExecPaneWithShell(shell)
			} else {
				m.PrepareExecPane()
			}
		}

		// for latency over ssh connections
		time.Sleep(500 * time.Millisecond)
		m.ExecPane.Refresh(m.GetMaxCaptureLines())
		m.Messages = []ChatMessage{}

		fmt.Println(m.ExecPane.String())
		m.parseExecPaneCommandHistory()

		logger.Debug("Parsed exec history:")
		for _, history := range m.ExecHistory {
			logger.Debug(fmt.Sprintf("Command: %s\nOutput: %s\nCode: %d\n", history.Command, history.Output, history.Code))
		}

		return

	case prefixMatch(commandPrefix, "/clear"):
		m.Messages = []ChatMessage{}
		_ = system.TmuxClearPane(m.PaneId)
		return

	case prefixMatch(commandPrefix, "/reset"):
		m.Status = ""
		m.Messages = []ChatMessage{}
		_ = system.TmuxClearPane(m.PaneId)
		_ = system.TmuxClearPane(m.ExecPane.Id)
		return

	case prefixMatch(commandPrefix, "/exit"):
		logger.Info("Exit command received, stopping watch mode (if active) and exiting.")
		os.Exit(0)
		return

	case prefixMatch(commandPrefix, "/squash"):
		m.squashHistory()
		return

	case prefixMatch(commandPrefix, "/watch") || commandPrefix == "/w":
		parts := strings.Fields(command)
		if len(parts) > 1 {
			watchDesc := strings.Join(parts[1:], " ")
			startWatch := `
1. Find out if there is new content in the pane based on chat history.
2. Comment only considering the new content in this pane output.

Watch for: ` + watchDesc
			m.Status = "running"
			m.WatchMode = true
			m.startWatchMode(startWatch)
			return
		}
		m.Println("Usage: /watch <description>")
		return

	case prefixMatch(commandPrefix, "/persona"):
		if len(parts) > 1 {
			m.switchPersona(parts[1])
		} else {
			m.listPersonas()
		}
		return

	case prefixMatch(commandPrefix, "/config"):
		// Helper function to check if a key is allowed
		isKeyAllowed := func(key string) bool {
			for _, k := range AllowedConfigKeys {
				if k == key {
					return true
				}
			}
			return false
		}

		// Check if it's "config set" for a specific key
		if len(parts) >= 3 && parts[1] == "set" {
			key := parts[2]
			if !isKeyAllowed(key) {
				m.Println(fmt.Sprintf("Cannot set '%s'. Only these keys are allowed: %s", key, strings.Join(AllowedConfigKeys, ", ")))
				return
			}
			value := strings.Join(parts[3:], " ")
			m.SessionOverrides[key] = config.TryInferType(key, value)
			m.Println(fmt.Sprintf("Set %s = %v", key, m.SessionOverrides[key]))
			return
		} else {
			code, _ := system.HighlightCode("yaml", m.FormatConfig())
			fmt.Println(code)
			return
		}

	default:
		m.Println(fmt.Sprintf("Unknown command: %s. Type '/help' to see available commands.", command))
		return
	}
}

// Helper function to check if a command matches a prefix
func prefixMatch(command, target string) bool {
	return strings.HasPrefix(target, command)
}

// listPersonas lists all available personas
func (m *Manager) listPersonas() {
	m.Println("Available personas:")
	for name, persona := range m.Config.Personas {
		m.Println(fmt.Sprintf("- %s: %s", name, persona.Description))
	}
	if len(m.Config.Personas) == 0 {
		m.Println("No personas loaded. Check config or personas directory.")
	}
}

// switchPersona switches to the specified persona
func (m *Manager) switchPersona(name string) {
	logger.Debug("Attempting to switch to persona: '%s'", name)
	logger.Debug("Current persona before switch: '%s'", m.CurrentPersona)

	if persona, ok := m.Config.Personas[name]; ok {
		m.CurrentPersona = name
		logger.Info("Successfully switched to persona: '%s'", name)
		m.Println(fmt.Sprintf("Switched to persona: %s - %s", name, persona.Description))
	} else {
		logger.Warn("Persona '%s' not found", name)
		keys := make([]string, 0, len(m.Config.Personas))
		for k := range m.Config.Personas {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		m.Println(fmt.Sprintf("Persona '%s' not found. Available: %s", name, strings.Join(keys, ", ")))
	}
}

// formats system information and tmux details into a readable string
func (m *Manager) formatInfo() {
	formatter := system.NewInfoFormatter()
	const labelWidth = 18 // Width of the label column
	formatLine := func(key string, value any) {
		fmt.Print(formatter.LabelColor.Sprintf("%-*s", labelWidth, key))
		fmt.Print("  ")
		fmt.Println(value)
	}
	// Display general information
	fmt.Println(formatter.FormatSection("\nGeneral"))
	formatLine("Version", Version)
	formatLine("Max Capture Lines", m.Config.MaxCaptureLines)
	formatLine("Wait Interval", m.Config.WaitInterval)

	// Display context information section
	fmt.Println(formatter.FormatSection("\nContext"))
	formatLine("Messages", len(m.Messages))
	var totalTokens int
	for _, msg := range m.Messages {
		totalTokens += system.EstimateTokenCount(msg.Content)
	}

	usagePercent := 0.0
	if m.GetMaxContextSize() > 0 {
		usagePercent = float64(totalTokens) / float64(m.GetMaxContextSize()) * 100
	}
	fmt.Print(formatter.LabelColor.Sprintf("%-*s", labelWidth, "Context Size~"))
	fmt.Print("  ") // Two spaces for separation
	fmt.Printf("%s\n", fmt.Sprintf("%d tokens", totalTokens))
	fmt.Printf("%-*s  %s\n", labelWidth, "", formatter.FormatProgressBar(usagePercent, 10))
	formatLine("Max Size", fmt.Sprintf("%d tokens", m.GetMaxContextSize()))

	// Display tmux panes section
	fmt.Println()
	fmt.Println(formatter.FormatSection("Tmux Window Panes"))

	panes, _ := m.GetTmuxPanes()
	for _, pane := range panes {
		pane.Refresh(m.GetMaxCaptureLines())
		fmt.Println(pane.FormatInfo(formatter))
	}
}
