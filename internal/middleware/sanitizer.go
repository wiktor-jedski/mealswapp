// Phase: phase-01 | Task: 11 | Architecture: ARCH-013 | Design: InputSanitizer

package middleware

import (
	"log"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type SanitizationConfig struct {
	AllowHTML         bool
	AllowedTags       []string
	AllowedAttributes map[string][]string
	MaxInputLength    int
	StripNullBytes    bool
	EscapeSQL         bool
	EscapeShell       bool
}

type SanitizationResult struct {
	Value     interface{}
	IsValid   bool
	Sanitized bool
	Warnings  []string
	Errors    []SanitizationError
}

type SanitizationError struct {
	Field   string
	Code    string
	Message string
	Value   interface{}
}

type Sanitizer struct {
	config SanitizationConfig
	logger *log.Logger
}

type InputType string

const (
	InputTypeString InputType = "string"
	InputTypeNumber InputType = "number"
	InputTypeBool   InputType = "bool"
	InputTypeArray  InputType = "array"
	InputTypeObject InputType = "object"
	InputTypeEmail  InputType = "email"
	InputTypeURL    InputType = "url"
	InputTypeHTML   InputType = "html"
)

type ValidationRule struct {
	Field     string
	InputType InputType
	Required  bool
	MinLength *int
	MaxLength *int
	MinValue  *float64
	MaxValue  *float64
	Pattern   *regexp.Regexp
	Custom    func(interface{}) bool
}

type FiberMiddlewareConfig struct {
	BodyFields    map[string]ValidationRule
	QueryFields   map[string]ValidationRule
	ParamsFields  map[string]ValidationRule
	HeadersFields map[string]ValidationRule
	OnError       func(c *fiber.Ctx, errors []SanitizationError) error
	Skipper       func(c *fiber.Ctx) bool
}

var (
	xssPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`),
		regexp.MustCompile(`(?i)<script[^>]*>`),
		regexp.MustCompile(`(?i)javascript:`),
		regexp.MustCompile(`(?i)on\w+\s*=`),
		regexp.MustCompile(`(?i)<iframe[^>]*>`),
		regexp.MustCompile(`(?i)<object[^>]*>`),
		regexp.MustCompile(`(?i)<embed[^>]*>`),
		regexp.MustCompile(`(?i)<svg[^>]*>`),
		regexp.MustCompile(`(?i)data:text/html`),
		regexp.MustCompile(`(?i)expression\s*\(`),
		regexp.MustCompile(`(?i)<meta[^>]*http-equiv`),
		regexp.MustCompile(`(?i)<link[^>]*rel`),
	}

	sqlInjectionPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(\bUNION\b|\bSELECT\b|\bINSERT\b|\bUPDATE\b|\bDELETE\b|\bDROP\b|\bTRUNCATE\b|\bALTER\b)`),
		regexp.MustCompile(`(?i)--`),
		regexp.MustCompile(`(?i)/\*|\*/`),
		regexp.MustCompile(`(?i)\bOR\b\s+1\s*=\s*1`),
		regexp.MustCompile(`(?i)\bAND\b\s+1\s*=\s*1`),
		regexp.MustCompile(`(?i)\bSLEEP\b\s*\(`),
		regexp.MustCompile(`(?i)\bBENCHMARK\b\s*\(`),
		regexp.MustCompile(`(?i)information_schema`),
		regexp.MustCompile(`(?i)\bEXEC\b\s*\(|\bEXECUTE\b\s*\(`),
	}

	shellCommandPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(rm|del|erase|unlink|mkfs|format)\b`),
		regexp.MustCompile(`(?i)\b(wget|curl|nc|netcat)\b`),
		regexp.MustCompile(`(?i)\b(sh|bash|powershell|cmd)\b`),
		regexp.MustCompile(`(?i)\b(sudo|su|passwd)\b`),
	}

	shellMetacharacters = []string{
		";", "|", "&", "$", "`", "\"", "'", "\\",
		"(", ")", "{", "}", "[", "]", "*", "?",
		"<", ">", "#", "~", "!", "%", "^",
	}

	emailRegex   = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	urlSafeRegex = regexp.MustCompile(`^https?://[^\s]+$`)
)

func DefaultConfig() SanitizationConfig {
	return SanitizationConfig{
		AllowHTML:         false,
		AllowedTags:       []string{},
		AllowedAttributes: map[string][]string{},
		MaxInputLength:    10000,
		StripNullBytes:    true,
		EscapeSQL:         true,
		EscapeShell:       true,
	}
}

func StrictConfig() SanitizationConfig {
	return SanitizationConfig{
		AllowHTML:         false,
		AllowedTags:       []string{},
		AllowedAttributes: map[string][]string{},
		MaxInputLength:    1000,
		StripNullBytes:    true,
		EscapeSQL:         true,
		EscapeShell:       true,
	}
}

func HTMLPermissiveConfig(allowedTags []string, allowedAttributes map[string][]string) SanitizationConfig {
	return SanitizationConfig{
		AllowHTML:         true,
		AllowedTags:       allowedTags,
		AllowedAttributes: allowedAttributes,
		MaxInputLength:    10000,
		StripNullBytes:    true,
		EscapeSQL:         true,
		EscapeShell:       true,
	}
}

func NewSanitizer(config SanitizationConfig, logger *log.Logger) *Sanitizer {
	if logger == nil {
		logger = log.Default()
	}
	return &Sanitizer{config: config, logger: logger}
}

func (s *Sanitizer) BlockXSSPatterns(value string) (string, bool) {
	detected := false
	result := value

	for _, pattern := range xssPatterns {
		if pattern.MatchString(result) {
			detected = true
			result = pattern.ReplaceAllString(result, "")
		}
	}

	result = strings.ReplaceAll(result, "&lt;script", "")
	result = strings.ReplaceAll(result, "&lt;script", "")
	result = strings.ReplaceAll(result, "&#60;script", "")

	return result, detected
}

func (s *Sanitizer) BlockSQLInjection(value string) (string, bool) {
	detected := false
	result := value

	for _, pattern := range sqlInjectionPatterns {
		if pattern.MatchString(result) {
			detected = true
			result = pattern.ReplaceAllString(result, "")
		}
	}

	result = strings.ReplaceAll(result, "'", "''")
	result = strings.ReplaceAll(result, "\\", "\\\\")
	result = strings.ReplaceAll(result, ";", "\\;")
	result = strings.ReplaceAll(result, "--", "\\--")
	result = strings.ReplaceAll(result, "/*", "\\/*")
	result = strings.ReplaceAll(result, "*/", "\\*/")

	return result, detected
}

func (s *Sanitizer) BlockShellInjection(value string) (string, bool) {
	detected := false
	result := value

	for _, pattern := range shellCommandPatterns {
		if pattern.MatchString(result) {
			detected = true
			result = pattern.ReplaceAllString(result, "")
		}
	}

	for _, char := range shellMetacharacters {
		result = strings.ReplaceAll(result, char, "\\"+char)
	}

	result = strings.ReplaceAll(result, "\n", "\\n")
	result = strings.ReplaceAll(result, "\r", "\\r")
	result = strings.ReplaceAll(result, "\t", "\\t")

	return result, detected
}

func (s *Sanitizer) SanitizeString(value string, allowHTML bool) (string, []SanitizationError) {
	var errors []SanitizationError

	if s.config.StripNullBytes {
		value = strings.ReplaceAll(value, "\x00", "")
	}

	if len(value) > s.config.MaxInputLength {
		errors = append(errors, SanitizationError{
			Field:   "",
			Code:    "InputTooLong",
			Message: "Input exceeds maximum length",
			Value:   value,
		})
		return value, errors
	}

	if !allowHTML && !s.config.AllowHTML {
		value = escapeHTML(value)
		value, _ = s.BlockXSSPatterns(value)
	}

	if s.config.EscapeSQL {
		value, _ = s.BlockSQLInjection(value)
	}

	if s.config.EscapeShell {
		value, _ = s.BlockShellInjection(value)
	}

	return value, errors
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&#x27;")
	s = strings.ReplaceAll(s, "/", "&#x2F;")
	return s
}

func (s *Sanitizer) SanitizeNumber(value interface{}, min, max *float64) (float64, []SanitizationError) {
	var errors []SanitizationError

	var numStr string
	switch v := value.(type) {
	case float64:
		numStr = strconv.FormatFloat(v, 'f', -1, 64)
	case float32:
		numStr = strconv.FormatFloat(float64(v), 'f', -1, 64)
	case int:
		numStr = strconv.Itoa(v)
	case int64:
		numStr = strconv.FormatInt(v, 10)
	case int32:
		numStr = strconv.FormatInt(int64(v), 10)
	case string:
		numStr = v
	default:
		errors = append(errors, SanitizationError{
			Field:   "",
			Code:    "InvalidType",
			Message: "Cannot parse input as number",
			Value:   value,
		})
		return 0, errors
	}

	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		errors = append(errors, SanitizationError{
			Field:   "",
			Code:    "InvalidType",
			Message: "Invalid number format",
			Value:   value,
		})
		return 0, errors
	}

	if min != nil && num < *min {
		errors = append(errors, SanitizationError{
			Field:   "",
			Code:    "ValueOutOfRange",
			Message: "Number below minimum value",
			Value:   num,
		})
		return num, errors
	}

	if max != nil && num > *max {
		errors = append(errors, SanitizationError{
			Field:   "",
			Code:    "ValueOutOfRange",
			Message: "Number above maximum value",
			Value:   num,
		})
		return num, errors
	}

	if math.IsInf(num, 0) || math.IsNaN(num) {
		errors = append(errors, SanitizationError{
			Field:   "",
			Code:    "ValueOutOfRange",
			Message: "Number is infinite or NaN",
			Value:   num,
		})
		return 0, errors
	}

	return num, errors
}

func (s *Sanitizer) SanitizeArray(arr []interface{}, itemType InputType) ([]interface{}, []SanitizationError) {
	var errors []SanitizationError
	result := make([]interface{}, 0, len(arr))

	for i, item := range arr {
		fieldName := strconv.Itoa(i)
		res := s.Sanitize(item, itemType, fieldName)
		result = append(result, res.Value)
		errors = append(errors, res.Errors...)
	}

	return result, errors
}

func (s *Sanitizer) SanitizeObject(obj map[string]interface{}, rules map[string]ValidationRule) (map[string]interface{}, []SanitizationError) {
	var errors []SanitizationError
	result := make(map[string]interface{})

	for fieldName, rule := range rules {
		value, exists := obj[fieldName]

		if rule.Required && !exists {
			errors = append(errors, SanitizationError{
				Field:   fieldName,
				Code:    "RequiredFieldMissing",
				Message: "Required field is missing",
				Value:   nil,
			})
			continue
		}

		if !exists {
			continue
		}

		res := s.Sanitize(value, rule.InputType, fieldName)
		result[fieldName] = res.Value
		errors = append(errors, res.Errors...)

		if rule.MinLength != nil {
			if str, ok := value.(string); ok && len(str) < *rule.MinLength {
				errors = append(errors, SanitizationError{
					Field:   fieldName,
					Code:    "ValueOutOfRange",
					Message: "String below minimum length",
					Value:   str,
				})
			}
		}

		if rule.MaxLength != nil {
			if str, ok := value.(string); ok && len(str) > *rule.MaxLength {
				errors = append(errors, SanitizationError{
					Field:   fieldName,
					Code:    "ValueOutOfRange",
					Message: "String exceeds maximum length",
					Value:   str,
				})
			}
		}

		if rule.Pattern != nil {
			if str, ok := value.(string); ok && !rule.Pattern.MatchString(str) {
				errors = append(errors, SanitizationError{
					Field:   fieldName,
					Code:    "PatternMismatch",
					Message: "Input does not match required pattern",
					Value:   str,
				})
			}
		}

		if rule.Custom != nil && !rule.Custom(value) {
			errors = append(errors, SanitizationError{
				Field:   fieldName,
				Code:    "CustomValidationFailed",
				Message: "Custom validation failed",
				Value:   value,
			})
		}
	}

	for key, value := range obj {
		if _, exists := rules[key]; !exists {
			res := s.Sanitize(value, InputTypeString, key)
			result[key] = res.Value
			errors = append(errors, res.Errors...)
		}
	}

	return result, errors
}

func (s *Sanitizer) ValidateEmail(email string) (bool, []SanitizationError) {
	var errors []SanitizationError

	email = strings.TrimSpace(email)

	if s.config.StripNullBytes {
		email = strings.ReplaceAll(email, "\x00", "")
	}

	if len(email) == 0 || len(email) > 254 {
		errors = append(errors, SanitizationError{
			Field:   "",
			Code:    "InvalidEmailFormat",
			Message: "Email address is empty or too long",
			Value:   email,
		})
		return false, errors
	}

	email, _ = s.BlockXSSPatterns(email)
	email, _ = s.BlockSQLInjection(email)

	if !emailRegex.MatchString(email) {
		errors = append(errors, SanitizationError{
			Field:   "",
			Code:    "InvalidEmailFormat",
			Message: "Invalid email format",
			Value:   email,
		})
		return false, errors
	}

	return true, errors
}

func (s *Sanitizer) ValidateURL(url string) (bool, []SanitizationError) {
	var errors []SanitizationError

	url = strings.TrimSpace(url)

	if s.config.StripNullBytes {
		url = strings.ReplaceAll(url, "\x00", "")
	}

	if len(url) == 0 || len(url) > 2048 {
		errors = append(errors, SanitizationError{
			Field:   "",
			Code:    "InvalidURLFormat",
			Message: "URL is empty or too long",
			Value:   url,
		})
		return false, errors
	}

	urlLower := strings.ToLower(url)
	if strings.HasPrefix(urlLower, "javascript:") ||
		strings.HasPrefix(urlLower, "data:") ||
		strings.HasPrefix(urlLower, "vbscript:") {
		errors = append(errors, SanitizationError{
			Field:   "",
			Code:    "InvalidURLFormat",
			Message: "URL contains dangerous protocol",
			Value:   url,
		})
		return false, errors
	}

	url, _ = s.BlockXSSPatterns(url)
	url, _ = s.BlockSQLInjection(url)

	if !urlSafeRegex.MatchString(url) {
		errors = append(errors, SanitizationError{
			Field:   "",
			Code:    "InvalidURLFormat",
			Message: "Invalid URL format or unsafe protocol",
			Value:   url,
		})
		return false, errors
	}

	return true, errors
}

func (s *Sanitizer) Sanitize(input interface{}, inputType InputType, fieldName string) SanitizationResult {
	var result SanitizationResult
	result.Warnings = []string{}
	result.Errors = []SanitizationError{}

	switch inputType {
	case InputTypeString:
		if str, ok := input.(string); ok {
			sanitized, errs := s.SanitizeString(str, false)
			result.Value = sanitized
			result.Errors = errs
			result.Sanitized = len(errs) > 0 || sanitized != str
		} else if input != nil {
			result.Value = ""
			result.Errors = append(result.Errors, SanitizationError{
				Field:   fieldName,
				Code:    "InvalidType",
				Message: "Expected string input",
				Value:   input,
			})
		}
		result.IsValid = len(result.Errors) == 0

	case InputTypeHTML:
		if str, ok := input.(string); ok {
			sanitized, errs := s.SanitizeString(str, s.config.AllowHTML)
			result.Value = sanitized
			result.Errors = errs
			result.Sanitized = len(errs) > 0 || sanitized != str
		} else if input != nil {
			result.Value = ""
			result.Errors = append(result.Errors, SanitizationError{
				Field:   fieldName,
				Code:    "InvalidType",
				Message: "Expected string input",
				Value:   input,
			})
		}
		result.IsValid = len(result.Errors) == 0

	case InputTypeNumber:
		num, errs := s.SanitizeNumber(input, nil, nil)
		result.Value = num
		result.Errors = errs
		result.Sanitized = false
		result.IsValid = len(result.Errors) == 0

	case InputTypeBool:
		if b, ok := input.(bool); ok {
			result.Value = b
		} else if str, ok := input.(string); ok {
			lower := strings.ToLower(str)
			result.Value = lower == "true" || lower == "1" || lower == "yes"
		} else {
			result.Value = false
			result.Errors = append(result.Errors, SanitizationError{
				Field:   fieldName,
				Code:    "InvalidType",
				Message: "Expected boolean input",
				Value:   input,
			})
		}
		result.IsValid = len(result.Errors) == 0

	case InputTypeArray:
		if arr, ok := input.([]interface{}); ok {
			sanitized, errs := s.SanitizeArray(arr, InputTypeString)
			result.Value = sanitized
			result.Errors = errs
			result.Sanitized = len(errs) > 0
		} else {
			result.Value = []interface{}{}
			result.Errors = append(result.Errors, SanitizationError{
				Field:   fieldName,
				Code:    "InvalidType",
				Message: "Expected array input",
				Value:   input,
			})
		}
		result.IsValid = len(result.Errors) == 0

	case InputTypeObject:
		if obj, ok := input.(map[string]interface{}); ok {
			sanitized, errs := s.SanitizeObject(obj, map[string]ValidationRule{})
			result.Value = sanitized
			result.Errors = errs
			result.Sanitized = len(errs) > 0
		} else {
			result.Value = map[string]interface{}{}
			result.Errors = append(result.Errors, SanitizationError{
				Field:   fieldName,
				Code:    "InvalidType",
				Message: "Expected object input",
				Value:   input,
			})
		}
		result.IsValid = len(result.Errors) == 0

	case InputTypeEmail:
		if str, ok := input.(string); ok {
			valid, errs := s.ValidateEmail(str)
			result.Value = str
			result.Errors = errs
			result.IsValid = valid
		} else {
			result.Value = ""
			result.Errors = append(result.Errors, SanitizationError{
				Field:   fieldName,
				Code:    "InvalidType",
				Message: "Expected string input for email",
				Value:   input,
			})
			result.IsValid = false
		}

	case InputTypeURL:
		if str, ok := input.(string); ok {
			valid, errs := s.ValidateURL(str)
			result.Value = str
			result.Errors = errs
			result.IsValid = valid
		} else {
			result.Value = ""
			result.Errors = append(result.Errors, SanitizationError{
				Field:   fieldName,
				Code:    "InvalidType",
				Message: "Expected string input for URL",
				Value:   input,
			})
			result.IsValid = false
		}

	default:
		result.Value = input
		result.IsValid = true
	}

	return result
}

func (s *Sanitizer) CreateFiberMiddleware(config FiberMiddlewareConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if config.Skipper != nil && config.Skipper(c) {
			return c.Next()
		}

		var allErrors []SanitizationError

		if config.BodyFields != nil {
			body := make(map[string]interface{})
			if err := c.BodyParser(&body); err == nil && body != nil {
				_, errors := s.SanitizeObject(body, config.BodyFields)
				allErrors = append(allErrors, errors...)
			}
		}

		if config.QueryFields != nil {
			query := c.Queries()
			queryMap := make(map[string]interface{})
			for k, v := range query {
				queryMap[k] = v
			}
			_, errors := s.SanitizeObject(queryMap, config.QueryFields)
			allErrors = append(allErrors, errors...)
		}

		if config.ParamsFields != nil {
			params := c.AllParams()
			paramsMap := make(map[string]interface{})
			for k, v := range params {
				paramsMap[k] = v
			}
			_, errors := s.SanitizeObject(paramsMap, config.ParamsFields)
			allErrors = append(allErrors, errors...)
		}

		if config.HeadersFields != nil {
			headers := make(map[string]interface{})
			c.Request().Header.VisitAll(func(key, value []byte) {
				headers[string(key)] = string(value)
			})
			_, errors := s.SanitizeObject(headers, config.HeadersFields)
			allErrors = append(allErrors, errors...)
		}

		if len(allErrors) > 0 {
			if config.OnError != nil {
				return config.OnError(c, allErrors)
			}
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":  "Invalid input",
				"errors": allErrors,
			})
		}

		return c.Next()
	}
}
