package clinicaltrials

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

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
		data, err := c.getEUJSON(ctx, "/retrieve/"+studyID)
		if err != nil {
			return "", err
		}
		return renderEUStudyDocument(studyID, data), nil
	}
	if !nctIDPattern.MatchString(normalizedNCTID) {
		return "", fmt.Errorf("study_id must be an NCT ID like NCT04280705 or an EU Clinical Trials CT number like 2025-523486-17-00")
	}

	values := url.Values{}
	values.Set("format", "json")
	data, err := c.getJSON(ctx, "/studies/"+normalizedNCTID, values)
	if err != nil {
		return "", err
	}

	return renderStudyDocument(normalizedNCTID, data), nil
}

func renderStudyDocument(nctID string, data map[string]any) string {
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
