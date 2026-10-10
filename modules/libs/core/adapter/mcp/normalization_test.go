package mcp_test

import (
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestToolArgumentsAreNormalizedAcrossCasingsAndSeparators(t *testing.T) {
	session, _ := newSession(t, map[string]string{
		"Entropy.md": "# Entropy\nEntropy always increases in an isolated system.\n",
	})

	t.Run("PascalCase parameter (Query, Limit)", func(t *testing.T) {
		res, err := session.CallTool(t.Context(), &sdk.CallToolParams{
			Name: "note_search",
			Arguments: map[string]any{
				"Query": "Entropy",
				"Limit": 5,
			},
		})
		if err != nil {
			t.Fatalf("unexpected protocol error: %v", err)
		}
		if res.IsError {
			t.Fatalf("tool error: %v", text(res))
		}
	})

	t.Run("SCREAMING_CASE parameter (QUERY)", func(t *testing.T) {
		res, err := session.CallTool(t.Context(), &sdk.CallToolParams{
			Name: "note_search",
			Arguments: map[string]any{
				"QUERY": "Entropy",
			},
		})
		if err != nil {
			t.Fatalf("unexpected protocol error: %v", err)
		}
		if res.IsError {
			t.Fatalf("tool error: %v", text(res))
		}
	})

	t.Run("PascalCase slice argument (Paths)", func(t *testing.T) {
		res, err := session.CallTool(t.Context(), &sdk.CallToolParams{
			Name: "note_read",
			Arguments: map[string]any{
				"Paths": []string{"Entropy.md"},
			},
		})
		if err != nil {
			t.Fatalf("unexpected protocol error: %v", err)
		}
		if res.IsError {
			t.Fatalf("tool error: %v", text(res))
		}
	})

	t.Run("Permissive extra benign parameters", func(t *testing.T) {
		res, err := session.CallTool(t.Context(), &sdk.CallToolParams{
			Name: "note_search",
			Arguments: map[string]any{
				"query":         "Entropy",
				"_context":      "some LLM context",
				"extra_ignored": 12345,
			},
		})
		if err != nil {
			t.Fatalf("unexpected protocol error: %v", err)
		}
		if res.IsError {
			t.Fatalf("tool error: %v", text(res))
		}
	})

	t.Run("Name lookup with uppercase (NAME)", func(t *testing.T) {
		res, err := session.CallTool(t.Context(), &sdk.CallToolParams{
			Name: "note_resolve",
			Arguments: map[string]any{
				"NAME": "Entropy",
			},
		})
		if err != nil {
			t.Fatalf("unexpected protocol error: %v", err)
		}
		if res.IsError {
			t.Fatalf("tool error: %v", text(res))
		}
	})
}
