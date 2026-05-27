package response

// ApiResponse is the standard JSON envelope used by all API endpoints.
// Used for swagger documentation only — the actual response is built
// by util/http.Resp.
type ApiResponse struct {
	// Stable errno used by the frontend for i18n. Success is 200; failures use codes from util/constant.Errno.
	StatusCode int `json:"status_code" example:"200"`
	// Default server-side errno text. Frontend should prefer status_code for display translation.
	Message string      `json:"message" example:"success"`
	Data    interface{} `json:"data"`
	// Request id echoed from X-Request-ID when present.
	RequestID string `json:"request_id" example:"req-123456"`
}
