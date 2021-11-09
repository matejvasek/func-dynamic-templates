package lib

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
)

type execRepo struct {
	cmd string
}

func (e execRepo) Runtimes(ctx context.Context) ([]string, error) {
	var runtimes []string
	cmd := exec.CommandContext(ctx, e.cmd, "runtimes")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(out, &runtimes)
	if err != nil {
		return nil, err
	}

	return runtimes, nil
}

func (e execRepo) Templates(ctx context.Context, runtime string) ([]string, error) {
	var templates []string
	cmd := exec.CommandContext(ctx, e.cmd, "templates", runtime)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(out, &templates)
	if err != nil {
		return nil, err
	}

	return templates, nil
}

type execTemplate struct {
	runtime, name, cmd string
}

func (e execTemplate) Name() string {
	return e.name
}

func (e execTemplate) Runtime() string {
	return e.runtime
}

func (e execTemplate) Write(ctx context.Context, name, destDir string) error {
	fun := FuncSpec{
		Name: name,
		Root: destDir,
		Runtime: e.Runtime(),
		Template: e.Name(),
	}
	data, err := json.Marshal(&fun)
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, e.cmd, "create", string(data))
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (e execRepo) Template(ctx context.Context, runtime, name string) (Template, error) {
	return execTemplate{name: name, runtime: runtime, cmd: e.cmd}, nil
}