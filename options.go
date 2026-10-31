package main

import (
	"flag"
	"fmt"
	"io"
	"time"
)

type options struct {
	domain        string
	configMapName string
	timeout       time.Duration
}

func parseOptions(args []string, getenv func(string) string, output io.Writer) (options, error) {
	var options options

	flags := flag.NewFlagSet("octelium-initializer", flag.ContinueOnError)

	flags.SetOutput(output)

	flags.StringVar(&options.domain, "domain", getenv("OCTELIUM_CLUSTER_DOMAIN"), "Domain used when creating a missing default ClusterConfig")
	flags.StringVar(&options.configMapName, "cluster-config-map", getenv("OCTELIUM_INITIALIZER_CLUSTER_CONFIG_MAP"), "Existing ConfigMap to receive the ClusterConfig name and UID")

	flags.DurationVar(&options.timeout, "timeout", 2*time.Minute, "Initialization timeout, including retries")

	if err := flags.Parse(args); err != nil {
		return options, err
	}

	if flags.NArg() != 0 {
		return options, fmt.Errorf("unexpected arguments: %v", flags.Args())
	}

	if options.timeout <= 0 {
		return options, fmt.Errorf("timeout must be positive")
	}

	return options, nil
}
