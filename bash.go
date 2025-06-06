package mcptools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/shaharia-lab/goai"
)

const BashToolName = "bash"

// Bash represents a wrapper around the system's bash command-line tool
type Bash struct {
	logger      goai.Logger
	cmdExecutor CommandExecutor
}

// NewBash creates a new instance of the Bash wrapper
func NewBash(logger goai.Logger) *Bash {
	return &Bash{
		logger:      logger,
		cmdExecutor: &RealCommandExecutor{},
	}
}

// BashAllInOneTool returns a goai.Tool that can execute bash commands
func (b *Bash) BashAllInOneTool() goai.Tool {
	return goai.Tool{
		Name:        BashToolName,
		Description: "Execute bash commands with specified script or command",
		InputSchema: json.RawMessage(`{
            "type": "object",
            "properties": {
                "command": {
                    "type": "string",
                    "description": "Bash command or script to execute"
                },
                "args": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    },
                    "description": "Additional arguments for the command"
                }
            },
            "required": ["command"]
        }`),
		Handler: func(ctx context.Context, params goai.CallToolParams) (goai.CallToolResult, error) {
			var input struct {
				Command string   `json:"command"`
				Args    []string `json:"args"`
			}

			b.logger.WithFields(map[string]interface{}{"tool": BashToolName}).Info("Received input", "input", string(params.Arguments))

			if err := json.Unmarshal(params.Arguments, &input); err != nil {
				b.logger.WithFields(map[string]interface{}{"tool": BashToolName}).Error("Failed to parse input", "error", err)
				return goai.CallToolResult{}, fmt.Errorf("failed to parse input: %w", err)
			}

			b.logger.Info("Executing bash command", "command", input.Command, "args", input.Args)
			cmd := exec.Command("bash", append([]string{"-c", input.Command}, input.Args...)...)
			output, err := b.cmdExecutor.ExecuteCommand(ctx, cmd)
			if err != nil {
				b.logger.WithFields(map[string]interface{}{"tool": BashToolName}).Error("Failed to execute bash command", "error", err)
				return returnErrorOutput(err), nil
			}

			o := string(output)
			b.logger.WithFields(map[string]interface{}{"tool": BashToolName, "output_length": len(o)}).Info("Bash command executed successfully")
			return goai.CallToolResult{
				Content: []goai.ToolResultContent{{Type: "text", Text: o}},
				IsError: false,
			}, nil
		},
	}
}
