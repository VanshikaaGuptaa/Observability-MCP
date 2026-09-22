package investigation

type DependencyStatus struct {
	Name                 string  `json:"name"`
	Status               string  `json:"status"` // Preserved for backward compatibility
	CurrentStatus        string  `json:"current_status"`
	WindowStatus         string  `json:"window_status"`
	HistoricalErrorCount float64 `json:"historical_error_count"`
	ErrorCount           float64 `json:"error_count"` // Preserved for backward compatibility
	RecentErrorCount     float64 `json:"recent_error_count"`
}

type DependencySummary struct {
	Dependencies []DependencyStatus `json:"dependencies"`
}
