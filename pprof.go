//go:build fnfpprof

package fnf

import (
	"net/http"
	_ "net/http/pprof"
)

func init() {
	AddSuffixToVersionTag("-PPROF")
	DebugPrintPersist("pprof", "true")
	go func() {
		FnfLogger.Println(http.ListenAndServe("localhost:6060", nil))
	}()
}
