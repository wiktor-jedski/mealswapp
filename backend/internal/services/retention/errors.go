package retention

import "errors"

var (
	ErrInvalidBackupRetention = errors.New("backup retention must be 30 days")
	ErrInvalidRule            = errors.New("retention rule is invalid")
	ErrMissingRule            = errors.New("retention policy is missing a required data class")
)
