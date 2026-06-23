package subhd

type ProviderError struct {
	Reason string
	Err    error
}

func (e *ProviderError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err == nil {
		return e.Reason
	}
	return e.Reason + ": " + e.Err.Error()
}

func (e *ProviderError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func wrapReason(reason string, err error) error {
	if err == nil {
		return &ProviderError{Reason: reason}
	}
	return &ProviderError{Reason: reason, Err: err}
}

func reasonOf(err error) string {
	if err == nil {
		return ""
	}
	if providerErr, ok := err.(*ProviderError); ok {
		return providerErr.Reason
	}
	return ReasonProbeFailed
}
