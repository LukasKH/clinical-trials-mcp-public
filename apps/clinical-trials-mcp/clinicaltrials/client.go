package clinicaltrials

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
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

func (c *Client) SearchTrials(ctx context.Context, params SearchTrialsParams) (string, error) {
	region, country, err := normalizeSearchScope(params)
	if err != nil {
		return "", err
	}
	params.Country = country

	searchCT := region != "EU"
	searchEU := region != "US"
	var ctData map[string]any
	var ctValues url.Values
	var ctErr error
	var euData map[string]any
	var euErr error
	if searchCT {
		ctData, ctValues, ctErr = c.searchClinicalTrialsGov(ctx, params, region)
	}
	if searchEU {
		euData, euErr = c.searchEUClinicalTrials(ctx, params)
	}
	if searchCT && searchEU && ctErr != nil && euErr != nil {
		return "", fmt.Errorf("search ClinicalTrials.gov: %v; search EU Clinical Trials: %w", ctErr, euErr)
	}
	if searchCT && !searchEU && ctErr != nil {
		return "", ctErr
	}
	if searchEU && !searchCT && euErr != nil {
		return "", euErr
	}

	return renderCombinedTrialSearchResults(region, ctData, ctValues, ctErr, euData, euErr), nil
}

func (c *Client) GetStudy(ctx context.Context, params GetStudyParams) (string, error) {
	studyID := strings.TrimSpace(params.StudyID)
	if studyID == "" {
		studyID = strings.TrimSpace(params.NCTID)
	}
	if studyID == "" {
		studyID = strings.TrimSpace(params.EUCTNumber)
	}
	normalizedNCTID := strings.ToUpper(studyID)
	if euCTNumberPattern.MatchString(studyID) {
		data, err := c.getEUClinicalTrialsStudy(ctx, studyID)
		if err != nil {
			return "", err
		}
		return renderEUStudyDocument(studyID, data), nil
	}
	if !nctIDPattern.MatchString(normalizedNCTID) {
		return "", fmt.Errorf("study_id must be an NCT ID like NCT04280705 or an EU Clinical Trials CT number like 2025-523486-17-00")
	}

	data, err := c.getClinicalTrialsGovStudy(ctx, normalizedNCTID)
	if err != nil {
		return "", err
	}
	return renderClinicalTrialsGovStudyDocument(normalizedNCTID, data), nil
}

func renderCombinedTrialSearchResults(region string, ctData map[string]any, ctValues url.Values, ctErr error, euData map[string]any, euErr error) string {
	lines := []string{"# Trial Search Results", ""}
	if region != "EU" {
		if ctErr != nil {
			lines = append(lines, "## ClinicalTrials.gov", "", "ClinicalTrials.gov search unavailable: "+ctErr.Error(), "")
		} else {
			lines = append(lines, renderClinicalTrialsGovSearchResults(ctData, ctValues), "")
		}
	}
	if region != "US" {
		if euErr != nil {
			lines = append(lines, "## EU Clinical Trials", "", "EU Clinical Trials search unavailable: "+euErr.Error())
		} else {
			lines = append(lines, renderEUSearchResults(euData))
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

var validSearchRegions = map[string]bool{
	"ALL": true,
	"EU":  true,
	"US":  true,
}

func normalizeSearchRegion(value string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	switch normalized {
	case "":
		return "ALL", nil
	case "USA", "UNITED STATES", "UNITED_STATES":
		return "US", nil
	case "EUROPE", "EUROPEAN UNION", "EUROPEAN_UNION":
		return "EU", nil
	}
	if !validSearchRegions[normalized] {
		return "", fmt.Errorf("invalid region %q", value)
	}
	return normalized, nil
}

func normalizeSearchScope(params SearchTrialsParams) (string, string, error) {
	region, err := normalizeSearchRegion(params.Region)
	if err != nil {
		return "", "", err
	}

	country := strings.TrimSpace(params.Country)
	location := strings.TrimSpace(params.Location)
	if country == "" {
		country = location
	}
	if params.Region != "" {
		return region, country, nil
	}

	if inferredRegion := regionFromLocation(location); inferredRegion != "" {
		if strings.EqualFold(country, location) {
			country = ""
		}
		return inferredRegion, country, nil
	}
	if inferredRegion := regionFromLocation(country); inferredRegion != "" {
		country = ""
		return inferredRegion, country, nil
	}
	return region, country, nil
}

func regionFromLocation(value string) string {
	normalized := normalizeLocationText(value)
	switch normalized {
	case "eu", "europe", "european union", "european_union":
		return "EU"
	case "us", "u s", "usa", "u s a", "america", "united states", "united states of america":
		return "US"
	}
	return ""
}

func normalizeLocationText(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.NewReplacer(".", " ", "_", " ", "-", " ").Replace(normalized)
	return strings.Join(strings.Fields(normalized), " ")
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
