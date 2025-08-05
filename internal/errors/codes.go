package errors

type ErrorCode int

const (
	Unknown ErrorCode = iota
	ServiceFailure
	BadCallingParameters
	ResourceAlreadyExist
	DoesNotExist
)
