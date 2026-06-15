package cli

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/indihood/yore/pkg/recipe"
	"github.com/indihood/yore/pkg/store"
)

// loadRecipes returns the merged recipe set: personal recipes overlaid by the
// project's .yorefile (project recipes win on name clashes).
func loadRecipes() []recipe.Recipe {
	var global, project []recipe.Recipe
	if gp, err := recipe.GlobalPath(); err == nil {
		global, _ = recipe.LoadFile(gp, "global")
	}
	if pp, ok := recipe.FindProjectFile(currentDir()); ok {
		project, _ = recipe.LoadFile(pp, "project")
	}
	return recipe.Merge(global, project)
}

const saveUsage = `Usage: yore save [flags] <name> [-- command...]

Save a command as a reusable recipe. Provide the command after --, or use
--last to capture the most recently recorded command. Write {placeholders} for
the parts that vary; 'yore run' will prompt for them.

  yore save deploy -- ./deploy.sh --env {env}
  yore save --last "fix the thing"
  yore save --project build -- make all      # write to ./.yorefile (team)

Flags:
  --desc TEXT     a short description
  --last          use the most recently recorded command
  --project       save to the project's .yorefile instead of personal recipes
`

// runSave handles `yore save`.
func runSave(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("yore save", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprint(stderr, saveUsage) }
	var (
		desc    string
		useLast bool
		project bool
	)
	fs.StringVar(&desc, "desc", "", "")
	fs.BoolVar(&useLast, "last", false, "")
	fs.BoolVar(&project, "project", false, "")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	rest := fs.Args()
	if len(rest) == 0 {
		fs.Usage()
		return 2
	}
	name := rest[0]
	cmdArgs := rest[1:]
	if len(cmdArgs) > 0 && cmdArgs[0] == "--" {
		cmdArgs = cmdArgs[1:]
	}

	command := strings.TrimSpace(strings.Join(cmdArgs, " "))
	if useLast {
		last, ok := lastRecordedCommand()
		if !ok {
			fmt.Fprintln(stderr, "yore: --last: no recorded commands yet (is recording enabled?)")
			return 1
		}
		command = last
	}
	if command == "" {
		fmt.Fprintln(stderr, "yore: no command given (use -- <command...> or --last)")
		fs.Usage()
		return 2
	}

	r := recipe.Recipe{Name: name, Command: command, Description: desc}

	path, err := saveTarget(project)
	if err != nil {
		fmt.Fprintf(stderr, "yore: %v\n", err)
		return 1
	}
	if err := recipe.AppendToFile(path, r); err != nil {
		fmt.Fprintf(stderr, "yore: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "saved recipe %q to %s\n", name, path)
	if params := recipe.ParamsIn(command); len(params) > 0 {
		fmt.Fprintf(stdout, "  parameters: %s\n", strings.Join(params, ", "))
	}
	fmt.Fprintf(stdout, "  run it with: yore run %q\n", name)
	return 0
}

// saveTarget resolves where a saved recipe should be written.
func saveTarget(project bool) (string, error) {
	if project {
		root := projectRoot(currentDir())
		return root + string(os.PathSeparator) + recipe.ProjectFile, nil
	}
	return recipe.GlobalPath()
}

// lastRecordedCommand returns the most recently recorded command, if any.
func lastRecordedCommand() (string, bool) {
	st, err := store.Default()
	if err != nil {
		return "", false
	}
	events, err := st.Load()
	if err != nil || len(events) == 0 {
		return "", false
	}
	return events[len(events)-1].Command, true
}

const runUsage = `Usage: yore run [flags] <name>

Run a saved recipe, prompting for any {parameters}.

Flags:
  --print    print the resolved command instead of running it
  --yes      do not prompt; use parameter defaults (blank if none)
`

// runRun handles `yore run`.
func runRun(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("yore run", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprint(stderr, runUsage) }
	var (
		printOnly bool
		assumeYes bool
	)
	fs.BoolVar(&printOnly, "print", false, "")
	fs.BoolVar(&assumeYes, "yes", false, "")

	// Pull the recipe name out first so flags may appear before or after it
	// (e.g. `yore run pf --print`). Safe because run's flags are all boolean.
	name, flagArgs := extractName(args)
	if err := fs.Parse(flagArgs); err != nil {
		return 2
	}

	recipes := loadRecipes()
	if name == "" {
		listRecipes(stdout, recipes)
		return 0
	}
	r, ok := recipe.Find(recipes, name)
	if !ok {
		fmt.Fprintf(stderr, "yore: no recipe named %q (see 'yore recipes')\n", name)
		return 1
	}

	values := promptParams(os.Stdin, stderr, r.Params, assumeYes)
	rendered := r.Render(values)

	if printOnly {
		fmt.Fprintln(stdout, rendered)
		return 0
	}
	return runShell(rendered)
}

// promptParams collects a value for each parameter. With assumeYes it uses
// defaults without prompting. Prompts and read echoes go to out (stderr) so a
// piped `--print` stays clean; values are read from in.
func promptParams(in io.Reader, out io.Writer, params []recipe.Param, assumeYes bool) map[string]string {
	values := make(map[string]string, len(params))
	reader := bufio.NewReader(in)
	for _, p := range params {
		if assumeYes {
			values[p.Name] = p.Default
			continue
		}
		if p.Default != "" {
			fmt.Fprintf(out, "%s [%s]: ", p.Name, p.Default)
		} else {
			fmt.Fprintf(out, "%s: ", p.Name)
		}
		line, _ := reader.ReadString('\n')
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			line = p.Default
		}
		values[p.Name] = line
	}
	return values
}

// extractName pulls the first non-flag argument (the recipe name) out of args,
// returning it and the remaining args in order. This lets flags appear before
// or after the name. It is safe only for flag sets whose flags are all boolean
// (none consume the following token as a value).
func extractName(args []string) (string, []string) {
	for i, a := range args {
		if !strings.HasPrefix(a, "-") {
			rest := append(append([]string{}, args[:i]...), args[i+1:]...)
			return a, rest
		}
	}
	return "", args
}

// runShell executes command in the user's shell and returns its exit code.
func runShell(command string) int {
	sh := os.Getenv("SHELL")
	if sh == "" {
		sh = "/bin/sh"
	}
	c := exec.Command(sh, "-c", command)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := c.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return ee.ExitCode()
		}
		return 1
	}
	return 0
}

// runRecipes handles `yore recipes`: list available recipes.
func runRecipes(_ []string, stdout, _ io.Writer) int {
	listRecipes(stdout, loadRecipes())
	return 0
}

func listRecipes(w io.Writer, recipes []recipe.Recipe) {
	if len(recipes) == 0 {
		fmt.Fprintln(w, "no recipes yet — create one with 'yore save'")
		return
	}
	for _, r := range recipes {
		tag := ""
		if r.Source == "project" {
			tag = "  (project)"
		}
		if r.Description != "" {
			fmt.Fprintf(w, "%-24s %s%s\n", r.Name, r.Description, tag)
		} else {
			fmt.Fprintf(w, "%-24s %s%s\n", r.Name, r.Command, tag)
		}
	}
}
