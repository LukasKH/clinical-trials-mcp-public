package clinicaltrials

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (c *Client) searchEUClinicalTrials(ctx context.Context, params SearchTrialsParams) (map[string]any, error) {
	return c.requestJSONWithHeaders(
		ctx,
		"EU Clinical Trials",
		http.MethodPost,
		c.euBaseURL+"/search",
		nil,
		c.euSearchRequestBody(params),
		map[string]string{"Origin": "https://euclinicaltrials.eu"},
	)
}

func (c *Client) getEUClinicalTrialsStudy(ctx context.Context, ctNumber string) (map[string]any, error) {
	return c.requestJSON(ctx, "EU Clinical Trials", http.MethodGet, c.euBaseURL+"/retrieve/"+ctNumber, nil, nil)
}

func (c *Client) euSearchRequestBody(params SearchTrialsParams) map[string]any {
	pageSize := params.PageSize
	if pageSize == 0 {
		pageSize = 5
	}
	pageSize = clamp(pageSize, 1, c.maxPageSize)

	return map[string]any{
		"pagination": map[string]any{
			"page": euSearchPage(params.PageToken),
			"size": pageSize,
		},
		"sort": map[string]any{
			"property":  "decisionDate",
			"direction": "DESC",
		},
		"searchCriteria": euSearchCriteria(params),
	}
}

func euSearchCriteria(params SearchTrialsParams) map[string]any {
	query := strings.TrimSpace(params.Query)
	criteria := map[string]any{
		"containAll":             query,
		"containAny":             nil,
		"containNot":             nil,
		"title":                  emptyNil(params.Title),
		"number":                 nil,
		"status":                 nil,
		"medicalCondition":       emptyNil(params.Condition),
		"sponsor":                emptyNil(params.Sponsor),
		"endPoint":               emptyNil(params.Outcome),
		"productName":            emptyNil(params.Intervention),
		"productRole":            nil,
		"populationType":         nil,
		"orphanDesignation":      nil,
		"msc":                    emptyNil(params.Country),
		"ageGroupCode":           nil,
		"therapeuticAreaCode":    nil,
		"trialPhaseCode":         nil,
		"sponsorTypeCode":        nil,
		"gender":                 nil,
		"eeaStartDateFrom":       nil,
		"eeaStartDateTo":         nil,
		"eeaEndDateFrom":         nil,
		"eeaEndDateTo":           nil,
		"protocolCode":           nil,
		"rareDisease":            nil,
		"pip":                    nil,
		"haveOrphanDesignation":  nil,
		"hasStudyResults":        nil,
		"hasClinicalStudyReport": nil,
		"isLowIntervention":      nil,
		"hasSeriousBreach":       nil,
		"hasUnexpectedEvent":     nil,
		"hasUrgentSafetyMeasure": nil,
		"isTransitioned":         nil,
		"eudraCtCode":            nil,
		"trialRegion":            nil,
		"vulnerablePopulation":   nil,
		"mscStatus":              nil,
	}
	if euCTNumberPattern.MatchString(query) {
		criteria["number"] = query
	}
	return criteria
}

func emptyNil(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func euSearchPage(pageToken string) int {
	token := strings.TrimSpace(strings.ToLower(pageToken))
	if !strings.HasPrefix(token, "eu:") {
		return 1
	}
	page, err := strconv.Atoi(strings.TrimPrefix(token, "eu:"))
	if err != nil || page < 1 {
		return 1
	}
	return page
}

func renderEUSearchResults(data map[string]any) string {
	trials, _ := data["data"].([]any)
	if len(trials) == 0 {
		return "## EU Clinical Trials\n\nNo EU Clinical Trials results found for the supplied search criteria.\nTry broader search terms or fewer filters."
	}

	lines := []string{"## EU Clinical Trials", ""}
	if pagination := object(data, "pagination"); pagination != nil {
		if totalRecords := text(pagination["totalRecords"]); totalRecords != "" {
			lines = append(lines, fmt.Sprintf("Total matching EU trials: %s", totalRecords), "")
		}
	}

	for index, rawTrial := range trials {
		trial, _ := rawTrial.(map[string]any)
		ctNumber := text(trial["ctNumber"])
		title := text(firstPresent(trial["ctTitle"], trial["shortTitle"], "Untitled EU trial"))
		status := euStatus(text(trial["ctStatus"]))
		conditions := text(trial["conditions"])
		phase := text(trial["trialPhase"])
		endpoint := shorten(text(firstPresent(trial["primaryEndPoint"], trial["endPoint"])), 420)

		lines = append(lines,
			fmt.Sprintf("### %d. %s: %s", index+1, ctNumber, title),
			fmt.Sprintf("- Registry: EU Clinical Trials"),
			fmt.Sprintf("- Status: %s", fallback(status, "Not provided")),
			fmt.Sprintf("- Conditions: %s", fallback(conditions, "Not provided")),
			fmt.Sprintf("- Phase / Study type: %s", fallback(phase, "Not provided")),
			fmt.Sprintf("- Sponsor: %s", fallback(text(trial["sponsor"]), "Not provided")),
			fmt.Sprintf("- Countries: %s", fallback(euCountries(trial["trialCountries"]), "Not provided")),
			fmt.Sprintf("- Short summary: %s", fallback(endpoint, "Not provided")),
			fmt.Sprintf("- Source URL: %s/%s?lang=en", euSourceBaseURL, ctNumber),
			"",
		)
	}

	if pagination := object(data, "pagination"); text(pagination["nextPage"]) == "true" {
		nextPage := text(pagination["currentPage"])
		if parsed, err := strconv.Atoi(nextPage); err == nil {
			lines = append(lines, "More EU results are available.", fmt.Sprintf("EU next page token: eu:%d", parsed+1))
		}
	}

	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func euStatus(value string) string {
	switch value {
	case "1":
		return "Under evaluation"
	case "2":
		return "Authorised"
	case "3":
		return "Refused"
	case "4":
		return "Withdrawn"
	default:
		return value
	}
}

func euCountries(value any) string {
	values, ok := value.([]any)
	if !ok || len(values) == 0 {
		return text(value)
	}
	countries := make([]string, 0, len(values))
	for _, item := range values {
		country := strings.TrimSpace(strings.Split(text(item), ":")[0])
		if country != "" {
			countries = append(countries, country)
		}
	}
	return strings.Join(countries, ", ")
}

func renderEUStudyDocument(ctNumber string, data map[string]any) string {
	application := object(data, "authorizedApplication")
	partI := object(application, "authorizedPartI")
	trialDetails := object(partI, "trialDetails")
	identifiers := object(trialDetails, "clinicalTrialIdentifiers")
	trialInfo := object(trialDetails, "trialInformation")
	trialCategory := object(trialInfo, "trialCategory")
	objective := object(trialInfo, "trialObjective")
	population := object(trialInfo, "populationOfTrialSubjects")
	duration := object(trialInfo, "trialDuration")

	title := text(firstPresent(identifiers["publicTitle"], identifiers["fullTitle"]))
	referencesText := listText(firstPresent(trialDetails["references"], trialDetails["pubmedUrl"], trialDetails["pubmedCode"]))
	if referencesText == "" {
		referencesText = "- Not provided"
	} else {
		referencesText = "- " + referencesText
	}
	secondaryIdentifiers := object(identifiers, "secondaryIdentifyingNumbers")
	linkedNCTID := text(object(secondaryIdentifiers, "nctNumber")["number"])
	lines := []string{
		fmt.Sprintf("# Study Document: %s", ctNumber),
		"",
		fmt.Sprintf("## Title\n%s", fallback(title, "Not provided")),
		"",
		"## Status",
		fmt.Sprintf("- Overall status: %s", fallback(text(data["ctStatus"]), "Not provided")),
		fmt.Sprintf("- Decision date: %s", fallback(text(data["decisionDate"]), "Not provided")),
		fmt.Sprintf("- Publish date: %s", fallback(text(data["publishDate"]), "Not provided")),
		"",
		"## Sponsor",
		fmt.Sprintf("- Lead sponsor: %s", fallback(euSponsorNames(partI["sponsors"], true), "Not provided")),
		fmt.Sprintf("- Other sponsor-related organisations: %s", fallback(euSponsorNames(partI["sponsors"], false), "Not provided")),
		"",
		"## Summaries",
		fmt.Sprintf("### Public title\n%s", fallback(text(identifiers["publicTitle"]), "Not provided")),
		"",
		fmt.Sprintf("### Main objective\n%s", fallback(text(objective["mainObjective"]), "Not provided")),
		"",
		fmt.Sprintf("### Secondary objectives\n%s", fallback(text(objective["secondaryObjectives"]), "Not provided")),
		"",
		"## Design",
		fmt.Sprintf("- Trial phase: %s", fallback(text(trialCategory["trialPhase"]), "Not provided")),
		fmt.Sprintf("- Trial category: %s", fallback(text(trialCategory["trialCategory"]), "Not provided")),
		fmt.Sprintf("- Low intervention: %s", fallback(text(partI["isLowIntervention"]), "Not provided")),
		fmt.Sprintf("- Estimated recruitment start date: %s", fallback(text(duration["estimatedRecruitmentStartDate"]), "Not provided")),
		fmt.Sprintf("- Estimated end date: %s", fallback(text(firstPresent(duration["estimatedEndDate"], duration["estimatedGlobalEndDate"])), "Not provided")),
		fmt.Sprintf("- Enrollment: %s", fallback(text(partI["rowSubjectCount"]), "Not provided")),
		"",
		"## Conditions",
		euNamedBullets(firstPresent(partI["medicalConditions"], object(trialInfo, "medicalCondition")["partIMedicalConditions"]), "medicalCondition"),
		"",
		"## Interventions",
		euProducts(partI["products"]),
		"",
		"## Outcomes",
		euEndpoints(trialInfo["endPoint"]),
		"",
		"## Eligibility",
		fmt.Sprintf("- Sex: %s", euSex(population)),
		fmt.Sprintf("- Population: %s", fallback(listText(population["clinicalTrialGroups"]), "Not provided")),
		fmt.Sprintf("- Vulnerable population selected: %s", fallback(text(population["isVulnerablePopulationSelected"]), "Not provided")),
		"",
		euEligibility(trialInfo["eligibilityCriteria"]),
		"",
		"## Locations",
		euLocations(application["authorizedPartsII"]),
		"",
		"## References",
		referencesText,
		"",
		"## Documents",
		euDocuments(data["documents"]),
		"",
		"## Source Metadata",
		fmt.Sprintf("- Source URL: %s/%s?lang=en", euSourceBaseURL, ctNumber),
		"- Data source: EU Clinical Trials CTIS public API",
		fmt.Sprintf("- CT number: %s", ctNumber),
		fmt.Sprintf("- Linked NCT ID: %s", fallback(linkedNCTID, "Not provided")),
	}

	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func euSponsorNames(value any, primaryOnly bool) string {
	values, ok := value.([]any)
	if !ok || len(values) == 0 {
		return ""
	}
	names := make([]string, 0, len(values))
	for _, item := range values {
		sponsor, _ := item.(map[string]any)
		if primaryOnly && text(sponsor["primary"]) != "true" {
			continue
		}
		if !primaryOnly && text(sponsor["primary"]) == "true" {
			continue
		}
		if name := text(object(sponsor, "organisation")["name"]); name != "" {
			names = append(names, name)
		}
	}
	return strings.Join(names, ", ")
}

func euNamedBullets(value any, nameKey string) string {
	values, ok := value.([]any)
	if !ok || len(values) == 0 {
		return "- Not provided"
	}
	lines := make([]string, 0, len(values))
	for _, item := range values {
		objectItem, _ := item.(map[string]any)
		lines = append(lines, "- "+fallback(text(objectItem[nameKey]), "Not provided"))
	}
	return strings.Join(lines, "\n")
}

func euProducts(value any) string {
	values, ok := value.([]any)
	if !ok || len(values) == 0 {
		return "- Not provided"
	}
	limit := min(len(values), 12)
	lines := make([]string, 0, limit)
	for _, item := range values[:limit] {
		product, _ := item.(map[string]any)
		lines = append(lines, fmt.Sprintf("- %s: %s", fallback(text(product["productName"]), "Product"), fallback(text(product["mpRoleInTrial"]), "role not provided")))
	}
	return strings.Join(lines, "\n")
}

func euEndpoints(value any) string {
	values, ok := value.([]any)
	if !ok || len(values) == 0 {
		return "- Not provided"
	}
	limit := min(len(values), 12)
	lines := make([]string, 0, limit)
	for _, item := range values[:limit] {
		endpoint, _ := item.(map[string]any)
		prefix := "Secondary"
		if text(endpoint["isPrimary"]) == "true" {
			prefix = "Primary"
		}
		lines = append(lines, fmt.Sprintf("- %s: %s", prefix, fallback(text(endpoint["endPoint"]), "Not provided")))
	}
	return strings.Join(lines, "\n")
}

func euSex(population map[string]any) string {
	sexes := make([]string, 0, 2)
	if text(population["isFemaleSubjects"]) == "true" {
		sexes = append(sexes, "Female")
	}
	if text(population["isMaleSubjects"]) == "true" {
		sexes = append(sexes, "Male")
	}
	if len(sexes) == 0 {
		return "Not provided"
	}
	return strings.Join(sexes, ", ")
}

func euEligibility(value any) string {
	eligibility, ok := value.(map[string]any)
	if !ok {
		return "Eligibility criteria not provided."
	}
	inclusion := text(eligibility["principalInclusionCriteria"])
	exclusion := text(eligibility["principalExclusionCriteria"])
	if inclusion == "" && exclusion == "" {
		return "Eligibility criteria not provided."
	}
	return strings.TrimSpace(fmt.Sprintf("### Inclusion criteria\n%s\n\n### Exclusion criteria\n%s", fallback(inclusion, "Not provided"), fallback(exclusion, "Not provided")))
}

func euLocations(value any) string {
	values, ok := value.([]any)
	if !ok || len(values) == 0 {
		return "- Not provided"
	}
	lines := make([]string, 0)
	for _, rawPartII := range values {
		partII, _ := rawPartII.(map[string]any)
		country := text(object(partII, "mscInfo")["countryName"])
		sites, _ := partII["trialSites"].([]any)
		if len(sites) == 0 {
			lines = append(lines, "- "+fallback(country, "Country not provided"))
			continue
		}
		for _, rawSite := range sites[:min(len(sites), 6)] {
			site, _ := rawSite.(map[string]any)
			addressInfo := object(site, "organisationAddressInfo")
			org := object(addressInfo, "organisation")
			address := object(addressInfo, "address")
			lines = append(lines, "- "+joinAvailable(text(org["name"]), text(address["city"]), fallback(text(address["countryName"]), country)))
			if len(lines) >= 20 {
				return strings.Join(lines, "\n")
			}
		}
	}
	return strings.Join(lines, "\n")
}

func euDocuments(value any) string {
	values, ok := value.([]any)
	if !ok || len(values) == 0 {
		return "- Not provided"
	}
	limit := min(len(values), 10)
	lines := make([]string, 0, limit)
	for _, item := range values[:limit] {
		document, _ := item.(map[string]any)
		lines = append(lines, fmt.Sprintf("- %s: %s (%s)", fallback(text(document["documentTypeLabel"]), "Document"), fallback(text(document["title"]), "Untitled"), fallback(text(document["fileType"]), "file type not provided")))
	}
	return strings.Join(lines, "\n")
}
