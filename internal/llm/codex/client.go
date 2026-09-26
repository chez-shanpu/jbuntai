// Package codex implements LLM backends through the Codex app-server protocol.
package codex

import (
	"bufio"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	codexCommand = "codex"

	// runTimeout bounds a whole run: process start, handshake, and the turn.
	runTimeout = 120 * time.Second

	// The app-server emits one JSON-RPC message per line; agent messages can be large.
	scannerInitialBufSize = 4 * 1024
	scannerMaxLineSize    = 10 * 1024 * 1024

	clientName    = "jbuntai"
	clientVersion = "0.1.0" // Required by the protocol; informational only.
)

// JSON-RPC methods of the Codex app-server protocol.
const (
	methodInitialize    = "initialize"
	methodInitialized   = "initialized"
	methodMCPServerList = "mcpServerStatus/list"
	methodThreadStart   = "thread/start"
	methodTurnStart     = "turn/start"
	methodItemCompleted = "item/completed"
	methodTurnCompleted = "turn/completed"
)

// Values of the Codex app-server protocol.
const (
	sandboxReadOnly      = "read-only"
	approvalPolicyNever  = "never"
	itemTypeAgentMessage = "agentMessage"
	phaseFinalAnswer     = "final_answer"
	turnStatusCompleted  = "completed"
	inputTypeText        = "text"
	mcpDetailToolsOnly   = "toolsAndAuthOnly"
)

// disabledFeatures are Codex features turned off so that the agent can only
// answer from the prompt. The input text is untrusted: instructions embedded
// in it must not be able to run commands, read local files, or reach
// external services. They are passed as `-c features.<name>=false`, which
// Codex versions that do not know a name ignore (whereas `--disable <name>`
// would make the app-server exit).
var disabledFeatures = []string{
	"shell_tool",
	"unified_exec",
	"view_image",
	"apps",
	"plugins",
	"browser_use",
	"computer_use",
	"in_app_browser",
	"memories",
	"image_generation",
	"hooks",
}

// configOverrides are `-c` overrides that complement disabledFeatures.
// MCP servers cannot be removed this way; run disables them per thread.
var configOverrides = []string{
	`web_search="disabled"`,
	`shell_environment_policy.inherit="none"`,
}

// client launches a fresh `codex app-server` process for each run.
type client struct {
	command string
	// args are prepended to the app-server arguments; tests use them to
	// re-exec the test binary as a fake app-server.
	args []string
}

func newClient() *client { return &client{command: codexCommand} }

// appServerArgs returns the arguments that start a locked-down app-server.
func (c *client) appServerArgs() []string {
	args := append([]string{}, c.args...)
	args = append(args, "app-server", "--stdio")
	for _, f := range disabledFeatures {
		args = append(args, "-c", "features."+f+"=false")
	}
	for _, o := range configOverrides {
		args = append(args, "-c", o)
	}
	return args
}

// run starts an isolated thread and returns its final agent message.
func (c *client) run(ctx context.Context, model, reasoningEffort, systemPrompt, userPrompt string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, runTimeout)
	defer cancel()

	conn, workDir, stop, err := c.startAppServer(ctx)
	if err != nil {
		return "", err
	}
	defer stop()

	if err := conn.call(methodInitialize, initializeParams{
		ClientInfo: clientInfo{Name: clientName, Title: clientName, Version: clientVersion},
	}, nil); err != nil {
		return "", err
	}
	if err := conn.notify(methodInitialized, struct{}{}); err != nil {
		return "", err
	}

	mcpServers, err := conn.listMCPServers()
	if err != nil {
		return "", err
	}

	var thread threadStartResult
	if err := conn.call(methodThreadStart, threadStartParams{
		Model:                 model,
		DeveloperInstructions: systemPrompt,
		Cwd:                   workDir,
		Sandbox:               sandboxReadOnly,
		ApprovalPolicy:        approvalPolicyNever,
		Ephemeral:             true,
		Config:                disableMCPServersConfig(mcpServers),
	}, &thread); err != nil {
		return "", err
	}
	if thread.Thread.ID == "" {
		return "", errors.New("thread/start response has no thread id")
	}

	if err := conn.call(methodTurnStart, turnStartParams{
		ThreadID: thread.Thread.ID,
		Input:    []inputItem{{Type: inputTypeText, Text: userPrompt}},
		Effort:   reasoningEffort,
	}, nil); err != nil {
		return "", err
	}

	return conn.awaitFinalAnswer(ctx)
}

// startAppServer starts the app-server in an empty temporary directory, so
// that no project files (AGENTS.md, .codex/, sources) are in its reach. The
// returned stop function terminates the process and removes the directory.
func (c *client) startAppServer(ctx context.Context) (*appServer, string, func(), error) {
	workDir, err := os.MkdirTemp("", "jbuntai-codex-")
	if err != nil {
		return nil, "", nil, fmt.Errorf("create app-server work dir: %w", err)
	}
	removeWorkDir := func() { _ = os.RemoveAll(workDir) }

	ctx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(ctx, c.command, c.appServerArgs()...)
	cmd.Dir = workDir
	cmd.Stderr = io.Discard
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		removeWorkDir()
		return nil, "", nil, fmt.Errorf("open app-server stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		removeWorkDir()
		return nil, "", nil, fmt.Errorf("open app-server stdout: %w", err)
	}
	if err := cmd.Start(); err != nil {
		cancel()
		removeWorkDir()
		return nil, "", nil, fmt.Errorf("start codex app-server: %w", err)
	}

	stop := func() {
		// Kill the process before Wait: stdout may not be drained on an
		// early return, and Wait would otherwise block on the pipe.
		cancel()
		_ = stdin.Close()
		_ = cmd.Wait()
		removeWorkDir()
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, scannerInitialBufSize), scannerMaxLineSize)
	return &appServer{writer: stdin, scanner: scanner}, workDir, stop, nil
}

// listMCPServers returns the names of all MCP servers configured for Codex.
func (s *appServer) listMCPServers() ([]string, error) {
	var names []string
	var cursor *string
	for {
		var page mcpServerListResult
		if err := s.call(methodMCPServerList, mcpServerListParams{Cursor: cursor, Detail: mcpDetailToolsOnly}, &page); err != nil {
			return nil, err
		}
		for _, server := range page.Data {
			names = append(names, server.Name)
		}
		if page.NextCursor == nil || *page.NextCursor == "" {
			return names, nil
		}
		cursor = page.NextCursor
	}
}

// disableMCPServersConfig returns a thread config that disables the given MCP
// servers. MCP tools (e.g. a Node.js REPL) would otherwise let instructions
// embedded in the input run code outside the read-only sandbox.
func disableMCPServersConfig(names []string) map[string]any {
	if len(names) == 0 {
		return nil
	}
	servers := make(map[string]any, len(names))
	for _, name := range names {
		servers[name] = map[string]bool{"enabled": false}
	}
	return map[string]any{"mcp_servers": servers}
}

// awaitFinalAnswer consumes events until the turn completes and returns the
// final agent message.
func (s *appServer) awaitFinalAnswer(ctx context.Context) (string, error) {
	var answer string
	for {
		message, err := s.nextEvent()
		if err != nil {
			if ctx.Err() != nil {
				return "", ctx.Err()
			}
			return "", fmt.Errorf("read app-server event: %w", err)
		}
		if message.ID != nil && message.Method != "" {
			return "", fmt.Errorf("unexpected app-server request: %s", message.Method)
		}
		switch message.Method {
		case methodItemCompleted:
			text, ok, err := finalAnswerText(message.Params)
			if err != nil {
				return "", err
			}
			if ok {
				answer = text
			}
		case methodTurnCompleted:
			if err := checkTurnCompleted(message.Params); err != nil {
				return "", err
			}
			answer = strings.TrimSpace(answer)
			if answer == "" {
				return "", errors.New("codex turn completed without an answer")
			}
			return answer, nil
		}
	}
}

// finalAnswerText extracts the text of an item/completed event if the item is
// the agent's final answer.
func finalAnswerText(params jsontext.Value) (string, bool, error) {
	var event itemCompletedEvent
	if err := json.Unmarshal(params, &event); err != nil {
		return "", false, fmt.Errorf("decode agent message: %w", err)
	}
	slog.Default().Debug("codex item completed", "type", event.Item.Type, "phase", event.Item.Phase)
	if event.Item.Type != itemTypeAgentMessage {
		return "", false, nil
	}
	if event.Item.Phase != phaseFinalAnswer && event.Item.Phase != "" {
		return "", false, nil
	}
	return event.Item.Text, true, nil
}

// checkTurnCompleted returns an error unless the turn/completed event reports success.
func checkTurnCompleted(params jsontext.Value) error {
	var event turnCompletedEvent
	if err := json.Unmarshal(params, &event); err != nil {
		return fmt.Errorf("decode completed turn: %w", err)
	}
	if event.Turn.Status == turnStatusCompleted {
		return nil
	}
	if event.Turn.Error != nil {
		return fmt.Errorf("codex turn %s: %s", event.Turn.Status, event.Turn.Error.Message)
	}
	return fmt.Errorf("codex turn %s", event.Turn.Status)
}

// appServer speaks newline-delimited JSON-RPC over the app-server's stdio.
// Notifications that arrive while waiting for a response are queued in
// pending and returned by nextEvent before any further reads.
type appServer struct {
	writer  io.Writer
	scanner *bufio.Scanner
	nextID  int
	pending []rpcMessage
}

// call sends a request and waits for its response. If result is non-nil,
// the response result is decoded into it.
func (s *appServer) call(method string, params, result any) error {
	id := s.nextID
	s.nextID++
	if err := s.write(rpcRequest{ID: &id, Method: method, Params: params}); err != nil {
		return err
	}
	raw, err := s.response(id)
	if err != nil {
		return fmt.Errorf("%s: %w", method, err)
	}
	if result == nil {
		return nil
	}
	if err := json.Unmarshal(raw, result); err != nil {
		return fmt.Errorf("decode %s response: %w", method, err)
	}
	return nil
}

// notify sends a notification, which has no response.
func (s *appServer) notify(method string, params any) error {
	return s.write(rpcRequest{Method: method, Params: params})
}

func (s *appServer) write(message rpcRequest) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("encode app-server request: %w", err)
	}
	data = append(data, '\n')
	if _, err := s.writer.Write(data); err != nil {
		return fmt.Errorf("write app-server request: %w", err)
	}
	return nil
}

// nextEvent returns the next queued notification, or reads a new message.
func (s *appServer) nextEvent() (rpcMessage, error) {
	if len(s.pending) > 0 {
		message := s.pending[0]
		s.pending = s.pending[1:]
		return message, nil
	}
	return s.readLine()
}

func (s *appServer) readLine() (rpcMessage, error) {
	if !s.scanner.Scan() {
		if err := s.scanner.Err(); err != nil {
			return rpcMessage{}, err
		}
		return rpcMessage{}, io.EOF
	}
	var message rpcMessage
	if err := json.Unmarshal(s.scanner.Bytes(), &message); err != nil {
		return rpcMessage{}, fmt.Errorf("decode app-server message: %w", err)
	}
	return message, nil
}

// response reads messages until the response with the given id arrives,
// queueing notifications received in the meantime.
func (s *appServer) response(id int) (jsontext.Value, error) {
	for {
		message, err := s.readLine()
		if err != nil {
			return nil, fmt.Errorf("read app-server response: %w", err)
		}
		if message.ID == nil {
			s.pending = append(s.pending, message)
			continue
		}
		if message.Method != "" {
			return nil, fmt.Errorf("unexpected app-server request: %s", message.Method)
		}
		if *message.ID != id {
			return nil, fmt.Errorf("unexpected app-server response id: %d", *message.ID)
		}
		if message.Error != nil {
			return nil, fmt.Errorf("app-server %d: %s", message.Error.Code, message.Error.Message)
		}
		return message.Result, nil
	}
}

type rpcRequest struct {
	ID     *int   `json:"id,omitempty"`
	Method string `json:"method"`
	Params any    `json:"params"`
}

type rpcMessage struct {
	ID     *int           `json:"id,omitempty"`
	Method string         `json:"method,omitempty"`
	Params jsontext.Value `json:"params,omitempty"`
	Result jsontext.Value `json:"result,omitempty"`
	Error  *rpcError      `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type clientInfo struct {
	Name    string `json:"name"`
	Title   string `json:"title"`
	Version string `json:"version"`
}

type initializeParams struct {
	ClientInfo clientInfo `json:"clientInfo"`
}

type threadStartParams struct {
	Model                 string         `json:"model"`
	DeveloperInstructions string         `json:"developerInstructions"`
	Cwd                   string         `json:"cwd"`
	Sandbox               string         `json:"sandbox"`
	ApprovalPolicy        string         `json:"approvalPolicy"`
	Ephemeral             bool           `json:"ephemeral"`
	Config                map[string]any `json:"config,omitempty"`
}

type mcpServerListParams struct {
	Cursor *string `json:"cursor,omitempty"`
	Detail string  `json:"detail"`
}

type mcpServerListResult struct {
	Data []struct {
		Name string `json:"name"`
	} `json:"data"`
	NextCursor *string `json:"nextCursor"`
}

type threadStartResult struct {
	Thread struct {
		ID string `json:"id"`
	} `json:"thread"`
}

type inputItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type turnStartParams struct {
	ThreadID string      `json:"threadId"`
	Input    []inputItem `json:"input"`
	Effort   string      `json:"effort,omitempty"`
}

type itemCompletedEvent struct {
	Item struct {
		Type  string `json:"type"`
		Text  string `json:"text"`
		Phase string `json:"phase"`
	} `json:"item"`
}

type turnCompletedEvent struct {
	Turn struct {
		Status string `json:"status"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	} `json:"turn"`
}
