package oauth

type ErrorCode string

const (
	ErrorCodeIncompatibleVersion ErrorCode = "incompatible_version"
	ErrorCodeAccessDenied        ErrorCode = "access_denied"
	ErrorCodeInvalidRequest      ErrorCode = "invalid_request"
	ErrorCodeAccountBanned       ErrorCode = "account_banned"
	ErrorCodeRateLimited         ErrorCode = "rate_limited"
)

const (
	ParamError            = "error"
	ParamErrorDescription = "error_description"
	ParamMinVersion       = "min_version"
	ParamClientVersion    = "client_version"
	ParamLocalPort        = "local_port"
)
