package i18n

// en is the reference catalog: every other catalog must define every key
// defined here (see catalog_test.go).
var en = map[MessageID]string{
	PlanCurrentVersion: "Current version",
	PlanDetectedChange: "Detected change",
	PlanNextVersion:    "Next version",
	PlanForced:         " (forced)",
	PlanPrerelease:     " (pre-release)",
	PlanFinalize:       " (finalize)",
	PlanCommitsSince:   "%d commits since %s",
}
