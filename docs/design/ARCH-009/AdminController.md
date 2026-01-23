# AdminController

**Traceability:** ARCH-009

## 1. Data Structures & Types

```go
package controller

import (
    "github.com/gofiber/fiber/v2"
    "mealswapp/internal/models"
    "mealswapp/internal/service"
)

// AdminController handles administrative operations for data curation, user management, and tag management.
type AdminController struct {
    dataImporter      *service.DataImporter
    itemCurator       *service.ItemCurator
    tagManager        *service.TagManager
    userAdminPanel    *service.UserAdminPanel
    externalSearch    *service.ExternalSearchProxy
    authService       service.AuthService
}

// NewAdminController creates a new AdminController instance with all required service dependencies.
func NewAdminController(
    dataImporter *service.DataImporter,
    itemCurator *service.ItemCurator,
    tagManager *service.TagManager,
    userAdminPanel *service.UserAdminPanel,
    externalSearch *service.ExternalSearchProxy,
    authService service.AuthService,
) *AdminController {
    return &AdminController{
        dataImporter:   dataImporter,
        itemCurator:    itemCurator,
        tagManager:     tagManager,
        userAdminPanel: userAdminPanel,
        externalSearch: externalSearch,
        authService:    authService,
    }
}

// adminMiddleware validates that the requesting user has 'Admin' role.
// Returns 403 Forbidden if user is not authenticated or lacks admin privileges.
func (c *AdminController) adminMiddleware(ctx *fiber.Ctx) error {
    userID, err := c.authService.GetCurrentUserID(ctx)
    if err != nil {
        return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
            "error": "Unauthorized: authentication required",
        })
    }

    hasRole, err := c.authService.HasRole(ctx, userID, "admin")
    if err != nil || !hasRole {
        return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
            "error": "Forbidden: admin role required",
        })
    }

    return ctx.Next()
}
```

```go
package models

// Item represents a food item with nutritional information, images, and tags.
type Item struct {
    ID          string    `json:"id" db:"id"`
    Name        string    `json:"name" db:"name"`
    Description string    `json:"description" db:"description"`
    Macros      Macros    `json:"macros"`
    Images      []Image   `json:"images"`
    Tags        []Tag     `json:"tags"`
    CreatedAt   time.Time `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// Macros represents nutritional macro-nutrients for a food item.
type Macros struct {
    Calories   float64 `json:"calories" db:"calories"`
    Protein    float64 `json:"protein" db:"protein"`
    Carbohydrates float64 `json:"carbohydrates" db:"carbohydrates"`
    Fat        float64 `json:"fat" db:"fat"`
    Fiber      float64 `json:"fiber" db:"fiber"`
    Sugar      float64 `json:"sugar" db:"sugar"`
    Sodium     float64 `json:"sodium" db:"sodium"`
}

// Image represents an image associated with a food item.
type Image struct {
    ID        string `json:"id" db:"id"`
    URL       string `json:"url" db:"url"`
    IsPrimary bool   `json:"is_primary" db:"is_primary"`
}

// Tag represents a categorization or functional tag for items.
type Tag struct {
    ID             string    `json:"id" db:"id"`
    Name           string    `json:"name" db:"name"`
    Type           TagType   `json:"type"`
    ParentTagID    *string   `json:"parent_tag_id,omitempty" db:"parent_tag_id"`
    CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// TagType defines the category of a tag.
type TagType string

const (
    TagTypeCategory     TagType = "category"
    TagTypeFunctionality TagType = "functionality"
    TagTypeDietary      TagType = "dietary"
)

// ExternalSearchResult represents an uncurated item returned from external APIs.
type ExternalSearchResult struct {
    ExternalID    string            `json:"external_id"`
    Source        string            `json:"source"` // "USDA" or "OpenFoodFacts"
    Name          string            `json:"name"`
    Description   string            `json:"description,omitempty"`
    Macros        Macros            `json:"macros"`
    Images        []ExternalImage   `json:"images,omitempty"`
    RawData       map[string]interface{} `json:"raw_data"`
}

// ExternalImage represents an image from external data sources.
type ExternalImage struct {
    URL      string `json:"url"`
    ThumbURL string `json:"thumb_url,omitempty"`
}

// User represents a user account with role information.
type User struct {
    ID        string    `json:"id" db:"id"`
    Email     string    `json:"email" db:"email"`
    Name      string    `json:"name" db:"name"`
    Roles     []string  `json:"roles"`
    IsActive  bool      `json:"is_active" db:"is_active"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
    UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// AuditLog represents an admin action audit entry.
type AuditLog struct {
    ID          string    `json:"id" db:"id"`
    AdminUserID string    `json:"admin_user_id" db:"admin_user_id"`
    Action      string    `json:"action" db:"action"`
    ResourceID  string    `json:"resource_id,omitempty" db:"resource_id"`
    ResourceType string   `json:"resource_type" db:"resource_type"`
    Details     string    `json:"details" db:"details"`
    IPAddress   string    `json:"ip_address" db:"ip_address"`
    CreatedAt   time.Time `json:"created_at" db:"created_at"`
}
```

## 2. Logic & Algorithms

### 2.1 Admin Middleware Validation Flow

```
1. Extract user ID from session/token via authService
2. IF extraction fails:
   - Return 401 Unauthorized with error message
3. Query authService to check if user has 'admin' role
4. IF user lacks admin role:
   - Return 403 Forbidden with error message
5. ELSE:
   - Continue to next handler
```

### 2.2 External Search Flow (SW-REQ-055)

```
1. Admin sends GET /admin/external-search?q={searchTerm}&source={USDA|OFF|both}
2. AdminMiddleware validates admin role
3. ExternalSearchProxy routes request to ARCH-012 (External Data Integration)
4. ARCH-012 queries USDA FoodData Central and/or OpenFoodFacts APIs
5. Results are normalized into []ExternalSearchResult
6. Return normalized results to admin UI for curation
7. Admin selects item, edits fields (name, tags, macros)
8. Admin confirms import via POST /admin/items/import
9. DataImporter:
   a. Receives curated ExternalSearchResult with modifications
   b. Validates required fields (name, macros)
   c. Transforms external data to internal Item model
   d. Calls DataRepository (ARCH-005) to save curated item
   e. Returns created Item with 201 Created status
```

### 2.3 Item CRUD Operations Flow

#### Create Item
```
1. Admin sends POST /admin/items with Item payload
2. AdminMiddleware validates admin role
3. ItemCurator validates payload:
   - Name is required and non-empty
   - Macros contain valid numeric values
   - Tags reference existing tag IDs (if provided)
4. ItemCurator calls DataRepository.CreateItem()
5. DataRepository inserts item into PostgreSQL via ARCH-005
6. Return created Item with 201 Created status
7. Create AuditLog entry for item creation
```

#### Update Item
```
1. Admin sends PUT /admin/items/:id with Item payload
2. AdminMiddleware validates admin role
3. ItemCurator:
   a. Fetches existing item by ID
   b. IF item not found: Return 404 Not Found
   c. Merges updates with existing item
   d. Validates updated fields
4. ItemCurator calls DataRepository.UpdateItem()
5. DataRepository updates item in PostgreSQL via ARCH-005
6. Return updated Item with 200 OK status
7. Create AuditLog entry for item update
```

#### Delete Item
```
1. Admin sends DELETE /admin/items/:id
2. AdminMiddleware validates admin role
3. ItemCurator:
   a. Verifies item exists
   b. IF item not found: Return 404 Not Found
   c. Checks for dependent records (e.g., meal items)
4. ItemCurator calls DataRepository.DeleteItem()
5. DataRepository soft-deletes or removes item from PostgreSQL via ARCH-005
6. Return 204 No Content status
7. Create AuditLog entry for item deletion
```

### 2.4 Tag Management Flow (SW-REQ-056)

#### Create Tag
```
1. Admin sends POST /admin/tags with Tag payload
2. AdminMiddleware validates admin role
3. TagManager validates payload:
   - Name is required and non-empty
   - Type is valid (category, functionality, dietary)
   - ParentTagID references existing tag (if provided)
4. TagManager calls DataRepository.CreateTag()
5. Return created Tag with 201 Created status
6. Create AuditLog entry for tag creation
```

#### Update Tag
```
1. Admin sends PUT /admin/tags/:id with Tag payload
2. AdminMiddleware validates admin role
3. TagManager:
   a. Verifies tag exists
   b. Validates parent tag relationship (no circular references)
4. TagManager calls DataRepository.UpdateTag()
5. Return updated Tag with 200 OK status
6. Create AuditLog entry for tag update
```

#### Delete Tag
```
1. Admin sends DELETE /admin/tags/:id
2. AdminMiddleware validates admin role
3. TagManager:
   a. Verifies tag exists
   b. Checks for items referencing this tag
   c. IF items reference tag: Return 409 Conflict or cascade delete
4. TagManager calls DataRepository.DeleteTag()
5. Return 204 No Content status
6. Create AuditLog entry for tag deletion
```

### 2.5 User Management Flow (SW-REQ-057)

#### List Users
```
1. Admin sends GET /admin/users
2. AdminMiddleware validates admin role
3. UserAdminPanel:
   a. Parse query params (page, limit, role filter, active filter)
   b. Call DataRepository.GetUsers(filters)
4. Return paginated User list with metadata
```

#### Update User Roles
```
1. Admin sends PUT /admin/users/:id/roles with {roles: []string}
2. AdminMiddleware validates admin role
3. UserAdminPanel:
   a. Verify user exists
   b. Validate role values (must be valid role constants)
   c. Prevent removing own admin role
4. UserAdminPanel calls DataRepository.UpdateUserRoles()
5. Return updated User with 200 OK status
6. Create AuditLog entry for role change
```

#### Deactivate User
```
1. Admin sends POST /admin/users/:id/deactivate
2. AdminMiddleware validates admin role
3. UserAdminPanel:
   a. Verify user exists
   b. Prevent self-deactivation
4. UserAdminPanel calls DataRepository.DeactivateUser()
5. Return 204 No Content status
6. Create AuditLog entry for deactivation
```

## 3. State Management & Error Handling

### 3.1 Error States and Transitions

| Error State | HTTP Status | Trigger Condition | User Action |
|-------------|-------------|-------------------|-------------|
| Unauthorized | 401 | Invalid/expired session token | Re-authenticate |
| Forbidden | 403 | User lacks admin role | Request admin access |
| Not Found | 404 | Resource ID does not exist | Verify resource ID |
| Validation Error | 400 | Missing/invalid input fields | Correct input data |
| Conflict | 409 | Resource has dependencies | Resolve dependencies first |
| Internal Error | 500 | Database or external service failure | Retry later, contact support |
| External Service Timeout | 504 | USDA/OFF API timeout | Retry search |

### 3.2 State Transitions

```
Initial State: No request in progress

Admin Request:
    ↓
Middleware Validation
    ├─→ Fail: 401/403 → Terminal (auth error)
    └─→ Pass: Continue to handler

Handler Processing:
    ├─→ Validation Fail: 400 → Terminal (client error)
    ├─→ Resource Not Found: 404 → Terminal (not found)
    ├─→ Dependency Conflict: 409 → Terminal (conflict)
    ├─→ Database Error: 500 → Terminal (server error)
    ├─→ External API Timeout: 504 → Terminal (upstream error)
    └─→ Success: 200/201/204 → Terminal (complete)

Audit Log: All admin actions create AuditLog entries regardless of outcome
```

### 3.3 Audit Logging

```go
func (c *AdminController) logAudit(ctx *fiber.Ctx, action string, resourceID string, resourceType string, details string) {
    adminUserID, _ := c.authService.GetCurrentUserID(ctx)

    auditLog := AuditLog{
        ID:           generateUUID(),
        AdminUserID:  adminUserID,
        Action:       action,
        ResourceID:   resourceID,
        ResourceType: resourceType,
        Details:      details,
        IPAddress:    ctx.IP(),
        CreatedAt:    time.Now(),
    }

    go c.dataRepository.CreateAuditLog(auditLog)
}
```

### 3.4 Cache Invalidation

```go
func (c *AdminController) invalidateItemCache(itemID string) {
    redisClient := config.GetRedisClient()
    keys := []string{
        fmt.Sprintf("item:%s", itemID),
        "items:list",           // Invalidate full list cache
        "items:recent",         // Invalidate recent items
        "tags:all",             // Invalidate tag list when items updated
    }
    redisClient.Del(ctx, keys...)
}

func (c *AdminController) invalidateTagCache() {
    redisClient := config.GetRedisClient()
    keys := []string{
        "tags:all",
        "tags:category",
        "tags:functionality",
        "tags:dietary",
    }
    redisClient.Del(ctx, keys...)
}
```

## 4. Component Interfaces

### 4.1 Controller Methods

```go
package controller

type AdminControllerInterface interface {
    // SetupRoutes registers all admin routes with the Fiber app
    SetupRoutes(app *fiber.App)

    // --- External Search ---
    SearchExternal(ctx *fiber.Ctx) error
    ImportExternalItem(ctx *fiber.Ctx) error

    // --- Item Management ---
    CreateItem(ctx *fiber.Ctx) error
    UpdateItem(ctx *fiber.Ctx) error
    DeleteItem(ctx *fiber.Ctx) error
    GetItem(ctx *fiber.Ctx) error
    ListItems(ctx *fiber.Ctx) error

    // --- Tag Management ---
    CreateTag(ctx *fiber.Ctx) error
    UpdateTag(ctx *fiber.Ctx) error
    DeleteTag(ctx *fiber.Ctx) error
    ListTags(ctx *fiber.Ctx) error

    // --- User Management ---
    ListUsers(ctx *fiber.Ctx) error
    GetUser(ctx *fiber.Ctx) error
    UpdateUserRoles(ctx *fiber.Ctx) error
    DeactivateUser(ctx *fiber.Ctx) error

    // --- Audit ---
    GetAuditLogs(ctx *fiber.Ctx) error
}
```

### 4.2 Route Definitions

```go
func (c *AdminController) SetupRoutes(app *fiber.App) {
    admin := app.Group("/admin", c.adminMiddleware)

    // External Search
    admin.Get("/external-search", c.SearchExternal)
    admin.Post("/items/import", c.ImportExternalItem)

    // Item Management
    admin.Get("/items", c.ListItems)
    admin.Get("/items/:id", c.GetItem)
    admin.Post("/items", c.CreateItem)
    admin.Put("/items/:id", c.UpdateItem)
    admin.Delete("/items/:id", c.DeleteItem)

    // Tag Management
    admin.Get("/tags", c.ListTags)
    admin.Post("/tags", c.CreateTag)
    admin.Put("/tags/:id", c.UpdateTag)
    admin.Delete("/tags/:id", c.DeleteTag)

    // User Management
    admin.Get("/users", c.ListUsers)
    admin.Get("/users/:id", c.GetUser)
    admin.Put("/users/:id/roles", c.UpdateUserRoles)
    admin.Post("/users/:id/deactivate", c.DeactivateUser)

    // Audit Logs
    admin.Get("/audit-logs", c.GetAuditLogs)
}
```

### 4.3 Method Signatures and Request/Response Types

```go
// SearchExternal handles GET /admin/external-search
// Query params: q (search term), source (USDA|OpenFoodFacts|both), limit (optional)
func (c *AdminController) SearchExternal(ctx *fiber.Ctx) error {
    query := ctx.Query("q")
    source := ctx.Query("source", "both")
    limit := ctx.QueryInt("limit", 50)

    if query == "" {
        return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Search query 'q' is required",
        })
    }

    results, err := c.externalSearch.Search(ctx, query, source, limit)
    if err != nil {
        return c.handleExternalSearchError(ctx, err)
    }

    return ctx.JSON(fiber.Map{
        "results": results,
        "count":   len(results),
        "query":   query,
        "source":  source,
    })
}

// ImportExternalItem handles POST /admin/items/import
// Request body: ExternalSearchResult with curated fields
func (c *AdminController) ImportExternalItem(ctx *fiber.Ctx) error {
    var req ExternalSearchResult
    if err := ctx.BodyParser(&req); err != nil {
        return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid request body: " + err.Error(),
        })
    }

    // Validate required fields
    if req.Name == "" {
        return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Item name is required",
        })
    }

    item, err := c.dataImporter.Import(ctx, &req)
    if err != nil {
        return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to import item: " + err.Error(),
        })
    }

    c.logAudit(ctx, "item_import", item.ID, "item",
        fmt.Sprintf("Imported item '%s' from %s", item.Name, req.Source))

    return ctx.Status(fiber.StatusCreated).JSON(item)
}

// CreateItem handles POST /admin/items
func (c *AdminController) CreateItem(ctx *fiber.Ctx) error {
    var item Item
    if err := ctx.BodyParser(&item); err != nil {
        return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid request body: " + err.Error(),
        })
    }

    created, err := c.itemCurator.Create(ctx, &item)
    if err != nil {
        return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Failed to create item: " + err.Error(),
        })
    }

    c.logAudit(ctx, "item_create", created.ID, "item",
        fmt.Sprintf("Created item '%s'", created.Name))

    return ctx.Status(fiber.StatusCreated).JSON(created)
}

// UpdateItem handles PUT /admin/items/:id
func (c *AdminController) UpdateItem(ctx *fiber.Ctx) error {
    id := ctx.Params("id")
    if id == "" {
        return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Item ID is required",
        })
    }

    var updates Item
    if err := ctx.BodyParser(&updates); err != nil {
        return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid request body: " + err.Error(),
        })
    }

    updated, err := c.itemCurator.Update(ctx, id, &updates)
    if err != nil {
        return c.handleItemError(ctx, err)
    }

    c.logAudit(ctx, "item_update", id, "item",
        fmt.Sprintf("Updated item '%s'", updated.Name))

    c.invalidateItemCache(id)

    return ctx.JSON(updated)
}

// DeleteItem handles DELETE /admin/items/:id
func (c *AdminController) DeleteItem(ctx *fiber.Ctx) error {
    id := ctx.Params("id")
    if id == "" {
        return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Item ID is required",
        })
    }

    err := c.itemCurator.Delete(ctx, id)
    if err != nil {
        return c.handleItemError(ctx, err)
    }

    c.logAudit(ctx, "item_delete", id, "item", "Deleted item")

    c.invalidateItemCache(id)

    return ctx.SendStatus(fiber.StatusNoContent)
}

// ListItems handles GET /admin/items
func (c *AdminController) ListItems(ctx *fiber.Ctx) error {
    page := ctx.QueryInt("page", 1)
    limit := ctx.QueryInt("limit", 50)
    tagFilter := ctx.Query("tag")
    searchQuery := ctx.Query("q")

    if page < 1 {
        page = 1
    }
    if limit < 1 || limit > 100 {
        limit = 50
    }

    items, total, err := c.itemCurator.List(ctx, page, limit, tagFilter, searchQuery)
    if err != nil {
        return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to list items: " + err.Error(),
        })
    }

    return ctx.JSON(fiber.Map{
        "items": items,
        "pagination": Pagination{
            Page:    page,
            Limit:   limit,
            Total:   total,
            TotalPages: (total + limit - 1) / limit,
        },
    })
}

// GetItem handles GET /admin/items/:id
func (c *AdminController) GetItem(ctx *fiber.Ctx) error {
    id := ctx.Params("id")
    if id == "" {
        return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Item ID is required",
        })
    }

    item, err := c.itemCurator.GetByID(ctx, id)
    if err != nil {
        return c.handleItemError(ctx, err)
    }

    return ctx.JSON(item)
}

// CreateTag handles POST /admin/tags
func (c *AdminController) CreateTag(ctx *fiber.Ctx) error {
    var tag Tag
    if err := ctx.BodyParser(&tag); err != nil {
        return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid request body: " + err.Error(),
        })
    }

    if tag.Name == "" {
        return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Tag name is required",
        })
    }

    if !isValidTagType(tag.Type) {
        return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid tag type. Must be: category, functionality, or dietary",
        })
    }

    created, err := c.tagManager.Create(ctx, &tag)
    if err != nil {
        return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Failed to create tag: " + err.Error(),
        })
    }

    c.logAudit(ctx, "tag_create", created.ID, "tag",
        fmt.Sprintf("Created tag '%s' of type %s", created.Name, created.Type))

    c.invalidateTagCache()

    return ctx.Status(fiber.StatusCreated).JSON(created)
}

// UpdateTag handles PUT /admin/tags/:id
func (c *AdminController) UpdateTag(ctx *fiber.Ctx) error {
    id := ctx.Params("id")
    if id == "" {
        return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Tag ID is required",
        })
    }

    var updates Tag
    if err := ctx.BodyParser(&updates); err != nil {
        return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid request body: " + err.Error(),
        })
    }

    updated, err := c.tagManager.Update(ctx, id, &updates)
    if err != nil {
        return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Failed to update tag: " + err.Error(),
        })
    }

    c.logAudit(ctx, "tag_update", id, "tag",
        fmt.Sprintf("Updated tag '%s'", updated.Name))

    c.invalidateTagCache()

    return ctx.JSON(updated)
}

// DeleteTag handles DELETE /admin/tags/:id
func (c *AdminController) DeleteTag(ctx *fiber.Ctx) error {
    id := ctx.Params("id")
    if id == "" {
        return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Tag ID is required",
        })
    }

    err := c.tagManager.Delete(ctx, id)
    if err != nil {
        return ctx.Status(fiber.StatusConflict).JSON(fiber.Map{
            "error": "Failed to delete tag: " + err.Error(),
        })
    }

    c.logAudit(ctx, "tag_delete", id, "tag", "Deleted tag")

    c.invalidateTagCache()

    return ctx.SendStatus(fiber.StatusNoContent)
}

// ListTags handles GET /admin/tags
func (c *AdminController) ListTags(ctx *fiber.Ctx) error {
    tagType := ctx.Query("type")

    tags, err := c.tagManager.List(ctx, tagType)
    if err != nil {
        return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to list tags: " + err.Error(),
        })
    }

    return ctx.JSON(fiber.Map{
        "tags": tags,
        "count": len(tags),
    })
}

// ListUsers handles GET /admin/users
func (c *AdminController) ListUsers(ctx *fiber.Ctx) error {
    page := ctx.QueryInt("page", 1)
    limit := ctx.QueryInt("limit", 50)
    roleFilter := ctx.Query("role")
    activeFilter := ctx.Query("active")

    if page < 1 {
        page = 1
    }
    if limit < 1 || limit > 100 {
        limit = 50
    }

    users, total, err := c.userAdminPanel.List(ctx, page, limit, roleFilter, activeFilter)
    if err != nil {
        return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to list users: " + err.Error(),
        })
    }

    return ctx.JSON(fiber.Map{
        "users": users,
        "pagination": Pagination{
            Page:    page,
            Limit:   limit,
            Total:   total,
            TotalPages: (total + limit - 1) / limit,
        },
    })
}

// GetUser handles GET /admin/users/:id
func (c *AdminController) GetUser(ctx *fiber.Ctx) error {
    id := ctx.Params("id")
    if id == "" {
        return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "User ID is required",
        })
    }

    user, err := c.userAdminPanel.GetByID(ctx, id)
    if err != nil {
        return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
            "error": "User not found",
        })
    }

    return ctx.JSON(user)
}

// UpdateUserRoles handles PUT /admin/users/:id/roles
func (c *AdminController) UpdateUserRoles(ctx *fiber.Ctx) error {
    id := ctx.Params("id")
    if id == "" {
        return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "User ID is required",
        })
    }

    var req struct {
        Roles []string `json:"roles"`
    }
    if err := ctx.BodyParser(&req); err != nil {
        return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid request body: " + err.Error(),
        })
    }

    // Prevent removing own admin role
    currentUserID, _ := c.authService.GetCurrentUserID(ctx)
    if currentUserID == id {
        hasAdmin := false
        for _, role := range req.Roles {
            if role == "admin" {
                hasAdmin = true
                break
            }
        }
        if !hasAdmin {
            return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
                "error": "Cannot remove your own admin role",
            })
        }
    }

    updated, err := c.userAdminPanel.UpdateRoles(ctx, id, req.Roles)
    if err != nil {
        return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Failed to update user roles: " + err.Error(),
        })
    }

    c.logAudit(ctx, "user_update_roles", id, "user",
        fmt.Sprintf("Updated roles to %v", req.Roles))

    return ctx.JSON(updated)
}

// DeactivateUser handles POST /admin/users/:id/deactivate
func (c *AdminController) DeactivateUser(ctx *fiber.Ctx) error {
    id := ctx.Params("id")
    if id == "" {
        return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "User ID is required",
        })
    }

    // Prevent self-deactivation
    currentUserID, _ := c.authService.GetCurrentUserID(ctx)
    if currentUserID == id {
        return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
            "error": "Cannot deactivate your own account",
        })
    }

    err := c.userAdminPanel.Deactivate(ctx, id)
    if err != nil {
        return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Failed to deactivate user: " + err.Error(),
        })
    }

    c.logAudit(ctx, "user_deactivate", id, "user", "Deactivated user")

    return ctx.SendStatus(fiber.StatusNoContent)
}

// GetAuditLogs handles GET /admin/audit-logs
func (c *AdminController) GetAuditLogs(ctx *fiber.Ctx) error {
    page := ctx.QueryInt("page", 1)
    limit := ctx.QueryInt("limit", 50)
    actionFilter := ctx.Query("action")
    resourceTypeFilter := ctx.Query("resource_type")
    adminUserID := ctx.Query("admin_user_id")

    if page < 1 {
        page = 1
    }
    if limit < 1 || limit > 100 {
        limit = 50
    }

    logs, total, err := c.dataRepository.GetAuditLogs(ctx, page, limit, actionFilter, resourceTypeFilter, adminUserID)
    if err != nil {
        return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to retrieve audit logs: " + err.Error(),
        })
    }

    return ctx.JSON(fiber.Map{
        "audit_logs": logs,
        "pagination": Pagination{
            Page:      page,
            Limit:     limit,
            Total:     total,
            TotalPages: (total + limit - 1) / limit,
        },
    })
}
```

### 4.4 Helper Methods

```go
func (c *AdminController) handleItemError(ctx *fiber.Ctx, err error) error {
    if errors.Is(err, service.ErrItemNotFound) {
        return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
            "error": "Item not found",
        })
    }
    if errors.Is(err, service.ErrItemHasDependencies) {
        return ctx.Status(fiber.StatusConflict).JSON(fiber.Map{
            "error": "Item cannot be deleted due to existing dependencies",
        })
    }
    return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
        "error": "Internal server error",
    })
}

func (c *AdminController) handleExternalSearchError(ctx *fiber.Ctx, err error) error {
    if errors.Is(err, service.ErrExternalAPITimeout) {
        return ctx.Status(fiber.StatusGatewayTimeout).JSON(fiber.Map{
            "error": "External search service timed out. Please try again.",
        })
    }
    if errors.Is(err, service.ErrExternalAPIUnavailable) {
        return ctx.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
            "error": "External search service is temporarily unavailable",
        })
    }
    return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
        "error": "Failed to search external sources",
    })
}

func isValidTagType(t TagType) bool {
    switch t {
    case TagTypeCategory, TagTypeFunctionality, TagTypeDietary:
        return true
    default:
        return false
    }
}

type Pagination struct {
    Page      int `json:"page"`
    Limit     int `json:"limit"`
    Total     int `json:"total"`
    TotalPages int `json:"total_pages"`
}
```
