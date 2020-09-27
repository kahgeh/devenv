package errors

type NoResponse struct {
	Message string
}

func (e *NoResponse) Error() string {
	return e.Message
}
