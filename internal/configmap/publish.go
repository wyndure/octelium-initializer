package configmap

import (
	"context"
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	coreclient "k8s.io/client-go/kubernetes/typed/core/v1"
)

// Publish patches the name and uid data keys in an existing ConfigMap.
// Requiring an existing object allows RBAC to restrict writes by resource name.
func Publish(ctx context.Context, client coreclient.ConfigMapInterface, configMapName string, clusterConfigName string, clusterConfigUID string) error {
	if configMapName == "" || clusterConfigName == "" || clusterConfigUID == "" {
		return fmt.Errorf("ConfigMap name, ClusterConfig name and UID must be nonempty")
	}

	patch, err := json.Marshal(map[string]any{"data": map[string]string{"name": clusterConfigName, "uid": clusterConfigUID}})
	if err != nil {
		return err
	}

	_, err = client.Patch(ctx, configMapName, types.MergePatchType, patch, metav1.PatchOptions{})

	return err
}
