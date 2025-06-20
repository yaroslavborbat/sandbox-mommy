package featuregate

import (
	"strings"

	"github.com/spf13/pflag"
)

type FeatureGate string

func (f FeatureGate) String() string {
	return string(f)
}

const (
	Kubevirt FeatureGate = "KUBEVIRT"
	DVP      FeatureGate = "DVP"
)

var defaultFeatureGates = FeatureGates{}

type FeatureGates struct {
	gates []string
}

func (f *FeatureGates) Enabled(feature FeatureGate) bool {
	for _, gate := range f.gates {
		if strings.ToUpper(gate) == feature.String() {
			return true
		}
	}
	return false
}

func (f *FeatureGates) AddFlags(fs *pflag.FlagSet) {
	fs.StringSliceVar(&f.gates, "feature-gate", nil, "A name of feature gate which should be enabled")
}

func AddFlags(fs *pflag.FlagSet) {
	defaultFeatureGates.AddFlags(fs)
}

func Enabled(feature FeatureGate) bool {
	return defaultFeatureGates.Enabled(feature)
}
