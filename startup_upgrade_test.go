package main

import (
	"errors"
	"reflect"
	"testing"
)

func TestRunDatabaseUpgradePipelineRunsPreBootstrapBeforeBootstrap(t *testing.T) {
	var calls []string
	err := runDatabaseUpgradePipeline(
		func() (bool, []string, error) {
			calls = append(calls, "scan-pre")
			return true, []string{"001_pre"}, nil
		},
		func() error {
			calls = append(calls, "run-pre")
			return nil
		},
		func() error {
			calls = append(calls, "bootstrap")
			return nil
		},
		func() error {
			calls = append(calls, "run-post")
			return nil
		},
		nil,
	)
	if err != nil {
		t.Fatalf("runDatabaseUpgradePipeline() error = %v", err)
	}

	want := []string{"scan-pre", "run-pre", "bootstrap", "run-post"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
}

func TestRunDatabaseUpgradePipelineSkipsPreBootstrapWhenNoSignals(t *testing.T) {
	var calls []string
	err := runDatabaseUpgradePipeline(
		func() (bool, []string, error) {
			calls = append(calls, "scan-pre")
			return false, nil, nil
		},
		func() error {
			t.Fatal("pre-bootstrap upgrades should not run without signals")
			return nil
		},
		func() error {
			calls = append(calls, "bootstrap")
			return nil
		},
		func() error {
			calls = append(calls, "run-post")
			return nil
		},
		nil,
	)
	if err != nil {
		t.Fatalf("runDatabaseUpgradePipeline() error = %v", err)
	}

	want := []string{"scan-pre", "bootstrap", "run-post"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
}

func TestRunDatabaseUpgradePipelineWarnsButContinuesWhenPreBootstrapFails(t *testing.T) {
	var calls []string
	warned := false
	err := runDatabaseUpgradePipeline(
		func() (bool, []string, error) {
			calls = append(calls, "scan-pre")
			return true, []string{"001_pre"}, nil
		},
		func() error {
			calls = append(calls, "run-pre")
			return errors.New("pre failed")
		},
		func() error {
			calls = append(calls, "bootstrap")
			return nil
		},
		func() error {
			calls = append(calls, "run-post")
			return nil
		},
		func(msg string, _ ...interface{}) {
			if msg != "" {
				warned = true
			}
		},
	)
	if err != nil {
		t.Fatalf("runDatabaseUpgradePipeline() error = %v", err)
	}
	if !warned {
		t.Fatal("expected warning when pre-bootstrap upgrades fail")
	}

	want := []string{"scan-pre", "run-pre", "bootstrap", "run-post"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
}
