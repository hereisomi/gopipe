// Package dispatch routes the gopipe command line to a subcommand handler.
//
// The routing table is data, not control flow: every subcommand is a Command
// value in a registry, and a later phase registers its real handler with
// Register instead of adding a case to a switch in cmd/gopipe/main.go.
//
// Spec: docs/phase1_unified_binary_architecture.md §3 (subcommand model),
// §6 (dispatch implementation), §10 (help output).
package dispatch

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"gopipe/internal/version"
)

// pipelineCommand is the subcommand used when no subcommand is named on the
// command line (spec §3.2, §3.3).
const pipelineCommand = "pipeline"

// Handler runs one subcommand.
//
// args is everything that follows the subcommand name. The return value is the
// process exit code: dispatch hands it back to main, so a handler must never
// call os.Exit itself (spec §6.3).
type Handler func(ctx context.Context, args []string) int

// Command is one entry in the dispatch table.
type Command struct {
	// Name is the canonical subcommand name, e.g. "pipeline".
	Name string
	// Aliases are alternate spellings that route to this command.
	Aliases []string
	// Short is the one-line description rendered in the top-level help block.
	Short string
	// Help is the full text printed by "gopipe help <name>". When it is empty
	// and Run is nil, the not-yet-implemented stub is printed instead.
	Help string
	// Run is the handler. A nil Run means "not implemented yet" (Phase 2+)
	// and is served by the stub handler below.
	Run Handler
	// Verbs holds the second-level verbs of a subcommand group. Only "db" uses
	// it (spec §3.4).
	Verbs []Command
	// DefaultVerb names the verb used when a group is invoked without one,
	// so "gopipe db --json task.json" means "gopipe db extract --json task.json".
	DefaultVerb string
}

// registry is the dispatch table, in registration order. Order matters only
// for the top-level help block, which must match spec §10.
var registry []Command

// Register adds a command to the dispatch table. It is the extension point for
// later phases: a phase 2 package calls it from its own init (wired in beside
// main.go) and gets routing, help and panic recovery for free.
//
// Register panics on a duplicate name, because that is a programmer error
// caught at process start, not a runtime condition to recover from.
func Register(c Command) {
	if find(registry, c.Name) != nil {
		panic(fmt.Sprintf("dispatch: subcommand %q registered twice", c.Name))
	}
	registry = append(registry, c)
}

// UpdateRun replaces the handler of an already-registered command. Updating
// a group also wires its default verb, so UpdateRun("db", extractHandler)
// serves both "db extract" and "db --json" without re-registering either.
// A later phase can update another verb with UpdateRun("db load", handler).
// A missing command is a programmer error and panics during initialization.
func UpdateRun(name string, h Handler) {
	parts := strings.Fields(name)
	if len(parts) > 0 {
		if c := find(registry, parts[0]); c != nil {
			if len(parts) == 1 {
				c.Run = h
				if c.DefaultVerb != "" {
					if v := find(c.Verbs, c.DefaultVerb); v != nil {
						v.Run = h
					}
				}
				return
			}
			if len(parts) == 2 {
				if v := find(c.Verbs, parts[1]); v != nil {
					v.Run = h
					return
				}
			}
		}
	}
	panic(fmt.Sprintf("dispatch: UpdateRun: subcommand %q not found", name))
}

// Commands returns the registered commands in registration order.
func Commands() []Command {
	out := make([]Command, len(registry))
	copy(out, registry)
	return out
}

func find(cmds []Command, name string) *Command {
	for i := range cmds {
		if cmds[i].Name == name {
			return &cmds[i]
		}
		for _, alias := range cmds[i].Aliases {
			if alias == name {
				return &cmds[i]
			}
		}
	}
	return nil
}

func init() {
	// Registration order is the help order of spec §10.
	Register(Command{
		Name:  "pipeline",
		Short: "Run a producer→consumer pipeline (default when no subcommand given)",
		// Phase 2 supplies Run.
	})
	Register(Command{
		Name:        "db",
		Short:       "Execute database queries or load data (db extract / db load)",
		DefaultVerb: "extract",
		Verbs: []Command{
			{Name: "extract", Short: "Execute a SQL query and write the result to a file"},
			{Name: "load", Short: "Bulk-load a file into a target database table"},
		},
	})
	Register(Command{Name: "run", Short: "Run a named pipeline from the registry"})
	Register(Command{Name: "registry", Short: "Manage named pipeline definitions"})
	Register(Command{Name: "shell", Short: "Interactive pipeline REPL"})
	Register(Command{Name: "gui", Short: "Launch native GUI (gopipe-gui.exe)"})
	Register(Command{
		Name:  "version",
		Short: "Print version and build information",
		Help:  "Print version and build information.\n\nUsage:\n  gopipe version\n",
		Run:   versionHandler,
	})
	Register(Command{
		Name:  "help",
		Short: "Print help for a subcommand",
		Help:  "Print help for a subcommand.\n\nUsage:\n  gopipe help [subcommand]\n",
		Run:   helpHandler,
	})
}

// Run routes args (os.Args[1:]) to a subcommand handler and returns the exit
// code the process should use. It never calls os.Exit and never panics: a
// panic inside a handler is recovered and reported as exit code 1 (spec §6.3).
func Run(ctx context.Context, args []string) int {
	return run(ctx, args, os.Stdout, os.Stderr)
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	ctx = withOutput(ctx, stdout, stderr)

	// Detection order is exactly spec §3.3.
	switch {
	case len(args) == 0:
		printHelp(stdout)
		return 0
	case args[0] == "--help" || args[0] == "-h":
		printHelp(stdout)
		return 0
	case args[0] == "--version" || args[0] == "-v":
		version.Fprint(stdout)
		return 0
	}

	if c := find(registry, args[0]); c != nil {
		return invoke(ctx, *c, c.Name, args[1:], stdout, stderr)
	}

	// Not a registered subcommand. Per §3.3 the default branch is the implicit
	// pipeline, in either shape:
	//   flags-first     gopipe --mode parallel -- python a.py -r python b.py
	//   producer-first  gopipe python a.py -r python b.py
	pipeline := find(registry, pipelineCommand)
	if pipeline == nil {
		fmt.Fprintf(stderr, "unknown subcommand %q — run 'gopipe help' for usage\n", args[0])
		return 1
	}
	return invoke(ctx, *pipeline, pipeline.Name, args, stdout, stderr)
}

// invoke runs one command, descending into a group's verbs first. display is
// the qualified name used in messages ("db extract").
func invoke(ctx context.Context, c Command, display string, args []string, stdout, stderr io.Writer) (code int) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(stderr, "gopipe: panic while running %q: %v\n", display, r)
			code = 1
		}
	}()

	// A group with an argument resolves its verb (spec §3.4). A group invoked
	// bare falls through to the group's own handler.
	if len(c.Verbs) > 0 && len(args) > 0 {
		verb, rest, ok := resolveVerb(c, args)
		if !ok {
			fmt.Fprintf(stderr, "unknown subcommand %q — run 'gopipe help' for usage\n", args[0])
			return 1
		}
		return invoke(ctx, *verb, display+" "+verb.Name, rest, stdout, stderr)
	}

	if c.Run == nil {
		return stub(display)(ctx, args)
	}
	return c.Run(ctx, args)
}

// resolveVerb picks the second-level verb of a group (spec §3.4): a
// flags-first group ("db --json task.json") falls back to DefaultVerb, a named
// verb ("db extract") selects that verb. args is non-empty.
func resolveVerb(c Command, args []string) (*Command, []string, bool) {
	if strings.HasPrefix(args[0], "-") {
		v := find(c.Verbs, c.DefaultVerb)
		if v == nil {
			return nil, nil, false
		}
		return v, args, true
	}
	if v := find(c.Verbs, args[0]); v != nil {
		return v, args[1:], true
	}
	return nil, nil, false
}

// stub is the Phase 1 placeholder for every subcommand implemented in a later
// phase: it reports itself and exits 1 (spec §12).
func stub(display string) Handler {
	return func(ctx context.Context, _ []string) int {
		_, stderr := Output(ctx)
		fmt.Fprintln(stderr, stubMessage(display))
		return 1
	}
}

func stubMessage(display string) string {
	return display + " subcommand — not yet implemented"
}

func versionHandler(ctx context.Context, _ []string) int {
	stdout, _ := Output(ctx)
	version.Fprint(stdout)
	return 0
}

func helpHandler(ctx context.Context, args []string) int {
	stdout, stderr := Output(ctx)

	if len(args) == 0 {
		printHelp(stdout)
		return 0
	}

	c := find(registry, args[0])
	if c == nil {
		fmt.Fprintf(stderr, "unknown subcommand %q — run 'gopipe help' for usage\n", args[0])
		return 1
	}

	if len(c.Verbs) > 0 && len(args) > 1 {
		verb := find(c.Verbs, args[1])
		if verb == nil {
			fmt.Fprintf(stderr, "unknown subcommand %q — run 'gopipe help' for usage\n", args[1])
			return 1
		}
		printCommandHelp(stdout, *verb, c.Name+" "+verb.Name)
		return 0
	}

	printCommandHelp(stdout, *c, c.Name)
	return 0
}

func printCommandHelp(w io.Writer, c Command, display string) {
	switch {
	case c.Run == nil:
		fmt.Fprintln(w, stubMessage(display))
	case c.Help != "":
		fmt.Fprint(w, c.Help)
	default:
		fmt.Fprintln(w, c.Short)
	}
}

// printHelp writes the top-level usage block, which must match spec §10
// byte for byte.
func printHelp(w io.Writer) {
	fmt.Fprintln(w, "gopipe — Windows CLI pipeline wrapper for RPA and ETL tools")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  gopipe <subcommand> [flags]")
	fmt.Fprintln(w, "  gopipe <producer...> -r <consumer...> [flags]   (implicit pipeline mode)")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Subcommands:")
	for _, c := range Commands() {
		fmt.Fprintf(w, "  %-12s%s\n", c.Name, c.Short)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintln(w, "  gopipe python export.py -r python ingest.py --mode parallel")
	fmt.Fprintln(w, "  gopipe db extract --json task.json")
	fmt.Fprintln(w, "  gopipe db load --json load_task.json")
	fmt.Fprintln(w, "  gopipe run nightly-cdr")
	fmt.Fprintln(w, "  gopipe shell")
	fmt.Fprintln(w, "  gopipe gui")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Run 'gopipe help <subcommand>' for subcommand-specific flags.")
}

// Handlers write through the context rather than os.Stdout/os.Stderr directly,
// so routing can be tested without capturing the process' file descriptors.
type outputKey struct{}

type writers struct {
	stdout io.Writer
	stderr io.Writer
}

func withOutput(ctx context.Context, stdout, stderr io.Writer) context.Context {
	return context.WithValue(ctx, outputKey{}, writers{stdout: stdout, stderr: stderr})
}

// Output returns the writers a handler should use. It falls back to the
// process' stdout and stderr when the context carries none.
func Output(ctx context.Context) (stdout, stderr io.Writer) {
	w, ok := ctx.Value(outputKey{}).(writers)
	if !ok || w.stdout == nil || w.stderr == nil {
		return os.Stdout, os.Stderr
	}
	return w.stdout, w.stderr
}
