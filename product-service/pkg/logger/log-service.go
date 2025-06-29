package logger

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type CustomLoggerService interface {
	Init(data LogDto)
	GetLogDto() LogDto
	Update(key string, value any)
	Info(log LoggerAction, data any, options ...MaskingOptionDto)
	Debug(log LoggerAction, data any, options ...MaskingOptionDto)
	Error(log LoggerAction, data any, options ...MaskingOptionDto)
	Flush()
	End(code int, message string)
	SetSummary(params LogEventTag) CustomLoggerService
}
type customLoggerService struct {
	logDto                    LogDto
	isSetSummaryLogParameters bool
	additionalSummary         map[string]any
	summaryLogAdditionalInfo  []Sequence

	detailLog      LoggerService
	summaryLog     LoggerService
	maskingService MaskingServiceInterface
	utilService    *Timer
}

type LogEventTag struct {
	Node        string
	Command     string
	Code        string
	Description string
	ResTime     int64
}

func EventTag(node, command, code, description string) LogEventTag {
	return LogEventTag{
		Node:        node,
		Command:     command,
		Code:        code,
		Description: description,
	}
}

type Timer struct {
	now   int64     // Unix timestamp in milliseconds
	begin time.Time // Duration since the start of the timer
}

func NewTimer() *Timer {
	return &Timer{
		now:   time.Now().UnixNano() / int64(time.Millisecond),
		begin: time.Now(),
	}
}

func NewCustomLogger(detailLog LoggerService, summaryLog LoggerService, time *Timer, maskingService MaskingServiceInterface) CustomLoggerService {
	return &customLoggerService{
		additionalSummary:         make(map[string]any),
		detailLog:                 detailLog,
		summaryLog:                summaryLog,
		maskingService:            maskingService,
		isSetSummaryLogParameters: false,
		utilService:               time,
		logDto:                    LogDto{},
	}
}

func (c *customLoggerService) Init(data LogDto) {
	c.logDto = data
}

func (c *customLoggerService) GetLogDto() LogDto {
	return c.logDto
}
func (c *customLoggerService) Update(key string, value any) {
	v := reflect.ValueOf(&c.logDto).Elem()
	field := v.FieldByName(key)
	if field.IsValid() && field.CanSet() {
		field.Set(reflect.ValueOf(value))
	}
}

func (c *customLoggerService) Info(action LoggerAction, data any, options ...MaskingOptionDto) {
	c.detailLog.Info(c.toStr(action, data, options...))
	c.logDto.Metadata = Metadata{}
	c.logDto.SubAction = ""
}

func (c *customLoggerService) toStr(action LoggerAction, data any, options ...MaskingOptionDto) string {
	cloned := cloneAndMask(data, options, c.maskingService)
	c.logDto.Action = action.Action
	c.logDto.ActionDescription = action.ActionDescription
	c.logDto.SubAction = action.SubAction
	c.logDto.Message = toJSON(cloned)
	c.logDto.Timestamp = ptrTime(time.Now())
	jsonBytes, err := json.MarshalIndent(c.logDto, "", "  ")
	if err != nil {
		jsonBytes = []byte(fmt.Sprintf(`{"error": "%s"}`, err.Error()))
	}
	return string(jsonBytes)
}
func (c *customLoggerService) Debug(action LoggerAction, data any, options ...MaskingOptionDto) {
	c.detailLog.Debug(c.toStr(action, data, options...))
	c.logDto.Metadata = Metadata{}
	c.logDto.SubAction = ""
}
func (c *customLoggerService) Error(action LoggerAction, data any, options ...MaskingOptionDto) {
	c.detailLog.Error(c.toStr(action, data, options...))
	c.logDto.Metadata = Metadata{}
	c.logDto.SubAction = ""
}
func (c *customLoggerService) Flush() {
	if c.detailLog != nil {
		c.detailLog.Sync()
	}
	if c.summaryLog != nil {
		c.summaryLog.Sync()
	}
}

func (c *customLoggerService) SetSummary(param LogEventTag) CustomLoggerService {
	if c.summaryLogAdditionalInfo == nil {
		c.summaryLogAdditionalInfo = make([]Sequence, 0)
	}

	if param.Command == "" && c.logDto.ActionDescription != "" {
		param.Command = c.logDto.ActionDescription
	}

	sequenceResult := []SequenceResult{{
		Result:  param.Code,
		Desc:    param.Description,
		ResTime: param.ResTime,
	}}

	if len(c.summaryLogAdditionalInfo) > 0 {
		for i, seq := range c.summaryLogAdditionalInfo {
			if seq.Node == param.Node && seq.Command == param.Command {
				// Update existing sequence result
				seq.Result = append(seq.Result, SequenceResult{
					Result:  param.Code,
					Desc:    param.Description,
					ResTime: param.ResTime,
				})
				c.summaryLogAdditionalInfo[i] = seq
				return c
			}
		}
	}
	c.summaryLogAdditionalInfo = append(c.summaryLogAdditionalInfo, Sequence{
		Node:    param.Node,
		Command: param.Command,
		Result:  sequenceResult,
	})
	return c

}

type resultCodeType struct {
	StatusCode string `json:"status"`
	ResultCode string `json:"resultCode"`
	Message    string `json:"message"`
}

func ConvertTTTTT(input string) string {
	return input + strings.Repeat("0", 5-len(input))

}

func mapHTTPStatusToSnakeCaseText(code int) string {
	text := http.StatusText(code)
	if text == "" {
		return "unknown"
	}
	return toSnakeCase(text)
}

// Converts "Not Found" → "not_found"
func toSnakeCase(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "-", "_")
	return s
}

func expandResultCode(code int) resultCodeType {
	switch code {
	case http.StatusOK:
		return resultCodeType{
			StatusCode: strconv.Itoa(http.StatusOK),
			ResultCode: ConvertTTTTT(strconv.Itoa(code)),
			Message:    "Success",
		}
	case http.StatusBadRequest:
		return resultCodeType{
			StatusCode: strconv.Itoa(http.StatusBadRequest),
			ResultCode: ConvertTTTTT(strconv.Itoa(code)),
			Message:    mapHTTPStatusToSnakeCaseText(code),
		}
	case http.StatusUnauthorized:
		return resultCodeType{
			StatusCode: strconv.Itoa(http.StatusUnauthorized),
			ResultCode: ConvertTTTTT(strconv.Itoa(code)),
			Message:    mapHTTPStatusToSnakeCaseText(code),
		}
	case http.StatusForbidden:
		return resultCodeType{
			StatusCode: strconv.Itoa(http.StatusForbidden),
			ResultCode: ConvertTTTTT(strconv.Itoa(code)),
			Message:    mapHTTPStatusToSnakeCaseText(code),
		}
	case http.StatusNotFound:
		return resultCodeType{
			StatusCode: strconv.Itoa(http.StatusNotFound),
			ResultCode: ConvertTTTTT(strconv.Itoa(code)),
			Message:    mapHTTPStatusToSnakeCaseText(code),
		}
	case http.StatusInternalServerError:
		return resultCodeType{
			StatusCode: strconv.Itoa(http.StatusInternalServerError),
			ResultCode: ConvertTTTTT(strconv.Itoa(code)),
			Message:    mapHTTPStatusToSnakeCaseText(code),
		}
	case http.StatusServiceUnavailable:
		return resultCodeType{
			StatusCode: strconv.Itoa(http.StatusServiceUnavailable),
			ResultCode: ConvertTTTTT(strconv.Itoa(code)),
			Message:    mapHTTPStatusToSnakeCaseText(code),
		}
	case http.StatusGatewayTimeout:
		return resultCodeType{
			StatusCode: strconv.Itoa(http.StatusGatewayTimeout),
			ResultCode: ConvertTTTTT(strconv.Itoa(code)),
			Message:    mapHTTPStatusToSnakeCaseText(code),
		}
	case http.StatusConflict:
		return resultCodeType{
			StatusCode: strconv.Itoa(http.StatusConflict),
			ResultCode: ConvertTTTTT(strconv.Itoa(code)),
			Message:    mapHTTPStatusToSnakeCaseText(code),
		}
	case http.StatusTooManyRequests:
		return resultCodeType{
			StatusCode: strconv.Itoa(http.StatusTooManyRequests),
			ResultCode: ConvertTTTTT(strconv.Itoa(code)),
			Message:    mapHTTPStatusToSnakeCaseText(code),
		}
	case http.StatusNotImplemented:
		return resultCodeType{
			StatusCode: strconv.Itoa(http.StatusNotImplemented),
			ResultCode: ConvertTTTTT(strconv.Itoa(code)),
			Message:    mapHTTPStatusToSnakeCaseText(code),
		}
	default:
		return resultCodeType{
			StatusCode: "Error",
			ResultCode: ConvertTTTTT(strconv.Itoa(code)),
			Message:    mapHTTPStatusToSnakeCaseText(code),
		}
	}
}

func (c *customLoggerService) End(code int, message string) {
	result := expandResultCode(code)
	if message == "" {
		message = result.Message
	}
	stack := Stack{
		Code:    result.ResultCode,
		Message: message,
		Status:  result.StatusCode,
	}
	summaryLog := NewSummaryLogService(c.summaryLog, c, c.maskingService)
	summaryLog.Init(c.logDto)
	summaryLog.Flush(stack)
	summaryLog = nil
	c.logDto = LogDto{} // Reset logDto after flushing
	c = nil
}

func isArrayOrSlice(data any) bool {
	kind := reflect.TypeOf(data).Kind()
	return kind == reflect.Slice || kind == reflect.Array
}

func cloneAndMask(data any, options []MaskingOptionDto, masker MaskingServiceInterface) any {
	if len(options) == 0 {
		return data
	}

	raw, err := json.Marshal(data)
	if err != nil {
		return data
	}

	// check is array
	if isArrayOrSlice(data) {
		var clones []map[string]any
		if err := json.Unmarshal(raw, &clones); err != nil {
			return data
		}

		for i := range clones {
			for _, opt := range options {
				if strings.Contains(opt.MaskingField, "*") {
					root := strings.TrimSuffix(strings.Split(opt.MaskingField, "*")[0], ".")
					lookupArr := GetObjectByStringKeys(clones[i], root)
					for index := range lookupArr {
						field := strings.Replace(opt.MaskingField, "*", strconv.Itoa(index), 1)
						SetNestedArrayProperty(clones[i], field, opt.MaskingType, masker)
					}
				} else {
					setNestedProperty(clones[i], opt.MaskingField, opt.MaskingType, masker)
				}
			}
		}

		return clones
	}

	var clone map[string]any
	if err := json.Unmarshal(raw, &clone); err != nil {
		return data
	}

	for _, opt := range options {
		if opt.IsArray && strings.Contains(opt.MaskingField, "*") {
			root := strings.TrimSuffix(strings.Split(opt.MaskingField, "*")[0], ".")
			suffix := strings.Split(opt.MaskingField, "*")[1]
			suffix = strings.TrimPrefix(suffix, ".")
			lookupArr := GetObjectByStringKeys(clone, root)
			for _, item := range lookupArr {
				elem, ok := item.(map[string]any)
				if !ok {
					continue
				}
				setNestedProperty(elem, suffix, opt.MaskingType, masker)
			}
			continue
		}
		setNestedProperty(clone, opt.MaskingField, opt.MaskingType, masker)
	}

	return clone
}

func SetNestedArrayProperty(obj map[string]any, path string, maskingType MaskingType, masker MaskingServiceInterface) {
	keys := strings.Split(path, ".")
	var current any = obj

	for i := 0; i < len(keys)-1; i++ {
		key := keys[i]
		currentMap, ok := current.(map[string]any)
		if !ok {
			return
		}
		current = currentMap[key]
	}

	// Last key
	lastKey := keys[len(keys)-1]
	parentMap, ok := current.(map[string]any)
	if !ok {
		return
	}

	// Handle if parentMap[lastKey] is array of maps
	arr, ok := parentMap[lastKey].([]any)
	if !ok {
		return
	}
	for i, v := range arr {
		elem, ok := v.(map[string]any)
		if !ok {
			continue
		}
		oldVal, _ := elem["key1"].(string)
		elem["key1"] = masker.Masking(oldVal, maskingType)
		arr[i] = elem
	}
	parentMap[lastKey] = arr
}

func GetObjectByStringKeys(obj map[string]any, path string) []any {
	keys := strings.Split(path, ".")
	current := any(obj)
	for _, key := range keys {
		switch cur := current.(type) {
		case map[string]any:
			current = cur[key]
		case []any:
			// If the key is an index in brackets e.g. [0], handle it here (optional)
			return nil // or handle properly
		default:
			return nil
		}
		if current == nil {
			return nil
		}
	}
	if arr, ok := current.([]any); ok {
		return arr
	}
	return nil
}

func setNestedProperty(obj map[string]any, path string, maskType MaskingType, masker MaskingServiceInterface) {
	keys := strings.Split(path, ".")
	current := obj
	for i := 0; i < len(keys)-1; i++ {
		if next, ok := current[keys[i]].(map[string]any); ok {
			current = next
		} else {
			return
		}
	}
	lastKey := keys[len(keys)-1]
	if val, ok := current[lastKey].(string); ok {
		current[lastKey] = masker.Masking(val, maskType)
	}
}

func toJSON(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case int, int64, float64, bool:
		return strings.TrimSpace(strings.ToLower(fmt.Sprintf("%v", val)))
	default:
		if v == nil {
			return ""
		}

		// b, _ := json.Marshal(v)
		jsonStr, _ := json.Marshal(v)
		strings := string(jsonStr)

		return strings
	}
}
