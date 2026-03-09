package php

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetQAStages_Default(t *testing.T) {
	stages := GetQAStages(QAOptions{})
	assert.Equal(t, []QAStage{QAStageQuick, QAStageStandard}, stages)
}

func TestGetQAStages_Quick(t *testing.T) {
	stages := GetQAStages(QAOptions{Quick: true})
	assert.Equal(t, []QAStage{QAStageQuick}, stages)
}

func TestGetQAStages_Full(t *testing.T) {
	stages := GetQAStages(QAOptions{Full: true})
	assert.Equal(t, []QAStage{QAStageQuick, QAStageStandard, QAStageFull}, stages)
}

func TestGetQAChecks_Quick(t *testing.T) {
	dir := t.TempDir()
	checks := GetQAChecks(dir, QAStageQuick)
	assert.Equal(t, []string{"audit", "fmt", "stan"}, checks)
}

func TestGetQAChecks_Standard_NoPsalm(t *testing.T) {
	dir := t.TempDir()
	checks := GetQAChecks(dir, QAStageStandard)
	assert.Equal(t, []string{"test"}, checks)
}

func TestGetQAChecks_Standard_WithPsalm(t *testing.T) {
	dir := t.TempDir()
	// Create vendor/bin/psalm
	vendorBin := filepath.Join(dir, "vendor", "bin")
	os.MkdirAll(vendorBin, 0755)
	os.WriteFile(filepath.Join(vendorBin, "psalm"), []byte("#!/bin/sh"), 0755)
	checks := GetQAChecks(dir, QAStageStandard)
	assert.Contains(t, checks, "psalm")
	assert.Contains(t, checks, "test")
}

func TestGetQAChecks_Full_NothingDetected(t *testing.T) {
	dir := t.TempDir()
	checks := GetQAChecks(dir, QAStageFull)
	assert.Empty(t, checks)
}

func TestGetQAChecks_Full_WithRectorAndInfection(t *testing.T) {
	dir := t.TempDir()
	vendorBin := filepath.Join(dir, "vendor", "bin")
	os.MkdirAll(vendorBin, 0755)
	os.WriteFile(filepath.Join(vendorBin, "rector"), []byte("#!/bin/sh"), 0755)
	os.WriteFile(filepath.Join(vendorBin, "infection"), []byte("#!/bin/sh"), 0755)
	checks := GetQAChecks(dir, QAStageFull)
	assert.Contains(t, checks, "rector")
	assert.Contains(t, checks, "infection")
}

func TestGetQAChecks_InvalidStage(t *testing.T) {
	checks := GetQAChecks(t.TempDir(), QAStage("invalid"))
	assert.Nil(t, checks)
}
