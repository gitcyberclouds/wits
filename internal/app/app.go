//go:build linux || darwin || freebsd || windows

package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/internal/output"
	"github.com/pranshuparmar/witr/internal/pipeline"
	procpkg "github.com/pranshuparmar/witr/internal/proc"
	"github.com/pranshuparmar/witr/internal/source"
	"github.com/pranshuparmar/witr/internal/target"
	"github.com/pranshuparmar/witr/internal/tui"
	"github.com/pranshuparmar/witr/pkg/model"
	"github.com/spf13/cobra"
)

var (
	version   = "v0.0.0-dev"
	commit    = "unknown"
	buildDate = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "witr [process name...]",
	Short: "Why is this running?",
	Long:  "witr explains why a process or port is running by tracing its ancestry.",
	Args:  cobra.ArbitraryArgs,
	CompletionOptions: cobra.CompletionOptions{
		HiddenDefaultCmd:  false,
		DisableDefaultCmd: false,
		DisableNoDescFlag: false,
	},
	Example: _genExamples(),
	RunE:    runApp,
}

func _genExamples() string {

	return `
  # Inspect a running process by name
  witr nginx

  # Look up a process by PID
  witr --pid 1234

  # Find the process listening on a specific port
  witr --port 5432

  # Find the process holding a lock on a file
  witr --file /var/lib/dpkg/lock

  # Inspect a process by name with exact matching (no fuzzy search)
  witr bun --exact

  # Show the full process ancestry (who started whom)
  witr postgres --tree

  # Show only warnings (suspicious env, arguments, parents)
  witr docker --warnings

  # Display only environment variables of the process
  witr node --env

  # Short, single-line output (useful for scripts)
  witr sshd --short

  # Disable colorized output (CI or piping)
  witr redis --no-color

  # Output machine-readable JSON
  witr chrome --json

  # Show extended process information (memory, I/O, file descriptors)
  witr mysql --verbose

  # Combine flags: inspect port, show environment variables, output JSON
  witr --port 8080 --env --json

  # Multiple inputs
  witr nginx node
  witr --port 8080 --port 3000
  witr --pid 1234 --pid 5678

  # Mixed inputs
  witr nginx --pid 1234 --port 8080
`
}

// Exit codes
const (
	ExitOK            = 0
	ExitWarnings      = 1
	ExitInternalError = 1
	ExitNotFound      = 2
	ExitPermission    = 3
	ExitInvalidInput  = 4
)

// exitCodeError wraps an error with a specific exit code.
type exitCodeError struct {
	code int
	err  error
}

func (e *exitCodeError) Error() string { return e.err.Error() }
func (e *exitCodeError) Unwrap() error { return e.err }

func withExitCode(code int, err error) error {
	return &exitCodeError{code: code, err: err}
}

func Execute() {
	err := rootCmd.Execute()
	if err == nil {
		return
	}

	var ece *exitCodeError
	if errors.As(err, &ece) {
		os.Exit(ece.code)
	}
	os.Exit(1)
}

func init() {
	rootCmd.InitDefaultCompletionCmd()
	rootCmd.Version = version
	rootCmd.SetVersionTemplate(fmt.Sprintf("witr {{.Version}} (commit %s, built %s)\n", commit, buildDate))
	rootCmd.SetErr(output.NewSafeTerminalWriter(os.Stderr))

	rootCmd.Flags().StringSliceP("pid", "p", nil, "pid(s) to look up (repeatable)")
	rootCmd.Flags().StringSliceP("port", "o", nil, "port(s) to look up (repeatable)")
	rootCmd.Flags().StringSliceP("file", "f", nil, "file path(s) to find process for (repeatable)")
	rootCmd.Flags().BoolP("short", "s", false, "show only ancestry")
	rootCmd.Flags().BoolP("tree", "t", false, "show only ancestry as a tree")
	rootCmd.Flags().Bool("json", false, "show result as JSON")
	rootCmd.Flags().Bool("warnings", false, "show only warnings")
	rootCmd.Flags().Bool("no-color", false, "disable colorized output")
	rootCmd.Flags().Bool("env", false, "show environment variables for the process")
	rootCmd.Flags().Bool("verbose", false, "show extended process information")
	rootCmd.Flags().BoolP("exact", "x", false, "use exact name matching (no substring search)")
	rootCmd.Flags().BoolP("interactive", "i", false, "interactive mode (TUI)")

}

// appFlags holds all parsed CLI flags for convenience.
type appFlags struct {
	short   bool
	tree    bool
	json    bool
	warn    bool
	noColor bool
	verbose bool
	exact   bool
	env     bool
}

func runApp(cmd *cobra.Command, args []string) error {
	interactiveFlag, _ := cmd.Flags().GetBool("interactive")
	if interactiveFlag {
		return runInteractive()
	}

	envFlag, _ := cmd.Flags().GetBool("env")
	pidFlags, _ := cmd.Flags().GetStringSlice("pid")
	portFlags, _ := cmd.Flags().GetStringSlice("port")
	fileFlags, _ := cmd.Flags().GetStringSlice("file")

	// Default to interactive mode if no arguments or relevant flags are provided
	if !envFlag && len(pidFlags) == 0 && len(portFlags) == 0 && len(fileFlags) == 0 && len(args) == 0 {
		return runInteractive()
	}

	flags := appFlags{
		env:     envFlag,
		exact:   boolFlag(cmd, "exact"),
		short:   boolFlag(cmd, "short"),
		tree:    boolFlag(cmd, "tree"),
		json:    boolFlag(cmd, "json"),
		warn:    boolFlag(cmd, "warnings"),
		noColor: boolFlag(cmd, "no-color"),
		verbose: boolFlag(cmd, "verbose"),
	}

	// Collect all targets preserving command-line order
	targets := collectTargetsInOrder(os.Args[1:], args)

	if len(targets) == 0 {
		return withExitCode(ExitInvalidInput, fmt.Errorf("must specify --pid, --port, --file, or a process name"))
	}

	outw := cmd.OutOrStdout()
	outp := output.NewPrinter(outw)
	multiMode := len(targets) > 1
	colorEnabled := !flags.noColor

	// For JSON multi-output, collect all JSON strings and wrap in array
	var jsonResults []string
	highestExit := ExitOK

	for i, t := range targets {
		if multiMode && !flags.json {
			printDivider(outp, t, colorEnabled, i > 0)
		}

		exitCode := processTarget(cmd, outw, outp, t, flags, multiMode, &jsonResults)
		if exitCode > highestExit {
			highestExit = exitCode
		}
	}

	// Emit JSON array for multi-target
	if flags.json && multiMode {
		indented := make([]string, len(jsonResults))
		for i, r := range jsonResults {
			lines := strings.Split(r, "\n")
			for j := range lines {
				if j > 0 {
					lines[j] = "  " + lines[j]
				}
			}
			indented[i] = "  " + strings.Join(lines, "\n")
		}
		fmt.Fprintf(outw, "[\n%s\n]\n", strings.Join(indented, ",\n"))
	}

	if highestExit > ExitOK {
		cmd.SilenceErrors = true
		return withExitCode(highestExit, fmt.Errorf("completed with exit code %d", highestExit))
	}
	return nil
}

func boolFlag(cmd *cobra.Command, name string) bool {
	v, _ := cmd.Flags().GetBool(name)
	return v
}

// collectTargetsInOrder walks the raw command-line arguments to build a target
// list that preserves the order the user typed them in.
func collectTargetsInOrder(rawArgs []string, positionalArgs []string) []model.Target {
	var targets []model.Target
	positionalIdx := 0

	// Map flag names to target types
	flagType := map[string]model.TargetType{
		"-p": model.TargetPID, "--pid": model.TargetPID,
		"-o": model.TargetPort, "--port": model.TargetPort,
		"-f": model.TargetFile, "--file": model.TargetFile,
	}

	// Track which positional args we've placed so we can insert them in order
	// between flag-based targets
	i := 0
	for i < len(rawArgs) {
		arg := rawArgs[i]

		// Check for --flag=value form
		if strings.HasPrefix(arg, "--") {
			if eqIdx := strings.Index(arg, "="); eqIdx >= 0 {
				flagName := arg[:eqIdx]
				flagVal := arg[eqIdx+1:]
				if tt, ok := flagType[flagName]; ok {
					for _, v := range strings.Split(flagVal, ",") {
						v = strings.TrimSpace(v)
						if v != "" {
							targets = append(targets, model.Target{Type: tt, Value: v})
						}
					}
				}
				i++
				continue
			}
		}

		// Check for -f value or --flag value form
		if tt, ok := flagType[arg]; ok {
			if i+1 < len(rawArgs) {
				i++
				for _, v := range strings.Split(rawArgs[i], ",") {
					v = strings.TrimSpace(v)
					if v != "" {
						targets = append(targets, model.Target{Type: tt, Value: v})
					}
				}
			}
			i++
			continue
		}

		// Skip known boolean flags and their short forms
		if strings.HasPrefix(arg, "-") {
			i++
			continue
		}

		// Positional argument — use it as a name target
		if positionalIdx < len(positionalArgs) {
			targets = append(targets, model.Target{Type: model.TargetName, Value: positionalArgs[positionalIdx]})
			positionalIdx++
		}
		i++
	}

	// Append any remaining positional args that weren't matched
	for positionalIdx < len(positionalArgs) {
		targets = append(targets, model.Target{Type: model.TargetName, Value: positionalArgs[positionalIdx]})
		positionalIdx++
	}

	return targets
}

// targetLabel returns a human-readable label for the divider.
func targetLabel(t model.Target) string {
	switch t.Type {
	case model.TargetPID:
		return fmt.Sprintf("pid: %s", t.Value)
	case model.TargetPort:
		return fmt.Sprintf("port: %s", t.Value)
	case model.TargetFile:
		return fmt.Sprintf("file: %s", t.Value)
	default:
		return fmt.Sprintf("name: %s", t.Value)
	}
}

func printDivider(outp output.Printer, t model.Target, colorEnabled bool, needsNewline bool) {
	label := targetLabel(t)
	if needsNewline {
		outp.Println()
	}
	if colorEnabled {
		outp.Printf("%s----- [%s] -----%s\n", output.ColorCyan, label, output.ColorReset)
	} else {
		outp.Printf("----- [%s] -----\n", label)
	}
}

// jsonErrorEntry returns a JSON string representing a failed target lookup.
func jsonErrorEntry(t model.Target, errMsg string) string {
	type errorEntry struct {
		Target model.Target
		Error  string
	}
	data, _ := json.MarshalIndent(errorEntry{Error: errMsg, Target: t}, "", "  ")
	return string(data)
}

// processTarget handles resolving and rendering a single target.
// Returns the exit code for this target.
func processTarget(cmd *cobra.Command, outw io.Writer, outp output.Printer, t model.Target, flags appFlags, multiMode bool, jsonResults *[]string) int {
	colorEnabled := !flags.noColor

	if flags.env {
		return processEnvTarget(outw, outp, t, flags, multiMode, jsonResults)
	}

	pids, err := target.Resolve(t, flags.exact)
	if err == nil && len(pids) == 0 {
		err = fmt.Errorf("no matching process found")
	}
	if err != nil {
		return handleResolveError(cmd, outw, outp, t, err, flags, multiMode, jsonResults)
	}

	if len(pids) > 1 {
		if multiMode && flags.json {
			*jsonResults = append(*jsonResults, jsonErrorEntry(t, fmt.Sprintf("multiple processes matched (%d results)", len(pids))))
		} else {
			hint := "witr --pid <pid>"
			if flags.env {
				hint = "witr --pid <pid> --env"
			}
			printMultiMatch(outp, pids, colorEnabled, hint)
		}
		return ExitInvalidInput
	}

	pid := pids[0]

	var systemdService string
	if t.Type == model.TargetPort && pid == 1 && source.IsSystemdRunning() {
		if portNum, err := strconv.Atoi(t.Value); err == nil {
			if svc, err := procpkg.ResolveSystemdService(portNum); err == nil && svc != "" {
				systemdService = svc
			}
		}
	}

	res, err := pipeline.AnalyzePID(pipeline.AnalyzeConfig{
		PID:     pid,
		Verbose: flags.verbose,
		Tree:    flags.tree,
		Target:  t,
	})

	if err != nil {
		if multiMode {
			if flags.json {
				*jsonResults = append(*jsonResults, jsonErrorEntry(t, err.Error()))
			} else {
				outp.Printf("Error: %v\n", err)
			}
			return classifyError(err)
		}
		errStr := err.Error()
		errorMsg := fmt.Sprintf("%s\n\nNo matching process or service found. Please check your query or try a different name/port/PID.\nFor usage and options, run: witr --help", errStr)
		cmd.PrintErrln(errorMsg)
		return classifyError(err)
	}

	if systemdService != "" {
		res.ResolvedTarget = strings.TrimSuffix(systemdService, ".service")
	}

	if t.Type == model.TargetPort {
		portNum := 0
		fmt.Sscanf(t.Value, "%d", &portNum)
		if portNum > 0 {
			res.SocketInfo = procpkg.GetSocketStateForPort(portNum)
			source.EnrichSocketInfo(res.SocketInfo)
		}
	}

	renderResult(outw, res, flags, multiMode, jsonResults)

	if len(res.Warnings) > 0 {
		return ExitWarnings
	}
	return ExitOK
}

// processEnvTarget handles the --env flag for a single target.
func processEnvTarget(outw io.Writer, outp output.Printer, t model.Target, flags appFlags, multiMode bool, jsonResults *[]string) int {
	colorEnabled := !flags.noColor

	pids, err := target.Resolve(t, flags.exact)
	if err != nil {
		if multiMode {
			if flags.json {
				*jsonResults = append(*jsonResults, jsonErrorEntry(t, err.Error()))
			} else {
				outp.Printf("Error: %v\n", err)
			}
			return classifyError(err)
		}
		outp.Printf("error: %v\n", err)
		return classifyError(err)
	}
	if len(pids) == 0 {
		if multiMode && flags.json {
			*jsonResults = append(*jsonResults, jsonErrorEntry(t, "no matching process found"))
			return ExitNotFound
		}
		outp.Println("No matching process found.")
		return ExitNotFound
	}
	if len(pids) > 1 {
		printMultiMatch(outp, pids, colorEnabled, "witr --pid <pid> --env")
		return ExitInvalidInput
	}

	pid := pids[0]
	procInfo, err := procpkg.ReadProcess(pid)
	if err != nil {
		outp.Printf("error: %v\n", err)
		return ExitInternalError
	}

	resEnv := model.Result{
		Process:  procInfo,
		Ancestry: []model.Process{procInfo},
	}

	if flags.json {
		jsonStr, err := output.ToEnvJSON(resEnv)
		if err != nil {
			outp.Printf("failed to generate json output: %v\n", err)
			return ExitInternalError
		}
		if multiMode {
			*jsonResults = append(*jsonResults, jsonStr)
		} else {
			fmt.Fprintln(outw, jsonStr)
		}
	} else {
		output.RenderEnvOnly(outw, resEnv, colorEnabled)
	}
	return ExitOK
}

// handleResolveError handles target resolution errors, including Docker fallback.
func handleResolveError(cmd *cobra.Command, outw io.Writer, outp output.Printer, t model.Target, err error, flags appFlags, multiMode bool, jsonResults *[]string) int {
	errStr := err.Error()
	colorEnabled := !flags.noColor

	if strings.Contains(errStr, "socket found but owning process not detected") {
		if t.Type == model.TargetPort {
			if portNum, convErr := strconv.Atoi(t.Value); convErr == nil {
				if match := procpkg.ResolveContainerByPort(portNum); match != nil {
					if flags.json {
						jsonStr, jsonErr := output.DockerFallbackToJSON(t.Value, match)
						if jsonErr != nil {
							outp.Printf("failed to generate json output: %v\n", jsonErr)
							return ExitInternalError
						}
						if multiMode {
							*jsonResults = append(*jsonResults, jsonStr)
						} else {
							fmt.Fprintln(outw, jsonStr)
						}
					} else if flags.short {
						output.RenderDockerFallbackShort(outw, t.Value, match, colorEnabled)
					} else {
						output.RenderDockerFallback(outw, t.Value, match, colorEnabled)
					}
					return ExitOK
				}
			}
		}
		if multiMode {
			if flags.json {
				*jsonResults = append(*jsonResults, jsonErrorEntry(t, "socket found but owning process not detected (try sudo)"))
			} else {
				outp.Printf("Error: socket found but owning process not detected (try sudo)\n")
			}
			return ExitPermission
		}
		errorMsg := fmt.Sprintf("%s\n\nA socket was found for the port, but the owning process could not be detected.\nThis may be due to insufficient permissions. Try running with sudo:\n  sudo %s", errStr, strings.Join(os.Args, " "))
		cmd.PrintErrln(errorMsg)
		return ExitPermission
	}

	if multiMode {
		if flags.json {
			*jsonResults = append(*jsonResults, jsonErrorEntry(t, errStr))
		} else {
			outp.Printf("Error: %v\n", err)
		}
		return classifyError(err)
	}
	errorMsg := fmt.Sprintf("%s\n\nNo matching process or service found. Please check your query or try a different name/port/PID.\nFor usage and options, run: witr --help", errStr)
	cmd.PrintErrln(errorMsg)
	return classifyError(err)
}

// renderResult renders a single result in the appropriate output mode.
func renderResult(outw io.Writer, res model.Result, flags appFlags, multiMode bool, jsonResults *[]string) {
	colorEnabled := !flags.noColor

	if flags.json {
		var jsonStr string
		var err error

		if flags.short {
			jsonStr, err = output.ToShortJSON(res)
		} else if flags.tree {
			jsonStr, err = output.ToTreeJSON(res)
		} else if flags.warn {
			jsonStr, err = output.ToWarningsJSON(res)
		} else {
			jsonStr, err = output.ToJSON(res)
		}

		if err != nil {
			fmt.Fprintf(outw, "failed to generate json output: %v\n", err)
			return
		}
		if multiMode {
			*jsonResults = append(*jsonResults, jsonStr)
		} else {
			fmt.Fprintln(outw, jsonStr)
		}
	} else if flags.warn {
		output.RenderWarnings(outw, res, colorEnabled)
	} else if flags.tree {
		output.PrintTree(outw, res.Ancestry, res.Children, colorEnabled)
	} else if flags.short {
		output.RenderShort(outw, res, colorEnabled)
	} else {
		output.RenderStandard(outw, res, colorEnabled, flags.verbose)
	}
}

func Root() *cobra.Command { return rootCmd }

func runInteractive() error {
	v := version
	if v == "v0.0.0-dev" {
		v = ""
	}
	return tui.Start(v)
}

func printMultiMatch(outp output.Printer, pids []int, colorEnabled bool, hint string) {
	outp.Print("Multiple matching processes found:\n\n")
	for i, pid := range pids {
		proc, err := procpkg.ReadProcess(pid)
		var command, cmdline string
		if err != nil {
			command = "unknown"
			cmdline = procpkg.GetCmdline(pid)
		} else {
			command = proc.Command
			cmdline = proc.Cmdline
		}
		if colorEnabled {
			outp.Printf("[%d] %s%s%s (%spid %d%s)\n    %s\n",
				i+1, output.ColorGreen, command, output.ColorReset,
				output.ColorDim, pid, output.ColorReset,
				cmdline)
		} else {
			outp.Printf("[%d] %s (pid %d)\n    %s\n", i+1, command, pid, cmdline)
		}
	}
	outp.Println("\nRe-run with:")
	outp.Printf("  %s\n", hint)
}

// classifyError maps common error strings to exit codes.
func classifyError(err error) int {
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "permission denied") ||
		strings.Contains(msg, "operation not permitted") ||
		strings.Contains(msg, "insufficient permissions"):
		return ExitPermission
	case strings.Contains(msg, "no matching") ||
		strings.Contains(msg, "no running process") ||
		strings.Contains(msg, "not found") ||
		strings.Contains(msg, "no process"):
		return ExitNotFound
	case strings.Contains(msg, "invalid") ||
		strings.Contains(msg, "must specify"):
		return ExitInvalidInput
	default:
		return ExitInternalError
	}
}

func SetVersion(v string, c string, bd string) {
	version = v
	commit = c
	buildDate = bd

	rootCmd.Version = version
	rootCmd.SetVersionTemplate(fmt.Sprintf("witr {{.Version}} (commit %s, built %s)\n", commit, buildDate))
	rootCmd.SilenceUsage = true
}
