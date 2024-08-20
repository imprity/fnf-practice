//go:build fnfdebug

package fnf

func init() {
	AddSuffixToVersionTag("-DEBUG")
	DebugPrintPersist("debug", "true")
}
