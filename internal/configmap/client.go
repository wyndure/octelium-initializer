package configmap

import (
	"fmt"
	"os"
	"strings"

	coreclient "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

const serviceAccountNamespacePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

// NewInClusterClient loads projected service-account credentials and the Pod namespace.
func NewInClusterClient() (coreclient.ConfigMapInterface, error) {
	credentials, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("load Kubernetes service-account credentials: %w", err)
	}

	namespace, err := loadNamespace(os.Getenv("POD_NAMESPACE"), os.ReadFile)
	if err != nil {
		return nil, err
	}

	client, err := coreclient.NewForConfig(credentials)
	if err != nil {
		return nil, fmt.Errorf("create Kubernetes client: %w", err)
	}

	return client.ConfigMaps(namespace), nil
}

func loadNamespace(podNamespace string, readFile func(string) ([]byte, error)) (string, error) {
	namespace := strings.TrimSpace(podNamespace)
	if namespace == "" {
		contents, err := readFile(serviceAccountNamespacePath)
		if err != nil {
			return "", fmt.Errorf("read service-account namespace: %w", err)
		}

		namespace = strings.TrimSpace(string(contents))
	}

	if namespace == "" {
		return "", fmt.Errorf("kubernetes namespace is empty")
	}

	return namespace, nil
}
