# FILE: UserAdminPanel.md
**Traceability:** ARCH-009

---

## 1. Data Structures & Types

### Go Backend Types

```go
// UserAdminPanel.go

type UserRole string

const (
    RoleUser  UserRole = "user"
    RoleAdmin UserRole = "admin"
)

type UserStatus string

const (
    StatusActive   UserStatus = "active"
    StatusInactive UserStatus = "inactive"
    StatusSuspended UserStatus = "suspended"
    StatusPending  UserStatus = "pending"
)

type User struct {
    ID          string     `json:"id" db:"id"`
    Email       string     `json:"email" db:"email"`
    Name        string     `json:"name" db:"name"`
    Role        UserRole   `json:"role" db:"role"`
    Status      UserStatus `json:"status" db:"status"`
    CreatedAt   time.Time  `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
    LastLoginAt *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
}

type UpdateUserRequest struct {
    UserID  string     `json:"user_id" validate:"required,uuid"`
    Email   *string    `json:"email,omitempty" validate:"omitempty,email"`
    Name    *string    `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
    Role    *UserRole  `json:"role,omitempty" validate:"omitempty,oneof=user admin"`
    Status  *UserStatus `json:"status,omitempty" validate:"omitempty,oneof=active inactive suspended pending"`
}

type CreateUserRequest struct {
    Email string    `json:"email" validate:"required,email"`
    Name  string    `json:"name" validate:"required,min=1,max=255"`
    Role  UserRole  `json:"role" validate:"required,oneof=user admin"`
}

type UserListResponse struct {
    Users      []UserSummary `json:"users"`
    TotalCount int           `json:"total_count"`
    Page       int           `json:"page"`
    PageSize   int           `json:"page_size"`
    TotalPages int           `json:"total_pages"`
}

type UserSummary struct {
    ID          string     `json:"id"`
    Email       string     `json:"email"`
    Name        string     `json:"name"`
    Role        UserRole   `json:"role"`
    Status      UserStatus `json:"status"`
    CreatedAt   time.Time  `json:"created_at"`
}

type UserActivity struct {
    UserID       string    `json:"user_id"`
    ActivityType string    `json:"activity_type"`
    Description  string    `json:"description"`
    Timestamp    time.Time `json:"timestamp"`
    Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

type UserStats struct {
    TotalUsers      int `json:"total_users"`
    ActiveUsers     int `json:"active_users"`
    SuspendedUsers  int `json:"suspended_users"`
    PendingUsers    int `json:"pending_users"`
    NewUsersToday   int `json:"new_users_today"`
    AdminsCount     int `json:"admins_count"`
}
```

### Svelte Frontend Types

```typescript
// UserAdminPanel.svelte.ts

interface AdminUser {
    id: string;
    email: string;
    name: string;
    role: 'user' | 'admin';
    status: 'active' | 'inactive' | 'suspended' | 'pending';
    created_at: string;
    last_login_at?: string;
}

interface UserFilters {
    search: string;
    role?: 'user' | 'admin' | 'all';
    status?: 'active' | 'inactive' | 'suspended' | 'pending' | 'all';
    dateFrom?: string;
    dateTo?: string;
}

interface UserListState {
    users: AdminUser[];
    totalCount: number;
    page: number;
    pageSize: number;
    loading: boolean;
    error: string | null;
    filters: UserFilters;
}

interface ActivityLogEntry {
    id: string;
    user_id: string;
    activity_type: 'login' | 'logout' | 'update' | 'create' | 'delete' | 'suspend' | 'activate';
    description: string;
    timestamp: string;
    metadata?: Record<string, unknown>;
}

interface UserStatsState {
    totalUsers: number;
    activeUsers: number;
    suspendedUsers: number;
    pendingUsers: number;
    newUsersToday: number;
    adminsCount: number;
    loading: boolean;
}
```

---

## 2. Logic & Algorithms (Step-by-Step)

### 2.1 List Users Endpoint

**Algorithm: `GET /api/admin/users`**

```
1. EXTRACT query parameters: page, pageSize, search, role, status, dateFrom, dateTo
2. VALIDATE pagination: page >= 1, pageSize in [10, 25, 50, 100], default pageSize = 25
3. BUILD base SQL query:
   SELECT id, email, name, role, status, created_at
   FROM users
   WHERE 1=1
4. IF search is provided:
   ADD clause: AND (email ILIKE $search OR name ILIKE $search)
5. IF role is provided and not 'all':
   ADD clause: AND role = $role
6. IF status is provided and not 'all':
   ADD clause: AND status = $status
7. IF dateFrom is provided:
   ADD clause: AND created_at >= $dateFrom
8. IF dateTo is provided:
   ADD clause: AND created_at <= $dateTo
9. EXECUTE COUNT query for total count with same filters
10. EXECUTE main query with LIMIT/OFFSET
11. RETURN JSON: { users, totalCount, page, pageSize, totalPages }
12. CATCH errors:
   - If database error: return 500 with error message
   - If validation error: return 400 with details
```

### 2.2 Get User Details Endpoint

**Algorithm: `GET /api/admin/users/:id`**

```
1. EXTRACT userID from path parameter
2. VALIDATE userID is valid UUID
3. QUERY database for user:
   SELECT id, email, name, role, status, created_at, updated_at, last_login_at
   FROM users
   WHERE id = $userID
4. IF no user found:
   RETURN 404 with { error: "User not found" }
5. QUERY user activity log for last 10 activities:
   SELECT * FROM user_activities
   WHERE user_id = $userID
   ORDER BY timestamp DESC
   LIMIT 10
6. RETURN JSON: { user, activity_log }
7. CATCH errors:
   - Database error: return 500
   - UUID validation error: return 400
```

### 2.3 Update User Endpoint

**Algorithm: `PATCH /api/admin/users/:id`**

```
1. EXTRACT userID from path and request body
2. VALIDATE userID matches body.user_id
3. VALIDATE request body fields (email, name, role, status)
4. FETCH current user from database
5. IF current user is the same as target user:
   RETURN 400 with { error: "Cannot modify own admin account" }
6. IF role is being changed to 'admin':
   CHECK if requester has admin role (already verified by middleware)
7. BUILD update query dynamically:
   SET updated_at = NOW()
   IF email provided: SET email = new_email
   IF name provided: SET name = new_name
   IF role provided: SET role = new_role
   IF status provided: SET status = new_status
8. EXECUTE UPDATE with updated fields only
9. CREATE activity log entry: { user_id: targetID, type: 'update', description: 'User details updated' }
10. RETURN 200 with { message: "User updated successfully" }
11. CATCH errors:
   - Duplicate email: return 409 with { error: "Email already in use" }
   - Database error: return 500
```

### 2.4 Suspend User Endpoint

**Algorithm: `POST /api/admin/users/:id/suspend`**

```
1. EXTRACT userID from path
2. VALIDATE userID is valid UUID
3. FETCH target user
4. IF target user is admin:
   RETURN 403 with { error: "Cannot suspend admin users" }
5. IF target user is already suspended:
   RETURN 400 with { error: "User is already suspended" }
6. UPDATE user SET status = 'suspended', updated_at = NOW()
7. INVALIDATE user sessions in Redis:
   DEL session:{userID}:*
8. CREATE activity log entry: { user_id: targetID, type: 'suspend', reason: optional_reason }
9. IF email notification enabled:
   SEND suspension email to user
10. RETURN 200 with { message: "User suspended successfully" }
```

### 2.5 Activate User Endpoint

**Algorithm: `POST /api/admin/users/:id/activate`**

```
1. EXTRACT userID from path
2. VALIDATE userID is valid UUID
3. FETCH target user
4. IF target user is not suspended and not inactive:
   RETURN 400 with { error: "User is not suspended or inactive" }
5. UPDATE user SET status = 'active', updated_at = NOW()
6. CREATE activity log entry: { user_id: targetID, type: 'activate', description: 'User account activated' }
7. IF user was pending, send welcome email
8. RETURN 200 with { message: "User activated successfully" }
```

### 2.6 Delete User Endpoint

**Algorithm: `DELETE /api/admin/users/:id`**

```
1. EXTRACT userID from path
2. VALIDATE userID is valid UUID
3. FETCH target user
4. IF target user is admin:
   RETURN 403 with { error: "Cannot delete admin users" }
5. SOFT DELETE approach (recommended):
   - UPDATE user SET status = 'deleted', email = CONCAT(email, '_deleted_', id), updated_at = NOW()
6. HARD DELETE approach (if required):
   - BEGIN TRANSACTION
   - DELETE FROM user_activities WHERE user_id = targetID
   - DELETE FROM user_sessions WHERE user_id = targetID
   - DELETE FROM user_preferences WHERE user_id = targetID
   - DELETE FROM users WHERE id = targetID
   - COMMIT
7. INVALIDATE all user sessions in Redis
8. CREATE activity log entry: { user_id: targetID, type: 'delete', description: 'User account deleted' }
9. RETURN 200 with { message: "User deleted successfully" }
```

### 2.7 Get User Stats Endpoint

**Algorithm: `GET /api/admin/users/stats`**

```
1. EXECUTE parallel queries:
   a. SELECT COUNT(*) FROM users WHERE status = 'active'
   b. SELECT COUNT(*) FROM users WHERE status = 'suspended'
   c. SELECT COUNT(*) FROM users WHERE status = 'pending'
   d. SELECT COUNT(*) FROM users WHERE role = 'admin'
   e. SELECT COUNT(*) FROM users
   f. SELECT COUNT(*) FROM users WHERE created_at >= CURRENT_DATE
2. RETURN aggregated stats object
3. CACHE stats in Redis for 60 seconds to reduce database load
```

---

## 3. State Management & Error Handling

### 3.1 Error States

| Error Type | Condition | HTTP Status | Response |
|------------|-----------|-------------|----------|
| Unauthorized | No valid session/token | 401 | `{ "error": "Authentication required" }` |
| Forbidden | User not admin role | 403 | `{ "error": "Admin access required" }` |
| NotFound | User ID not found | 404 | `{ "error": "User not found" }` |
| Validation | Invalid input format | 400 | `{ "error": "Validation failed", "details": [...] }` |
| Conflict | Email already exists | 409 | `{ "error": "Email already in use" }` |
| DatabaseError | Query execution failed | 500 | `{ "error": "Internal server error" }` |
| RateLimit | Too many requests | 429 | `{ "error": "Too many requests" }` |

### 3.2 State Transitions

```
User Status State Machine:
┌─────────┐     ┌─────────┐     ┌───────────┐
│ pending │────>│  active │<───>│  inactive │
└─────────┘     └────┬────┘     └───────────┘
                     │
                     │ suspend()
                     ▼
                 ┌─────────┐
                 │suspended│
                 └─────────┘
                      │
                      │ activate()
                      ▼
                 ┌─────────┐
                 │  active │
                 └─────────┘

Role Transition:
- user → admin (requires existing admin)
- admin → user (self-demotion not allowed)
```

### 3.3 Frontend State Management (Svelte + TanStack Query)

```typescript
// User list state machine
type UserListAction =
    | { type: 'FETCH_START' }
    | { type: 'FETCH_SUCCESS'; payload: { users: AdminUser[]; totalCount: number } }
    | { type: 'FETCH_ERROR'; error: string }
    | { type: 'SET_FILTERS'; filters: UserFilters }
    | { type: 'SET_PAGE'; page: number }
    | { type: 'DELETE_USER_SUCCESS'; userId: string }
    | { type: 'UPDATE_USER_SUCCESS'; user: AdminUser };

function userListReducer(state: UserListState, action: UserListAction): UserListState {
    switch (action.type) {
        case 'FETCH_START':
            return { ...state, loading: true, error: null };
        case 'FETCH_SUCCESS':
            return { ...state, loading: false, users: action.payload.users, totalCount: action.payload.totalCount };
        case 'FETCH_ERROR':
            return { ...state, loading: false, error: action.error };
        case 'SET_FILTERS':
            return { ...state, filters: action.filters, page: 1 }; // Reset to page 1
        case 'SET_PAGE':
            return { ...state, page: action.page };
        case 'DELETE_USER_SUCCESS':
            return { ...state, users: state.users.filter(u => u.id !== action.userId) };
        case 'UPDATE_USER_SUCCESS':
            return { ...state, users: state.users.map(u => u.id === action.user.id ? action.user : u) };
        default:
            return state;
    }
}
```

### 3.4 Error Boundary Implementation

```svelte
<script lang="ts">
    import { useQuery } from '@tanstack/svelte-query';

    let userId: string;

    const userQuery = useQuery({
        queryKey: ['adminUser', userId],
        queryFn: () => fetchUser(userId),
        retry: 2,
        staleTime: 30000,
    });

    $: if ($userQuery.isError) {
        toast.error($userQuery.error.message);
    }
</script>

{#if $userQuery.isLoading}
    <LoadingSpinner />
{:else if $userQuery.isError}
    <ErrorDisplay message={$userQuery.error.message} onRetry={() => $userQuery.refetch()} />
{:else}
    <UserDetailView user={$userQuery.data} />
{/if}
```

---

## 4. Component Interfaces

### 4.1 Backend Go Interfaces

```go
// UserAdminPanel.go

type UserAdminPanel interface {
    ListUsers(ctx fiber.Ctx) error
    GetUser(ctx fiber.Ctx) error
    CreateUser(ctx fiber.Ctx) error
    UpdateUser(ctx fiber.Ctx) error
    DeleteUser(ctx fiber.Ctx) error
    SuspendUser(ctx fiber.Ctx) error
    ActivateUser(ctx fiber.Ctx) error
    GetUserStats(ctx fiber.Ctx) error
    GetUserActivity(ctx fiber.Ctx) error
}

type UserRepository interface {
    FindByID(id string) (*User, error)
    FindAll(filters UserFilters, page, pageSize int) ([]User, int, error)
    Create(req CreateUserRequest) (*User, error)
    Update(id string, req UpdateUserRequest) error
    SoftDelete(id string) error
    HardDelete(id string) error
    Count(filters UserFilters) (int, error)
    UpdateStatus(id string, status UserStatus) error
}

type UserActivityRepository interface {
    Log(userID, activityType, description string, metadata map[string]interface{}) error
    GetByUserID(userID string, limit int) ([]UserActivity, error)
}

type UserAdminPanelImpl struct {
    userRepo UserRepository
    activityRepo UserActivityRepository
    cache *redis.Client
    emailService EmailService
}
```

### 4.2 Backend Function Signatures

```go
// ListUsers returns paginated list of users
func (p *UserAdminPanelImpl) ListUsers(c *fiber.Ctx) error {
    // Extract pagination and filter params
    // Validate params
    // Call repository with filters
    // Return response
}

// GetUser returns user details with activity log
func (p *UserAdminPanelImpl) GetUser(c *fiber.Ctx) error {
    userID := c.Params("id")
    // Validate UUID
    // Fetch user and activity log
    // Return combined response
}

// UpdateUser modifies user details
func (p *UserAdminPanelImpl) UpdateUser(c *fiber.Ctx) error {
    userID := c.Params("id")
    var req UpdateUserRequest
    // Parse and validate body
    // Check self-modification prevention
    // Update user and log activity
}

// SuspendUser suspends a user account
func (p *UserAdminPanelImpl) SuspendUser(c *fiber.Ctx) error {
    userID := c.Params("id")
    // Validate user exists and is not admin
    // Update status to suspended
    // Invalidate sessions in Redis
    // Send notification email
}

// ActivateUser activates a suspended user
func (p *UserAdminPanelImpl) ActivateUser(c *fiber.Ctx) error {
    userID := c.Params("id")
    // Validate user exists and is suspended
    // Update status to active
    // Log activity
}

// DeleteUser removes a user
func (p *UserAdminPanelImpl) DeleteUser(c *fiber.Ctx) error {
    userID := c.Params("id")
    // Validate user exists and is not admin
    // Perform soft delete
    // Invalidate sessions
    // Log activity
}

// GetUserStats returns user statistics
func (p *UserAdminPanelImpl) GetUserStats(c *fiber.Ctx) error {
    // Fetch counts from cache or database
    // Return stats object
}
```

### 4.3 Svelte Frontend Component Interfaces

```typescript
// UserAdminPanel.svelte.ts

interface UserAdminPanelProps {
    initialFilters?: UserFilters;
}

interface UserTableRow {
    user: AdminUser;
    onEdit: (user: AdminUser) => void;
    onSuspend: (user: AdminUser) => void;
    onActivate: (user: AdminUser) => void;
    onDelete: (user: AdminUser) => void;
}

interface UserFilterPanelProps {
    filters: UserFilters;
    onFilterChange: (filters: UserFilters) => void;
    onReset: () => void;
}

interface UserStatsPanelProps {
    stats: UserStatsState;
    onRefresh: () => void;
}

// API Service Interface
interface UserAdminAPI {
    listUsers(filters: UserFilters, page: number, pageSize: number): Promise<UserListResponse>;
    getUser(id: string): Promise<{ user: AdminUser; activity: ActivityLogEntry[] }>;
    updateUser(id: string, data: Partial<AdminUser>): Promise<AdminUser>;
    suspendUser(id: string, reason?: string): Promise<void>;
    activateUser(id: string): Promise<void>;
    deleteUser(id: string): Promise<void>;
    getStats(): Promise<UserStats>;
}

// Svelte Stores
const userListStore = writable<UserListState>(initialState);
const selectedUserStore = writable<AdminUser | null>(null);
const userStatsStore = writable<UserStatsState>(initialStatsState);
```

### 4.4 Svelte Component Structure

```svelte
<!-- UserAdminPanel.svelte -->
<script lang="ts">
    import { userListStore, userStatsStore } from './stores';
    import UserTable from './UserTable.svelte';
    import UserFilterPanel from './UserFilterPanel.svelte';
    import UserStatsPanel from './UserStatsPanel.svelte';
    import UserDetailModal from './UserDetailModal.svelte';
    import { useQuery, useMutation, useQueryClient } from '@tanstack/svelte-query';

    let showDetailModal = false;
    let selectedUserId: string | null = null;
    const queryClient = useQueryClient();

    const usersQuery = useQuery({
        queryKey: ['adminUsers', $userListStore.filters, $userListStore.page],
        queryFn: () => api.listUsers($userListStore.filters, $userListStore.page, $userListStore.pageSize),
    });

    const statsQuery = useQuery({
        queryKey: ['adminUserStats'],
        queryFn: api.getStats,
        refetchInterval: 60000, // Refresh every minute
    });

    const suspendMutation = useMutation({
        mutationFn: (id: string) => api.suspendUser(id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['adminUsers'] });
            queryClient.invalidateQueries({ queryKey: ['adminUserStats'] });
        },
    });

    function handleEditUser(user: AdminUser) {
        selectedUserId = user.id;
        showDetailModal = true;
    }

    function handleSuspendUser(user: AdminUser) {
        if (confirm(`Are you sure you want to suspend ${user.name}?`)) {
            $suspendMutation.mutate(user.id);
        }
    }
</script>

<div class="user-admin-panel">
    <header>
        <h1>User Administration</h1>
    </header>

    <UserStatsPanel stats={$statsQuery.data} loading={$statsQuery.isLoading} />

    <UserFilterPanel
        filters={$userListStore.filters}
        onFilterChange={(f) => userListStore.update(s => ({ ...s, filters: f, page: 1 }))}
        onReset={() => userListStore.update(s => ({ ...s, filters: initialFilters, page: 1 }))}
    />

    <UserTable
        users={$usersQuery.data?.users || []}
        loading={$usersQuery.isLoading}
        onEdit={handleEditUser}
        onSuspend={handleSuspendUser}
        onActivate={(u) => activateMutation.mutate(u.id)}
        onDelete={(u) => deleteMutation.mutate(u.id)}
    />

    <Pagination
        currentPage={$userListStore.page}
        totalPages={Math.ceil(($usersQuery.data?.totalCount || 0) / $userListStore.pageSize)}
        onPageChange={(p) => userListStore.update(s => ({ ...s, page: p }))}
    />

    {#if showDetailModal}
        <UserDetailModal
            userId={selectedUserId}
            onClose={() => { showDetailModal = false; selectedUserId = null; }}
            onUpdate={(user) => {
                queryClient.invalidateQueries({ queryKey: ['adminUsers'] });
            }}
        />
    {/if}
</div>
```

### 4.5 API Route Definitions (Fiber)

```go
// routes/admin/users.go

func SetupUserAdminRoutes(app *fiber.App, panel UserAdminPanel) {
    users := app.Group("/api/admin/users")
    users.Use(middleware.RequireAdmin())

    users.Get("/", panel.ListUsers)
    users.Get("/stats", panel.GetUserStats)
    users.Get("/:id", panel.GetUser)
    users.Get("/:id/activity", panel.GetUserActivity)
    users.Post("/", panel.CreateUser)
    users.Patch("/:id", panel.UpdateUser)
    users.Post("/:id/suspend", panel.SuspendUser)
    users.Post("/:id/activate", panel.ActivateUser)
    users.Delete("/:id", panel.DeleteUser)
}
```

### 4.6 Middleware Dependencies

```go
// Required middleware for all UserAdminPanel routes

func RequireAdmin() fiber.Handler {
    return func(c *fiber.Ctx) error {
        // 1. Extract session/token from Authorization header or cookie
        // 2. Validate session exists in Redis
        // 3. Fetch user role from session
        // 4. If role != "admin": return 403
        // 5. Set user context for logging
        return c.Next()
    }
}

func AuditLog() fiber.Handler {
    return func(c *fiber.Ctx) error {
        start := time.Now()
        err := c.Next()
        duration := time.Since(start)

        // Log admin action for compliance
        if c.Locals("user_id") != nil {
            logAdminAction(AdminAuditLog{
                UserID:    c.Locals("user_id").(string),
                Action:    c.Method() + " " + c.Path(),
                Status:    c.Response().StatusCode(),
                Duration:  duration,
                IP:        c.IP(),
                UserAgent: c.Get("User-Agent"),
                Timestamp: time.Now(),
            })
        }

        return err
    }
}
```
