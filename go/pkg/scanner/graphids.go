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
	if kind == protocol.EntryKindHTTP {
		return fmt.Sprintf("entry:http:%s:%s", method, path)
	}
	return fmt.Sprintf("entry:%s:%s:%s", kind, method, path)
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
