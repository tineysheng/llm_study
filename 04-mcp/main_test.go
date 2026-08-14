package main

import (
	"testing"
)

func TestMCPServer_Initialize(t *testing.T) {
	server := NewMCPServer(nil, nil)
	rawReq := []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`)
	resp := server.handleMessage(rawReq)

	if resp == nil || resp.ID != float64(1) && resp.ID != 1 {
		t.Fatalf("expected response id 1, got %+v", resp)
	}

	initRes, ok := resp.Result.(InitializeResult)
	if !ok {
		t.Fatalf("expected InitializeResult type, got %T", resp.Result)
	}
	if initRes.ProtocolVersion == "" || initRes.ServerInfo.Name == "" {
		t.Fatalf("unexpected init result: %+v", initRes)
	}
}

func TestMCPServer_ToolsList(t *testing.T) {
	server := NewMCPServer(nil, nil)
	rawReq := []byte(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	resp := server.handleMessage(rawReq)

	if resp == nil || resp.Error != nil {
		t.Fatalf("expected success, got error: %+v", resp)
	}

	listRes, ok := resp.Result.(ToolsListResult)
	if !ok {
		t.Fatalf("expected ToolsListResult, got %T", resp.Result)
	}
	if len(listRes.Tools) < 2 {
		t.Fatalf("expected at least 2 tools, got %d", len(listRes.Tools))
	}
}

func TestMCPServer_ToolsCall_Calculator(t *testing.T) {
	server := NewMCPServer(nil, nil)
	rawReq := []byte(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"calculator","arguments":{"left_operand":10,"right_operand":5,"operation":"multiply"}}}`)
	resp := server.handleMessage(rawReq)

	if resp == nil || resp.Error != nil {
		t.Fatalf("expected success, got error: %+v", resp)
	}

	callRes, ok := resp.Result.(CallToolResult)
	if !ok {
		t.Fatalf("expected CallToolResult, got %T", resp.Result)
	}
	if len(callRes.Content) == 0 || callRes.Content[0].Text != "50" {
		t.Fatalf("expected calculation result '50', got %+v", callRes)
	}
}

func TestMCPServer_ToolsCall_DivideZero(t *testing.T) {
	server := NewMCPServer(nil, nil)
	rawReq := []byte(`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"calculator","arguments":{"left_operand":10,"right_operand":0,"operation":"divide"}}}`)
	resp := server.handleMessage(rawReq)

	callRes, ok := resp.Result.(CallToolResult)
	if !ok {
		t.Fatalf("expected CallToolResult, got %T", resp.Result)
	}
	if !callRes.IsError || callRes.Content[0].Text != "除数不能为0" {
		t.Fatalf("expected isError true with message '除数不能为0', got %+v", callRes)
	}
}
