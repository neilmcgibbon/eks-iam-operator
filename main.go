/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"errors"
	"flag"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	eksiamoperatorv1beta1 "github.com/neilmcgibbon/eks-iam-operator/api/v1beta1"
	"github.com/neilmcgibbon/eks-iam-operator/controllers"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(eksiamoperatorv1beta1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var err error
	var configFlag string
	flag.StringVar(&configFlag, "config", "", "The controller will load its configuration from this file.")

	opts := zap.Options{Development: true}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	ctrlConfig := eksiamoperatorv1beta1.Config{}
	options := ctrl.Options{Scheme: scheme}
	if options, err = options.AndFrom(ctrl.ConfigFile().AtPath(configFlag).OfKind(&ctrlConfig)); err != nil {
		setupLog.Error(err, "unable to load the config file")
		os.Exit(1)
	}

	if err := validateConfig(ctrlConfig); err != nil {
		setupLog.Error(err, "invalid config")
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.RoleReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Log:    ctrl.Log.WithName("eks-iam-controller"),

		RolePrefix:         ctrlConfig.RoleNameOptions.Prefix,
		RoleSuffix:         ctrlConfig.RoleNameOptions.Suffix,
		InlinePolicyPrefix: ctrlConfig.InlinePolicyNameOptions.Prefix,
		InlinePolicySuffix: ctrlConfig.InlinePolicyNameOptions.Suffix,
		OIDCIssuerURL:      ctrlConfig.OIDC.IssuerURL,
		OIDCProviderARN:    ctrlConfig.OIDC.ProviderARN,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Role")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func validateConfig(cfg eksiamoperatorv1beta1.Config) error {

	// check OIDC provider ARN
	if len(cfg.OIDC.ProviderARN) == 0 {
		return errors.New("<config> oidc.providerArn must be set")
	}

	// check OIDC issuer URL
	if len(cfg.OIDC.IssuerURL) == 0 {
		return errors.New("<config> oidc.issuerURL must be set")
	}

	return nil
}
