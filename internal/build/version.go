package build

// Version is set at build time via ldflags:
// go build -ldflags "-X github.com/geomark27/deploy-doc/internal/build.Version=v1.0.0"
var Version = "dev"
