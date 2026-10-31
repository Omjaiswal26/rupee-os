package classification

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ModelStatus reports whether configured Ollama models are pulled locally.
type ModelStatus struct {
	OK              bool   `json:"ok"`
	Reachable       bool   `json:"reachable"`
	ClassifyModel   string `json:"classify_model"`
	ClassifyPresent bool   `json:"classify_present"`
	HealthModel     string `json:"health_model"`
	HealthPresent   bool   `json:"health_present"`
	Hint            string `json:"hint,omitempty"`
	Error           string `json:"error,omitempty"`
}

type ollamaTagsResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

// CheckOllamaModels calls GET /api/tags and checks required model names.
func CheckOllamaModels(ctx context.Context, baseURL, classifyModel, healthModel string) ModelStatus {
	baseURL = strings.TrimRight(baseURL, "/")
	out := ModelStatus{
		ClassifyModel: classifyModel,
		HealthModel:   healthModel,
		Hint:          "run ./scripts/pull-ollama-model.sh",
	}
	if baseURL == "" {
		out.Error = "OLLAMA_BASE_URL is empty"
		return out
	}

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/tags", nil)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	resp, err := client.Do(req)
	if err != nil {
		out.Error = fmt.Sprintf("ollama unreachable: %v", err)
		return out
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		out.Error = fmt.Sprintf("ollama tags status %d: %s", resp.StatusCode, truncate(string(raw), 120))
		return out
	}

	var tags ollamaTagsResponse
	if err := json.Unmarshal(raw, &tags); err != nil {
		out.Error = "decode /api/tags: " + err.Error()
		return out
	}

	out.Reachable = true
	names := make([]string, 0, len(tags.Models))
	for _, m := range tags.Models {
		names = append(names, m.Name)
	}
	out.ClassifyPresent = modelPresent(names, classifyModel)
	if healthModel == "" || healthModel == classifyModel {
		out.HealthPresent = out.ClassifyPresent
		out.HealthModel = classifyModel
	} else {
		out.HealthPresent = modelPresent(names, healthModel)
	}

	out.OK = out.ClassifyPresent
	if !out.ClassifyPresent {
		out.Hint = fmt.Sprintf("Classify model %q not pulled — run ./scripts/pull-ollama-model.sh", classifyModel)
	} else if !out.HealthPresent {
		out.Hint = fmt.Sprintf("Health model %q not pulled — month health needs it. Run ./scripts/pull-ollama-model.sh", healthModel)
		// Imports still work; health generate will fail until pulled.
		out.OK = true
	} else {
		out.Hint = ""
	}
	return out
}

func modelPresent(names []string, want string) bool {
	want = strings.TrimSpace(want)
	if want == "" {
		return false
	}
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n == want || strings.HasPrefix(n, want+":") {
			return true
		}
	}
	return false
}
