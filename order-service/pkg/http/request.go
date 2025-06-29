package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

const (
	defaultMaxMemory = 32 << 20 // 32 MB
)

var (
	errNoFileFound    = errors.New("no files were bounded")
	errNonPointerBind = errors.New("bind error, cannot bind to a non pointer type")
	errNonSliceBind   = errors.New("bind error: input is not a pointer to a byte slice")
)

type Request struct {
	req           *http.Request
	pathParams    map[string]string
	TransactionID string
	SessionID     string
	RequestID     string
}

// NewRequest creates a new GoFr Request instance from the given http.Request.
func NewRequest(r *http.Request) *Request {
	xTId := r.Header.Get("X-Transaction-ID")
	if xTId == "" {
		xTId = uuid.NewString()
		r.Header.Set("X-Transaction-ID", xTId)
	}

	xSId := r.Header.Get("X-Session-ID")
	if xSId == "" {
		xSId = uuid.NewString()
		r.Header.Set("X-Session-ID", xSId)
	}

	xRId := r.Header.Get("X-Request-ID")
	if xRId == "" {
		xRId = uuid.NewString()
		r.Header.Set("X-Request-ID", xRId)
	}

	return &Request{
		req:           r,
		pathParams:    mux.Vars(r),
		TransactionID: xTId,
		SessionID:     xSId,
		RequestID:     xRId,
	}
}

// Param returns the query parameter with the given key.
func (r *Request) Param(key string) string {
	return r.req.URL.Query().Get(key)
}

func (r *Request) UserAgent() string {
	return r.req.UserAgent()
}

// Method returns the HTTP method of the request.
func (r *Request) Method() string {
	return r.req.Method
}

// Referer returns the referer of the request.
func (r *Request) Referer() string {
	return r.req.Referer()
}

// URL returns the URL of the request.
func (r *Request) URL() string {
	if r.req.URL == nil {
		return ""
	}

	url := r.req.URL.String()
	if r.req.URL.RawQuery != "" {
		url += "?" + r.req.URL.RawQuery
	}
	return url
}

// Header returns the value of the specified header key.
func (r *Request) Header(key string) string {
	return r.req.Header.Get(key)
}

// Headers returns all headers of the request.
func (r *Request) Headers() map[string]any {
	headers := make(map[string]any)
	for key, values := range r.req.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		} else {
			headers[key] = nil
		}
	}
	return headers
}

// Context returns the context of the request.
func (r *Request) Context() context.Context {
	return r.req.Context()
}

func (r *Request) TransactionId() string {
	return r.TransactionID
}

// SessionId returns the session ID from the request, if available.
func (r *Request) SessionId() string {
	return r.SessionID
}

// RequestId returns the request ID from the request, if available.
func (r *Request) RequestId() string {
	return r.RequestID
}

// PathParam retrieves a path parameter from the request.
func (r *Request) PathParam(key string) string {
	return r.pathParams[key]
}

// Bind parses the request body and binds it to the provided interface.
func (r *Request) Bind(i any) error {
	v := r.req.Header.Get("Content-Type")
	contentType := strings.Split(v, ";")[0]

	switch contentType {
	case "application/json":
		body, err := r.body()
		if err != nil {
			return err
		}

		return json.Unmarshal(body, &i)
	case "multipart/form-data":
		return r.bindMultipart(i)
	case "application/x-www-form-urlencoded":
		return r.bindFormURLEncoded(i)
	case "binary/octet-stream":
		return r.bindBinary(i)
	}

	return nil
}

// HostName retrieves the hostname from the request.
func (r *Request) HostName() string {
	proto := r.req.Header.Get("X-Forwarded-Proto")
	if proto == "" {
		proto = "http"
	}

	return fmt.Sprintf("%s://%s", proto, r.req.Host)
}

func (r *Request) ClientIP() string {
	xff := r.req.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	ip := r.req.RemoteAddr
	if colon := strings.LastIndex(ip, ":"); colon != -1 {
		ip = ip[:colon]
	}
	return ip
}

// Params returns a slice of strings containing the values associated with the given query parameter key.
// If the parameter is not present, an empty slice is returned.
func (r *Request) Params(key string) []string {
	values := r.req.URL.Query()[key]

	var result []string

	for _, value := range values {
		result = append(result, strings.Split(value, ",")...)
	}

	return result
}

func (r *Request) body() ([]byte, error) {
	bodyBytes, err := io.ReadAll(r.req.Body)
	if err != nil {
		return nil, err
	}

	r.req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	return bodyBytes, nil
}

func (r *Request) bindMultipart(ptr any) error {
	return r.bindForm(ptr, true)
}

func (r *Request) bindFormURLEncoded(ptr any) error {
	return r.bindForm(ptr, false)
}

func (r *Request) bindForm(ptr any, isMultipart bool) error {
	ptrVal := reflect.ValueOf(ptr)
	if ptrVal.Kind() != reflect.Ptr {
		return errNonPointerBind
	}

	ptrVal = ptrVal.Elem()

	var fd formData

	if isMultipart {
		if err := r.req.ParseMultipartForm(defaultMaxMemory); err != nil {
			return err
		}

		fd = formData{files: r.req.MultipartForm.File, fields: r.req.MultipartForm.Value}
	} else {
		if err := r.req.ParseForm(); err != nil {
			return err
		}

		fd = formData{fields: r.req.Form}
	}

	ok, err := fd.mapStruct(ptrVal, nil)
	if err != nil {
		return err
	}

	if !ok {
		if isMultipart {
			return errNoFileFound
		}

		return errFieldsNotSet
	}

	return nil
}

// bindBinary handles binding for binary/octet-stream content type.
func (r *Request) bindBinary(raw any) error {
	// Ensure raw is a pointer to a byte slice
	byteSlicePtr, ok := raw.(*[]byte)
	if !ok {
		return fmt.Errorf("%w: %v", errNonSliceBind, raw)
	}

	body, err := r.body()
	if err != nil {
		return fmt.Errorf("failed to read request body: %w", err)
	}

	// Assign the body to the provided slice
	*byteSlicePtr = body

	return nil
}
