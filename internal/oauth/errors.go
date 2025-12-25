package oauth

// ErrorCode represents OAuth error codes used in client-proxy communication.
type ErrorCode string

const (
	ErrorCodeIncompatibleVersion ErrorCode = "incompatible_version"
	ErrorCodeAccessDenied        ErrorCode = "access_denied"
	ErrorCodeInvalidRequest      ErrorCode = "invalid_request"
)

// Query parameter keys for OAuth flow.
const (
	ParamError            = "error"
	ParamErrorDescription = "error_description"
	ParamMinVersion       = "min_version"
	ParamClientVersion    = "client_version"
	ParamLocalPort        = "local_port"
)
