-- Phase: phase-01 | Task: 4 | Architecture: ARCH-005 | Design: FoodItemEntity
-- Create food_item_category_tags junction table

CREATE TABLE food_item_category_tags (
    food_item_id UUID NOT NULL REFERENCES food_items(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (food_item_id, tag_id)
);

CREATE INDEX idx_food_item_category_tags_food_item_id ON food_item_category_tags(food_item_id);
CREATE INDEX idx_food_item_category_tags_tag_id ON food_item_category_tags(tag_id);
