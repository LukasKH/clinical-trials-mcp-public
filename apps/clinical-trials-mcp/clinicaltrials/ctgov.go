package clinicaltrials

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

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

func (c *Client) searchClinicalTrialsGov(ctx context.Context, params SearchTrialsParams, region string) (map[string]any, url.Values, error) {
	values, err := c.clinicalTrialsGovSearchValues(params, region)
	if err != nil {
		return nil, nil, err
	}
	data, err := c.requestJSON(ctx, "ClinicalTrials.gov", http.MethodGet, c.baseURL+"/studies", values, nil)
	return data, values, err
}

func (c *Client) getClinicalTrialsGovStudy(ctx context.Context, nctID string) (map[string]any, error) {
	values := url.Values{}
	values.Set("format", "json")
	return c.requestJSON(ctx, "ClinicalTrials.gov", http.MethodGet, c.baseURL+"/studies/"+nctID, values, nil)
}

func (c *Client) clinicalTrialsGovSearchValues(params SearchTrialsParams, region string) (url.Values, error) {
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

func renderClinicalTrialsGovStudyDocument(nctID string, data map[string]any) string {
	protocol := object(data, "protocolSection")
	derived := object(data, "derivedSection")
	identification := object(protocol, "identificationModule")
	status := object(protocol, "statusModule")
	sponsor := object(protocol, "sponsorCollaboratorsModule")
	description := object(protocol, "descriptionModule")
	design := object(protocol, "designModule")
	arms := object(protocol, "armsInterventionsModule")
	outcomes := object(protocol, "outcomesModule")
	eligibility := object(protocol, "eligibilityModule")
	contacts := object(protocol, "contactsLocationsModule")
	references := object(protocol, "referencesModule")
	documents := object(protocol, "documentSection")

	title := text(firstPresent(identification["briefTitle"], identification["officialTitle"]))
	startDateValue := status["startDateStruct"]
	startDate := fallback(text(startDateValue), "Not provided")
	if startDateStruct, ok := startDateValue.(map[string]any); ok {
		startDate = fallback(text(startDateStruct["date"]), "Not provided")
	}
	completionDateValue := status["completionDateStruct"]
	completionDate := fallback(text(completionDateValue), "Not provided")
	if completionDateStruct, ok := completionDateValue.(map[string]any); ok {
		completionDate = fallback(text(completionDateStruct["date"]), "Not provided")
	}
	leadSponsorValue := sponsor["leadSponsor"]
	leadSponsor := fallback(text(leadSponsorValue), "Not provided")
	if leadSponsorObject, ok := leadSponsorValue.(map[string]any); ok {
		leadSponsor = fallback(text(leadSponsorObject["name"]), "Not provided")
	}
	enrollmentInfo := object(design, "enrollmentInfo")
	enrollmentText := joinAvailable(text(enrollmentInfo["count"]), text(enrollmentInfo["type"]))
	lines := []string{
		fmt.Sprintf("# Study Document: %s", nctID),
		"",
		fmt.Sprintf("## Title\n%s", fallback(title, "Not provided")),
		"",
		"## Status",
		fmt.Sprintf("- Overall status: %s", fallback(text(status["overallStatus"]), "Not provided")),
		fmt.Sprintf("- Start date: %s", startDate),
		fmt.Sprintf("- Completion date: %s", completionDate),
		fmt.Sprintf("- Last update submitted: %s", fallback(text(status["lastUpdateSubmitDate"]), "Not provided")),
		"",
		"## Sponsor",
		fmt.Sprintf("- Lead sponsor: %s", leadSponsor),
		fmt.Sprintf("- Collaborators: %s", orgList(sponsor["collaborators"])),
		"",
		"## Summaries",
		fmt.Sprintf("### Brief Summary\n%s", fallback(text(description["briefSummary"]), "Not provided")),
		"",
		fmt.Sprintf("### Detailed Description\n%s", fallback(text(description["detailedDescription"]), "Not provided")),
		"",
		"## Design",
		fmt.Sprintf("- Study type: %s", fallback(text(design["studyType"]), "Not provided")),
		fmt.Sprintf("- Phases: %s", fallback(listText(design["phases"]), "Not provided")),
		fmt.Sprintf("- Enrollment: %s", enrollmentText),
		fmt.Sprintf("- Design allocation: %s", fallback(text(object(design, "designInfo")["allocation"]), "Not provided")),
		fmt.Sprintf("- Masking: %s", fallback(text(object(object(design, "designInfo"), "maskingInfo")["masking"]), "Not provided")),
		"",
		"## Conditions",
		bullets(object(protocol, "conditionsModule")["conditions"]),
		"",
		"## Interventions",
		interventions(arms["interventions"]),
		"",
		"## Outcomes",
		outcomeSections(outcomes),
		"",
		"## Eligibility",
		fmt.Sprintf("- Sex: %s", fallback(text(eligibility["sex"]), "Not provided")),
		fmt.Sprintf("- Minimum age: %s", fallback(text(eligibility["minimumAge"]), "Not provided")),
		fmt.Sprintf("- Maximum age: %s", fallback(text(eligibility["maximumAge"]), "Not provided")),
		fmt.Sprintf("- Healthy volunteers: %s", fallback(text(eligibility["healthyVolunteers"]), "Not provided")),
		"",
		fallback(text(eligibility["eligibilityCriteria"]), "Eligibility criteria not provided."),
		"",
		"## Locations",
		locations(contacts["locations"]),
		"",
		"## References",
		referenceList(references["references"]),
		"",
		"## Documents",
		documentList(documents),
		"",
		"## Source Metadata",
		fmt.Sprintf("- Source URL: %s/%s", sourceBaseURL, nctID),
		"- Data source: ClinicalTrials.gov API v2",
		fmt.Sprintf("- Results first submitted: %s", fallback(text(status["resultsFirstSubmitDate"]), "Not provided")),
		fmt.Sprintf("- Derived intervention types: %s", fallback(listText(object(object(derived, "interventionBrowseModule"), "meshes")), "Not provided")),
	}

	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func orgList(value any) string {
	values, ok := value.([]any)
	if !ok || len(values) == 0 {
		return "Not provided"
	}
	names := make([]any, 0, len(values))
	for _, item := range values {
		if objectItem, ok := item.(map[string]any); ok {
			names = append(names, objectItem["name"])
		} else {
			names = append(names, item)
		}
	}
	return listText(names)
}

func bullets(value any) string {
	values, ok := value.([]any)
	if !ok || len(values) == 0 {
		return "- Not provided"
	}
	lines := make([]string, 0, len(values))
	for _, item := range values {
		lines = append(lines, "- "+text(item))
	}
	return strings.Join(lines, "\n")
}

func interventions(value any) string {
	values, ok := value.([]any)
	if !ok || len(values) == 0 {
		return "- Not provided"
	}
	lines := make([]string, 0, len(values)*2)
	for _, item := range values {
		objectItem, _ := item.(map[string]any)
		lines = append(lines, fmt.Sprintf("- %s: %s", fallback(text(objectItem["type"]), "Intervention"), fallback(text(objectItem["name"]), "Not provided")))
		if description := text(objectItem["description"]); description != "" {
			lines = append(lines, "  - Description: "+description)
		}
	}
	return strings.Join(lines, "\n")
}

func outcomeSections(values map[string]any) string {
	sections := make([]string, 0)
	for _, entry := range []struct {
		label string
		key   string
	}{
		{"Primary outcomes", "primaryOutcomes"},
		{"Secondary outcomes", "secondaryOutcomes"},
		{"Other outcomes", "otherOutcomes"},
	} {
		outcomes, ok := values[entry.key].([]any)
		if !ok || len(outcomes) == 0 {
			continue
		}
		sections = append(sections, "### "+entry.label)
		for _, item := range outcomes {
			outcome, _ := item.(map[string]any)
			measure := fallback(text(outcome["measure"]), "Not provided")
			timeframe := fallback(text(outcome["timeFrame"]), "Not provided")
			sections = append(sections, fmt.Sprintf("- %s (time frame: %s)", measure, timeframe))
			if description := text(outcome["description"]); description != "" {
				sections = append(sections, "  - "+description)
			}
		}
	}
	if len(sections) == 0 {
		return "- Not provided"
	}
	return strings.Join(sections, "\n")
}

func locations(value any) string {
	values, ok := value.([]any)
	if !ok || len(values) == 0 {
		return "- Not provided"
	}
	limit := min(len(values), 20)
	lines := make([]string, 0, limit)
	for _, item := range values[:limit] {
		location, _ := item.(map[string]any)
		facility := strings.TrimSpace(text(location["facility"]))
		lines = append(lines, "- "+joinAvailable(facility, text(location["city"]), text(location["state"]), text(location["country"])))
	}
	return strings.Join(lines, "\n")
}

func referenceList(value any) string {
	values, ok := value.([]any)
	if !ok || len(values) == 0 {
		return "- Not provided"
	}
	limit := min(len(values), 10)
	lines := make([]string, 0, limit)
	for _, item := range values[:limit] {
		reference, _ := item.(map[string]any)
		pmid := text(reference["pmid"])
		pmidText := ""
		if pmid != "" {
			pmidText = "PMID " + pmid
		}
		lines = append(lines, "- "+joinAvailable(text(reference["citation"]), pmidText))
	}
	return strings.Join(lines, "\n")
}

func documentList(value any) string {
	documentSection, ok := value.(map[string]any)
	if !ok {
		return "- Not provided"
	}
	largeDocuments := object(documentSection, "largeDocumentModule")
	docs, ok := largeDocuments["largeDocs"].([]any)
	if !ok || len(docs) == 0 {
		return "- Not provided"
	}
	lines := make([]string, 0, len(docs))
	for _, item := range docs {
		doc, _ := item.(map[string]any)
		lines = append(lines, fmt.Sprintf("- %s: %s", fallback(text(doc["typeAbbrev"]), "Document"), fallback(text(doc["url"]), "No URL")))
	}
	return strings.Join(lines, "\n")
}

func addFilter(values url.Values, key string, value string) {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		values.Set(key, trimmed)
	}
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
