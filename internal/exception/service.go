package exception

type exception string

func (e exception) Error() string {
	return string(e)
}

const (
	InvalidArguments exception = "Service called with invalid arguments"
	AlreadyExist     exception = "Resource already exist"
	DoesNotExist     exception = "Resource does not exist"
	ExternalFailure  exception = "Failure due to external exception"
	Unknown          exception = "Unknown service error"
)
