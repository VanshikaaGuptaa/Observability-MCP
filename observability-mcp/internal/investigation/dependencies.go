package investigation

type DependencyStatus struct {
	Name       string  `json:"name"`
	Status     string  `json:"status"`
	ErrorCount float64 `json:"error_count"`
}

type DependencySummary struct {
	Dependencies []DependencyStatus `json:"dependencies"`
}
