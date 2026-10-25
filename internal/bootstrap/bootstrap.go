package bootstrap

import (
	"context"
	"fmt"

	core "github.com/octelium/octelium/apis/main/corev1"
	meta "github.com/octelium/octelium/apis/main/metav1"
	rmeta "github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/clusterconfig"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/grpcerr"
)

// Store is satisfied by the upstream resource-server implementation.
type Store interface {
	GetResource(context.Context, *rmeta.GetOptions, string, string, string) (umetav1.ResourceObjectI, error)
	CreateResource(context.Context, umetav1.ResourceObjectI, string, string, string) (umetav1.ResourceObjectI, error)
}

// Ensure creates only the missing singleton, using upstream persistence and network defaults.
// An existing ClusterConfig is never modified.
func Ensure(ctx context.Context, store Store, domain string) (bool, error) {
	_, err := store.GetResource(ctx, &rmeta.GetOptions{Name: "default"}, ucorev1.API, ucorev1.Version, ucorev1.KindClusterConfig)
	if err == nil {
		return false, nil
	}
	if !grpcerr.IsNotFound(err) {
		return false, err
	}

	if domain == "" {
		return false, fmt.Errorf("domain is required when creating the initial ClusterConfig")
	}

	cc := &core.ClusterConfig{Metadata: &meta.Metadata{Name: "default", IsSystem: true}, Spec: &core.ClusterConfig_Spec{}, Status: &core.ClusterConfig_Status{Domain: domain}}
	if err = clusterconfig.SetClusterSubnets(cc); err != nil {
		return false, err
	}

	_, err = store.CreateResource(ctx, cc, ucorev1.API, ucorev1.Version, ucorev1.KindClusterConfig)
	if grpcerr.AlreadyExists(err) {
		return false, nil
	}

	return err == nil, err
}
