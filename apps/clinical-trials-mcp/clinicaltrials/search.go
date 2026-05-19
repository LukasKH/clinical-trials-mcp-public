package clinicaltrials

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

var validSearchRegions = map[string]bool{
	"ALL": true,
	"EU":  true,
	"US":  true,
}

var searchFields = []string{
	"NCTId",
	"BriefTitle",
	"OfficialTitle",
	"OverallStatus",
	"Condition",
	"Phase",
	"StudyType",
	"BriefSummary",
}

func (c *Client) SearchTrials(ctx context.Context, params SearchTrialsParams) (string, error) {
	region, country, err := normalizeSearchScope(params)
	if err != nil {
		return "", err
	}
	params.Country = country

	values, err := c.searchValues(params, region)
	if err != nil {
		return "", err
	}

	searchCT := region != "EU"
	searchEU := region != "US"
	var ctData map[string]any
	var ctErr error
	var euData map[string]any
	var euErr error
	if searchCT {
		ctData, ctErr = c.getJSON(ctx, "/studies", values)
	}
	if searchEU {
		euData, euErr = c.searchEUTrials(ctx, params)
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

	return renderCombinedTrialSearchResults(region, ctData, ctErr, euData, euErr, values), nil
}

func (c *Client) searchValues(params SearchTrialsParams, region string) (url.Values, error) {
	if strings.TrimSpace(params.Query) == "" {
		return nil, fmt.Errorf("query is required")
	}

	pageSize := params.PageSize
	if pageSize == 0 {
		pageSize = 5
	}
	pageSize = clamp(pageSize, 1, c.maxPageSize)

	values := url.Values{}
	values.Set("pageSize", fmt.Sprintf("%d", pageSize))
	values.Set("countTotal", "true")
	values.Set("fields", strings.Join(searchFields, "|"))
	values.Set("format", "json")
	addFilter(values, "query.term", params.Query)
	addFilter(values, "query.cond", params.Condition)
	addFilter(values, "query.intr", params.Intervention)
	addFilter(values, "query.spons", params.Sponsor)
	addFilter(values, "query.titles", params.Title)
	addFilter(values, "query.outc", params.Outcome)
	if region == "US" {
		addFilter(values, "filter.advanced", locationCountryFilter(fallback(params.Country, "United States")))
	} else {
		addFilter(values, "filter.advanced", locationCountryFilter(params.Country))
	}
	if !strings.HasPrefix(strings.TrimSpace(strings.ToLower(params.PageToken)), "eu:") {
		addFilter(values, "pageToken", params.PageToken)
	}
	return values, nil
}

func renderCombinedTrialSearchResults(region string, ctData map[string]any, ctErr error, euData map[string]any, euErr error, values url.Values) string {
	lines := []string{"# Trial Search Results", ""}
	if region != "EU" {
		if ctErr != nil {
			lines = append(lines, "## ClinicalTrials.gov", "", "ClinicalTrials.gov search unavailable: "+ctErr.Error(), "")
		} else {
			lines = append(lines, renderClinicalTrialsGovSearchResults(ctData, values), "")
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

func renderClinicalTrialsGovSearchResults(data map[string]any, values url.Values) string {
	studies, _ := data["studies"].([]any)
	if len(studies) == 0 {
		return "## ClinicalTrials.gov\n\n" + emptySearchResult(values)
	}

	lines := []string{"## ClinicalTrials.gov", ""}
	if totalCount, ok := data["totalCount"]; ok {
		lines = append(lines, fmt.Sprintf("Total matching studies: %s", text(totalCount)), "")
	}

	for index, rawStudy := range studies {
		study, _ := rawStudy.(map[string]any)
		protocol := object(study, "protocolSection")
		identification := object(protocol, "identificationModule")
		status := object(protocol, "statusModule")
		design := object(protocol, "designModule")
		conditions := object(protocol, "conditionsModule")
		description := object(protocol, "descriptionModule")

		nctID := text(identification["nctId"])
		title := text(firstPresent(identification["briefTitle"], identification["officialTitle"], "Untitled study"))
		overallStatus := text(status["overallStatus"])
		conditionNames := listText(conditions["conditions"])
		phases := listText(design["phases"])
		studyType := text(design["studyType"])
		summary := shorten(text(description["briefSummary"]), 420)

		lines = append(lines,
			fmt.Sprintf("## %d. %s: %s", index+1, nctID, title),
			fmt.Sprintf("- Status: %s", fallback(overallStatus, "Not provided")),
			fmt.Sprintf("- Conditions: %s", fallback(conditionNames, "Not provided")),
			fmt.Sprintf("- Phase / Study type: %s", joinAvailable(phases, studyType)),
			fmt.Sprintf("- Short summary: %s", fallback(summary, "Not provided")),
			fmt.Sprintf("- Source URL: %s/%s", sourceBaseURL, nctID),
			"",
		)
	}

	if nextPageToken := text(data["nextPageToken"]); nextPageToken != "" {
		lines = append(lines, "More results are available.", fmt.Sprintf("Next page token: %s", nextPageToken))
	}

	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func emptySearchResult(values url.Values) string {
	keys := make([]string, 0)
	for key := range values {
		if strings.HasPrefix(key, "query.") || strings.HasPrefix(key, "filter.") {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	lines := []string{"No Trial Search Results found for the supplied search criteria."}
	if len(keys) > 0 {
		criteria := make([]string, 0, len(keys))
		for _, key := range keys {
			criteria = append(criteria, fmt.Sprintf("%s=%s", key, values.Get(key)))
		}
		lines = append(lines, "Criteria: "+strings.Join(criteria, ", "))
	}
	lines = append(lines, "Try broader search terms or fewer filters.")
	return strings.Join(lines, "\n")
}

func addFilter(values url.Values, key string, value string) {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		values.Set(key, trimmed)
	}
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

func locationCountryFilter(country string) string {
	if trimmed := strings.TrimSpace(country); trimmed != "" {
		return "AREA[LocationCountry]" + trimmed
	}
	return ""
}

func clamp(value int, minValue int, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func shorten(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return strings.TrimSpace(value[:limit-3]) + "..."
}
