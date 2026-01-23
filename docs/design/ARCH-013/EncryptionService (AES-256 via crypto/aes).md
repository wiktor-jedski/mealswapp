# EncryptionService (AES-256 via crypto/aes)

**Traceability:** ARCH-013

---

## 1. Data Structures & Types

### 1.1 Key Types

```go
package encryption

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "encoding/base64"
    "errors"
    "io"
)

type KeySize int

const (
    KeySize256 KeySize = 32 // 256 bits = 32 bytes
)

type EncryptionMode int

const (
    ModeGCM EncryptionMode = iota // Galois/Counter Mode - authenticated encryption
)

type Service struct {
    gcm     cipher.AEAD
    keySize KeySize
}

type EncryptedData struct {
    Ciphertext []byte
    Nonce      []byte
    Version    int
}

type ServiceOption func(*Service)

func WithMode(mode EncryptionMode) ServiceOption {
    return func(s *Service) {
        // Mode configuration - GCM is the only supported mode
    }
}

func WithKeySize(size KeySize) ServiceOption {
    return func(s *Service) {
        s.keySize = size
    }
}
```

### 1.2 Error Types

```go
var (
    ErrInvalidKeySize       = errors.New("encryption: invalid key size")
    ErrInvalidNonceSize     = errors.New("encryption: invalid nonce size")
    ErrEncryptionFailed     = errors.New("encryption: encryption operation failed")
    ErrDecryptionFailed     = errors.New("encryption: decryption operation failed")
    ErrAuthenticationFailed = errors.New("encryption: authentication failed - data may be tampered")
    ErrNilInput             = errors.New("encryption: nil or empty input data")
    ErrKeyNotInitialized    = errors.New("encryption: service not initialized with encryption key")
    ErrDataTooShort         = errors.New("encryption: encrypted data too short")
    ErrUnsupportedVersion   = errors.New("encryption: unsupported encrypted data version")
)
```

### 1.3 Configuration Types

```go
type Config struct {
    MasterKey string // Base64-encoded 256-bit key from GCP Secret Manager
    NonceSize int    // Default: 12 bytes for GCM
}
```

---

## 2. Logic & Algorithms

### 2.1 Service Initialization Flow

```
INITIALIZE ENCRYPTION SERVICE
1.  Receive master key from configuration (Base64-encoded)
2.  Validate key length equals 32 bytes (256 bits)
3.  Decode Base64-encoded key to raw bytes
4.  Create AES-256 block cipher from decoded key
5.  Initialize GCM cipher mode (Galois/Counter Mode)
6.  Store cipher in service struct
7.  Return initialized Service instance
```

**Pseudocode:**
```
function NewService(config Config) (*Service, error):
    if config.MasterKey is empty:
        return nil, ErrKeyNotInitialized
    
    keyBytes := base64Decode(config.MasterKey)
    
    if length(keyBytes) != KeySize256:
        return nil, ErrInvalidKeySize
    
    block, err := aes.NewCipher(keyBytes)
    if err != nil:
        return nil, err
    
    gcm, err := cipher.NewGCMWithNonceSize(block, config.NonceSize)
    if err != nil:
        return nil, err
    
    return &Service{gcm: gcm, keySize: KeySize256}, nil
```

### 2.2 Encryption Flow

```
ENCRYPT DATA (AES-256-GCM)
1.  Validate input plaintext is not nil/empty
2.  Generate cryptographically secure random nonce (12 bytes)
3.  Call GCM Seal operation with nonce + plaintext
4.  Return combined nonce + ciphertext
5.  Include version marker for future compatibility
```

**Pseudocode:**
```
function Encrypt(plaintext []byte) (*EncryptedData, error):
    if plaintext is nil or empty:
        return nil, ErrNilInput
    
    nonce := generateRandomBytes(12)
    
    ciphertext := gcm.Seal(
        dst: make([]byte, 0, len(plaintext)+gcm.Overhead()),
        nonce: nonce,
        plaintext: plaintext,
        additionalData: nil
    )
    
    return &EncryptedData{
        Ciphertext: ciphertext,
        Nonce: nonce,
        Version: 1
    }, nil
```

### 2.3 Decryption Flow

```
DECRYPT DATA (AES-256-GCM)
1.  Validate encrypted data structure is valid
2.  Extract nonce from beginning of ciphertext (first 12 bytes)
3.  Extract actual ciphertext (remaining bytes)
4.  Call GCM Open operation for authenticated decryption
5.  Verify authentication tag during decryption
6.  Return decrypted plaintext on success
7.  Return error if authentication fails (tampering detected)
```

**Pseudocode:**
```
function Decrypt(data *EncryptedData) ([]byte, error):
    if data is nil:
        return nil, ErrNilInput
    
    if data.Version > 1:
        return nil, ErrUnsupportedVersion
    
    if len(data.Ciphertext) < data.NonceSize + gcm.Overhead():
        return nil, ErrDataTooShort
    
    nonce := data.Ciphertext[:data.NonceSize]
    ciphertext := data.Ciphertext[data.NonceSize:]
    
    plaintext, err := gcm.Open(
        dst: make([]byte, 0, len(ciphertext)),
        nonce: nonce,
        ciphertext: ciphertext,
        additionalData: nil
    )
    
    if err != nil:
        return nil, ErrAuthenticationFailed
    
    return plaintext, nil
```

### 2.4 Secure Random Generation

```
GENERATE SECURE RANDOM BYTES
1.  Allocate byte slice of requested size
2.  Use crypto/rand.Reader for cryptographically secure random
3.  Read exactly the requested number of bytes
4.  Return random bytes or error
```

**Pseudocode:**
```
function generateRandomBytes(size int) []byte:
    bytes := make([]byte, size)
    _, err := io.ReadFull(rand.Reader, bytes)
    if err != nil:
        panic("encryption: failed to generate random bytes")
    return bytes
```

---

## 3. State Management & Error Handling

### 3.1 Service States

| State | Condition | Behavior |
| :--- | :--- | :--- |
| **Uninitialized** | gcm is nil | All Encrypt/Decrypt operations fail with ErrKeyNotInitialized |
| **Initialized** | gcm is set | Normal operation - Encrypt/Decrypt available |
| **Error** | Any operation returns error | Service remains usable; caller handles errors |

### 3.2 Error Handling Matrix

| Operation | Error Condition | Handling Strategy |
| :--- | :--- | :--- |
| NewService | Empty master key | Fail startup - no encryption available |
| NewService | Invalid key size | Fail startup - misconfiguration |
| Encrypt | Nil/empty plaintext | Return error; caller must provide valid input |
| Decrypt | Nil encrypted data | Return error; invalid request |
| Decrypt | Ciphertext too short | Return error; possible data corruption |
| Decrypt | GCM Open fails | Return ErrAuthenticationFailed; possible tampering |
| Decrypt | Future version | Return ErrUnsupportedVersion; requires migration |

### 3.3 Security Considerations

- **Never** expose raw encryption key in error messages
- **Always** use authenticated encryption (GCM) to detect tampering
- **Generate** fresh nonce for each encryption operation
- **Never** reuse nonces with the same key
- **Fail fast** on authentication errors - do not attempt recovery
- **Panic** only on internal errors (random generation failure)

### 3.4 State Transitions

```
                    +----------------+
                    |  Uninitialized |
                    +----------------+
                           |
                    Load master key
                           |
                           v
                    +----------------+
    +-------------->|  Initialized   |<---------------+
    |               +----------------+               |
    |                      |                          |
    |               Encrypt/Decrypt                  |
    |                      |                          |
    |                      v                          |
    |               +----------------+               |
    +---------------|   Operational  |<--------------+
                    +----------------+
                           |
                    Any operation error
                    (returns error, state unchanged)
```

---

## 4. Component Interfaces

### 4.1 Public Interface

```go
// NewService creates a new EncryptionService instance with the provided configuration.
// The master key must be a Base64-encoded 256-bit (32-byte) key.
// Returns ErrInvalidKeySize if the decoded key is not exactly 32 bytes.
// Returns ErrKeyNotInitialized if the master key is empty.
func NewService(config Config) (*Service, error)

// Encrypt encrypts plaintext using AES-256-GCM.
// The nonce is generated randomly for each call.
// Returns ErrNilInput if plaintext is nil or empty.
func (s *Service) Encrypt(plaintext []byte) ([]byte, error)

// EncryptToBase64 encrypts plaintext and returns Base64-encoded result.
// Convenience method combining Encrypt and Base64 encoding.
func (s *Service) EncryptToBase64(plaintext []byte) (string, error)

// Decrypt decrypts ciphertext that was encrypted with Encrypt.
// The nonce must be the first bytes of the ciphertext (12 bytes for GCM).
// Returns ErrAuthenticationFailed if the data has been tampered with.
// Returns ErrNilInput if data is nil.
// Returns ErrDataTooShort if ciphertext is too short to contain nonce and auth tag.
func (s *Service) Decrypt(ciphertext []byte) ([]byte, error)

// DecryptFromBase64 decrypts Base64-encoded ciphertext.
func (s *Service) DecryptFromBase64(encoded string) ([]byte, error)
```

### 4.2 Internal Helper Functions

```go
// generateRandomBytes generates cryptographically secure random bytes.
// Panics only if the cryptographic random source fails (should never happen).
func (s *Service) generateRandomBytes(size int) []byte

// encodeBase64 encodes bytes to Base64 string without padding.
func (s *Service) encodeBase64(data []byte) string

// decodeBase64 decodes Base64 string, handling padding.
func (s *Service) decodeBase64(encoded string) ([]byte, error)
```

### 4.3 Usage Examples

#### Basic Encryption/Decryption

```go
config := encryption.Config{
    MasterKey: os.Getenv("ENCRYPTION_MASTER_KEY"),
    NonceSize: 12,
}

service, err := encryption.NewService(config)
if err != nil {
    log.Fatalf("Failed to initialize encryption service: %v", err)
}

// Encrypt
plaintext := []byte("user@example.com")
encrypted, err := service.Encrypt(plaintext)
if err != nil {
    log.Fatalf("Encryption failed: %v", err)
}

// Decrypt
decrypted, err := service.Decrypt(encrypted)
if err != nil {
    log.Fatalf("Decryption failed: %v", err)
}

fmt.Println(string(decrypted) == string(plaintext)) // true
```

#### Database Field Encryption

```go
type User struct {
    ID           int64  `json:"id"`
    Email        []byte `json:"-"` // Encrypted storage
    PhoneNumber  []byte `json:"-"` // Encrypted storage
}

func (r *UserRepository) Create(ctx context.Context, email, phone string) error {
    encryptedEmail, err := r.encryption.Encrypt([]byte(email))
    if err != nil {
        return fmt.Errorf("encrypt email: %w", err)
    }

    encryptedPhone, err := r.encryption.Encrypt([]byte(phone))
    if err != nil {
        return fmt.Errorf("encrypt phone: %w", err)
    }

    _, err = r.db.ExecContext(ctx, `
        INSERT INTO users (email, phone_number) VALUES ($1, $2)
    `, encryptedEmail, encryptedPhone)
    return err
}

func (r *UserRepository) GetEmail(ctx context.Context, id int64) (string, error) {
    var encryptedEmail []byte
    err := r.db.QueryRowContext(ctx, `
        SELECT email FROM users WHERE id = $1
    `, id).Scan(&encryptedEmail)
    if err != nil {
        return "", err
    }

    decrypted, err := r.encryption.Decrypt(encryptedEmail)
    if err != nil {
        return "", fmt.Errorf("decrypt email: %w", err)
    }

    return string(decrypted), nil
}
```

#### Base64 Convenience

```go
// Encrypt to Base64 for JSON storage
plaintext := []byte("PII data")
encoded, err := service.EncryptToBase64(plaintext)
// Result: "a1b2c3d4e5f6..."

// Decrypt from Base64
decrypted, err := service.DecryptFromBase64(encoded)
```

### 4.4 Integration Points

| Component | Interface | Purpose |
| :--- | :--- | :--- |
| Database | Encrypt/Decrypt | Encrypt PII fields before DB insert; decrypt on fetch |
| User Service | EncryptToBase64/DecryptFromBase64 | Handle JSON serialization of encrypted fields |
| Configuration | NewService | Initialize with master key from GCP Secret Manager |

### 4.5 Configuration Requirements

The service requires:
- Master key: 32-byte (256-bit) key encoded as Base64
- Nonce size: 12 bytes (standard for GCM)
- Source: GCP Secret Manager with appropriate IAM permissions

**Environment setup:**
```bash
# Generate a new 256-bit key (32 bytes, Base64 encoded)
openssl rand -base64 32

# Store in GCP Secret Manager
gcloud secrets create encryption-master-key --replication-policy="automatic"
gcloud secrets versions add encryption-master-key --data-file="-"
```
