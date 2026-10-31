package main

import (
	"errors"
	"flag"
	"io"
	"testing"
	"time"
)

func TestParseOptions(t *testing.T) {
	cases := []struct {
		name        string
		args        []string
		environment map[string]string
		want        options
		wantError   bool
	}{
		{name: "defaults", want: options{timeout: 2 * time.Minute}},
		{name: "environment", environment: map[string]string{"OCTELIUM_CLUSTER_DOMAIN": "access.example.com", "OCTELIUM_INITIALIZER_CLUSTER_CONFIG_MAP": "cluster"}, want: options{domain: "access.example.com", configMapName: "cluster", timeout: 2 * time.Minute}},
		{name: "flags override environment", args: []string{"--domain=other.example.com", "--cluster-config-map=another", "--timeout=10s"}, environment: map[string]string{"OCTELIUM_CLUSTER_DOMAIN": "ignored.example.com", "OCTELIUM_INITIALIZER_CLUSTER_CONFIG_MAP": "ignored"}, want: options{domain: "other.example.com", configMapName: "another", timeout: 10 * time.Second}},
		{name: "explicitly disable publication", args: []string{"--cluster-config-map="}, environment: map[string]string{"OCTELIUM_INITIALIZER_CLUSTER_CONFIG_MAP": "cluster"}, want: options{timeout: 2 * time.Minute}},
		{name: "zero timeout", args: []string{"--timeout=0s"}, wantError: true},
		{name: "negative timeout", args: []string{"--timeout=-1s"}, wantError: true},
		{name: "positional argument", args: []string{"unexpected"}, wantError: true},
		{name: "unknown flag", args: []string{"--unknown"}, wantError: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseOptions(tc.args, func(name string) string { return tc.environment[name] }, io.Discard)

			if (err != nil) != tc.wantError {
				t.Fatalf("error = %v, want error %t", err, tc.wantError)
			}

			if err == nil && got != tc.want {
				t.Fatalf("options = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestHelp(t *testing.T) {
	_, err := parseOptions([]string{"--help"}, func(string) string { return "" }, io.Discard)

	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("error = %v, want help", err)
	}
}
