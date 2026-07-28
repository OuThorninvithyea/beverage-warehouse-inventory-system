package httpx

type Envelope struct {
	Success bool       `json:"success"`
	Data    any        `json:"data,omitempty"`
	Error   *ErrorBody `json:"error,omitempty"`
}

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func Success(data any) Envelope {
	return Envelope{Success: true, Data: data}
}

func Failure(code, message string) Envelope {
	return Envelope{
		Success: false,
		Error:   &ErrorBody{Code: code, Message: message},
	}
}
