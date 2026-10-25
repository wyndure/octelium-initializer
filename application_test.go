package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	core "github.com/octelium/octelium/apis/main/corev1"
	meta "github.com/octelium/octelium/apis/main/metav1"
	rmeta "github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	coreclient "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

type testStore struct {
	getError error
	reads    int
	closed   bool
}

func (s *testStore) GetResource(context.Context, *rmeta.GetOptions, string, string, string) (umetav1.ResourceObjectI, error) {
	s.reads++

	if s.getError != nil {
		return nil, s.getError
	}

	return &core.ClusterConfig{Metadata: &meta.Metadata{Name: "default", Uid: "existing-uid"}}, nil
}

func (s *testStore) CreateResource(context.Context, umetav1.ResourceObjectI, string, string, string) (umetav1.ResourceObjectI, error) {
	return nil, errors.New("unexpected creation")
}

func (s *testStore) Close() {
	s.closed = true
}

func TestInitializeClosesStorage(t *testing.T) {
	failure := errors.New("read failed")
	for _, tc := range []struct {
		name      string
		getError  error
		wantError bool
	}{{name: "existing"}, {name: "read failure", getError: failure, wantError: true}} {
		t.Run(tc.name, func(t *testing.T) {
			store := &testStore{getError: tc.getError}

			created, err := initializeOnce(t.Context(), options{}, nil, func(context.Context) (clusterConfigStore, error) { return store, nil })
			if created || (err != nil) != tc.wantError || !store.closed {
				t.Fatalf("created = %t, error = %v, closed = %t", created, err, store.closed)
			}

			if tc.getError != nil && !errors.Is(err, tc.getError) {
				t.Fatalf("storage error was lost: %v", err)
			}
		})
	}
}

func TestInitializeConnectionFailure(t *testing.T) {
	failure := errors.New("connection refused")

	_, err := initializeOnce(t.Context(), options{}, nil, func(context.Context) (clusterConfigStore, error) { return nil, failure })
	if !errors.Is(err, failure) {
		t.Fatalf("error = %v", err)
	}
}

func TestInitializePublishesAndCloses(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusForbidden} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			patches := 0

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				patches++

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)

				if status == http.StatusOK {
					_, _ = w.Write([]byte(`{"kind":"ConfigMap","apiVersion":"v1","metadata":{"name":"cluster"}}`))
					return
				}
				_, _ = w.Write([]byte(`{"kind":"Status","apiVersion":"v1","status":"Failure","code":403}`))
			}))

			defer server.Close()

			client, err := coreclient.NewForConfig(&rest.Config{Host: server.URL})
			if err != nil {
				t.Fatal(err)
			}

			store := &testStore{}

			_, err = initializeOnce(t.Context(), options{configMapName: "cluster"}, client.ConfigMaps("control"), func(context.Context) (clusterConfigStore, error) {
				return store, nil
			})

			if (err != nil) != (status != http.StatusOK) || !store.closed || store.reads != 2 || patches != 1 {
				t.Fatalf("error = %v, store = %+v, patches = %d", err, store, patches)
			}
		})
	}
}
