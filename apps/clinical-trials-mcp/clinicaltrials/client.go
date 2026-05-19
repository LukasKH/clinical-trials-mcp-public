package clinicaltrials

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const sourceBaseURL = "https://clinicaltrials.gov/study"
const euSourceBaseURL = "https://euclinicaltrials.eu/ctis-public/view"

var nctIDPattern = regexp.MustCompile(`^NCT[0-9]{8}$`)
var euCTNumberPattern = regexp.MustCompile(`^[0-9]{4}-[0-9]{6}-[0-9]{2}-[0-9]{2}$`)

type Client struct {
	baseURL     string
	euBaseURL   string
	httpClient  *http.Client
	maxPageSize int
}

type SearchTrialsParams struct {
	Query        string `json:"query" jsonschema:"Required. Broad keyword search across clinical trial registry records, such as a disease, treatment, sponsor, product, or phrase. This is the only mandatory search parameter."`
	Condition    string `json:"condition,omitempty" jsonschema:"Optional shared filter for the condition or disease being studied, such as diabetes, breast cancer, or COVID-19. Use only when the user clearly asks to narrow by condition."`
	Intervention string `json:"intervention,omitempty" jsonschema:"Optional shared filter for the intervention, investigational product, or treatment, such as metformin, chemotherapy, remdesivir, surgery, or behavioral therapy. Use only when the user clearly asks to narrow by treatment or product."`
	Sponsor      string `json:"sponsor,omitempty" jsonschema:"Optional shared filter for the trial sponsor or collaborator, such as NIH, National Cancer Institute, or a pharmaceutical company name. Use only when the user names a sponsor."`
	Title        string `json:"title,omitempty" jsonschema:"Optional shared filter for words expected in the official, public, or brief study title. Use only when the user asks for a title-specific search."`
	Outcome      string `json:"outcome,omitempty" jsonschema:"Optional shared filter for an outcome measure or endpoint, such as HbA1c, overall survival, or viral load. Use only when the user asks for studies measuring a specific endpoint or result."`
	Country      string `json:"country,omitempty" jsonschema:"Optional shared filter for trial country, such as United States, Denmark, Germany, or Japan. Use country names, not country codes."`
	Location     string `json:"location,omitempty" jsonschema:"Optional natural-language trial location. Europe, European Union, EU, United States, U.S., USA, and America are treated as registry region selectors; other values are treated as country filters when country is omitted."`
	Region       string `json:"region,omitempty" jsonschema:"Optional registry region selector. Valid values are ALL, US, and EU. Defaults to ALL. Use US for US-only searches, EU for EU or European Union-only searches, and omit when no region is specified."`
	PageToken    string `json:"page_token,omitempty" jsonschema:"Optional pagination token copied exactly from a previous search_trials response. Omit for the first page."`
	PageSize     int    `json:"page_size,omitempty" jsonschema:"Optional number of search results to return from each searched registry. Defaults to 5 and is capped by the server."`
}

type GetStudyParams struct {
	NCTID      string `json:"nct_id,omitempty" jsonschema:"Optional ClinicalTrials.gov NCT identifier for one study. Must be NCT followed by 8 digits, such as NCT04280705. Provide this, eu_ct_number, or study_id."`
	EUCTNumber string `json:"eu_ct_number,omitempty" jsonschema:"Optional EU Clinical Trials CT number for one study. Must look like 2025-523486-17-00. Provide this, nct_id, or study_id."`
	StudyID    string `json:"study_id,omitempty" jsonschema:"Optional generic study identifier. Can be either a ClinicalTrials.gov NCT ID or an EU Clinical Trials CT number. Use this when the user provides one identifier without naming the registry."`
}

type SearchTrialsOutput struct {
	Markdown string `json:"markdown" jsonschema:"Compact markdown Trial Search Results labelled by source registry with identifiers, titles, statuses, conditions, phase or study type, summaries, source URLs, and pagination details."`
}

type GetStudyOutput struct {
	Markdown string `json:"markdown" jsonschema:"Curated markdown Study Document labelled with source metadata, not raw registry JSON."`
}

func NewClient(baseURL string, euBaseURL string, requestTimeout time.Duration, maxPageSize int) *Client {
	return &Client{
		baseURL:   strings.TrimRight(baseURL, "/"),
		euBaseURL: strings.TrimRight(euBaseURL, "/"),
		httpClient: &http.Client{
			Timeout: requestTimeout,
		},
		maxPageSize: maxPageSize,
	}
}

func object(values map[string]any, key string) map[string]any {
	if values == nil {
		return map[string]any{}
	}
	if objectValue, ok := values[key].(map[string]any); ok {
		return objectValue
	}
	return map[string]any{}
}

func firstPresent(values ...any) any {
	for _, value := range values {
		if text(value) != "" {
			return value
		}
	}
	return nil
}

func text(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return strings.Join(strings.Fields(typed), " ")
	case float64:
		if typed == float64(int64(typed)) {
			return fmt.Sprintf("%d", int64(typed))
		}
		return fmt.Sprintf("%v", typed)
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func listText(value any) string {
	values, ok := value.([]any)
	if !ok || len(values) == 0 {
		return text(value)
	}
	names := make([]string, 0, len(values))
	for _, item := range values {
		if objectItem, ok := item.(map[string]any); ok {
			item = objectItem["name"]
		}
		if itemText := text(item); itemText != "" {
			names = append(names, itemText)
		}
	}
	return strings.Join(names, ", ")
}

func fallback(value string, fallbackValue string) string {
	if value == "" {
		return fallbackValue
	}
	return value
}

func joinAvailable(values ...string) string {
	available := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			available = append(available, value)
		}
	}
	if len(available) == 0 {
		return "Not provided"
	}
	return strings.Join(available, " / ")
}
