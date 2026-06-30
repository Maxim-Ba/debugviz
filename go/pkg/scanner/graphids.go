package scanner

import (
	"fmt"
	"strings"

	"github.com/Maxim-Ba/debugviz/go/pkg/protocol"
)

func functionNodeID(relFile, funcName string) string {
	return fmt.Sprintf("func:%s:%s", relFile, funcName)
}

func middlewareNodeID(relFile, funcName string) string {
	return fmt.Sprintf("mw:%s:%s", relFile, funcName)
}

func entryNodeID(kind protocol.EntryKind, method, path string) string {
	switch kind {
	case protocol.EntryKindHTTP:
		return fmt.Sprintf("entry:http:%s:%s", method, path)
	case protocol.EntryKindGRPC:
		return fmt.Sprintf("entry:grpc:%s:%s", method, path)
	case protocol.EntryKindCLI:
		return fmt.Sprintf("entry:cli:%s", slugCLICommand(method))
	case protocol.EntryKindWorker:
		return fmt.Sprintf("entry:worker:%s", slugCLICommand(method))
	default:
		return fmt.Sprintf("entry:%s:%s:%s", kind, method, path)
	}
}

func entryHandlesEdgeID(entryID, handlerID string) string {
	return "edge:entry_handles:" + slugImportPath(entryID) + ":" + slugImportPath(handlerID)
}

func middlewareChainEdgeID(entryID, middlewareID string, order int) string {
	return fmt.Sprintf("edge:middleware:%s:%s:%d", slugImportPath(entryID), slugImportPath(middlewareID), order)
}

func httpEntryName(method, path string) string {
	return strings.TrimSpace(method + " " + path)
}

func grpcEntryName(service, method string) string {
	return service + "/" + method
}

func slugCLICommand(command string) string {
	return strings.NewReplacer(" ", "-", "/", "-", ":", "-").Replace(command)
}
