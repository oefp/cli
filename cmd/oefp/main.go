package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/oefp/cli/internal/lint"
	"github.com/oefp/cli/internal/submission"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "oefp:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		usage()
		return errors.New("a command is required")
	}
	switch args[0] {
	case "lint":
		flags := flag.NewFlagSet("oefp lint", flag.ContinueOnError)
		root := flags.String("root", ".", "OEFP repository root")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		if flags.NArg() != 0 {
			return errors.New("lint does not accept positional arguments")
		}
		absolute, err := filepath.Abs(*root)
		if err != nil {
			return err
		}
		result := lint.Check(absolute)
		if len(result.Errors) != 0 {
			for _, problem := range result.Errors {
				fmt.Fprintln(os.Stderr, problem)
			}
			return fmt.Errorf("lint failed with %d error(s)", len(result.Errors))
		}
		fmt.Printf("Lint passed: %d OEFP Lean entry file(s) checked.\n", result.Files)
		return nil

	case "generate":
		flags := flag.NewFlagSet("oefp generate", flag.ContinueOnError)
		root := flags.String("root", ".", "Git repository root")
		base := flags.String("base", submission.DefaultBase, "base commit or ref")
		output := flags.String("output", "submission.oefp", "output archive path")
		force := flags.Bool("force", false, "replace an existing output archive")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		if flags.NArg() != 0 {
			return errors.New("generate does not accept positional arguments")
		}
		if err := submission.Generate(ctx, submission.GenerateOptions{
			Root: *root, Base: *base, Output: *output, Force: *force,
		}); err != nil {
			return err
		}
		absolute, _ := filepath.Abs(filepath.Join(*root, *output))
		if filepath.IsAbs(*output) {
			absolute = *output
		}
		fmt.Printf("Generated %s\n", absolute)
		return nil

	case "help", "-h", "--help":
		usage()
		return nil
	default:
		usage()
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `Usage: oefp <command> [options]

Commands:
  lint       Validate OEFP entry paths and YAML front matter
  generate   Generate a submission archive from committed changes`)
}
