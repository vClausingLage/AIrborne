package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"airborne/internal/logging"
)

type Config struct {
	Base      string
	Key       string
	Model     string
	MaxTokens int
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Message      struct {
			Content          string `json:"content"`
			Reasoning        string `json:"reasoning"`
			ReasoningContent string `json:"reasoning_content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func LoadEnv(candidates ...string) error {
	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			k, v, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			k = strings.TrimSpace(k)
			v = strings.TrimSpace(v)
			if len(v) >= 2 && (v[0] == '"' && v[len(v)-1] == '"' || v[0] == '\'' && v[len(v)-1] == '\'') {
				v = v[1 : len(v)-1]
			} else if idx := strings.Index(v, "#"); idx >= 0 && (idx == 0 || v[idx-1] == ' ' || v[idx-1] == '\t') {
				v = strings.TrimSpace(v[:idx])
			}
			if _, exists := os.LookupEnv(k); !exists {
				os.Setenv(k, v)
			}
		}
	}
	return nil
}

func LoadConfig() Config {
	base := os.Getenv("AI_API_BASE")
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	maxTokens := 30000
	if v := os.Getenv("AI_MAX_TOKENS"); v != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			maxTokens = n
		}
	}
	return Config{
		Base:      strings.TrimRight(base, "/"),
		Key:       os.Getenv("AI_API_KEY"),
		Model:     os.Getenv("AI_MODEL"),
		MaxTokens: maxTokens,
	}
}

func (c Config) Chat(ctx context.Context, messages []Message) (string, error) {
	if c.Model == "" {
		return "", fmt.Errorf("AI_MODEL ist nicht gesetzt (.env)")
	}
	model := c.Model
	body, err := json.Marshal(chatRequest{Model: model, Messages: messages, Temperature: 0.7, MaxTokens: c.MaxTokens})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Base+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Key != "" {
		req.Header.Set("Authorization", "Bearer "+c.Key)
	}
	client := &http.Client{Timeout: 300 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logging.Errorf("LLM-Verbindung fehlgeschlagen (%s, model=%s): %v", c.Base, model, err)
		return "", fmt.Errorf("LLM-Anfrage fehlgeschlagen: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	logging.Infof("LLM-HTTP %d, %d bytes", resp.StatusCode, len(data))
	var cr chatResponse
	if err := json.Unmarshal(data, &cr); err != nil {
		logging.Errorf("LLM-Antwort nicht parsebar (HTTP %d): %s", resp.StatusCode, truncate(string(data), 300))
		return "", fmt.Errorf("ungültige LLM-Antwort (HTTP %d): %s", resp.StatusCode, truncate(string(data), 300))
	}
	if cr.Error != nil {
		logging.Errorf("LLM-Fehler: %s", cr.Error.Message)
		return "", fmt.Errorf("LLM-Fehler: %s", cr.Error.Message)
	}
	if resp.StatusCode != http.StatusOK {
		logging.Errorf("LLM-HTTP %d: %s", resp.StatusCode, truncate(string(data), 300))
		return "", fmt.Errorf("LLM-HTTP %d: %s", resp.StatusCode, truncate(string(data), 300))
	}
	if len(cr.Choices) == 0 {
		return "", fmt.Errorf("LLM-Antwort ohne choices")
	}
	content := cr.Choices[0].Message.Content
	if strings.TrimSpace(content) == "" {
		if cr.Choices[0].Message.Reasoning != "" {
			logging.Infof("content leer, nutze reasoning-Feld (%d bytes)", len(cr.Choices[0].Message.Reasoning))
			content = cr.Choices[0].Message.Reasoning
		} else if cr.Choices[0].Message.ReasoningContent != "" {
			logging.Infof("content leer, nutze reasoning_content-Feld (%d bytes)", len(cr.Choices[0].Message.ReasoningContent))
			content = cr.Choices[0].Message.ReasoningContent
		}
	}
	if strings.TrimSpace(content) == "" {
		logging.Block("LLM RAW BODY (content leer)", truncate(string(data), 8000))
		return "", fmt.Errorf("LLM lieferte leeren content (finish=%s) - ggf. AI_MAX_TOKENS erhoehen (Details im Log)", cr.Choices[0].FinishReason)
	}
	return content, nil
}

func FindEnvFile(startDirs ...string) []string {
	var out []string
	for _, dir := range startDirs {
		if dir == "" {
			continue
		}
		out = append(out, filepath.Join(dir, ".env"))
	}
	return out
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}

func ExtractJSON(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "```"); i >= 0 {
		if j := strings.Index(s[i+3:], "```"); j >= 0 {
			inner := s[i+3 : i+3+j]
			inner = strings.TrimPrefix(inner, "json")
			s = strings.TrimSpace(inner)
		}
	}
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start < 0 || end <= start {
		return nil, fmt.Errorf("kein JSON-Objekt in der Antwort gefunden")
	}
	return []byte(s[start : end+1]), nil
}
