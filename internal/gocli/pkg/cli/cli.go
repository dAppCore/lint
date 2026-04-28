package cli

import (
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/spf13/cobra"
)

type Command = cobra.Command

type CommandSetup func(root *Command)
type CommandRegistration func(root *Command)

type AnsiStyle struct{}

const (
	ColourAmber500  = "#f59e0b"
	ColourGray500   = "#6b7280"
	ColourOrange500 = "#f97316"
	ColourRed500    = "#ef4444"
)

var (
	appName            = "core"
	registeredCommands []CommandRegistration
)

var (
	DimStyle     = NewStyle()
	ErrorStyle   = NewStyle()
	HeaderStyle  = NewStyle()
	RepoStyle    = NewStyle()
	SuccessStyle = NewStyle()
	TitleStyle   = NewStyle()
	ValueStyle   = NewStyle()
	WarningStyle = NewStyle()
)

func NewStyle() *AnsiStyle { return &AnsiStyle{} }

func (s *AnsiStyle) Bold() *AnsiStyle {
	return s
}

func (s *AnsiStyle) Dim() *AnsiStyle {
	return s
}

func (s *AnsiStyle) Foreground(string) *AnsiStyle {
	return s
}

func (s *AnsiStyle) Render(text string) string {
	return text
}

func WithAppName(name string) {
	if name != "" {
		appName = name
	}
}

func WithCommands(name string, register func(root *Command), _ ...fs.FS) CommandSetup {
	return func(root *Command) {
		register(root)
	}
}

func RegisterCommands(fn CommandRegistration, _ ...fs.FS) {
	registeredCommands = append(registeredCommands, fn)
}

func Main(commands ...CommandSetup) {
	root := &cobra.Command{Use: appName}
	for _, register := range registeredCommands {
		register(root)
	}
	for _, setup := range commands {
		setup(root)
	}
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func NewCommand(use, short, long string, run func(cmd *Command, args []string) error) *Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Long:  long,
		RunE:  run,
	}
}

func NewGroup(use, short, long string) *Command {
	return &cobra.Command{Use: use, Short: short, Long: long}
}

func StringFlag(cmd *Command, ptr *string, name, short, def, usage string) {
	if short != "" {
		cmd.Flags().StringVarP(ptr, name, short, def, usage)
		return
	}
	cmd.Flags().StringVar(ptr, name, def, usage)
}

func BoolFlag(cmd *Command, ptr *bool, name, short string, def bool, usage string) {
	if short != "" {
		cmd.Flags().BoolVarP(ptr, name, short, def, usage)
		return
	}
	cmd.Flags().BoolVar(ptr, name, def, usage)
}

func StringSliceFlag(cmd *Command, ptr *[]string, name, short string, def []string, usage string) {
	if short != "" {
		cmd.Flags().StringSliceVarP(ptr, name, short, def, usage)
		return
	}
	cmd.Flags().StringSliceVar(ptr, name, def, usage)
}

func Blank() {
	fmt.Println()
}

func Print(format string, args ...any) {
	fmt.Printf(format, args...)
}

func Text(args ...any) {
	fmt.Println(args...)
}

func Warnf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func Err(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}

func Sprintf(format string, args ...any) string {
	return fmt.Sprintf(format, args...)
}

func Glyph(code string) string {
	switch code {
	case ":check:":
		return "OK"
	case ":cross:":
		return "X"
	default:
		return code
	}
}

func FormatAge(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dw ago", int(d.Hours()/(24*7)))
	default:
		return fmt.Sprintf("%dmo ago", int(d.Hours()/(24*30)))
	}
}
