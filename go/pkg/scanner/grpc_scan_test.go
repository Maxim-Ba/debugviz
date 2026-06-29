package scanner

import (
	"testing"

	"github.com/Maxim-Ba/debugviz/go/pkg/protocol"
)

func TestScanDemoGRPC(t *testing.T) {
	root := repoRoot(t)

	graph, err := Scan([]string{"./demo/grpc"}, Options{
		Dir:       root,
		Framework: "grpc",
	})
	if err != nil {
		t.Fatal(err)
	}

	wantMethods := []string{
		"user.v1.UserService/GetUser",
		"user.v1.UserService/ListUsers",
		"user.v1.UserService/DeleteUser",
	}

	found := make(map[string]protocol.Node)
	for _, node := range graph.Nodes {
		if node.Type != protocol.NodeTypeEntryPoint || node.Kind != protocol.EntryKindGRPC {
			continue
		}
		found[node.Name] = node
	}

	for _, want := range wantMethods {
		node, ok := found[want]
		if !ok {
			t.Fatalf("missing gRPC entry %q", want)
		}
		service, _ := node.Metadata["service"].(string)
		method, _ := node.Metadata["method"].(string)
		if service == "" || method == "" {
			t.Fatalf("entry %q missing service/method metadata: %#v", want, node.Metadata)
		}
		if service+"/"+method != want {
			t.Fatalf("entry %q metadata = %s/%s", want, service, method)
		}
	}

	wantHandler := "func:demo/grpc/internal/server/user.go:GetUser"
	entryID := entryNodeID(protocol.EntryKindGRPC, "user.v1.UserService", "GetUser")
	handlerLinked := false
	for _, edge := range graph.Edges {
		if edge.Type != protocol.EdgeTypeEntryHandles || edge.Source != entryID {
			continue
		}
		if edge.Target == wantHandler {
			handlerLinked = true
			break
		}
	}
	if !handlerLinked {
		t.Fatalf("missing entry_handles edge %s -> %s", entryID, wantHandler)
	}
}
