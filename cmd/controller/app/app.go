package app

import (
	"context"
	"fmt"
	"log/slog"

	dvpcorev1alpha2 "github.com/deckhouse/virtualization/api/core/v1alpha2"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	kubernetesscheme "k8s.io/client-go/kubernetes/scheme"
	virtv1 "kubevirt.io/api/core/v1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	ctrlConfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/yaroslavborbat/sandbox-mommy/api/core/v1alpha1"
	"github.com/yaroslavborbat/sandbox-mommy/internal/controller/sandbox"
	"github.com/yaroslavborbat/sandbox-mommy/internal/controller/sandboxtemplate"
	"github.com/yaroslavborbat/sandbox-mommy/internal/featuregate"
	"github.com/yaroslavborbat/sandbox-mommy/pkg/config"
	scontrollerutil "github.com/yaroslavborbat/sandbox-mommy/pkg/controller/util"
	"github.com/yaroslavborbat/sandbox-mommy/pkg/logging"
)

const (
	appName = "sandbox-controller"
)

type sandboxControllerOptions struct {
	Base           config.BaseOpts
	LeaderElection bool
}

func (o *sandboxControllerOptions) AddFlags(fs *pflag.FlagSet) {
	o.Base.AddFlags(fs)
	fs.BoolVar(&o.LeaderElection, "leader-election", true, "Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	featuregate.AddFlags(fs)
}

func NewSandboxControllerCommand() *cobra.Command {
	opts := &sandboxControllerOptions{}

	cmd := &cobra.Command{
		Use:   appName,
		Short: appName,
		Args:  cobra.NoArgs,
		PreRunE: func(_ *cobra.Command, _ []string) error {
			return logging.SetupSlogLogger(
				logging.Output(opts.Base.LogOutput),
				logging.Format(opts.Base.LogFormat),
				opts.Base.LogLevel,
			)
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runCommand(cmd.Context(), opts)
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	flagset := cmd.Flags()
	opts.AddFlags(flagset)

	return cmd
}

func runCommand(ctx context.Context, opts *sandboxControllerOptions) (err error) {
	log := slog.With(logging.SlogApp(appName))

	scheme := runtime.NewScheme()
	for _, f := range []func(*runtime.Scheme) error{
		kubernetesscheme.AddToScheme,
		v1alpha1.AddToScheme,
		cdiv1beta1.AddToScheme,
		virtv1.AddToScheme,
		dvpcorev1alpha2.AddToScheme,
	} {
		err = f(scheme)
		if err != nil {
			return fmt.Errorf("failed to setup scheme: %w", err)
		}
	}

	namespace := opts.Base.Namespace
	if namespace == "" {
		namespace, err = scontrollerutil.GetNamespaceFromFs()
		if err != nil {
			return err
		}
	}

	managerOpts := manager.Options{
		LeaderElection:             opts.LeaderElection,
		LeaderElectionNamespace:    namespace,
		LeaderElectionID:           appName,
		LeaderElectionResourceLock: "leases",
		Scheme:                     scheme,
		Metrics: metricsserver.Options{
			BindAddress: opts.Base.MetricsBindAddress,
		},
		PprofBindAddress: opts.Base.PprofBindAddress,
	}

	cfg, err := ctrlConfig.GetConfig()
	if err != nil {
		return err
	}

	_, err = kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}

	mgr, err := manager.New(cfg, managerOpts)
	if err != nil {
		return err
	}

	log.Info("Registering Components.")

	if err = sandbox.SetupController(mgr, log); err != nil {
		return fmt.Errorf("failed to setup Sandbox controller %w", err)
	}
	if err = sandboxtemplate.SetupController(mgr, log); err != nil {
		return fmt.Errorf("failed to setup SandboxTemplate controller %w", err)
	}

	if err = mgr.Start(ctx); err != nil {
		return err
	}

	return nil
}
