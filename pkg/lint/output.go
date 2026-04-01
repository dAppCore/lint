package lint

// ResolveRunOutputFormat resolves the report writer from the run input and project config.
//
//	format, err := lint.ResolveRunOutputFormat(lint.RunInput{Path: ".", CI: true})
func ResolveRunOutputFormat(input RunInput) (string, error) {
	if input.Output != "" {
		return input.Output, nil
	}
	if input.CI {
		return "github", nil
	}
	config, _, err := LoadProjectConfig(input.Path, input.Config)
	if err != nil {
		return "", err
	}
	schedule, err := ResolveSchedule(config, input.Schedule)
	if err != nil {
		return "", err
	}
	if schedule != nil && schedule.Output != "" {
		return schedule.Output, nil
	}
	if config.Output != "" {
		return config.Output, nil
	}
	return "text", nil
}
