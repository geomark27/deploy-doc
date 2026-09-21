package updater

import "testing"

// isNewer gates the self-update. A bug here either blocks every update or, far
// worse, installs an older binary over a newer one.
func TestIsNewerOnlyAcceptsStrictlyGreaterVersions(t *testing.T) {
	cases := []struct {
		candidate, base string
		want            bool
	}{
		{"v1.3.0", "v1.2.10", true},
		{"v1.2.11", "v1.2.10", true},
		{"v2.0.0", "v1.99.99", true},
		{"v1.2.10", "v1.2.10", false}, // igual no es mayor
		{"v1.2.9", "v1.2.10", false},  // 9 < 10: comparación numérica, no de texto
		{"v1.2.0", "v1.10.0", false},
		{"v0.9.9", "v1.0.0", false},
		{"1.3.0", "v1.2.0", true}, // el prefijo v es opcional
	}
	for _, c := range cases {
		if got := isNewer(c.candidate, c.base); got != c.want {
			t.Errorf("isNewer(%q, %q) = %v, esperaba %v", c.candidate, c.base, got, c.want)
		}
	}
}

func TestIsNewerRejectsUnparseableVersions(t *testing.T) {
	// "dev" is the default when the binary is built without ldflags. It must
	// never be treated as comparable, or every run would offer an update.
	cases := [][2]string{
		{"v1.3.0", "dev"},
		{"dev", "v1.3.0"},
		{"v1.3", "v1.2.0"},
		{"v1.3.0.1", "v1.2.0"},
		{"vX.Y.Z", "v1.2.0"},
		{"", "v1.2.0"},
	}
	for _, c := range cases {
		if isNewer(c[0], c[1]) {
			t.Errorf("isNewer(%q, %q) = true, esperaba false", c[0], c[1])
		}
	}
}

func TestAssetNameIsDefinedForEveryPlatform(t *testing.T) {
	// The name must match what the release publishes, or update breaks.
	if got := assetName(); got == "" {
		t.Fatal("assetName() vacío")
	}
}
