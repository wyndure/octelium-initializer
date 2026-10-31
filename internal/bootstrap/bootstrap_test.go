package bootstrap

import (
	"context"
	"errors"
	"testing"

	core "github.com/octelium/octelium/apis/main/corev1"
	meta "github.com/octelium/octelium/apis/main/metav1"
	rmeta "github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type fakeStore struct {
	t           *testing.T
	current     umetav1.ResourceObjectI
	readError   error
	createError error
	creates     int
}

func (s *fakeStore) GetResource(_ context.Context, options *rmeta.GetOptions, api string, version string, kind string) (umetav1.ResourceObjectI, error) {
	s.t.Helper()

	if options.Name != "default" || api != "core" || version != "v1" || kind != "ClusterConfig" {
		s.t.Fatalf("unexpected resource request: %+v %s/%s %s", options, api, version, kind)
	}

	return s.current, s.readError
}
func (s *fakeStore) CreateResource(_ context.Context, resource umetav1.ResourceObjectI, api string, version string, kind string) (umetav1.ResourceObjectI, error) {
	s.t.Helper()

	if api != "core" || version != "v1" || kind != "ClusterConfig" {
		s.t.Fatalf("unexpected creation: %s/%s %s", api, version, kind)
	}

	s.creates++
	if s.createError != nil {
		return nil, s.createError
	}

	s.current = resource

	return resource, nil
}

func TestEnsureKeepsExistingConfiguration(t *testing.T) {
	for _, domain := range []string{"", "different.example.com"} {
		t.Run(domain, func(t *testing.T) {
			original := &core.ClusterConfig{Metadata: &meta.Metadata{Name: "default", Uid: "existing-uid"}, Status: &core.ClusterConfig_Status{Domain: "existing.example.com"}}

			before := proto.Clone(original)

			store := &fakeStore{t: t, current: original}

			created, err := Ensure(t.Context(), store, domain)
			if err != nil || created || store.creates != 0 {
				t.Fatalf("created = %t, writes = %d, error = %v", created, store.creates, err)
			}

			if !proto.Equal(original, before) {
				t.Fatal("existing configuration was changed")
			}
		})
	}
}

func TestEnsureCreatesDefaultConfiguration(t *testing.T) {
	store := &fakeStore{t: t, readError: status.Error(codes.NotFound, "missing")}

	created, err := Ensure(t.Context(), store, "access.example.com")
	if err != nil || !created || store.creates != 1 {
		t.Fatalf("created = %t, writes = %d, error = %v", created, store.creates, err)
	}

	config := store.current.(*core.ClusterConfig)
	if config.Metadata.Name != "default" || !config.Metadata.IsSystem || config.Spec == nil || config.Status.Domain != "access.example.com" {
		t.Fatalf("unexpected configuration: %v", config)
	}

	if got := config.GetStatus().GetNetwork().GetClusterNetwork().GetV4(); got != "100.64.0.0/10" {
		t.Fatalf("network = %q", got)
	}
}

func TestEnsureFailuresAndConcurrentCreation(t *testing.T) {
	unavailable := errors.New("database unavailable")
	cases := []struct {
		name        string
		domain      string
		readError   error
		createError error
		wantWrites  int
		wantError   bool
	}{
		{name: "missing domain", readError: status.Error(codes.NotFound, "missing"), wantError: true},
		{name: "read failure", readError: unavailable, wantError: true},
		{name: "write failure", domain: "access.example.com", readError: status.Error(codes.NotFound, "missing"), createError: unavailable, wantWrites: 1, wantError: true},
		{name: "concurrent creation", domain: "access.example.com", readError: status.Error(codes.NotFound, "missing"), createError: status.Error(codes.AlreadyExists, "created concurrently"), wantWrites: 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := &fakeStore{t: t, readError: tc.readError, createError: tc.createError}

			created, err := Ensure(t.Context(), store, tc.domain)
			if created || (err != nil) != tc.wantError || store.creates != tc.wantWrites {
				t.Fatalf("created = %t, writes = %d, error = %v", created, store.creates, err)
			}

			if (tc.readError == unavailable || tc.createError == unavailable) && !errors.Is(err, unavailable) {
				t.Fatalf("storage error was lost: %v", err)
			}
		})
	}
}
