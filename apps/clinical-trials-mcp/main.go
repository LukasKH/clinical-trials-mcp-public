package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/lukas/clinical-trials-mcp/apps/clinical-trials-mcp/clinicaltrials"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)

	host := envString("MCP_HOST", "127.0.0.1")
	port := envString("PORT", envString("MCP_PORT", "8001"))
	path := envString("MCP_PATH", "/mcp")

	client := clinicaltrials.NewClient(
		envString("CT_API_BASE_URL", "https://clinicaltrials.gov/api/v2"),
		envString("EU_CT_API_BASE_URL", "https://euclinicaltrials.eu/ctis-public-api"),
		envDurationSeconds("CT_REQUEST_TIMEOUT_SECONDS", 30*time.Second),
		envInt("CT_MAX_PAGE_SIZE", 25),
	)
	server := newMCPServer(client)

	handler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		if req.URL.Path != path {
			return nil
		}
		return server
	}, &mcp.StreamableHTTPOptions{Stateless: true})

	mux := http.NewServeMux()
	mux.Handle(path, handler)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	addr := fmt.Sprintf("%s:%s", host, port)
	log.Printf("Starting ClinicalTrials.gov MCP server on %s%s", addr, path)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("MCP server failed: %v", err)
	}
}

func newMCPServer(client *clinicaltrials.Client) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "clinicaltrials-gov-mcp",
		Version: "0.1.0",
	}, &mcp.ServerOptions{
		Instructions: "Expose controlled Trial Tools backed by ClinicalTrials.gov and EU Clinical Trials. Keep the search interface simple: query is required, shared filters are optional, and region is optional. region=US searches ClinicalTrials.gov, region=EU searches EU Clinical Trials, and omitted region or region=ALL searches both registries. Registry-specific API mappings stay inside the MCP server.",
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_trials",
		Description: "Search public clinical trial registry records from ClinicalTrials.gov and EU Clinical Trials. Requires query. Optional shared filters are condition, intervention, sponsor, title, outcome, country, and location. Optional region can be ALL, US, or EU: US searches ClinicalTrials.gov, EU searches EU Clinical Trials, and ALL or omitted region searches both registries. Natural-language location values such as Europe, European Union, EU, United States, U.S., USA, and America are normalized to EU or US when region is omitted; other location values are treated as country filters. Use US or EU only when the user asks for that registry region. Returns compact Trial Search Results labelled by source registry with identifiers, titles, statuses, conditions, phase or study type, summaries, source URLs, and pagination details.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params clinicaltrials.SearchTrialsParams) (*mcp.CallToolResult, any, error) {
		log.Printf("MCP tool call: search_trials query=%q region=%q page_size=%d", params.Query, params.Region, params.PageSize)
		result, err := client.SearchTrials(ctx, params)
		if err != nil {
			return nil, nil, err
		}
		return textResult(result), nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_study",
		Description: "Retrieve one public study record by ClinicalTrials.gov NCT ID or EU Clinical Trials CT number. Provide nct_id, eu_ct_number, or study_id. Returns a curated markdown Study Document labelled with source metadata, not raw registry JSON.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params clinicaltrials.GetStudyParams) (*mcp.CallToolResult, any, error) {
		log.Printf("MCP tool call: get_study nct_id=%q eu_ct_number=%q study_id=%q", params.NCTID, params.EUCTNumber, params.StudyID)
		result, err := client.GetStudy(ctx, params)
		if err != nil {
			return nil, nil, err
		}
		return textResult(result), nil, nil
	})

	return server
}

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}
}

func envString(name string, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func envInt(name string, fallback int) int {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		log.Printf("Invalid %s=%q; using %d", name, value, fallback)
		return fallback
	}
	return parsed
}

func envDurationSeconds(name string, fallback time.Duration) time.Duration {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		log.Printf("Invalid %s=%q; using %s", name, value, fallback)
		return fallback
	}
	return time.Duration(parsed * float64(time.Second))
}
