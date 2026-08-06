package domain

// Activity represents a collection activity log entry.
type Activity struct {
	ID            string   `json:"id,omitempty"`
	LoanID        string   `json:"loan_id"`
	ClientID      string   `json:"client_id,omitempty"`
	AgentID       string   `json:"agent_id"`
	AgentName     string   `json:"agent_name,omitempty"`
	ActivityType  string   `json:"activity_type"`
	Subject       string   `json:"subject"`
	Description   string   `json:"description,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	Channel       string   `json:"channel,omitempty"`
	ScheduledDate string   `json:"scheduled_date,omitempty"`
	CompletedDate string   `json:"completed_date,omitempty"`
	Status        string   `json:"status,omitempty"`
	CreatedAt     string   `json:"created_at,omitempty"`
	UpdatedAt     string   `json:"updated_at,omitempty"`
}

// ActivityListResponse represents a paginated activity list.
type ActivityListResponse struct {
	Activities []Activity `json:"activities"`
	Total      int        `json:"total"`
	Limit      int        `json:"limit"`
	Offset     int        `json:"offset"`
}

// Escalation represents an escalation record.
type Escalation struct {
	ID             string `json:"id,omitempty"`
	LoanID         string `json:"loan_id"`
	ClientID       string `json:"client_id,omitempty"`
	AgentID        string `json:"agent_id"`
	AgentName      string `json:"agent_name,omitempty"`
	EscalationType string `json:"escalation_type"`
	Severity       string `json:"severity,omitempty"`
	Subject        string `json:"subject"`
	Description    string `json:"description,omitempty"`
	Status         string `json:"status,omitempty"`
	Decision       string `json:"decision,omitempty"`
	DecidedBy      string `json:"decided_by,omitempty"`
	CreatedAt      string `json:"created_at,omitempty"`
	UpdatedAt      string `json:"updated_at,omitempty"`
}

// PaymentPromise represents a payment promise record.
type PaymentPromise struct {
	ID            string  `json:"id,omitempty"`
	LoanID        string  `json:"loan_id"`
	ClientID      string  `json:"client_id,omitempty"`
	AgentID       string  `json:"agent_id"`
	AgentName     string  `json:"agent_name,omitempty"`
	PromiseDate   string  `json:"promise_date"`
	PromiseAmount float64 `json:"promise_amount"`
	Currency      string  `json:"currency,omitempty"`
	Status        string  `json:"status,omitempty"`
	Notes         string  `json:"notes,omitempty"`
	CreatedAt     string  `json:"created_at,omitempty"`
	UpdatedAt     string  `json:"updated_at,omitempty"`
}

// AgentKPI represents an agent key performance indicator.
type AgentKPI struct {
	AgentID             string  `json:"agent_id"`
	AgentName           string  `json:"agent_name,omitempty"`
	Day                 string  `json:"day"`
	ActivitiesMade      int     `json:"activities_made"`
	ActivitiesContacted int     `json:"activities_contacted"`
	MetPromiseRate      float64 `json:"met_promise_rate"`
	ActivitiesPerDay    float64 `json:"activities_per_day"`
}

// AgentGoal represents an agent performance goal.
type AgentGoal struct {
	AgentID   string `json:"agent_id"`
	AgentName string `json:"agent_name,omitempty"`
	GoalType  string `json:"goal_type"`
	Target    int    `json:"target"`
	Current   int    `json:"current"`
	Period    string `json:"period,omitempty"`
}

// AgentRankingEntry represents a single entry in the agent ranking.
type AgentRankingEntry struct {
	Rank        int     `json:"rank"`
	AgentID     string  `json:"agent_id"`
	AgentName   string  `json:"agent_name,omitempty"`
	Score       float64 `json:"score"`
	Activities  int     `json:"activities"`
	MetPromises int     `json:"met_promises"`
}

// WorkloadEntry represents an agent's current workload.
type WorkloadEntry struct {
	AgentID           string `json:"agent_id"`
	AgentName         string `json:"agent_name,omitempty"`
	AssignedLoans     int    `json:"assigned_loans"`
	ActiveEscalations int    `json:"active_escalations"`
	PendingPromises   int    `json:"pending_promises"`
}

// Notification represents a notification record.
type Notification struct {
	ID        string         `json:"id"`
	UserID    string         `json:"user_id"`
	Title     string         `json:"title"`
	Message   string         `json:"message"`
	Severity  string         `json:"severity,omitempty"`
	Type      string         `json:"type"`
	State     string         `json:"state"`
	Data      map[string]any `json:"data,omitempty"`
	CreatedAt string         `json:"created_at,omitempty"`
	UpdatedAt string         `json:"updated_at,omitempty"`
}

// NotificationDevice represents a registered notification device.
type NotificationDevice struct {
	InstallationID string `json:"installation_id"`
	UserID         string `json:"user_id"`
	Platform       string `json:"platform"`
	Token          string `json:"token,omitempty"`
	CreatedAt      string `json:"created_at,omitempty"`
}

// DashboardSummary represents the dashboard summary response.
type DashboardSummary struct {
	TotalLoans        int     `json:"total_loans"`
	TotalClients      int     `json:"total_clients"`
	TotalActivities   int     `json:"total_activities"`
	ActiveEscalations int     `json:"active_escalations"`
	PendingPromises   int     `json:"pending_promises"`
	MetPromiseRate    float64 `json:"met_promise_rate"`
}

// DashboardAlert represents a dashboard alert.
type DashboardAlert struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Severity  string `json:"severity"`
	Message   string `json:"message"`
	LoanID    string `json:"loan_id,omitempty"`
	ClientID  string `json:"client_id,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}
