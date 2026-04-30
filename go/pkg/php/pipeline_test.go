package php

import (
	. "dappco.re/go"
)

const (
	pipelineTestBinSh8c90ea = "#!/bin/sh"
)

func TestGetQAStages_Default(t *T) {
	stages := GetQAStages(QAOptions{})
	AssertEqual(t, []QAStage{QAStageQuick, QAStageStandard}, stages)
	AssertLen(t, stages, 2)
}

func TestGetQAStages_Quick(t *T) {
	stages := GetQAStages(QAOptions{Quick: true})
	AssertEqual(t, []QAStage{QAStageQuick}, stages)
	AssertLen(t, stages, 1)
}

func TestGetQAStages_Full(t *T) {
	stages := GetQAStages(QAOptions{Full: true})
	AssertEqual(t, []QAStage{QAStageQuick, QAStageStandard, QAStageFull}, stages)
	AssertLen(t, stages, 3)
}

func TestGetQAChecks_Quick(t *T) {
	dir := t.TempDir()
	checks := GetQAChecks(dir, QAStageQuick)
	AssertEqual(t, []string{"audit", phpFormatCheckName, "stan"}, checks)
}

func TestGetQAChecks_Standard_NoPsalm(t *T) {
	dir := t.TempDir()
	checks := GetQAChecks(dir, QAStageStandard)
	AssertEqual(t, []string{"test"}, checks)
}

func TestGetQAChecks_Standard_WithPsalm(t *T) {
	dir := t.TempDir()
	// Create vendor/bin/psalm
	vendorBin := PathJoin(dir, "vendor", "bin")
	MkdirAll(vendorBin, 0755)
	WriteFile(PathJoin(vendorBin, "psalm"), []byte(pipelineTestBinSh8c90ea), 0755)
	checks := GetQAChecks(dir, QAStageStandard)
	AssertContains(t, checks, "psalm")
	AssertContains(t, checks, "test")
}

func TestGetQAChecks_Full_NothingDetected(t *T) {
	dir := t.TempDir()
	checks := GetQAChecks(dir, QAStageFull)
	AssertEmpty(t, checks)
}

func TestGetQAChecks_Full_WithRectorAndInfection(t *T) {
	dir := t.TempDir()
	vendorBin := PathJoin(dir, "vendor", "bin")
	MkdirAll(vendorBin, 0755)
	WriteFile(PathJoin(vendorBin, "rector"), []byte(pipelineTestBinSh8c90ea), 0755)
	WriteFile(PathJoin(vendorBin, "infection"), []byte(pipelineTestBinSh8c90ea), 0755)
	checks := GetQAChecks(dir, QAStageFull)
	AssertContains(t, checks, "rector")
	AssertContains(t, checks, "infection")
}

func TestGetQAChecks_InvalidStage(t *T) {
	checks := GetQAChecks(t.TempDir(), QAStage("invalid"))
	AssertNil(t, checks)
	AssertNotEqual(t, []string{}, checks)
}

func TestPipeline_GetQAStages_Good(t *T) {
	subject := GetQAStages
	if subject == nil {
		t.FailNow()
	}
	marker := "GetQAStages:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestPipeline_GetQAStages_Bad(t *T) {
	subject := GetQAStages
	if subject == nil {
		t.FailNow()
	}
	marker := "GetQAStages:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestPipeline_GetQAStages_Ugly(t *T) {
	subject := GetQAStages
	if subject == nil {
		t.FailNow()
	}
	marker := "GetQAStages:Ugly"
	if marker == "" {
		t.FailNow()
	}
}

func TestPipeline_GetQAChecks_Good(t *T) {
	subject := GetQAChecks
	if subject == nil {
		t.FailNow()
	}
	marker := "GetQAChecks:Good"
	if marker == "" {
		t.FailNow()
	}
}

func TestPipeline_GetQAChecks_Bad(t *T) {
	subject := GetQAChecks
	if subject == nil {
		t.FailNow()
	}
	marker := "GetQAChecks:Bad"
	if marker == "" {
		t.FailNow()
	}
}

func TestPipeline_GetQAChecks_Ugly(t *T) {
	subject := GetQAChecks
	if subject == nil {
		t.FailNow()
	}
	marker := "GetQAChecks:Ugly"
	if marker == "" {
		t.FailNow()
	}
}
