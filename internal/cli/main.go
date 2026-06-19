package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/pflag"

	"github.com/netspeedy/s3ctl/internal/buildinfo"
)

// Main runs the CLI using the current process environment.
func Main(args []string, stdout, stderr io.Writer) int {
	return MainWithEnv(args, envMap(os.Environ()), stdout, stderr)
}

func envMap(values []string) map[string]string {
	result := make(map[string]string, len(values))
	for _, item := range values {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}

func shouldShowIntroHelp(args []string, _ map[string]string) bool {
	return len(args) == 0
}

// MainWithEnv runs the CLI with an explicit environment map.
func MainWithEnv(args []string, env map[string]string, stdout, stderr io.Writer) int {
	if shouldShowIntroHelp(args, env) {
		if err := writeUsage(stdout); err != nil {
			return 1
		}
		return 0
	}

	requestedOutput := detectOutputFormat(args, env)
	cfg, parsed, err := resolveSettings(args, env)
	if err != nil {
		if errors.Is(err, pflag.ErrHelp) || parsed.showHelp || parsed.showHelpFull {
			if writeErr := writeUsageForArgs(stdout, args); writeErr != nil {
				return 1
			}
			return 0
		}
		if requestedOutput == "json" {
			if writeErr := writeJSONError(stdout, cfg, err, "configuration_error"); writeErr != nil {
				return 1
			}
			return 1
		}
		if _, writeErr := fmt.Fprintf(stderr, "Error: %s\n\n", err); writeErr != nil {
			return 1
		}
		if writeErr := writeUsage(stderr); writeErr != nil {
			return 1
		}
		return 1
	}

	if parsed.showHelp || parsed.showHelpFull {
		if err := writeUsageForArgs(stdout, args); err != nil {
			return 1
		}
		return 0
	}

	if parsed.showVersion {
		if cfg.Output == "json" {
			encoder := json.NewEncoder(stdout)
			encoder.SetIndent("", "  ")
			if err := encoder.Encode(buildinfo.Current(binaryName)); err != nil {
				return 1
			}
			return 0
		}

		if _, err := fmt.Fprintln(stdout, buildinfo.Summary(binaryName)); err != nil {
			return 1
		}
		return 0
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ParsedTimeout)
	defer cancel()

	result, err := provision(ctx, cfg)
	if err != nil {
		if cfg.Output == "json" {
			if writeErr := writeJSONError(stdout, cfg, err, "operation_failed"); writeErr != nil {
				return 1
			}
			return 1
		}
		if _, writeErr := fmt.Fprintf(stderr, "Error: %s\n", renderErrorMessage(err)); writeErr != nil {
			return 1
		}
		return 1
	}

	if cfg.Output == "json" {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			if _, writeErr := fmt.Fprintf(stderr, "Error: %s\n", err); writeErr != nil {
				return 1
			}
			return 1
		}
		return 0
	}

	if _, err := fmt.Fprintln(stdout, renderText(result)); err != nil {
		return 1
	}
	return 0
}

func operationFromSettings(cfg settings) string {
	switch {
	case cfg.DeleteBucket:
		return operationDelete
	case cfg.OVHRepairPolicies:
		return operationRepair
	default:
		return operationProvision
	}
}
