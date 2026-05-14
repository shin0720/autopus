package run

import qaevidence "github.com/insajin/autopus-adk/pkg/qa/evidence"

func sanitizeResult(result Result) Result {
	for i := range result.Checks {
		result.Checks[i].Expected = qaevidence.RedactText(result.Checks[i].Expected)
		result.Checks[i].Actual = qaevidence.RedactText(result.Checks[i].Actual)
		result.Checks[i].FailureSummary = qaevidence.RedactText(result.Checks[i].FailureSummary)
	}
	for i := range result.AdapterResults {
		result.AdapterResults[i].FailureSummary = qaevidence.RedactText(result.AdapterResults[i].FailureSummary)
		if result.AdapterResults[i].SetupGap != nil {
			result.AdapterResults[i].SetupGap.Reason = qaevidence.RedactText(result.AdapterResults[i].SetupGap.Reason)
		}
	}
	for i := range result.SetupGaps {
		result.SetupGaps[i].Reason = qaevidence.RedactText(result.SetupGaps[i].Reason)
	}
	return result
}
