package main

import (
	"strings"

	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"

	"github.com/tidwall/gjson"
)

type vmContext struct {
	// Embed the default VM context here,
	// so that we don't need to reimplement all the methods.
	types.DefaultVMContext
}

type pluginContext struct {
	// Embed the default plugin context here,
	// so that we don't need to reimplement all the methods.
	types.DefaultPluginContext

	// headerName and headerValue are the header to be added to response. They are configured via
	// plugin configuration during OnPluginStart.
	headerName  string
	headerValue string

	// addingPath is where to check the header, it can only request or response.
	addingPath string
}

type httpHeaders struct {
	// Embed the default http context here,
	// so that we don't need to reimplement all the methods.
	types.DefaultHttpContext
	contextID   uint32
	headerName  string
	headerValue string
	addingPath  string
}

// Override types.DefaultVMContext.
func (*vmContext) NewPluginContext(contextID uint32) types.PluginContext {
	return &pluginContext{}
}

// Override types.DefaultPluginContext.
func (p *pluginContext) NewHttpContext(contextID uint32) types.HttpContext {
	return &httpHeaders{
		contextID:   contextID,
		headerName:  p.headerName,
		headerValue: p.headerValue,
		addingPath:  p.addingPath,
	}
}

func (p *pluginContext) OnPluginStart(pluginConfigurationSize int) types.OnPluginStartStatus {
	proxywasm.LogDebug("loading plugin config")
	data, err := proxywasm.GetPluginConfiguration()
	if data == nil {
		return types.OnPluginStartStatusOK
	}

	if err != nil {
		proxywasm.LogCriticalf("error reading plugin configuration: %v", err)
		return types.OnPluginStartStatusFailed
	}

	if !gjson.Valid(string(data)) {
		proxywasm.LogCritical(`invalid configuration format; expected {"header": "<header name>", "value": "<header value>",  "path": "<adding path>"}`)
		return types.OnPluginStartStatusFailed
	}

	p.headerName = strings.TrimSpace(gjson.Get(string(data), "header").Str)
	p.headerValue = strings.TrimSpace(gjson.Get(string(data), "value").Str)
	p.addingPath = strings.TrimSpace(gjson.Get(string(data), "path").Str)

	if p.headerName == "" || p.headerValue == "" {
		proxywasm.LogCritical(`invalid configuration format; expected {"header": "<header name>", "value": "<header value>"}`)
		return types.OnPluginStartStatusFailed
	}

	if p.addingPath != "request" && p.addingPath != "response" {
		proxywasm.LogCritical(`invalid configuration format; path can only request or response`)
		return types.OnPluginStartStatusFailed
	}

	proxywasm.LogInfof("header from config: %s = %s, path: %s", p.headerName, p.headerValue, p.addingPath)

	return types.OnPluginStartStatusOK
}

// Override types.DefaultHttpContext.
func (ctx *httpHeaders) OnHttpRequestHeaders(numHeaders int, endOfStream bool) types.Action {
	err := proxywasm.ReplaceHttpRequestHeader("x-request-header", "changed/created by wasm")
	if err != nil {
		proxywasm.LogCritical("failed to set request header: x-request-header")
	}

	if ctx.addingPath == "request" {
		proxywasm.LogWarnf("adding header to request: %s=%s", ctx.headerName, ctx.headerValue)

		if err := proxywasm.AddHttpRequestHeader(ctx.headerName, ctx.headerValue); err != nil {
			proxywasm.LogCriticalf("failed to set request headers: %v", err)
		}
	}

	hs, err := proxywasm.GetHttpRequestHeaders()
	if err != nil {
		proxywasm.LogCriticalf("failed to get request headers: %v", err)
	}

	for _, h := range hs {
		proxywasm.LogInfof("request header --> %s: %s", h[0], h[1])
	}

	return types.ActionContinue
}

// Override types.DefaultHttpContext.
func (ctx *httpHeaders) OnHttpResponseHeaders(_ int, _ bool) types.Action {
	proxywasm.LogInfof("adding header: %s=%s", ctx.headerName, ctx.headerValue)

	// Add a hardcoded header
	if err := proxywasm.AddHttpResponseHeader("x-proxy-wasm-go-sdk-example", "http_headers"); err != nil {
		proxywasm.LogCriticalf("failed to set response constant header: %v", err)
	}

	// Add headers at the response path
	if ctx.addingPath == "response" {
		if err := proxywasm.AddHttpResponseHeader(ctx.headerName, ctx.headerValue); err != nil {
			proxywasm.LogCriticalf("failed to set response headers: %v", err)
		}
	}

	// Get and log the headers
	hs, err := proxywasm.GetHttpResponseHeaders()
	if err != nil {
		proxywasm.LogCriticalf("failed to get response headers: %v", err)
	}

	for _, h := range hs {
		proxywasm.LogInfof("response header <-- %s: %s", h[0], h[1])
	}
	return types.ActionContinue
}

// Override types.DefaultHttpContext.
func (ctx *httpHeaders) OnHttpStreamDone() {
	proxywasm.LogInfof("%d finished", ctx.contextID)
}

func main() {
	proxywasm.SetVMContext(&vmContext{})
}
