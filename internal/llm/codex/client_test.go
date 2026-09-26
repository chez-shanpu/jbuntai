package codex

import (
	"bufio"
	"encoding/json/v2"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
)

func TestClientRun(t *testing.T) {
	tests := []struct {
		name      string
		mode      string
		want      string
		wantError string
	}{
		{name: "completed", mode: "completed", want: "完成文"},
		{name: "failed", mode: "failed", wantError: "model unavailable"},
		{name: "empty", mode: "empty", wantError: "without an answer"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("JBUNTAI_FAKE_CODEX", tt.mode)
			tmp := t.TempDir()
			t.Setenv("TMPDIR", tmp)
			c := &client{command: os.Args[0], args: []string{"-test.run=^TestAppServerHelper$"}}
			got, err := c.run(t.Context(), "example-model", "low", "System prompt", "User prompt")
			if entries, _ := os.ReadDir(tmp); len(entries) != 0 {
				t.Errorf("app-server work dir was not removed: %v", entries)
			}
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("error = %v, want substring %q", err, tt.wantError)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("answer = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAppServerHelper(t *testing.T) {
	mode := os.Getenv("JBUNTAI_FAKE_CODEX")
	if mode == "" {
		return
	}
	// The app-server must be locked down (see disabledFeatures).
	if !slices.Contains(os.Args, "features.shell_tool=false") || !slices.Contains(os.Args, `web_search="disabled"`) {
		os.Exit(2)
	}
	respond := func(id *int, result string) {
		fmt.Fprintf(os.Stdout, `{"id":%d,"result":%s}`+"\n", *id, result)
	}
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var request struct {
			ID     *int           `json:"id"`
			Method string         `json:"method"`
			Params map[string]any `json:"params"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &request); err != nil {
			os.Exit(2)
		}
		switch request.Method {
		case "initialize":
			respond(request.ID, `{}`)
		case "initialized":
			// Notifications have no response.
		case "mcpServerStatus/list":
			if request.Params["cursor"] == nil {
				respond(request.ID, `{"data":[{"name":"repl"}],"nextCursor":"page-2"}`)
			} else {
				respond(request.ID, `{"data":[{"name":"ide"}],"nextCursor":null}`)
			}
		case "thread/start":
			if request.Params["model"] != "example-model" ||
				request.Params["developerInstructions"] != "System prompt" ||
				request.Params["sandbox"] != "read-only" ||
				request.Params["approvalPolicy"] != "never" ||
				request.Params["ephemeral"] != true {
				os.Exit(2)
			}
			// Every MCP server must be disabled for the thread.
			if !mcpServersDisabled(request.Params["config"], "repl", "ide") {
				os.Exit(2)
			}
			// The thread must run in the app-server's (empty, temporary) working directory.
			cwd, _ := request.Params["cwd"].(string)
			if !isWorkingDir(cwd) {
				os.Exit(2)
			}
			fmt.Fprintln(os.Stdout, `{"method":"thread/started","params":{}}`)
			respond(request.ID, `{"thread":{"id":"thread-1"}}`)
		case "turn/start":
			if request.Params["threadId"] != "thread-1" || request.Params["effort"] != "low" {
				os.Exit(2)
			}
			input, ok := request.Params["input"].([]any)
			if !ok || len(input) != 1 {
				os.Exit(2)
			}
			item, ok := input[0].(map[string]any)
			if !ok || item["type"] != "text" || item["text"] != "User prompt" {
				os.Exit(2)
			}
			// An item can arrive before the turn/start response.
			if mode != "empty" {
				fmt.Fprintln(os.Stdout, `{"method":"item/completed","params":{"item":{"type":"agentMessage","phase":"final_answer","text":"完成文"}}}`)
			}
			respond(request.ID, `{"turn":{"id":"turn-1"}}`)
			if mode == "failed" {
				fmt.Fprintln(os.Stdout, `{"method":"turn/completed","params":{"turn":{"status":"failed","error":{"message":"model unavailable"}}}}`)
			} else {
				fmt.Fprintln(os.Stdout, `{"method":"turn/completed","params":{"turn":{"status":"completed"}}}`)
			}
			os.Exit(0)
		default:
			os.Exit(2)
		}
	}
	os.Exit(2)
}

// isWorkingDir reports whether dir is the process's working directory and is empty.
func isWorkingDir(dir string) bool {
	wd, err := os.Getwd()
	if err != nil || dir == "" {
		return false
	}
	a, errA := os.Stat(dir)
	b, errB := os.Stat(wd)
	if errA != nil || errB != nil || !os.SameFile(a, b) {
		return false
	}
	entries, err := os.ReadDir(dir)
	return err == nil && len(entries) == 0
}

// mcpServersDisabled reports whether the thread config disables all of the given MCP servers.
func mcpServersDisabled(config any, names ...string) bool {
	cfg, _ := config.(map[string]any)
	servers, _ := cfg["mcp_servers"].(map[string]any)
	for _, name := range names {
		server, _ := servers[name].(map[string]any)
		if server["enabled"] != false {
			return false
		}
	}
	return true
}
