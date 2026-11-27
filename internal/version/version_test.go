package version

import "testing"

func TestStringUsesBuildVersion(t *testing.T) {
	original := Version
	t.Cleanup(func() { Version = original })
	Version = "v1.2.3"

	if got := String(); got != "v1.2.3" {
		t.Fatalf("String() = %q, want %q", got, "v1.2.3")
	}
}

func TestResolveVersionFallbacks(t *testing.T) {
	tests := []struct {
		name          string
		buildVersion  string
		moduleVersion string
		want          string
	}{
		{name: "build version", buildVersion: "v1.2.3", moduleVersion: "v1.0.0", want: "v1.2.3"},
		{name: "trim build version", buildVersion: " v1.2.3\n", moduleVersion: "v1.0.0", want: "v1.2.3"},
		{name: "module version", buildVersion: "dev", moduleVersion: "v1.2.3", want: "v1.2.3"},
		{name: "development build", buildVersion: "dev", moduleVersion: "(devel)", want: "dev"},
		{name: "empty versions", buildVersion: "", moduleVersion: "", want: "dev"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolve(tt.buildVersion, tt.moduleVersion); got != tt.want {
				t.Fatalf("resolve(%q, %q) = %q, want %q", tt.buildVersion, tt.moduleVersion, got, tt.want)
			}
		})
	}
}
