# TagManager - Detailed Design

**Traceability:** ARCH-009

---

## 1. Data Structures & Types

### 1.1 Core Structs

```go
package tag

type Tag struct {
    ID          string    `json:"id" db:"id"`
    Name        string    `json:"name" db:"name"`
    Slug        string    `json:"slug" db:"slug"`
    Type        TagType   `json:"type" db:"type"`
    Description string    `json:"description" db:"description"`
    Color       string    `json:"color" db:"color"`
    Icon        string    `json:"icon" db:"icon"`
    ParentID    *string   `json:"parent_id,omitempty" db:"parent_id"`
    SortOrder   int       `json:"sort_order" db:"sort_order"`
    IsActive    bool      `json:"is_active" db:"is_active"`
    CreatedAt   time.Time `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type TagType string

const (
    TagTypeCategory     TagType = "category"
    TagTypeFunctionality TagType = "functionality"
)

type TagHierarchy struct {
    Tag         *Tag           `json:"tag"`
    Children    []*TagHierarchy `json:"children"`
    ItemCount   int            `json:"item_count"`
}

type TagWithItems struct {
    Tag      *Tag     `json:"tag"`
    ItemIDs  []string `json:"item_ids"`
    Total    int      `json:"total"`
}

type BulkTagOperation struct {
    TagIDs    []string `json:"tag_ids"`
    Operation string   `json:"operation"` // "activate", "deactivate", "delete"
}

type MergeTagsRequest struct {
    SourceTagID string `json:"source_tag_id"`
    TargetTagID string `json:"target_tag_id"`
}

type CreateTagRequest struct {
    Name        string    `json:"name" validate:"required,min=1,max=100"`
    Type        TagType   `json:"type" validate:"required,oneof=category functionality"`
    Description string    `json:"description" validate:"max=500"`
    Color       string    `json:"color" validate:"omitempty,hexcolor"`
    Icon        string    `json:"icon" validate:"omitempty,max=50"`
    ParentID    *string   `json:"parent_id,omitempty"`
    SortOrder   int       `json:"sort_order"`
}

type UpdateTagRequest struct {
    Name        *string   `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
    Description *string   `json:"description,omitempty" validate:"omitempty,max=500"`
    Color       *string   `json:"color,omitempty" validate:"omitempty,hexcolor"`
    Icon        *string   `json:"icon,omitempty" validate:"omitempty,max=50"`
    ParentID    *string   `json:"parent_id,omitempty"`
    SortOrder   *int      `json:"sort_order,omitempty"`
    IsActive    *bool     `json:"is_active,omitempty"`
}

type TagFilter struct {
    Type      *TagType  `json:"type,omitempty"`
    ParentID  *string   `json:"parent_id,omitempty"`
    IsActive  *bool     `json:"is_active,omitempty"`
    Search    string    `json:"search,omitempty"`
    Limit     int       `json:"limit"`
    Offset    int       `json:"offset"`
}

type PaginatedTags struct {
    Tags     []*Tag  `json:"tags"`
    Total    int     `json:"total"`
    Limit    int     `json:"limit"`
    Offset   int     `json:"offset"`
}
```

### 1.2 Repository Interface

```go
type TagRepository interface {
    Create(ctx context.Context, tag *Tag) error
    GetByID(ctx context.Context, id string) (*Tag, error)
    GetBySlug(ctx context.Context, slug string, tagType TagType) (*Tag, error)
    Update(ctx context.Context, tag *Tag) error
    Delete(ctx context.Context, id string) error
    List(ctx context.Context, filter TagFilter) (*PaginatedTags, error)
    ListByIDs(ctx context.Context, ids []string) ([]*Tag, error)
    GetChildren(ctx context.Context, parentID string) ([]*Tag, error)
    GetHierarchy(ctx context.Context, tagType TagType) ([]*TagHierarchy, error)
    GetItemCount(ctx context.Context, tagID string) (int, error)
    BulkActivate(ctx context.Context, ids []string) error
    BulkDeactivate(ctx context.Context, ids []string) error
    MergeTags(ctx context.Context, sourceID, targetID string) error
    AssignToItem(ctx context.Context, itemID string, tagIDs []string) error
    RemoveFromItem(ctx context.Context, itemID string, tagIDs []string) error
    GetByItem(ctx context.Context, itemID string) ([]*Tag, error)
    Exists(ctx context.Context, id string) (bool, error)
    ExistsBySlug(ctx context.Context, slug string, tagType TagType, excludeID string) (bool, error)
}
```

### 1.3 Service Interface

```go
type TagService interface {
    CreateTag(ctx context.Context, req CreateTagRequest) (*Tag, error)
    GetTag(ctx context.Context, id string) (*Tag, error)
    UpdateTag(ctx context.Context, id string, req UpdateTagRequest) (*Tag, error)
    DeleteTag(ctx context.Context, id string) error
    ListTags(ctx context.Context, filter TagFilter) (*PaginatedTags, error)
    GetTagHierarchy(ctx context.Context, tagType TagType) ([]*TagHierarchy, error)
    GetTagsByItem(ctx context.Context, itemID string) ([]*Tag, error)
    BulkOperation(ctx context.Context, op BulkTagOperation) error
    MergeTags(ctx context.Context, req MergeTagsRequest) error
    AssignTagsToItem(ctx context.Context, itemID string, tagIDs []string) error
    RemoveTagsFromItem(ctx context.Context, itemID string, tagIDs []string) error
    ValidateTags(ctx context.Context, tagIDs []string, tagType TagType) ([]string, error)
    GenerateSlug(name string) string
}
```

---

## 2. Logic & Algorithms

### 2.1 CreateTag Flow

```
1. Validate CreateTagRequest
   - Name: required, 1-100 characters
   - Type: required, must be "category" or "functionality"
   - Description: max 500 characters
   - Color: optional, must be valid hex color
   - Icon: optional, max 50 characters
   - ParentID: optional, must exist if provided

2. Generate slug from name using GenerateSlug()
   - Lowercase name
   - Replace spaces with hyphens
   - Remove special characters
   - Truncate to 100 characters

3. Check for slug uniqueness
   - Call ExistsBySlug(slug, tagType, "")
   - If exists, append incrementing suffix (e.g., name-2)

4. If ParentID provided, validate parent exists
   - Call Exists(parentID)
   - Verify parent.Type == tag.Type (hierarchy must match type)

5. Create Tag struct with generated values
   - ID: generate UUID v4
   - Slug: generated slug
   - IsActive: true
   - CreatedAt, UpdatedAt: current time

6. Call repository.Create(tag)

7. Return created tag
```

### 2.2 UpdateTag Flow

```
1. Validate tag exists
   - Call GetByID(id)
   - If not found, return ErrTagNotFound

2. Validate UpdateTagRequest fields
   - Check required fields are not empty
   - Validate format of optional fields

3. If Name is being updated:
   - Generate new slug
   - Check slug uniqueness: ExistsBySlug(newSlug, tag.Type, id)
   - If exists, append suffix

4. If ParentID is being updated:
   - If ParentID is nil, clear parent
   - If ParentID is set:
     - Validate parent exists: Exists(parentID)
     - Verify parent.Type == tag.Type
     - Verify parent.ID != tag.ID (no self-reference)
     - Verify no circular reference (traverse up parent chain)

5. Update Tag struct with new values
   - UpdatedAt: current time

6. Call repository.Update(tag)

7. Return updated tag
```

### 2.3 DeleteTag Flow

```
1. Validate tag exists
   - Call GetByID(id)
   - If not found, return ErrTagNotFound

2. Check for child tags
   - Call GetChildren(id)
   - If children exist, return ErrTagHasChildren

3. Check for associated items
   - Call GetItemCount(id)
   - If items exist, return ErrTagHasItems

4. Call repository.Delete(id)

5. Return nil (success)
```

### 2.4 MergeTags Flow

```
1. Validate both tags exist
   - GetByID(sourceID)
   - GetByID(targetID)

2. Validate both tags are same type
   - source.Type == target.Type
   - If not, return ErrTagTypeMismatch

3. Validate target tag is not being deleted
   - sourceID != targetID

4. Begin transaction

5. Reassign all items from source to target
   - Update item_tag associations in database

6. Reassign child tags from source to target
   - Update parent_id for all child tags

7. Delete source tag
   - Call repository.Delete(sourceID)

8. Commit transaction

9. Return nil (success)
```

### 2.5 GetTagHierarchy Flow

```
1. Fetch all tags of given type
   - Call List(TagFilter{Type: &tagType, IsActive: &true})

2. Build hierarchy tree
   - Create map[ID]*TagNode
   - Create list of root nodes (ParentID == nil)
   - For each tag:
     - If ParentID exists, add to parent's Children
     - Else add to roots

3. For each node, calculate item counts
   - Call GetItemCount(tag.ID)

4. Return hierarchical structure
```

### 2.6 GenerateSlug Algorithm

```go
func GenerateSlug(name string) string {
    slug := strings.ToLower(name)
    slug = strings.ReplaceAll(slug, " ", "-")
    
    var result strings.Builder
    for _, r := range slug {
        if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
            result.WriteRune(r)
        }
        if result.Len() >= 100 {
            break
        }
    }
    
    slug = strings.Trim(result.String(), "-")
    slug = strings.ReplaceAll(slug, "--", "-")
    
    return slug
}
```

---

## 3. State Management & Error Handling

### 3.1 Error Types

```go
var (
    ErrTagNotFound         = fiber.NewError(404, "tag not found")
    ErrTagAlreadyExists    = fiber.NewError(409, "tag already exists")
    ErrTagHasChildren      = fiber.NewError(409, "tag has child tags")
    ErrTagHasItems         = fiber.NewError(409, "tag is assigned to items")
    ErrTagTypeMismatch     = fiber.NewError(400, "tags must be same type")
    ErrCircularReference   = fiber.NewError(400, "circular reference detected")
    ErrInvalidTagType      = fiber.NewError(400, "invalid tag type")
    ErrParentNotFound      = fiber.NewError(404, "parent tag not found")
    ErrSelfReference       = fiber.NewError(400, "tag cannot be parent of itself")
    ErrSlugAlreadyExists   = fiber.NewError(409, "slug already exists")
    ErrTagInactive         = fiber.NewError(400, "tag is not active")
)
```

### 3.2 State Transitions

| Current State | Event | Next State | Action |
|---|---|---|---|
| None | CreateTag | Active | Create tag with IsActive=true |
| Active | UpdateTag | Active | Update tag fields |
| Active | DeleteTag | Deleted | Remove tag from database |
| Active | BulkDeactivate | Inactive | Set IsActive=false |
| Inactive | BulkActivate | Active | Set IsActive=true |
| Inactive | UpdateTag | Inactive | Update tag fields |
| Inactive | DeleteTag | Deleted | Remove tag from database |

### 3.3 Validation States

```
CreateTag Validation:
├── Name required
├── Name length 1-100
├── Type required
├── Type valid (category|functionality)
├── Description max 500
├── Color valid hex
├── Icon max 50
├── ParentID valid if provided
└── Slug unique

UpdateTag Validation:
├── At least one field to update
├── Name valid if provided
├── Description valid if provided
├── Color valid if provided
├── Icon valid if provided
├── ParentID valid if provided
├── No circular reference
└── Slug unique (if name updated)

MergeTags Validation:
├── Source tag exists
├── Target tag exists
├── Same tag type
├── Different tags
└── No items on source (optional - can merge with items)
```

### 3.4 Concurrency Handling

- Use database transactions for merge operations
- Implement optimistic locking with version field on Tag struct
- Use row-level locking when updating parent hierarchies
- Validate circular references within transaction

---

## 4. Component Interfaces

### 4.1 Handler Signatures

```go
type TagHandler struct {
    service TagService
}

func NewTagHandler(service TagService) *TagHandler {
    return &TagHandler{service: service}
}

func (h *TagHandler) Create(c *fiber.Ctx) error {
    var req CreateTagRequest
    if err := c.BodyParser(&req); err != nil {
        return fiber.NewError(400, "invalid request body")
    }
    
    tag, err := h.service.CreateTag(c.Context(), req)
    if err != nil {
        return err
    }
    
    return c.Status(201).JSON(tag)
}

func (h *TagHandler) GetByID(c *fiber.Ctx) error {
    id := c.Params("id")
    
    tag, err := h.service.GetTag(c.Context(), id)
    if err != nil {
        return err
    }
    
    return c.JSON(tag)
}

func (h *TagHandler) Update(c *fiber.Ctx) error {
    id := c.Params("id")
    var req UpdateTagRequest
    if err := c.BodyParser(&req); err != nil {
        return fiber.NewError(400, "invalid request body")
    }
    
    tag, err := h.service.UpdateTag(c.Context(), id, req)
    if err != nil {
        return err
    }
    
    return c.JSON(tag)
}

func (h *TagHandler) Delete(c *fiber.Ctx) error {
    id := c.Params("id")
    
    if err := h.service.DeleteTag(c.Context(), id); err != nil {
        return err
    }
    
    return c.SendStatus(204)
}

func (h *TagHandler) List(c *fiber.Ctx) error {
    filter := TagFilter{
        Type:    tagTypeFromQuery(c),
        ParentID: parentIDFromQuery(c),
        IsActive: isActiveFromQuery(c),
        Search:   c.Query("search"),
        Limit:    limitFromQuery(c, 50),
        Offset:   offsetFromQuery(c),
    }
    
    result, err := h.service.ListTags(c.Context(), filter)
    if err != nil {
        return err
    }
    
    return c.JSON(result)
}

func (h *TagHandler) GetHierarchy(c *fiber.Ctx) error {
    tagType := TagType(c.Params("type"))
    if tagType != TagTypeCategory && tagType != TagTypeFunctionality {
        return fiber.NewError(400, "invalid tag type")
    }
    
    hierarchy, err := h.service.GetTagHierarchy(c.Context(), tagType)
    if err != nil {
        return err
    }
    
    return c.JSON(hierarchy)
}

func (h *TagHandler) BulkOperation(c *fiber.Ctx) error {
    var op BulkTagOperation
    if err := c.BodyParser(&op); err != nil {
        return fiber.NewError(400, "invalid request body")
    }
    
    if err := h.service.BulkOperation(c.Context(), op); err != nil {
        return err
    }
    
    return c.SendStatus(204)
}

func (h *TagHandler) Merge(c *fiber.Ctx) error {
    var req MergeTagsRequest
    if err := c.BodyParser(&req); err != nil {
        return fiber.NewError(400, "invalid request body")
    }
    
    if err := h.service.MergeTags(c.Context(), req); err != nil {
        return err
    }
    
    return c.SendStatus(204)
}

func (h *TagHandler) GetByItem(c *fiber.Ctx) error {
    itemID := c.Params("item_id")
    
    tags, err := h.service.GetTagsByItem(c.Context(), itemID)
    if err != nil {
        return err
    }
    
    return c.JSON(tags)
}

func (h *TagHandler) AssignToItem(c *fiber.Ctx) error {
    itemID := c.Params("item_id")
    var req struct {
        TagIDs []string `json:"tag_ids"`
    }
    if err := c.BodyParser(&req); err != nil {
        return fiber.NewError(400, "invalid request body")
    }
    
    if err := h.service.AssignTagsToItem(c.Context(), itemID, req.TagIDs); err != nil {
        return err
    }
    
    return c.SendStatus(204)
}

func (h *TagHandler) RemoveFromItem(c *fiber.Ctx) error {
    itemID := c.Params("item_id")
    var req struct {
        TagIDs []string `json:"tag_ids"`
    }
    if err := c.BodyParser(&req); err != nil {
        return fiber.NewError(400, "invalid request body")
    }
    
    if err := h.service.RemoveTagsFromItem(c.Context(), itemID, req.TagIDs); err != nil {
        return err
    }
    
    return c.SendStatus(204)
}
```

### 4.2 Route Registration

```go
func RegisterTagRoutes(router fiber.Router, handler *TagHandler, authMiddleware fiber.Handler) {
    admin := router.Group("/tags", authMiddleware.RequireAdmin())
    
    admin.Post("/", handler.Create)
    admin.Get("/", handler.List)
    admin.Get("/:id", handler.GetByID)
    admin.Put("/:id", handler.Update)
    admin.Delete("/:id", handler.Delete)
    
    admin.Get("/type/:type/hierarchy", handler.GetHierarchy)
    admin.Post("/bulk", handler.BulkOperation)
    admin.Post("/merge", handler.Merge)
    
    admin.Get("/item/:item_id", handler.GetByItem)
    admin.Post("/item/:item_id", handler.AssignToItem)
    admin.Delete("/item/:item_id", handler.RemoveFromItem)
}
```

### 4.3 Service Implementation Skeleton

```go
type tagService struct {
    repo TagRepository
}

func NewTagService(repo TagRepository) TagService {
    return &tagService{repo: repo}
}

func (s *tagService) CreateTag(ctx context.Context, req CreateTagRequest) (*Tag, error) {
    // Implementation from Section 2.1
}

func (s *tagService) GetTag(ctx context.Context, id string) (*Tag, error) {
    // Implementation from Section 2.2 (step 1)
}

func (s *tagService) UpdateTag(ctx context.Context, id string, req UpdateTagRequest) (*Tag, error) {
    // Implementation from Section 2.2
}

func (s *tagService) DeleteTag(ctx context.Context, id string) error {
    // Implementation from Section 2.3
}

func (s *tagService) ListTags(ctx context.Context, filter TagFilter) (*PaginatedTags, error) {
    return s.repo.List(ctx, filter)
}

func (s *tagService) GetTagHierarchy(ctx context.Context, tagType TagType) ([]*TagHierarchy, error) {
    // Implementation from Section 2.5
}

func (s *tagService) GetTagsByItem(ctx context.Context, itemID string) ([]*Tag, error) {
    return s.repo.GetByItem(ctx, itemID)
}

func (s *tagService) BulkOperation(ctx context.Context, op BulkTagOperation) error {
    switch op.Operation {
    case "activate":
        return s.repo.BulkActivate(ctx, op.TagIDs)
    case "deactivate":
        return s.repo.BulkDeactivate(ctx, op.TagIDs)
    case "delete":
        for _, id := range op.TagIDs {
            if err := s.DeleteTag(ctx, id); err != nil {
                return err
            }
        }
        return nil
    default:
        return fiber.NewError(400, "invalid operation")
    }
}

func (s *tagService) MergeTags(ctx context.Context, req MergeTagsRequest) error {
    // Implementation from Section 2.4
}

func (s *tagService) AssignTagsToItem(ctx context.Context, itemID string, tagIDs []string) error {
    return s.repo.AssignToItem(ctx, itemID, tagIDs)
}

func (s *tagService) RemoveTagsFromItem(ctx context.Context, itemID string, tagIDs []string) error {
    return s.repo.RemoveFromItem(ctx, itemID, tagIDs)
}

func (s *tagService) ValidateTags(ctx context.Context, tagIDs []string, tagType TagType) ([]string, error) {
    var valid []string
    for _, id := range tagIDs {
        tag, err := s.repo.GetByID(ctx, id)
        if err != nil {
            continue
        }
        if tag.Type == tagType && tag.IsActive {
            valid = append(valid, id)
        }
    }
    return valid, nil
}

func (s *tagService) GenerateSlug(name string) string {
    // Implementation from Section 2.6
}
```

### 4.4 Repository Implementation Skeleton

```go
type tagRepository struct {
    db *sql.DB
}

func NewTagRepository(db *sql.DB) TagRepository {
    return &tagRepository{db: db}
}

func (r *tagRepository) Create(ctx context.Context, tag *Tag) error {
    query := `
        INSERT INTO tags (id, name, slug, type, description, color, icon, parent_id, sort_order, is_active, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
    `
    _, err := r.db.ExecContext(ctx, query, tag.ID, tag.Name, tag.Slug, tag.Type, tag.Description, tag.Color, tag.Icon, tag.ParentID, tag.SortOrder, tag.IsActive, tag.CreatedAt, tag.UpdatedAt)
    return err
}

func (r *tagRepository) GetByID(ctx context.Context, id string) (*Tag, error) {
    query := `SELECT id, name, slug, type, description, color, icon, parent_id, sort_order, is_active, created_at, updated_at FROM tags WHERE id = $1`
    var tag Tag
    err := r.db.QueryRowContext(ctx, query, id).Scan(&tag.ID, &tag.Name, &tag.Slug, &tag.Type, &tag.Description, &tag.Color, &tag.Icon, &tag.ParentID, &tag.SortOrder, &tag.IsActive, &tag.CreatedAt, &tag.UpdatedAt)
    if err == sql.ErrNoRows {
        return nil, ErrTagNotFound
    }
    if err != nil {
        return nil, err
    }
    return &tag, nil
}

func (r *tagRepository) Update(ctx context.Context, tag *Tag) error {
    query := `
        UPDATE tags SET name = $1, slug = $2, description = $3, color = $4, icon = $5, parent_id = $6, sort_order = $7, is_active = $8, updated_at = $9
        WHERE id = $10
    `
    _, err := r.db.ExecContext(ctx, query, tag.Name, tag.Slug, tag.Description, tag.Color, tag.Icon, tag.ParentID, tag.SortOrder, tag.IsActive, tag.UpdatedAt, tag.ID)
    return err
}

func (r *tagRepository) Delete(ctx context.Context, id string) error {
    query := `DELETE FROM tags WHERE id = $1`
    _, err := r.db.ExecContext(ctx, query, id)
    return err
}

func (r *tagRepository) List(ctx context.Context, filter TagFilter) (*PaginatedTags, error) {
    // Implementation with dynamic WHERE clause building
}

func (r *tagRepository) GetChildren(ctx context.Context, parentID string) ([]*Tag, error) {
    query := `SELECT id, name, slug, type, description, color, icon, parent_id, sort_order, is_active, created_at, updated_at FROM tags WHERE parent_id = $1 ORDER BY sort_order`
    rows, err := r.db.QueryContext(ctx, query, parentID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var tags []*Tag
    for rows.Next() {
        var tag Tag
        if err := rows.Scan(&tag.ID, &tag.Name, &tag.Slug, &tag.Type, &tag.Description, &tag.Color, &tag.Icon, &tag.ParentID, &tag.SortOrder, &tag.IsActive, &tag.CreatedAt, &tag.UpdatedAt); err != nil {
            return nil, err
        }
        tags = append(tags, &tag)
    }
    return tags, rows.Err()
}

// Additional repository methods...
```
