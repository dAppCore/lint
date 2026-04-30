package cli

import (
	"bytes"
	"os"
	"strings"
	"time"

	core "dappco.re/go"
)

func ax7ResetCLI(t *core.T) {
	t.Helper()
	oldArgs := append([]string(nil), os.Args...)
	oldAppName := appName
	oldRegistered := append([]CommandRegistration(nil), registeredCommands...)
	t.Cleanup(func() {
		os.Args = oldArgs
		appName = oldAppName
		registeredCommands = oldRegistered
	})
	os.Args = []string{"core"}
	appName = "core"
	registeredCommands = nil
}

func ax7CaptureStdout(t *core.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	reader, writer, err := os.Pipe()
	core.RequireNoError(t, err)
	os.Stdout = writer
	fn()
	core.RequireNoError(t, writer.Close())
	os.Stdout = old
	var buffer bytes.Buffer
	_, err = buffer.ReadFrom(reader)
	core.RequireNoError(t, err)
	return buffer.String()
}

func ax7CaptureStderr(t *core.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	reader, writer, err := os.Pipe()
	core.RequireNoError(t, err)
	os.Stderr = writer
	fn()
	core.RequireNoError(t, writer.Close())
	os.Stderr = old
	var buffer bytes.Buffer
	_, err = buffer.ReadFrom(reader)
	core.RequireNoError(t, err)
	return buffer.String()
}

func TestCLI_NewStyle_Good(t *core.T) {
	style := NewStyle()
	core.AssertNotNil(t, style)
	core.AssertEqual(t, "agent", style.Render("agent"))
}

func TestCLI_NewStyle_Bad(t *core.T) {
	style := NewStyle()
	core.AssertNotNil(t, style)
	core.AssertNotEqual(t, (*AnsiStyle)(nil), style)
}

func TestCLI_NewStyle_Ugly(t *core.T) {
	style := NewStyle()
	got := style.Bold().Dim().Foreground(ColourRed500).Render("")
	core.AssertEqual(t, "", got)
	core.AssertNotNil(t, style)
}

func TestCLI_AnsiStyle_Bold_Good(t *core.T) {
	style := NewStyle()
	got := style.Bold()
	core.AssertEqual(t, style, got)
	core.AssertEqual(t, "ready", got.Render("ready"))
}

func TestCLI_AnsiStyle_Bold_Bad(t *core.T) {
	style := NewStyle()
	got := style.Bold().Bold()
	core.AssertEqual(t, style, got)
	core.AssertNotNil(t, got)
}

func TestCLI_AnsiStyle_Bold_Ugly(t *core.T) {
	style := &AnsiStyle{}
	got := style.Bold()
	core.AssertEqual(t, style, got)
	core.AssertEqual(t, "", got.Render(""))
}

func TestCLI_AnsiStyle_Dim_Good(t *core.T) {
	style := NewStyle()
	got := style.Dim()
	core.AssertEqual(t, style, got)
	core.AssertEqual(t, "ready", got.Render("ready"))
}

func TestCLI_AnsiStyle_Dim_Bad(t *core.T) {
	style := NewStyle()
	got := style.Dim().Dim()
	core.AssertEqual(t, style, got)
	core.AssertNotNil(t, got)
}

func TestCLI_AnsiStyle_Dim_Ugly(t *core.T) {
	style := &AnsiStyle{}
	got := style.Dim()
	core.AssertEqual(t, style, got)
	core.AssertEqual(t, "", got.Render(""))
}

func TestCLI_AnsiStyle_Foreground_Good(t *core.T) {
	style := NewStyle()
	got := style.Foreground(ColourAmber500)
	core.AssertEqual(t, style, got)
	core.AssertEqual(t, "ready", got.Render("ready"))
}

func TestCLI_AnsiStyle_Foreground_Bad(t *core.T) {
	style := NewStyle()
	got := style.Foreground("")
	core.AssertEqual(t, style, got)
	core.AssertNotNil(t, got)
}

func TestCLI_AnsiStyle_Foreground_Ugly(t *core.T) {
	style := NewStyle()
	got := style.Foreground("#not-a-real-colour")
	core.AssertEqual(t, style, got)
	core.AssertEqual(t, "text", got.Render("text"))
}

func TestCLI_AnsiStyle_Render_Good(t *core.T) {
	style := NewStyle()
	got := style.Render("agent")
	core.AssertEqual(t, "agent", got)
	core.AssertNotEqual(t, "", got)
}

func TestCLI_AnsiStyle_Render_Bad(t *core.T) {
	style := NewStyle()
	got := style.Render("")
	core.AssertEqual(t, "", got)
	core.AssertNotNil(t, style)
}

func TestCLI_AnsiStyle_Render_Ugly(t *core.T) {
	style := &AnsiStyle{}
	got := style.Render(strings.Repeat("x", 4))
	core.AssertEqual(t, "xxxx", got)
	core.AssertLen(t, got, 4)
}

func TestCLI_WithAppName_Good(t *core.T) {
	ax7ResetCLI(t)
	WithAppName("qa")
	core.AssertEqual(t, "qa", appName)
	core.AssertNotEqual(t, "core", appName)
}

func TestCLI_WithAppName_Bad(t *core.T) {
	ax7ResetCLI(t)
	WithAppName("")
	core.AssertEqual(t, "core", appName)
	core.AssertNotEqual(t, "", appName)
}

func TestCLI_WithAppName_Ugly(t *core.T) {
	ax7ResetCLI(t)
	WithAppName("core qa")
	core.AssertEqual(t, "core qa", appName)
	core.AssertContains(t, appName, "qa")
}

func TestCLI_WithCommands_Good(t *core.T) {
	root := NewGroup("root", "", "")
	setup := WithCommands("qa", func(root *Command) { root.AddCommand(NewGroup("qa", "", "")) })
	setup(root)
	core.AssertLen(t, root.Commands(), 1)
	core.AssertEqual(t, "qa", root.Commands()[0].Use)
}

func TestCLI_WithCommands_Bad(t *core.T) {
	root := NewGroup("root", "", "")
	setup := WithCommands("", func(root *Command) {
		// Empty callback verifies blank command names are skipped.
	})
	setup(root)
	core.AssertEmpty(t, root.Commands())
	core.AssertNotNil(t, root)
}

func TestCLI_WithCommands_Ugly(t *core.T) {
	root := NewGroup("root", "", "")
	setup := WithCommands("ignored", func(root *Command) { root.Use = "changed" })
	setup(root)
	core.AssertEqual(t, "changed", root.Use)
	core.AssertEmpty(t, root.Commands())
}

func TestCLI_RegisterCommands_Good(t *core.T) {
	ax7ResetCLI(t)
	RegisterCommands(func(root *Command) { root.AddCommand(NewGroup("qa", "", "")) })
	core.AssertLen(t, registeredCommands, 1)
	core.AssertNotNil(t, registeredCommands[0])
}

func TestCLI_RegisterCommands_Bad(t *core.T) {
	ax7ResetCLI(t)
	RegisterCommands(nil)
	core.AssertLen(t, registeredCommands, 1)
	core.AssertNil(t, registeredCommands[0])
}

func TestCLI_RegisterCommands_Ugly(t *core.T) {
	ax7ResetCLI(t)
	RegisterCommands(func(root *Command) { root.Use = "first" })
	RegisterCommands(func(root *Command) { root.Use = "second" })
	core.AssertLen(t, registeredCommands, 2)
}

func TestCLI_Main_Good(t *core.T) {
	ax7ResetCLI(t)
	var called bool
	RegisterCommands(func(root *Command) { called = root.Use == "core" })
	Main()
	core.AssertTrue(t, called)
	core.AssertEqual(t, "core", appName)
}

func TestCLI_Main_Bad(t *core.T) {
	ax7ResetCLI(t)
	Main()
	core.AssertEqual(t, "core", appName)
	core.AssertEmpty(t, registeredCommands)
}

func TestCLI_Main_Ugly(t *core.T) {
	ax7ResetCLI(t)
	os.Args = []string{"core", "noop"}
	var ran bool
	Main(func(root *Command) {
		root.AddCommand(NewCommand("noop", "", "", func(_ *Command, _ []string) error {
			ran = true
			return nil
		}))
	})
	core.AssertTrue(t, ran)
}

func TestCLI_NewCommand_Good(t *core.T) {
	cmd := NewCommand("run", "short", "long", func(_ *Command, _ []string) error { return nil })
	core.AssertEqual(t, "run", cmd.Use)
	core.AssertEqual(t, "short", cmd.Short)
	core.AssertNotNil(t, cmd.RunE)
}

func TestCLI_NewCommand_Bad(t *core.T) {
	cmd := NewCommand("", "", "", nil)
	core.AssertEqual(t, "", cmd.Use)
	core.AssertEqual(t, "", cmd.Short)
	core.AssertNil(t, cmd.RunE)
}

func TestCLI_NewCommand_Ugly(t *core.T) {
	cmd := NewCommand("run [args...]", "", "long", func(_ *Command, args []string) error { return nil })
	core.AssertContains(t, cmd.Use, "[args")
	core.AssertEqual(t, "long", cmd.Long)
}

func TestCLI_NewGroup_Good(t *core.T) {
	cmd := NewGroup("qa", "Quality", "Long")
	core.AssertEqual(t, "qa", cmd.Use)
	core.AssertEqual(t, "Quality", cmd.Short)
	core.AssertEqual(t, "Long", cmd.Long)
}

func TestCLI_NewGroup_Bad(t *core.T) {
	cmd := NewGroup("", "", "")
	core.AssertEqual(t, "", cmd.Use)
	core.AssertNil(t, cmd.RunE)
}

func TestCLI_NewGroup_Ugly(t *core.T) {
	cmd := NewGroup("qa", "", "")
	cmd.AddCommand(NewGroup("watch", "", ""))
	core.AssertLen(t, cmd.Commands(), 1)
	core.AssertEqual(t, "watch", cmd.Commands()[0].Use)
}

func TestCLI_StringFlag_Good(t *core.T) {
	cmd := NewGroup("root", "", "")
	var value string
	StringFlag(cmd, &value, "name", "n", "codex", "name")
	core.AssertNoError(t, cmd.Flags().Set("name", "hades"))
	core.AssertEqual(t, "hades", value)
}

func TestCLI_StringFlag_Bad(t *core.T) {
	cmd := NewGroup("root", "", "")
	var value string
	StringFlag(cmd, &value, "name", "", "codex", "name")
	core.AssertNoError(t, cmd.Flags().Set("name", ""))
	core.AssertEqual(t, "", value)
}

func TestCLI_StringFlag_Ugly(t *core.T) {
	cmd := NewGroup("root", "", "")
	var value string
	StringFlag(cmd, &value, "name", "n", "", "name")
	core.AssertNoError(t, cmd.Flags().Set("name", "short"))
	core.AssertEqual(t, "short", value)
	core.AssertEqual(t, "n", cmd.Flags().Lookup("name").Shorthand)
}

func TestCLI_BoolFlag_Good(t *core.T) {
	cmd := NewGroup("root", "", "")
	var value bool
	BoolFlag(cmd, &value, "json", "j", false, "json")
	core.AssertNoError(t, cmd.Flags().Set("json", "true"))
	core.AssertTrue(t, value)
}

func TestCLI_BoolFlag_Bad(t *core.T) {
	cmd := NewGroup("root", "", "")
	var value bool
	BoolFlag(cmd, &value, "json", "", true, "json")
	core.AssertNoError(t, cmd.Flags().Set("json", "false"))
	core.AssertFalse(t, value)
}

func TestCLI_BoolFlag_Ugly(t *core.T) {
	cmd := NewGroup("root", "", "")
	var value bool
	BoolFlag(cmd, &value, "json", "j", false, "json")
	err := cmd.Flags().Set("json", "not-bool")
	core.AssertError(t, err)
	core.AssertFalse(t, value)
}

func TestCLI_StringSliceFlag_Good(t *core.T) {
	cmd := NewGroup("root", "", "")
	var value []string
	StringSliceFlag(cmd, &value, "path", "p", []string{"."}, "paths")
	core.AssertNoError(t, cmd.Flags().Set("path", "a,b"))
	core.AssertEqual(t, []string{"a", "b"}, value)
}

func TestCLI_StringSliceFlag_Bad(t *core.T) {
	cmd := NewGroup("root", "", "")
	var value []string
	StringSliceFlag(cmd, &value, "path", "", nil, "paths")
	core.AssertNoError(t, cmd.Flags().Set("path", ""))
	core.AssertEmpty(t, value)
}

func TestCLI_StringSliceFlag_Ugly(t *core.T) {
	cmd := NewGroup("root", "", "")
	var value []string
	StringSliceFlag(cmd, &value, "path", "p", []string{}, "paths")
	core.AssertNoError(t, cmd.Flags().Set("path", "a"))
	core.AssertEqual(t, []string{"a"}, value)
	core.AssertEqual(t, "p", cmd.Flags().Lookup("path").Shorthand)
}

func TestCLI_Blank_Good(t *core.T) {
	output := ax7CaptureStdout(t, Blank)
	core.AssertEqual(t, "\n", output)
	core.AssertLen(t, output, 1)
}

func TestCLI_Blank_Bad(t *core.T) {
	output := ax7CaptureStdout(t, func() {
		Blank()
		Blank()
	})
	core.AssertEqual(t, "\n\n", output)
	core.AssertLen(t, output, 2)
}

func TestCLI_Blank_Ugly(t *core.T) {
	output := ax7CaptureStdout(t, func() {
		for i := 0; i < 3; i++ {
			Blank()
		}
	})
	core.AssertEqual(t, "\n\n\n", output)
	core.AssertLen(t, output, 3)
}

func TestCLI_Print_Good(t *core.T) {
	output := ax7CaptureStdout(t, func() { Print("hello %s", "agent") })
	core.AssertEqual(t, "hello agent", output)
	core.AssertContains(t, output, "agent")
}

func TestCLI_Print_Bad(t *core.T) {
	output := ax7CaptureStdout(t, func() { Print("") })
	core.AssertEqual(t, "", output)
	core.AssertLen(t, output, 0)
}

func TestCLI_Print_Ugly(t *core.T) {
	output := ax7CaptureStdout(t, func() { Print("%d %s", 7, "items") })
	core.AssertEqual(t, "7 items", output)
	core.AssertContains(t, output, "items")
}

func TestCLI_Text_Good(t *core.T) {
	output := ax7CaptureStdout(t, func() { Text("hello", "agent") })
	core.AssertContains(t, output, "hello")
	core.AssertContains(t, output, "agent")
}

func TestCLI_Text_Bad(t *core.T) {
	output := ax7CaptureStdout(t, func() { Text() })
	core.AssertEqual(t, "\n", output)
	core.AssertLen(t, output, 1)
}

func TestCLI_Text_Ugly(t *core.T) {
	output := ax7CaptureStdout(t, func() { Text([]string{"a", "b"}) })
	core.AssertContains(t, output, "[a b]")
	core.AssertContains(t, output, "\n")
}

func TestCLI_Warnf_Good(t *core.T) {
	output := ax7CaptureStderr(t, func() { Warnf("warn %s", "agent") })
	core.AssertEqual(t, "warn agent\n", output)
	core.AssertContains(t, output, "warn")
}

func TestCLI_Warnf_Bad(t *core.T) {
	output := ax7CaptureStderr(t, func() { Warnf("") })
	core.AssertEqual(t, "\n", output)
	core.AssertLen(t, output, 1)
}

func TestCLI_Warnf_Ugly(t *core.T) {
	output := ax7CaptureStderr(t, func() { Warnf("%d", 42) })
	core.AssertEqual(t, "42\n", output)
	core.AssertContains(t, output, "\n")
}

func TestCLI_Err_Good(t *core.T) {
	err := Err("failed %s", "agent")
	core.AssertError(t, err)
	core.AssertEqual(t, "failed agent", err.Error())
}

func TestCLI_Err_Bad(t *core.T) {
	err := Err("")
	core.AssertError(t, err)
	core.AssertEqual(t, "", err.Error())
}

func TestCLI_Err_Ugly(t *core.T) {
	err := Err("%d/%d", 1, 0)
	core.AssertError(t, err)
	core.AssertEqual(t, "1/0", err.Error())
}

func TestCLI_Sprintf_Good(t *core.T) {
	got := Sprintf("hello %s", "agent")
	core.AssertEqual(t, "hello agent", got)
	core.AssertContains(t, got, "agent")
}

func TestCLI_Sprintf_Bad(t *core.T) {
	got := Sprintf("")
	core.AssertEqual(t, "", got)
	core.AssertLen(t, got, 0)
}

func TestCLI_Sprintf_Ugly(t *core.T) {
	got := Sprintf("%04d", 7)
	core.AssertEqual(t, "0007", got)
	core.AssertLen(t, got, 4)
}

func TestCLI_Glyph_Good(t *core.T) {
	got := Glyph(":check:")
	core.AssertEqual(t, "OK", got)
	core.AssertNotEqual(t, ":check:", got)
}

func TestCLI_Glyph_Bad(t *core.T) {
	got := Glyph(":missing:")
	core.AssertEqual(t, ":missing:", got)
	core.AssertContains(t, got, "missing")
}

func TestCLI_Glyph_Ugly(t *core.T) {
	got := Glyph("")
	core.AssertEqual(t, "", got)
	core.AssertLen(t, got, 0)
}

func TestCLI_FormatAge_Good(t *core.T) {
	got := FormatAge(time.Now().Add(-2 * time.Minute))
	core.AssertContains(t, got, "m ago")
	core.AssertNotEqual(t, "just now", got)
}

func TestCLI_FormatAge_Bad(t *core.T) {
	got := FormatAge(time.Now())
	core.AssertEqual(t, "just now", got)
	core.AssertNotContains(t, got, "ago")
}

func TestCLI_FormatAge_Ugly(t *core.T) {
	got := FormatAge(time.Now().Add(-45 * 24 * time.Hour))
	core.AssertContains(t, got, "mo ago")
	core.AssertNotEqual(t, "", got)
}
