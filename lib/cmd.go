package lib

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"strings"
)

func newRootCmd(repository Repository) (*cobra.Command, error) {
	rootCmd := cobra.Command{
		Use:   "quarkus-func-template",
		Short: "CLI for scaffolding func projects.",
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true, DisableNoDescFlag: true, DisableDescriptions: true},
	}

	rootCmd.AddCommand(newListRuntimesCmd(repository))
	rootCmd.AddCommand(newListTemplatesCmd(repository))
	rootCmd.AddCommand(newCreateCmd(repository))
	return &rootCmd, nil
}

func newCreateCmd(repository Repository) *cobra.Command {
	validTemplates := make(map[struct{ Runtime, Template string }]bool)
	return &cobra.Command{
		Use:   "create <func-spec-JSON>",
		Short: "Create a func project for template",
		Args:  cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			rs, err := repository.Runtimes(cmd.Context())
			if err != nil {
				return err
			}
			for _, r := range rs {
				ts, err := repository.Templates(cmd.Context(), r)
				if err != nil {
					return err
				}
				for _, t := range ts {
					validTemplates[struct{ Runtime, Template string }{Runtime: r, Template: t}] = true
				}
			}
			return nil
		},
		RunE:  func (cmd *cobra.Command, args []string) error {
			var (
				err error
				fun FuncSpec
			)

			dec := json.NewDecoder(strings.NewReader(args[0]))
			err = dec.Decode(&fun)
			if err != nil {
				return err
			}

			if _, ok := validTemplates[struct{ Runtime, Template string }{Runtime: fun.Runtime, Template: fun.Template}]; !ok {
				return fmt.Errorf("invalid runtime/template combination: %s/%s", fun.Runtime, fun.Template)
			}

			_, err = os.Stat(filepath.Join(fun.Root, "func.yaml"))
			if err == nil {
				return fmt.Errorf("destiantion directory already contains function")
			}

			template, err := repository.Template(cmd.Context(), fun.Runtime, fun.Template)
			if err != nil {
				return err
			}
			return template.Write(cmd.Context(), fun.Name, fun.Root)
		},
	}

}

func newListRuntimesCmd(repository Repository) *cobra.Command {
	return &cobra.Command{
		Use:   "runtimes",
		Short: "List available runtimes",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			rs, err := repository.Runtimes(cmd.Context())
			if err != nil {
				return err
			}

			return json.NewEncoder(cmd.OutOrStdout()).Encode(rs)
		},
	}
}

func newListTemplatesCmd(repository Repository) *cobra.Command {
	return &cobra.Command{
		Use:   "templates <runtime>",
		Short: "List available templates for given runtime",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ts, err := repository.Templates(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			return json.NewEncoder(cmd.OutOrStdout()).Encode(ts)
		},
	}
}
