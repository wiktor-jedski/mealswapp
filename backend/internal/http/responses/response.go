package responses

type Envelope struct {
	Data    any            `json:"data,omitempty"`
	Error   *ErrorResponse `json:"error,omitempty"`
	Meta    *Meta          `json:"meta,omitempty"`
	Success bool           `json:"success"`
}

type ErrorResponse struct {
	Code      string `json:"code"`
	Category  string `json:"category,omitempty"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
	RequestID string `json:"requestId,omitempty"`
	Fields    any    `json:"fields,omitempty"`
}

type Meta struct {
	RequestID string `json:"requestId,omitempty"`
}

func Success(data any, requestID string) Envelope {
	return Envelope{
		Data:    data,
		Meta:    &Meta{RequestID: requestID},
		Success: true,
	}
}

func Failure(code string, message string, requestID string) Envelope {
	return Envelope{
		Error: &ErrorResponse{
			Code:      code,
			Message:   message,
			RequestID: requestID,
		},
		Meta:    &Meta{RequestID: requestID},
		Success: false,
	}
}

func ValidationFailure(message string, requestID string, fields any) Envelope {
	envelope := Failure("validation_error", message, requestID)
	envelope.Error.Fields = fields
	return envelope
}
