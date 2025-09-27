package exceptions

type serverException string

func (e serverException) Error() string {
	return string(e)
}

const (
	InvalidArguments serverException = "Service called with invalid arguments"
	AlreadyExist     serverException = "Resource already exist"
	DoesNotExist     serverException = "Resource does not exist"
	NotModified      serverException = "Resource was not modified"
	ExternalFailure  serverException = "Failure due to an external exception"
	InternalFailure  serverException = "Failure due to an internal exception"
	Unknown          serverException = "Unknown service error"
)
