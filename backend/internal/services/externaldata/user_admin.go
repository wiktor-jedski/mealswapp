package externaldata

import (
	"context"
	"errors"
	"strings"

	"mealswapp/backend/internal/repositories"

	"github.com/google/uuid"
)

var ErrUserAdminInvalidUser = errors.New("user id is required")

type UserAdminUserStore interface {
	GetByID(ctx context.Context, id uuid.UUID) (repositories.UserEntity, error)
	List(ctx context.Context, query repositories.PageQuery) ([]repositories.UserEntity, int, error)
	Update(ctx context.Context, user repositories.UserEntity) error
}

type UserAdminEntitlementStore interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (repositories.EntitlementEntity, error)
}

type UserAdminAuditStore interface {
	ListByTarget(ctx context.Context, target string, query repositories.PageQuery) ([]repositories.AuditLogEntity, int, error)
}

type AccountLockoutResetter interface {
	ResetAccount(accountKey string)
}

type UserAdminPanel struct {
	users        UserAdminUserStore
	entitlements UserAdminEntitlementStore
	audits       UserAdminAuditStore
	lockouts     AccountLockoutResetter
}

type UserAdminListResult struct {
	Users []repositories.UserEntity `json:"users"`
	Total int                       `json:"total"`
	Page  int                       `json:"page"`
	Limit int                       `json:"limit"`
}

type UserAdminDetail struct {
	User        repositories.UserEntity         `json:"user"`
	Entitlement *repositories.EntitlementEntity `json:"entitlement,omitempty"`
}

type UserAuditHistory struct {
	Entries []repositories.AuditLogEntity `json:"entries"`
	Total   int                           `json:"total"`
	Page    int                           `json:"page"`
	Limit   int                           `json:"limit"`
}

func NewUserAdminPanel(users UserAdminUserStore, entitlements UserAdminEntitlementStore, audits UserAdminAuditStore, lockouts AccountLockoutResetter) UserAdminPanel {
	return UserAdminPanel{users: users, entitlements: entitlements, audits: audits, lockouts: lockouts}
}

func (panel UserAdminPanel) List(ctx context.Context, query string, page int, limit int) (UserAdminListResult, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	users, total, err := panel.users.List(ctx, repositories.PageQuery{Text: query, Limit: limit, Offset: (page - 1) * limit})
	if err != nil {
		return UserAdminListResult{}, err
	}
	return UserAdminListResult{Users: users, Total: total, Page: page, Limit: limit}, nil
}

func (panel UserAdminPanel) Detail(ctx context.Context, userID uuid.UUID) (UserAdminDetail, error) {
	if userID == uuid.Nil {
		return UserAdminDetail{}, ErrUserAdminInvalidUser
	}
	user, err := panel.users.GetByID(ctx, userID)
	if err != nil {
		return UserAdminDetail{}, err
	}
	detail := UserAdminDetail{User: user}
	if panel.entitlements != nil {
		entitlement, err := panel.entitlements.GetByUserID(ctx, userID)
		if err == nil {
			detail.Entitlement = &entitlement
		}
	}
	return detail, nil
}

func (panel UserAdminPanel) Disable(ctx context.Context, userID uuid.UUID) (repositories.UserEntity, error) {
	if userID == uuid.Nil {
		return repositories.UserEntity{}, ErrUserAdminInvalidUser
	}
	user, err := panel.users.GetByID(ctx, userID)
	if err != nil {
		return repositories.UserEntity{}, err
	}
	user.Disabled = true
	if err := panel.users.Update(ctx, user); err != nil {
		return repositories.UserEntity{}, err
	}
	return user, nil
}

func (panel UserAdminPanel) ResetLockout(ctx context.Context, userID uuid.UUID) error {
	if userID == uuid.Nil {
		return ErrUserAdminInvalidUser
	}
	user, err := panel.users.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if panel.lockouts != nil {
		panel.lockouts.ResetAccount(strings.ToLower(strings.TrimSpace(user.Email)))
	}
	return nil
}

func (panel UserAdminPanel) AuditHistory(ctx context.Context, userID uuid.UUID, page int, limit int) (UserAuditHistory, error) {
	if userID == uuid.Nil {
		return UserAuditHistory{}, ErrUserAdminInvalidUser
	}
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	if panel.audits == nil {
		return UserAuditHistory{Entries: []repositories.AuditLogEntity{}, Page: page, Limit: limit}, nil
	}
	entries, total, err := panel.audits.ListByTarget(ctx, "user:"+userID.String(), repositories.PageQuery{Limit: limit, Offset: (page - 1) * limit})
	if err != nil {
		return UserAuditHistory{}, err
	}
	return UserAuditHistory{Entries: entries, Total: total, Page: page, Limit: limit}, nil
}
