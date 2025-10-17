package context

// ContextKey is a custom type for context keys to avoid collisions
type ContextKey string

const (
	// UserContextKey is the key for storing the authenticated user in the request context
	UserContextKey ContextKey = "authenticatedUser"

	// FlashMessageContextKey is the key for storing flash messages in the request context
	FlashMessageContextKey ContextKey = "flashMessage"

	// RequestIDContextKey is the key for storing the request ID in the request context
	RequestIDContextKey ContextKey = "requestID"
)