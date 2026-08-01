package lint

import (
	core "dappco.re/go"
)

// newStartedService builds a Core-attached lint service and runs OnStartup so
// the lint.* actions are registered on c.
//
//	c, svc := newStartedService(t)
//	r := c.Action("lint.tools").Run(ctx, core.NewOptions())
func newStartedService(t *core.T) (*core.Core, *Service) {
	t.Helper()
	c := core.New()
	result := NewServiceFor(LintConfigOptions{})(c)
	RequireResultOK(t, result)
	svc := result.Value.(*Service)
	RequireResultOK(t, c.RegisterService("lint", svc))
	RequireResultOK(t, svc.OnStartup(t.Context()))
	return c, svc
}

// TestHandleTools_Good_DispatchReturnsToolInfo dispatches lint.tools and asserts
// the action returns a non-empty []ToolInfo.
func TestHandleTools_Good_DispatchReturnsToolInfo(t *core.T) {
	c, _ := newStartedService(t)
	result := c.Action("lint.tools").Run(t.Context(), core.NewOptions(
		core.Option{Key: "languages", Value: []string{"go"}},
	))
	RequireResultOK(t, result)
	tools := result.Value.([]ToolInfo)
	core.AssertNotEmpty(t, tools)
}

// TestHandleWriteDefaultConfig_Good_DispatchWritesConfig dispatches
// lint.write_default_config and asserts the config file is written.
func TestHandleWriteDefaultConfig_Good_DispatchWritesConfig(t *core.T) {
	c, _ := newStartedService(t)
	dir := t.TempDir()
	result := c.Action("lint.write_default_config").Run(t.Context(), core.NewOptions(
		core.Option{Key: "path", Value: dir},
	))
	RequireResultOK(t, result)
	core.AssertContains(t, result.Value.(string), DefaultConfigPath)
}

// TestHandleWriteDefaultConfig_Bad_DispatchExistingFails dispatches twice and
// asserts the second dispatch fails without force.
func TestHandleWriteDefaultConfig_Bad_DispatchExistingFails(t *core.T) {
	c, _ := newStartedService(t)
	dir := t.TempDir()
	opts := core.NewOptions(core.Option{Key: "path", Value: dir})
	RequireResultOK(t, c.Action("lint.write_default_config").Run(t.Context(), opts))
	result := c.Action("lint.write_default_config").Run(t.Context(), opts)
	core.AssertFalse(t, result.OK)
}

// TestHandleRun_Good_DispatchProducesReport dispatches lint.run over a fixture
// directory and asserts a Report is returned.
func TestHandleRun_Good_DispatchProducesReport(t *core.T) {
	c, _ := newStartedService(t)
	dir := t.TempDir()
	RequireResultOK(t, core.WriteFile(core.PathJoin(dir, "go.mod"), []byte("module example.com/test\n"), 0o644))
	RequireResultOK(t, core.WriteFile(core.PathJoin(dir, "clean.go"), []byte("package sample\n\nfunc Clean() {}\n"), 0o644))
	t.Setenv("PATH", t.TempDir())

	result := c.Action("lint.run").Run(t.Context(), core.NewOptions(
		core.Option{Key: "path", Value: dir},
		core.Option{Key: "output", Value: "json"},
		core.Option{Key: "files", Value: []string{"clean.go"}},
	))
	RequireResultOK(t, result)
	report := result.Value.(Report)
	core.AssertNotNil(t, report.Summary)
}

// TestHandleInstallRemoveHook_Good_DispatchRoundTrips dispatches
// lint.install_hook then lint.remove_hook against a fresh git repo.
func TestHandleInstallRemoveHook_Good_DispatchRoundTrips(t *core.T) {
	if !(core.App{}).Find("git", "git").OK {
		t.Skip("git not available")
	}
	c, _ := newStartedService(t)
	dir := t.TempDir()
	toolkit := NewToolkit(dir)
	if run := toolkit.Run("git", "init").Value.(CommandOutput); run.ExitCode != 0 {
		t.Skip("git init failed")
	}

	install := c.Action("lint.install_hook").Run(t.Context(), core.NewOptions(
		core.Option{Key: "path", Value: dir},
	))
	RequireResultOK(t, install)

	remove := c.Action("lint.remove_hook").Run(t.Context(), core.NewOptions(
		core.Option{Key: "path", Value: dir},
	))
	RequireResultOK(t, remove)
}

// TestHandlers_Ugly_NilServiceFail asserts each handler fails fast when the
// service receiver is nil.
func TestHandlers_Ugly_NilServiceFail(t *core.T) {
	var svc *Service
	ctx := t.Context()
	opts := core.NewOptions()
	core.AssertFalse(t, svc.handleRun(ctx, opts).OK)
	core.AssertFalse(t, svc.handleTools(ctx, opts).OK)
	core.AssertFalse(t, svc.handleInstallHook(ctx, opts).OK)
	core.AssertFalse(t, svc.handleRemoveHook(ctx, opts).OK)
	core.AssertFalse(t, svc.handleWriteDefaultConfig(ctx, opts).OK)
}
