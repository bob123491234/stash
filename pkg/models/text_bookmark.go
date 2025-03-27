package models

type TextBookmarkFilterType struct {
	// Filter to only include text bookmarks with this tag
	TagID *string `json:"tag_id"`
	// Filter to only include text bookmarks with these tags
	Tags *HierarchicalMultiCriterionInput `json:"tags"`
	// Filter to only include text bookmarks attached to a text with these tags
	TextTags *HierarchicalMultiCriterionInput `json:"text_tags"`
	// Filter to only include text bookmarks with these performers
	Performers *MultiCriterionInput `json:"performers"`
	// Filter by created at
	CreatedAt *TimestampCriterionInput `json:"created_at"`
	// Filter by updated at
	UpdatedAt *TimestampCriterionInput `json:"updated_at"`
	// Filter by texts date
	TextDate *DateCriterionInput `json:"text_date"`
	// Filter by texts created at
	TextCreatedAt *TimestampCriterionInput `json:"text_created_at"`
	// Filter by texts updated at
	TextUpdatedAt *TimestampCriterionInput `json:"text_updated_at"`
	// Filter by related texts that meet this criteria
	TextFilter *TextFilterType `json:"text_filter"`
}

type BookmarkStringsResultType struct {
	Count int    `json:"count"`
	ID    string `json:"id"`
	Title string `json:"title"`
}
