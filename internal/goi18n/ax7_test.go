package i18n

import (
	"testing/fstest"

	core "dappco.re/go"
)

func TestI18N_RegisterLocales_Good(t *core.T) {
	locales := fstest.MapFS{"en.json": {Data: []byte(`{"ax7":{"register":"ready"}}`)}}
	RegisterLocales(locales, ".")
	got := T("ax7.register")
	core.AssertEqual(t, "ready", got)
	core.AssertNotEqual(t, "ax7.register", got)
}

func TestI18N_RegisterLocales_Bad(t *core.T) {
	RegisterLocales(nil, ".")
	got := T("ax7.register.nil")
	core.AssertEqual(t, "ax7.register.nil", got)
	core.AssertNotNil(t, messages)
}

func TestI18N_RegisterLocales_Ugly(t *core.T) {
	locales := fstest.MapFS{"en.json": {Data: []byte(`{"ax7":{"emptydir":"ready"}}`)}}
	RegisterLocales(locales, "")
	got := T("ax7.emptydir")
	core.AssertEqual(t, "ready", got)
	core.AssertTrue(t, len(messages) > 0)
}

func TestI18N_T_Good(t *core.T) {
	locales := fstest.MapFS{"en.json": {Data: []byte(`{"ax7":{"template":"Hello {{.Name}}"}}`)}}
	RegisterLocales(locales, ".")
	got := T("ax7.template", map[string]string{"Name": "Codex"})
	core.AssertEqual(t, "Hello Codex", got)
	core.AssertNotEqual(t, "ax7.template", got)
}

func TestI18N_T_Bad(t *core.T) {
	got := T("ax7.missing.message")
	core.AssertEqual(t, "ax7.missing.message", got)
	core.AssertNotEqual(t, "", got)
}

func TestI18N_T_Ugly(t *core.T) {
	locales := fstest.MapFS{"en.json": {Data: []byte(`{"ax7":{"badtemplate":"{{"}}`)}}
	RegisterLocales(locales, ".")
	got := T("ax7.badtemplate", map[string]string{"Name": "Codex"})
	core.AssertEqual(t, "{{", got)
	core.AssertNotEqual(t, "Codex", got)
}

func TestI18N_Label_Good(t *core.T) {
	locales := fstest.MapFS{"en.json": {Data: []byte(`{"common":{"label":{"agent":"Agent"}}}`)}}
	RegisterLocales(locales, ".")
	got := Label("agent")
	core.AssertEqual(t, "Agent", got)
	core.AssertNotEqual(t, "agent", got)
}

func TestI18N_Label_Bad(t *core.T) {
	got := Label("unregistered")
	core.AssertEqual(t, "unregistered", got)
	core.AssertNotEqual(t, "common.label.unregistered", got)
}

func TestI18N_Label_Ugly(t *core.T) {
	got := Label("")
	core.AssertEqual(t, "", got)
	core.AssertNotEqual(t, "common.label.", got)
}
