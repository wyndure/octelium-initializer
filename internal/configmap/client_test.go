package configmap

import (
	"errors"
	"testing"
)

func TestLoadNamespace(t *testing.T) {
	denied := errors.New("permission denied")
	cases := []struct {
		name      string
		env       string
		file      string
		readError error
		want      string
		wantError bool
	}{
		{name: "environment", env: " control ", want: "control"},
		{name: "projected namespace", file: "control\n", want: "control"},
		{name: "empty environment falls back", env: " ", file: "control", want: "control"},
		{name: "empty namespace", file: "\n", wantError: true},
		{name: "unreadable namespace", readError: denied, wantError: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := loadNamespace(tc.env, func(path string) ([]byte, error) {
				if path != serviceAccountNamespacePath {
					t.Fatalf("unexpected path %q", path)
				}

				if tc.name == "environment" {
					t.Fatal("namespace file read despite environment")
				}

				return []byte(tc.file), tc.readError
			})

			if (err != nil) != tc.wantError || got != tc.want {
				t.Fatalf("namespace = %q, error = %v", got, err)
			}

			if tc.readError != nil && !errors.Is(err, tc.readError) {
				t.Fatalf("file error was lost: %v", err)
			}
		})
	}
}
