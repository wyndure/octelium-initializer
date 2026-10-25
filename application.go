package main

import (
	"context"
	"fmt"
	"log"
	"time"

	rmeta "github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/rscserver/rscserver"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/wyndure/octelium-initializer/internal/bootstrap"
	"github.com/wyndure/octelium-initializer/internal/configmap"
	coreclient "k8s.io/client-go/kubernetes/typed/core/v1"
)

const retryInterval = 2 * time.Second

type clusterConfigStore interface {
	bootstrap.Store
	Close()
}

type storageSession struct {
	*rscserver.Server
}

func openStorage(ctx context.Context) (clusterConfigStore, error) {
	server, err := rscserver.NewServer(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("connect to storage: %w", err)
	}

	return &storageSession{Server: server}, nil
}

func (session *storageSession) Close() {
	if err := session.GetDB().Close(); err != nil {
		log.Printf("close PostgreSQL connection: %v", err)
	}

	if err := session.GetRedisC().Close(); err != nil {
		log.Printf("close Redis connection: %v", err)
	}
}

func run(ctx context.Context, options options) error {
	ctx, cancel := context.WithTimeout(ctx, options.timeout)

	defer cancel()

	var client coreclient.ConfigMapInterface
	if options.configMapName != "" {
		var err error

		client, err = configmap.NewInClusterClient()
		if err != nil {
			return err
		}
	}

	err := retry(ctx, retryInterval, func(ctx context.Context) error {
		created, err := initializeOnce(ctx, options, client, openStorage)
		if err == nil {
			log.Printf("default ClusterConfig ready (created=%t)", created)
		}

		return err
	})

	if err != nil {
		return fmt.Errorf("initialize ClusterConfig: %w", err)
	}

	return nil
}

// initializeOnce releases its storage session before returning to the retry loop.
func initializeOnce(ctx context.Context, options options, client coreclient.ConfigMapInterface, connect func(context.Context) (clusterConfigStore, error)) (bool, error) {
	store, err := connect(ctx)
	if err != nil {
		return false, err
	}

	defer store.Close()

	created, err := bootstrap.Ensure(ctx, store, options.domain)
	if err != nil {
		return false, fmt.Errorf("ensure default ClusterConfig: %w", err)
	}

	if client == nil {
		return created, nil
	}

	clusterConfig, err := store.GetResource(ctx, &rmeta.GetOptions{Name: "default"}, ucorev1.API, ucorev1.Version, ucorev1.KindClusterConfig)
	if err != nil {
		return created, fmt.Errorf("read initialized ClusterConfig: %w", err)
	}

	metadata := clusterConfig.GetMetadata()
	if err := configmap.Publish(ctx, client, options.configMapName, metadata.GetName(), metadata.GetUid()); err != nil {
		return created, fmt.Errorf("publish ClusterConfig to ConfigMap %q: %w", options.configMapName, err)
	}

	return created, nil
}
