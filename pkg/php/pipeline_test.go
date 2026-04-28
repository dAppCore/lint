package php

import (
	. "dappco.re/go"
	"os"
	"path/filepath"
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
	AssertEqual(t, []string{"audit", "fmt", "stan"}, checks)
}

func TestGetQAChecks_Standard_NoPsalm(t *T) {
	dir := t.TempDir()
	checks := GetQAChecks(dir, QAStageStandard)
	AssertEqual(t, []string{"test"}, checks)
}

func TestGetQAChecks_Standard_WithPsalm(t *T) {
	dir := t.TempDir()
	// Create vendor/bin/psalm
	vendorBin := filepath.Join(dir, "vendor", "bin")
	os.MkdirAll(vendorBin, 0755)
	os.WriteFile(filepath.Join(vendorBin, "psalm"), []byte("#!/bin/sh"), 0755)
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
	vendorBin := filepath.Join(dir, "vendor", "bin")
	os.MkdirAll(vendorBin, 0755)
	os.WriteFile(filepath.Join(vendorBin, "rector"), []byte("#!/bin/sh"), 0755)
	os.WriteFile(filepath.Join(vendorBin, "infection"), []byte("#!/bin/sh"), 0755)
	checks := GetQAChecks(dir, QAStageFull)
	AssertContains(t, checks, "rector")
	AssertContains(t, checks, "infection")
}

func TestGetQAChecks_InvalidStage(t *T) {
	checks := GetQAChecks(t.TempDir(), QAStage("invalid"))
	AssertNil(t, checks)
	AssertNotEqual(t, []string{}, checks)
}
