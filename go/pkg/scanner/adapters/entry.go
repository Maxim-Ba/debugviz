package adapters

import (
	"golang.org/x/tools/go/packages"

	"github.com/Maxim-Ba/debugviz/go/pkg/protocol"
)

// HandlerRef identifies a Go function used as an HTTP handler or middleware.
type HandlerRef struct {
	File    string
	Line    int
	Name    string
	Package string
}

// EntryPoint is a discovered runtime entry (HTTP route, gRPC method, etc.).
type EntryPoint struct {
	Kind       protocol.EntryKind
	Method     string
	Path       string
	Handler    HandlerRef
	HasHandler bool
	Middleware []HandlerRef
}

// EntryDiscoverer finds entry points in loaded Go packages.
type EntryDiscoverer interface {
	Name() string
	Discover(ctx *ScanContext, pkgs []*packages.Package) ([]EntryPoint, error)
}
