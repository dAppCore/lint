package php

import (
	"context"

	core "dappco.re/go"
)

type phpCommandOutput struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Err      error
}

func runPHPCommand(ctx context.Context, dir string, name string, args []string, output core.Writer, env []string) core.Result {
	if output == nil {
		output = core.Stdout()
	}
	cmd := buildPHPCommand(ctx, dir, name, args)
	cmd.Stdout = output
	cmd.Stderr = output
	cmd.Stdin = core.Stdin()
	if len(env) > 0 {
		cmd.Env = append(core.Environ(), env...)
	}
	err := cmd.Run()
	if err != nil {
		return core.Fail(err)
	}
	return core.Ok(nil)
}

func outputPHPCommand(ctx context.Context, dir string, name string, args []string) phpCommandOutput {
	stdout := core.NewBuffer()
	stderr := core.NewBuffer()
	cmd := buildPHPCommand(ctx, dir, name, args)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	return phpCommandOutput{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: phpCommandExitCode(err),
		Err:      err,
	}
}

func buildPHPCommand(ctx context.Context, dir string, name string, args []string) *core.Cmd {
	_ = ctx
	path := name
	if found := findPHPExecutable(name); found.OK {
		path = found.Value.(string)
	}
	return &core.Cmd{
		Path: path,
		Args: append([]string{name}, args...),
		Dir:  dir,
	}
}

func phpCommandExitCode(err error) int {
	if err == nil {
		return 0
	}
	if exitErr, ok := err.(interface{ ExitCode() int }); ok {
		return exitErr.ExitCode()
	}
	return -1
}

func findPHPExecutable(name string) core.Result {
	if name == "" {
		return core.Fail(core.E("findPHPExecutable", "empty executable name", nil))
	}
	if core.Contains(name, string(core.PathSeparator)) {
		if executablePHPPath(name) {
			return core.Ok(name)
		}
		return core.Fail(core.E("findPHPExecutable", core.Sprintf("%s is not executable", name), nil))
	}
	for _, dir := range core.Split(core.Getenv("PATH"), string(core.PathListSeparator)) {
		if dir == "" {
			dir = "."
		}
		candidate := core.PathJoin(dir, name)
		if executablePHPPath(candidate) {
			return core.Ok(candidate)
		}
	}
	return core.Fail(core.E("findPHPExecutable", core.Sprintf("%s was not found in PATH", name), nil))
}

func executablePHPPath(path string) bool {
	stat := core.Stat(path)
	if !stat.OK {
		return false
	}
	info := stat.Value.(core.FsFileInfo)
	return !info.IsDir() && info.Mode()&0111 != 0
}
