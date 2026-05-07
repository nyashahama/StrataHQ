package response

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

type SuccessResponse struct {
	Data any   `json:"data"`
	Meta *Meta `json:"meta,omitempty"`
}

type Meta struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Total   int `json:"total"`
}

type ErrorResponse struct {
	Err ErrorBody `json:"error"`
}

type ErrorBody struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"requestId,omitempty"`
}

func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(SuccessResponse{Data: data})
}

func JSONList(w http.ResponseWriter, status int, data any, meta Meta) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(SuccessResponse{Data: data, Meta: &meta})
}

func Error(w http.ResponseWriter, status int, code string, message string) {
	writeError(w, status, ErrorBody{Code: code, Message: message})
}

func ErrorWithRequest(w http.ResponseWriter, r *http.Request, status int, code string, message string) {
	requestID := ""
	if r != nil {
		requestID = r.Header.Get("X-Request-ID")
	}
	writeError(w, status, ErrorBody{Code: code, Message: message, RequestID: requestID})
}

func writeError(w http.ResponseWriter, status int, body ErrorBody) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{Err: body})
}

// DecodeJSON decodes a JSON request body into dst, rejecting unknown fields.
func DecodeJSON(r io.Reader, dst any) error {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("trailing tokens after JSON value")
		}
		return err
	}
	return nil
}
