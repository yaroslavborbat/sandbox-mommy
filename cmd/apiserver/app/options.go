package app

import (
	"fmt"
	"net"
	"strings"

	"github.com/yaroslavborbat/sandbox-mommy/internal/apiserver/api"
	generatedopenapi "github.com/yaroslavborbat/sandbox-mommy/internal/apiserver/api/generated"
	"github.com/yaroslavborbat/sandbox-mommy/internal/apiserver/server"
	"github.com/yaroslavborbat/sandbox-mommy/internal/featuregate"
	"k8s.io/apimachinery/pkg/types"
	openapinamer "k8s.io/apiserver/pkg/endpoints/openapi"
	genericapiserver "k8s.io/apiserver/pkg/server"
	genericoptions "k8s.io/apiserver/pkg/server/options"
	utilversion "k8s.io/apiserver/pkg/util/version"
	"k8s.io/client-go/rest"
	"k8s.io/component-base/cli/flag"
	"k8s.io/component-base/logs"
	logsapi "k8s.io/component-base/logs/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type Options struct {
	// genericoptions.RecomendedOptions - EtcdOptions
	SecureServing  *genericoptions.SecureServingOptionsWithLoopback
	Authentication *genericoptions.DelegatingAuthenticationOptions
	Authorization  *genericoptions.DelegatingAuthorizationOptions
	Audit          *genericoptions.AuditOptions
	Features       *genericoptions.FeatureOptions
	Logging        *logs.Options

	ServiceAccount types.NamespacedName

	ShowVersion bool
	// Only to be used to for testing
	DisableAuthForTesting bool
}

func (o *Options) Validate() error {
	return logsapi.ValidateAndApply(o.Logging, nil)
}

func (o *Options) Flags() (fs flag.NamedFlagSets) {
	msfs := fs.FlagSet("sandbox-api server")
	msfs.BoolVar(&o.ShowVersion, "version", false, "Show version")
	msfs.StringVar(&o.ServiceAccount.Name, "service-account-name", "", "Service account name")
	msfs.StringVar(&o.ServiceAccount.Namespace, "service-account-namespace", "", "Service account namespace")

	featuregate.AddFlags(fs.FlagSet("sabdbox-api feature-gates"))

	o.SecureServing.AddFlags(fs.FlagSet("sandbox-api secure serving"))
	o.Authentication.AddFlags(fs.FlagSet("sandbox-api authentication"))
	o.Authorization.AddFlags(fs.FlagSet("sandbox-api authorization"))
	o.Audit.AddFlags(fs.FlagSet("sandbox-api audit log"))
	o.Features.AddFlags(fs.FlagSet("features"))
	logsapi.AddFlags(o.Logging, fs.FlagSet("logging"))

	return fs
}

func NewOptions() *Options {
	return &Options{
		SecureServing:  genericoptions.NewSecureServingOptions().WithLoopback(),
		Authentication: genericoptions.NewDelegatingAuthenticationOptions(),
		Authorization:  genericoptions.NewDelegatingAuthorizationOptions(),
		Features:       genericoptions.NewFeatureOptions(),
		Audit:          genericoptions.NewAuditOptions(),

		Logging: logs.NewOptions(),
	}
}

func (o *Options) ServerConfig() (*server.Config, error) {
	apiserver, err := o.ApiserverConfig()
	if err != nil {
		return nil, err
	}
	restConfig, err := o.RestConfig()
	if err != nil {
		return nil, err
	}

	conf := &server.Config{
		Apiserver:      apiserver,
		Rest:           restConfig,
		ServiceAccount: o.ServiceAccount,
	}
	if err := conf.Validate(); err != nil {
		return nil, err
	}
	return conf, nil
}

func (o *Options) ApiserverConfig() (*genericapiserver.Config, error) {
	if err := o.SecureServing.MaybeDefaultWithSelfSignedCerts("localhost", nil, []net.IP{net.ParseIP("127.0.0.1")}); err != nil {
		return nil, fmt.Errorf("error creating self-signed certificates: %w", err)
	}

	serverConfig := genericapiserver.NewConfig(api.Codecs)
	if err := o.SecureServing.ApplyTo(&serverConfig.SecureServing, &serverConfig.LoopbackClientConfig); err != nil {
		return nil, err
	}

	if !o.DisableAuthForTesting {
		if err := o.Authentication.ApplyTo(&serverConfig.Authentication, serverConfig.SecureServing, nil); err != nil {
			return nil, err
		}
		if err := o.Authorization.ApplyTo(&serverConfig.Authorization); err != nil {
			return nil, err
		}
	}

	if err := o.Audit.ApplyTo(serverConfig); err != nil {
		return nil, err
	}

	versionGet := utilversion.DefaultBuildEffectiveVersion()
	serverConfig.EffectiveVersion = versionGet
	serverConfig.OpenAPIConfig = genericapiserver.DefaultOpenAPIConfig(generatedopenapi.GetOpenAPIDefinitions, openapinamer.NewDefinitionNamer(api.Scheme))
	serverConfig.OpenAPIV3Config = genericapiserver.DefaultOpenAPIV3Config(generatedopenapi.GetOpenAPIDefinitions, openapinamer.NewDefinitionNamer(api.Scheme))
	serverConfig.OpenAPIConfig.Info.Title = "SandboxAPI"
	serverConfig.OpenAPIV3Config.Info.Title = "SandboxAPI"
	serverConfig.OpenAPIConfig.Info.Version = strings.Split(versionGet.String(), "-")[0]
	serverConfig.OpenAPIV3Config.Info.Version = strings.Split(versionGet.String(), "-")[0]

	return serverConfig, nil
}

func (o *Options) RestConfig() (*rest.Config, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	// Use protobufs for communication with apiserver
	// cfg.ContentType = "application/vnd.kubernetes.protobuf"
	err = rest.SetKubernetesDefaults(cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}
