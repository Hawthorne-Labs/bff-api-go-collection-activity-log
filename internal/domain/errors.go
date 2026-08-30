package domain

// Error codes for the activity-log BFF.
// These are numeric application codes mapped from the Python BFF error codes.
const (
	// Activities
	ActivitiesListFailed = 5001
	ActivityCreateFailed = 5002
	ActivityBatchFailed  = 5003
	ActivityNotFound     = 5004
	// Escalations
	EscalationCreateFailed = 5010
	EscalationUpdateFailed = 5011
	EscalationNotFound     = 5012
	EscalationDecideFailed = 5013
	EscalationListFailed   = 5014
	// Payment Promises
	PaymentPromisesListFailed = 5020
	// Agent Performance
	AgentKPIsFailed     = 5030
	AgentGoalsFailed    = 5031
	AgentRankingFailed  = 5032
	AgentWorkloadFailed = 5033
	AgentReportFailed   = 5034
	// Notifications
	NotificationsListFailed    = 5040
	NotificationDetailFailed   = 5041
	NotificationCreateFailed   = 5042
	NotificationReadFailed     = 5043
	NotificationRegisterFailed = 5044
	NotificationRevokeFailed   = 5045
	// Dashboard
	DashboardSummaryFailed = 5050
	DashboardAlertsFailed  = 5051
	// Contacts
	ContactSubmitFailed = 5080
	// General
	CollectionsRequestFailed = 5060
	// Auth
	MissingAuthToken = 5070
	InvalidAuthToken = 5071
	AccessDenied     = 5072
	// Public catalog (shared with Python BFF)
	ValidationError          = 90005
	IdempotencyKeyRequired   = 90020
	IdempotencyKeyInvalid    = 90021
	IdempotencyKeyConflict   = 90022
	IdempotencyInProgress    = 90023
)

// BusinessError represents a user-facing error with a numeric code.
type BusinessError struct {
	Code       int    `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"-"`
}

func (e *BusinessError) Error() string {
	return e.Message
}

// NewBusinessError creates a BusinessError from a code string.
func NewBusinessError(code int, message string) *BusinessError {
	return &BusinessError{Code: code, Message: message}
}

// NewHTTPBusinessError creates a BusinessError with an explicit HTTP status.
func NewHTTPBusinessError(httpStatus, code int, message string) *BusinessError {
	return &BusinessError{Code: code, Message: message, HTTPStatus: httpStatus}
}

// Status returns the HTTP status for this error (defaults to 502 for upstream failures).
func (e *BusinessError) Status() int {
	if e == nil {
		return 502
	}
	if e.HTTPStatus > 0 {
		return e.HTTPStatus
	}
	switch e.Code {
	case MissingAuthToken, InvalidAuthToken:
		return 401
	case AccessDenied:
		return 403
	case ActivityNotFound, EscalationNotFound:
		return 404
	case ValidationError:
		return 422
	case IdempotencyKeyRequired, IdempotencyKeyInvalid:
		return 400
	case IdempotencyKeyConflict, IdempotencyInProgress:
		return 409
	default:
		if e.Code >= 5001 && e.Code < 5090 {
			return 400
		}
		return 502
	}
}
