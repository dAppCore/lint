package log

type Err struct {
	Operation string
	Message   string
	Cause     error
}

func (e *Err) Error() string {
	prefix := ""
	if e.Operation != "" {
		prefix = e.Operation + ": "
	}
	if e.Cause != nil {
		return prefix + e.Message + ": " + e.Cause.Error()
	}
	return prefix + e.Message
}

func (e *Err) Unwrap() error {
	return e.Cause
}

func E(op, msg string, err error) error {
	return &Err{Operation: op, Message: msg, Cause: err}
}

func Wrap(err error, op, msg string) error {
	if err == nil {
		return nil
	}
	return E(op, msg, err)
}
