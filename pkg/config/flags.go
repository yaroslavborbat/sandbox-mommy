package config

import (
	"os"

	"github.com/spf13/pflag"

	"github.com/yaroslavborbat/sandbox-mommy/pkg/logging"
)

const (
	metricsBindAddressFlag = "metrics-bind-address"
	pprofBindAddressFlag   = "pprof-bind-address"
	logFormatFlag          = "log-format"
	logLevelFlag           = "log-level"
	LogOutputFlag          = "log-output"
	namespaceFlag          = "namespace"

	metricBindAddressEnv = "METRICS_BIND_ADDRESS"
	pprofBindAddressEnv  = "PPROF_BIND_ADDRESS"
	logFormatEnv         = "LOG_FORMAT"
	logLevelEnv          = "LOG_LEVEL"
	logOutputEnv         = "LOG_OUTPUT"
	namespaceEnv         = "NAMESPACE"
)

type BaseOpts struct {
	MetricsBindAddress string
	PprofBindAddress   string
	LogFormat          string
	LogLevel           string
	LogOutput          string
	Namespace          string
}

func (o *BaseOpts) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.MetricsBindAddress, metricsBindAddressFlag, getEnvWithDefault(metricBindAddressEnv, ""), "The address the metric endpoint binds to.")
	fs.StringVar(&o.PprofBindAddress, pprofBindAddressFlag, os.Getenv(pprofBindAddressEnv), "The address the pprof endpoint binds to.")
	fs.StringVar(&o.LogFormat, logFormatFlag, getEnvWithDefault(logFormatEnv, logging.Text.String()), "The log format.")
	fs.StringVar(&o.LogLevel, logLevelFlag, getEnvWithDefault(logLevelEnv, ""), "The log level.")
	fs.StringVar(&o.LogOutput, LogOutputFlag, getEnvWithDefault(logOutputEnv, logging.Stdout.String()), "The log output.")
	fs.StringVar(&o.Namespace, namespaceFlag, os.Getenv(namespaceEnv), "The namespace.")
}

func getEnvWithDefault(key string, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}
