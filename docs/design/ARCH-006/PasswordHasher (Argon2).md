# FILE: PasswordHasher (Argon2).md

**Traceability:** ARCH-006

## 1. Data Structures & Types

```go
package auth

import (
	"golang.org/x/crypto/argon2"
	"crypto/rand"
	"encoding/base64"
	"strings"
)

// Config holds Argon2 configuration parameters
type HasherConfig struct {
	Memory      uint32 // Memory usage in kibibytes (e.g., 64MB = 65536 KiB)
	Iterations  uint32 // Number of iterations (time cost)
	Parallelism uint8  // Number of parallel threads
	SaltLen     uint32 // Salt length in bytes
	KeyLen      uint32 // Derived key length in bytes (output hash length)
}

// PasswordHash represents a stored password hash with its parameters
type PasswordHash struct {
	Hash       string // Base64 encoded hash
	Salt       string // Base64 encoded salt
	Iterations uint32 // Argon2 iterations used
	Memory     uint32 // Memory usage in KiB
	Parallelism uint8 // Parallel threads used
	Variant    argon2.ID // Argon2 variant (id or d)
}

// NewHasherConfig creates a production-ready Argon2 configuration
func NewHasherConfig() HasherConfig {
	return HasherConfig{
		Memory:      65536,    // 64 MiB
		Iterations:  3,        // Number of passes
		Parallelism: 2,        // Parallel threads
		SaltLen:     16,       // 16 bytes salt
		KeyLen:      32,       // 32 bytes = 256-bit hash
	}
}
```

## 2. Logic & Algorithms (Step-by-Step)

### 2.1 Hash Password

```
FUNCTION HashPassword(password string, config HasherConfig) -> PasswordHash, error

1. GENERATE SALT
   - Generate cryptographically random bytes of config.SaltLen
   - If generation fails, RETURN error "failed to generate salt"

2. COMPUTE ARGON2ID HASH
   - Call argon2.IDKey(
       password: password,
       salt: salt,
       time: config.Iterations,
       memory: config.Memory,
       threads: config.Parallelism,
       keyLen: config.KeyLen
     )
   - If computation fails, RETURN error "failed to compute hash"

3. ENCODE TO BASE64
   - Base64 encode the hash bytes -> hashBase64
   - Base64 encode the salt bytes -> saltBase64

4. CONSTRUCT PasswordHash STRUCTURE
   - hash = hashBase64
   - salt = saltBase64
   - iterations = config.Iterations
   - memory = config.Memory
   - parallelism = config.Parallelism
   - variant = argon2.IDKey (Argon2id)

5. RETURN PasswordHash, nil
END FUNCTION
```

### 2.2 Verify Password

```
FUNCTION VerifyPassword(password string, storedHash PasswordHash) -> bool, error

1. DECODE STORED SALT
   - Decode storedHash.Salt from Base64 -> saltBytes
   - If decode fails, RETURN false, error "invalid salt encoding"

2. COMPUTE HASH FROM INPUT PASSWORD
   - Call argon2.IDKey(
       password: password,
       salt: saltBytes,
       time: storedHash.Iterations,
       memory: storedHash.Memory,
       threads: storedHash.Parallelism,
       keyLen: uint32(len(storedHash.Hash)) // decoded length
     )
   - If computation fails, RETURN false, error "hash computation failed"

3. DECODE STORED HASH
   - Decode storedHash.Hash from Base64 -> storedHashBytes
   - If decode fails, RETURN false, error "invalid hash encoding"

4. CONSTANT-TIME COMPARISON
   - Use subtle.ConstantTimeCompare(hashBytes, storedHashBytes)
   - If equal, RETURN true, nil
   - If not equal, RETURN false, nil
END FUNCTION
```

### 2.3 Format and Parse

```
FUNCTION FormatHash(h PasswordHash) -> string
- Concatenate fields with "$" delimiter:
  format: "$argon2id$v=<iterations>,m=<memory>,p=<parallelism>$<salt>$<hash>"
- RETURN formatted string
END FUNCTION

FUNCTION ParseHash(formatted string) -> PasswordHash, error
- Split formatted string by "$"
- Extract parameters from second segment (v, m, p)
- Extract salt from third segment
- Extract hash from fourth segment
- RETURN parsed PasswordHash
END FUNCTION
```

## 3. State Management & Error Handling

### 3.1 Error States

| Error Condition | Error Message | Severity | Recovery Action |
| :--- | :--- | :--- | :--- |
| Random bytes generation fails | "failed to generate salt: [underlying error]" | Critical | Log error, return to caller, do not attempt retry |
| Argon2 computation fails | "failed to compute hash: [underlying error]" | Critical | Log error, return to caller, do not attempt retry |
| Invalid salt encoding in Verify | "invalid salt encoding" | High | Reject authentication, log potential tampering |
| Invalid hash encoding in Verify | "invalid hash encoding" | High | Reject authentication, log potential tampering |
| Base64 decode error | "failed to decode base64: [field]" | Medium | Log warning, treat as failed verification |

### 3.2 State Transitions

```
IDLE -> HASHING
  Trigger: HashPassword() called
  Action: Generate salt, compute Argon2id hash

HASHING -> COMPLETE
  Trigger: Hash computation succeeded
  Action: Return PasswordHash struct with encoded values

HASHING -> ERROR
  Trigger: Salt generation or hash computation failed
  Action: Return error, no state persisted

IDLE -> VERIFYING
  Trigger: VerifyPassword() called
  Action: Decode stored salt, compute candidate hash

VERIFYING -> COMPLETE
  Trigger: Hash comparison completed
  Action: Return boolean result

VERIFYING -> ERROR
  Trigger: Decode failure or computation failure
  Action: Return error, failed verification
```

### 3.3 Security Considerations

- **Constant-time comparison:** Always use `subtle.ConstantTimeCompare` to prevent timing attacks
- **No information leakage:** Error messages must not reveal whether username exists or password is wrong
- **Salt uniqueness:** Each password hash uses a cryptographically random 16-byte salt
- **Config immutability:** HasherConfig should be validated at initialization, not per-operation
- **Memory handling:** Argon2id memory is automatically freed by Go garbage collector

## 4. Component Interfaces

```go
package auth

import (
	"golang.org/x/crypto/argon2"
)

// IPasswordHasher defines the interface for password hashing operations
type IPasswordHasher interface {
	// Hash creates a secure hash of the given password
	Hash(password string) (*PasswordHash, error)

	// Verify checks if the password matches the stored hash
	Verify(password string, storedHash *PasswordHash) (bool, error)

	// Format serializes a password hash to a storage-safe string
	Format(h *PasswordHash) string

	// Parse deserializes a stored hash string back to PasswordHash
	Parse(s string) (*PasswordHash, error)
}

// Argon2Hasher implements IPasswordHasher using Argon2id
type Argon2Hasher struct {
	config HasherConfig
}

// NewArgon2Hasher creates a new Argon2Hasher with production defaults
func NewArgon2Hasher() *Argon2Hasher {
	return &Argon2Hasher{
		config: NewHasherConfig(),
	}
}

// NewArgon2HasherWithConfig creates a new Argon2Hasher with custom configuration
func NewArgon2HasherWithConfig(cfg HasherConfig) *Argon2Hasher {
	return &Argon2Hasher{
		config: cfg,
	}
}

// Hash implements IPasswordHasher.Hash
// Input: password string (plaintext password)
// Output: PasswordHash struct with encoded hash and salt, error if generation fails
func (h *Argon2Hasher) Hash(password string) (*PasswordHash, error) {
	salt := make([]byte, h.config.SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		h.config.Iterations,
		h.config.Memory,
		h.config.Parallelism,
		h.config.KeyLen,
	)

	return &PasswordHash{
		Hash:       base64.RawStdEncoding.EncodeToString(hash),
		Salt:       base64.RawStdEncoding.EncodeToString(salt),
		Iterations: h.config.Iterations,
		Memory:     h.config.Memory,
		Parallelism: h.config.Parallelism,
		Variant:    argon2.IDKey,
	}, nil
}

// Verify implements IPasswordHasher.Verify
// Input: password string (plaintext), storedHash *PasswordHash (previously hashed)
// Output: bool indicating match, error if decoding fails
func (h *Argon2Hasher) Verify(password string, storedHash *PasswordHash) (bool, error) {
	salt, err := base64.RawStdEncoding.DecodeString(storedHash.Salt)
	if err != nil {
		return false, err
	}

	storedHashBytes, err := base64.RawStdEncoding.DecodeString(storedHash.Hash)
	if err != nil {
		return false, err
	}

	candidateHash := argon2.IDKey(
		[]byte(password),
		salt,
		storedHash.Iterations,
		storedHash.Memory,
		storedHash.Parallelism,
		uint32(len(storedHashBytes)),
	)

	return subtle.ConstantTimeCompare(candidateHash, storedHashBytes) == 1, nil
}

// Format implements IPasswordHasher.Format
// Input: PasswordHash struct
// Output: string in format "$argon2id$v=<iter>,m=<mem>,p=<par>$<salt>$<hash>"
func (h *Argon2Hasher) Format(ph *PasswordHash) string {
	return fmt.Sprintf(
		"$argon2id$v=%d,m=%d,p=%d$%s$%s",
		ph.Iterations,
		ph.Memory,
		ph.Parallelism,
		ph.Salt,
		ph.Hash,
	)
}

// Parse implements IPasswordHasher.Parse
// Input: formatted string
// Output: PasswordHash struct, error if format is invalid
func (h *Argon2Hasher) Parse(s string) (*PasswordHash, error) {
	parts := strings.Split(s, "$")
	if len(parts) != 5 {
		return nil, fmt.Errorf("invalid hash format")
	}

	// parts[0] = "" (empty before first $)
	// parts[1] = "argon2id"
	// parts[2] = "v=<iter>,m=<mem>,p=<par>"
	// parts[3] = <salt>
	// parts[4] = <hash>

	params := strings.Split(parts[2], ",")
	if len(params) != 3 {
		return nil, fmt.Errorf("invalid parameters format")
	}

	var iter, mem uint32
	var par uint8
	_, err := fmt.Sscanf(params[0], "v=%d", &iter)
	if err != nil {
		return nil, err
	}
	_, err = fmt.Sscanf(params[1], "m=%d", &mem)
	if err != nil {
		return nil, err
	}
	_, err = fmt.Sscanf(params[2], "p=%d", &par)
	if err != nil {
		return nil, err
	}

	return &PasswordHash{
		Hash:       parts[4],
		Salt:       parts[3],
		Iterations: iter,
		Memory:     mem,
		Parallelism: par,
		Variant:    argon2.IDKey,
	}, nil
}
```

## 5. Usage Example

```go
// Initialization (typically in wire/bootstrap)
hasher := auth.NewArgon2Hasher()

// Registration flow
func RegisterUser(email, password string) error {
	hash, err := hasher.Hash(password)
	if err != nil {
		return fmt.Errorf("password hashing failed: %w", err)
	}
	
	// Store hash.UserID = userID, hash.Salt, hash.Iterations, etc. in user record
	return userRepository.Create(email, hash)
}

// Login flow
func LoginUser(email, password string) (*Token, error) {
	user, err := userRepository.FindByEmail(email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	
	matched, err := hasher.Verify(password, user.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("verification failed: %w", err)
	}
	
	if !matched {
		return nil, ErrInvalidCredentials
	}
	
	// Issue JWT tokens
	return tokenIssuer.IssueTokens(user.ID)
}
```
