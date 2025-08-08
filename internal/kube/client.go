package kube

import (
	"flag"
	"os"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// NewClient creates a Kubernetes clientset using in-cluster config if present,
// otherwise falls back to KUBECONFIG/default loading rules.
func NewClient() (*kubernetes.Clientset, *rest.Config, error) {
	if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/token"); err == nil {
		cfg, err := rest.InClusterConfig()
		if err != nil {
			return nil, nil, err
		}
		cs, err := kubernetes.NewForConfig(cfg)
		if err != nil {
			return nil, nil, err
		}
		return cs, cfg, nil
	}
	// out-of-cluster
	kubeconfig := ""
	if env := os.Getenv("KUBECONFIG"); env != "" {
		kubeconfig = env
	}
	// quiet warnings from clientcmd by setting flag.CommandLine to a new FlagSet if needed
	_ = flag.CommandLine
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, nil, err
	}
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, nil, err
	}
	return cs, cfg, nil
}
