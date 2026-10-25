package configmap

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	coreclient "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

func TestPublish(t *testing.T) {
	cases := []struct {
		name      string
		status    int
		wantError bool
	}{
		{name: "published", status: http.StatusOK},
		{name: "permission denied", status: http.StatusForbidden, wantError: true},
		{name: "ConfigMap missing", status: http.StatusNotFound, wantError: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPatch || r.URL.Path != "/api/v1/namespaces/control/configmaps/cluster" {
					t.Errorf("unexpected request %s %s", r.Method, r.URL)
				}

				if r.Header.Get("Content-Type") != "application/merge-patch+json" {
					t.Errorf("unexpected patch content type %q", r.Header.Get("Content-Type"))
				}

				var patch map[string]map[string]string
				if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
					t.Error(err)
				}

				if len(patch) != 1 || len(patch["data"]) != 2 || patch["data"]["name"] != "default" || patch["data"]["uid"] != "existing-uid" {
					t.Errorf("unexpected patch %#v", patch)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)

				if tc.status != http.StatusOK {
					_ = json.NewEncoder(w).Encode(map[string]any{"kind": "Status", "apiVersion": "v1", "status": "Failure", "code": tc.status})
					return
				}

				_ = json.NewEncoder(w).Encode(map[string]any{"kind": "ConfigMap", "apiVersion": "v1", "metadata": map[string]string{"name": "cluster"}})
			}))

			defer server.Close()

			client, err := coreclient.NewForConfig(&rest.Config{Host: server.URL})
			if err != nil {
				t.Fatal(err)
			}

			err = Publish(t.Context(), client.ConfigMaps("control"), "cluster", "default", "existing-uid")
			if (err != nil) != tc.wantError {
				t.Fatalf("error = %v, want error %t", err, tc.wantError)
			}

			if tc.status == http.StatusForbidden && !apierrors.IsForbidden(err) {
				t.Fatalf("permission error was lost: %v", err)
			}

			if tc.status == http.StatusNotFound && !apierrors.IsNotFound(err) {
				t.Fatalf("not-found error was lost: %v", err)
			}
		})
	}
}
func TestPublishRejectsMissingIdentity(t *testing.T) {
	for _, input := range [][3]string{{"", "default", "uid"}, {"cluster", "", "uid"}, {"cluster", "default", ""}} {
		if err := Publish(t.Context(), nil, input[0], input[1], input[2]); err == nil {
			t.Fatalf("missing identity accepted: %v", input)
		}
	}
}
