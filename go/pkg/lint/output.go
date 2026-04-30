package lint

import core "dappco.re/go"

// ResolveRunOutputFormat resolves the report writer from the run input and project config.
//
//	format, err := lint.ResolveRunOutputFormat(lint.RunInput{Path: ".", CI: true})
//	format, err := lint.ResolveRunOutputFormat(lint.RunInput{Path: ".", Schedule: "nightly"})
func ResolveRunOutputFormat(input RunInput) core.Result {
	if input.Output != "" {
		return core.Ok(input.Output)
	}
	if input.CI {
		return core.Ok("github")
	}
	configResult := LoadProjectConfig(input.Path, input.Config)
	if !configResult.OK {
		return configResult
	}
	config := configResult.Value.(projectConfigResult).Config
	scheduleResult := ResolveSchedule(config, input.Schedule)
	if !scheduleResult.OK {
		return scheduleResult
	}
	schedule, _ := scheduleResult.Value.(*Schedule)
	if schedule != nil && schedule.Output != "" {
		return core.Ok(schedule.Output)
	}
	if config.Output != "" {
		return core.Ok(config.Output)
	}
	return core.Ok("text")
}
