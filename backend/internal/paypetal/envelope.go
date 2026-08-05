package paypetal

import "encoding/json"

// PayPetal's endpoints reply in one of two different envelope shapes
// depending on which product group answered:
//
//	style 1 (auth, customer, trustvault, payment): {status: bool, statusCode, message, dataResponse}
//	style 2 (TrustCore, bank):                     {status: "success"|"error", responseCode, message, responseData}
//
// decode() hides that difference from every call site: it figures out
// whether the call succeeded, unmarshals whichever data field is present
// into out, and on failure returns an *APIError with the best error code it
// can find (style 2's error responses nest one at responseData.code).
type rawEnvelope struct {
	Status       json.RawMessage `json:"status"`
	Message      *string         `json:"message"`
	DataResponse json.RawMessage `json:"dataResponse"`
	ResponseData json.RawMessage `json:"responseData"`
}

type style2ErrorData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func decode(body []byte, httpStatus int, out interface{}) error {
	var env rawEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return &APIError{HTTPStatus: httpStatus, Message: "unrecognized response from paypetal"}
	}

	ok := isOK(env.Status)
	data := env.DataResponse
	if len(data) == 0 || string(data) == "null" {
		data = env.ResponseData
	}

	if !ok {
		message := "request failed"
		if env.Message != nil && *env.Message != "" {
			message = *env.Message
		}
		code := ""
		// Style 2 errors nest a {code, message} object in the data field.
		var errData style2ErrorData
		if len(data) > 0 && json.Unmarshal(data, &errData) == nil && errData.Message != "" {
			code = errData.Code
			message = errData.Message
		} else {
			// Style 1 errors sometimes carry the real, specific reason as a
			// plain string in dataResponse instead (e.g. "Phone number is
			// required") while `message` itself is just a generic "Fail".
			var errString string
			if len(data) > 0 && json.Unmarshal(data, &errString) == nil && errString != "" {
				message = errString
			}
		}
		return &APIError{HTTPStatus: httpStatus, Code: code, Message: message}
	}

	if out == nil || len(data) == 0 || string(data) == "null" {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return &APIError{HTTPStatus: httpStatus, Message: "could not parse paypetal response: " + err.Error()}
	}
	return nil
}

// isOK handles status being either a JSON bool (style 1: true/false) or a
// JSON string (style 2: "success"/"error").
func isOK(raw json.RawMessage) bool {
	var b bool
	if err := json.Unmarshal(raw, &b); err == nil {
		return b
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s == "success"
	}
	return false
}
