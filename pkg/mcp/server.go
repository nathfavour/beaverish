package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/nathfavour/beaverish/pkg/logger"
)

type Server struct {
	reader  *bufio.Reader
	writer  io.Writer
	handler *Handler
}

func NewServer(in io.Reader, out io.Writer, handler *Handler) *Server {
	if in == nil {
		in = os.Stdin
	}
	if out == nil {
		out = os.Stdout
	}
	return &Server{
		reader:  bufio.NewReader(in),
		writer:  out,
		handler: handler,
	}
}

func (s *Server) Start(ctx context.Context) error {
	logger.Infof("MCP JSON-RPC 2.0 Server initialized on stdio")

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		line, err := s.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		if len(line) == 0 || line == "\n" || line == "\r\n" {
			continue
		}

		var req Request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			s.sendError(nil, -32700, "Parse error", err.Error())
			continue
		}

		s.handleRequest(ctx, req)
	}
}

func (s *Server) handleRequest(ctx context.Context, req Request) {
	switch req.Method {
	case "initialize":
		s.sendResult(req.ID, map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"serverInfo": map[string]string{
				"name":    "beaverish-mcp",
				"version": "1.0.0",
			},
			"capabilities": map[string]interface{}{
				"tools": map[string]bool{
					"listChanged": true,
				},
			},
		})

	case "notifications/initialized":
		// No response required for notifications
		return

	case "tools/list":
		tools := s.handler.GetTools()
		s.sendResult(req.ID, map[string]interface{}{
			"tools": tools,
		})

	case "tools/call":
		var params ToolCallParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.sendError(req.ID, -32602, "Invalid params", err.Error())
			return
		}

		res, err := s.handler.CallTool(ctx, params.Name, params.Arguments)
		if err != nil {
			s.sendResult(req.ID, res)
			return
		}
		s.sendResult(req.ID, res)

	case "ping":
		s.sendResult(req.ID, map[string]string{"status": "pong"})

	default:
		s.sendError(req.ID, -32601, "Method not found", fmt.Sprintf("unsupported method: %s", req.Method))
	}
}

func (s *Server) sendResult(id interface{}, result interface{}) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	bytes, _ := json.Marshal(resp)
	fmt.Fprintf(s.writer, "%s\n", string(bytes))
}

func (s *Server) sendError(id interface{}, code int, message string, data interface{}) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	bytes, _ := json.Marshal(resp)
	fmt.Fprintf(s.writer, "%s\n", string(bytes))
}
