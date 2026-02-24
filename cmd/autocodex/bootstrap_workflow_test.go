package main

import (
	"errors"
	"reflect"
	"testing"
)

func TestRunBootstrapWorkflowOrder(t *testing.T) {
	originalRepo := bootstrapRepoRunner
	originalReady := bootstrapReadyChecksRunner
	originalSmoke := bootstrapSmokeTaskRunner
	defer func() {
		bootstrapRepoRunner = originalRepo
		bootstrapReadyChecksRunner = originalReady
		bootstrapSmokeTaskRunner = originalSmoke
	}()

	var calls []string
	bootstrapRepoRunner = func(configPath string, requestedProfile string, force bool, initGit bool, initBD bool) error {
		calls = append(calls, "repo:"+configPath+":"+requestedProfile)
		return nil
	}
	bootstrapReadyChecksRunner = func(configPath string) error {
		calls = append(calls, "ready:"+configPath)
		return nil
	}
	bootstrapSmokeTaskRunner = func(configPath, task string) error {
		calls = append(calls, "smoke:"+configPath+":"+task)
		return nil
	}

	err := runBootstrapWorkflow("autocodex.yaml", "max_capability", false, true, true, true, "ship it")
	if err != nil {
		t.Fatalf("runBootstrapWorkflow: %v", err)
	}

	want := []string{
		"repo:autocodex.yaml:max_capability",
		"ready:autocodex.yaml",
		"smoke:autocodex.yaml:ship it",
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("unexpected call order\nwant: %#v\ngot:  %#v", want, calls)
	}
}

func TestRunBootstrapWorkflowSkipsOptionalSteps(t *testing.T) {
	originalRepo := bootstrapRepoRunner
	originalReady := bootstrapReadyChecksRunner
	originalSmoke := bootstrapSmokeTaskRunner
	defer func() {
		bootstrapRepoRunner = originalRepo
		bootstrapReadyChecksRunner = originalReady
		bootstrapSmokeTaskRunner = originalSmoke
	}()

	var calls []string
	bootstrapRepoRunner = func(configPath string, requestedProfile string, force bool, initGit bool, initBD bool) error {
		calls = append(calls, "repo")
		return nil
	}
	bootstrapReadyChecksRunner = func(configPath string) error {
		calls = append(calls, "ready")
		return nil
	}
	bootstrapSmokeTaskRunner = func(configPath, task string) error {
		calls = append(calls, "smoke")
		return nil
	}

	err := runBootstrapWorkflow("autocodex.yaml", "", false, true, true, false, "   ")
	if err != nil {
		t.Fatalf("runBootstrapWorkflow: %v", err)
	}
	want := []string{"repo"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("unexpected call order\nwant: %#v\ngot:  %#v", want, calls)
	}
}

func TestRunBootstrapWorkflowShortCircuitsOnReadyFailure(t *testing.T) {
	originalRepo := bootstrapRepoRunner
	originalReady := bootstrapReadyChecksRunner
	originalSmoke := bootstrapSmokeTaskRunner
	defer func() {
		bootstrapRepoRunner = originalRepo
		bootstrapReadyChecksRunner = originalReady
		bootstrapSmokeTaskRunner = originalSmoke
	}()

	var calls []string
	bootstrapRepoRunner = func(configPath string, requestedProfile string, force bool, initGit bool, initBD bool) error {
		calls = append(calls, "repo")
		return nil
	}
	bootstrapReadyChecksRunner = func(configPath string) error {
		calls = append(calls, "ready")
		return errors.New("ready failed")
	}
	bootstrapSmokeTaskRunner = func(configPath, task string) error {
		calls = append(calls, "smoke")
		return nil
	}

	err := runBootstrapWorkflow("autocodex.yaml", "", false, true, true, true, "task")
	if err == nil || err.Error() != "ready failed" {
		t.Fatalf("expected ready failure, got %v", err)
	}
	want := []string{"repo", "ready"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("unexpected call order\nwant: %#v\ngot:  %#v", want, calls)
	}
}

func TestRunBootstrapWorkflowReturnsRepoFailure(t *testing.T) {
	originalRepo := bootstrapRepoRunner
	originalReady := bootstrapReadyChecksRunner
	originalSmoke := bootstrapSmokeTaskRunner
	defer func() {
		bootstrapRepoRunner = originalRepo
		bootstrapReadyChecksRunner = originalReady
		bootstrapSmokeTaskRunner = originalSmoke
	}()

	bootstrapRepoRunner = func(configPath string, requestedProfile string, force bool, initGit bool, initBD bool) error {
		return errors.New("repo failed")
	}
	bootstrapReadyChecksRunner = func(configPath string) error {
		t.Fatal("ready should not be called")
		return nil
	}
	bootstrapSmokeTaskRunner = func(configPath, task string) error {
		t.Fatal("smoke should not be called")
		return nil
	}

	err := runBootstrapWorkflow("autocodex.yaml", "", false, true, true, true, "task")
	if err == nil || err.Error() != "repo failed" {
		t.Fatalf("expected repo failure, got %v", err)
	}
}
