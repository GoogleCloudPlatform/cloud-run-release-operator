package knative

import (
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	k8smachinery "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Namespaces retrieves the list of namespaces in the Kubernetes cluster.
func Namespaces(config *rest.Config) ([]v1.Namespace, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize Kubernetes client set")
	}

	namespaces, err := clientset.CoreV1().Namespaces().List(k8smachinery.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get list of namespaces")
	}
	return namespaces.Items, nil
}
