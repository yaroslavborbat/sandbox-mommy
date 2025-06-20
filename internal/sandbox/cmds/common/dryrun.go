package common

import (
	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var dryRun = false

func SetDryRun(fs *pflag.FlagSet) {
	fs.BoolVar(&dryRun, "dry-run", false, "Execute with dry-run")
}

func GetDryRun() []string {
	if dryRun {
		return []string{metav1.DryRunAll}
	}
	return nil
}

func IsDryRun() bool {
	return dryRun
}
