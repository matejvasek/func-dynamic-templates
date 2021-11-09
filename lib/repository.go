package lib

import (
	"context"
	"github.com/spf13/cobra"
)

// FuncSpec is structurally sub-type of func.Function
type FuncSpec struct {
	Name     string `json:"name"`
	Root     string `json:"root"`
	Runtime  string `json:"runtime"`
	Template string `json:"template"`
}

type Template interface {
	Name() string
	Runtime() string
	// Write creates all project files including func.yaml
	Write(ctx context.Context, name, destDir string) error
}

type Repository interface {
	Runtimes(ctx context.Context) ([]string, error)
	Templates(ctx context.Context, runtime string) ([]string, error)
	Template(ctx context.Context, runtime, name string) (Template, error)
}

// NewRepositoryFromExecutable creates an instance of Repository that is backed by executable.
//
// The executable shall satisfy following contract:
//
// The executable shall support three sub-commands: runtimes, templates and create.
//
// The runtimes sub-command shall take no parameters.
// The sub-command shall print JSON array of string to standard output containing list of supported runtimes.
//
// The templates sub-command shall take exactly one parameter: runtime name.
// The sub-command shall print JSON array of templates for given runtime.
//
// The create sub-command shall accept exactly one argument: JSON func-spec see the FuncSpec structure.
// The sub-command shall create func project with given FuncSpec.Name and at specified FuncSpec.Root
// using runtime/template specified by FuncSpec.Runtime/FuncSpec.Template.
//
// The create sub-command is may use stdio to prompt user for detail is needed for project scaffolding.
func NewRepositoryFromExecutable(cmd string) (Repository, error) {
	return execRepo{cmd}, nil
}

// NewCommandFromRepository creates cobra command which is backed by an instance of Repository.
// This is convenience function to ease development of "executable repositories".
// Such executables are expected to be saved in ~/.config/func/repositories.
//
// Applications using this function to create their root command in their main function
// should satisfy contract specified in NewRepositoryFromExecutable.
func NewCommandFromRepository(repository Repository) (*cobra.Command, error) {
	return newRootCmd(repository)
}
