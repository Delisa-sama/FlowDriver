package flowerror

type FlowError interface {
	Message() string
	Code() string
}

type flowError struct {
	code string
	msg  string
}

func New(code, msg string) FlowError {
	return &flowError{
		code: code,
		msg:  msg,
	}
}

func (e *flowError) Message() string {
	return e.msg
}

func (e *flowError) Code() string {
	return e.code
}
