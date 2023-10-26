package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/proxytest"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
)

func TestHttpHeaders_OnHttpRequestHeaders(t *testing.T) {
	vmTest(t, func(t *testing.T, vm types.VMContext) {
		opt := proxytest.NewEmulatorOption().WithVMContext(vm)
		host, reset := proxytest.NewHostEmulator(opt)
		defer reset()

		// Initialize http context.
		id := host.InitializeHttpContext()

		// Call OnHttpResponseHeaders.
		hs := [][2]string{{"x-request-header", "test"}}
		action := host.CallOnRequestHeaders(id,
			hs, false)
		require.Equal(t, types.ActionContinue, action)

		// Check headers.
		resultHeaders := host.GetCurrentRequestHeaders(id)
		var found bool
		for _, val := range resultHeaders {
			if val[0] == "x-request-header" {
				require.Equal(t, "changed/created by wasm", val[1])
				found = true
			}
		}
		require.True(t, found)

		// Call OnHttpStreamDone.
		host.CompleteHttpContext(id)

		// Check Envoy logs.
		logs := host.GetInfoLogs()
		require.Contains(t, logs, fmt.Sprintf("%d finished", id))
		require.Contains(t, logs, "request header --> x-request-header: changed/created by wasm")
	})
}

// vmTest executes f twice, once with a types.VMContext that executes plugin code directly
// in the host, and again by executing the plugin code within the compiled main.wasm binary.
// Execution with main.wasm will be skipped if the file cannot be found.
func vmTest(t *testing.T, f func(*testing.T, types.VMContext)) {
	t.Helper()

	t.Run("go", func(t *testing.T) {
		f(t, &vmContext{})
	})

	t.Run("wasm", func(t *testing.T) {
		wasm, err := os.ReadFile("main.wasm")
		if err != nil {
			t.Skip("wasm not found")
		}
		v, err := proxytest.NewWasmVMContext(wasm)
		require.NoError(t, err)
		defer v.Close()
		f(t, v)
	})
}