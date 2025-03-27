package models

import "context"

type TextFilterType struct {
	OperatorFilter[TextFilterType]
	ID           *IntCriterionInput    `json:"id"`
	Title        *StringCriterionInput `json:"title"`
	TagLine      *StringCriterionInput `json:"tag_line"`
	Code         *StringCriterionInput `json:"code"`
	Details      *StringCriterionInput `json:"details"`
	Author       *StringCriterionInput `json:"author"`
	LanguageCode *StringCriterionInput `json:"language_code"`
	// Filter by file oshash
	Oshash *StringCriterionInput `json:"oshash"`
	// Filter by file checksum
	Checksum *StringCriterionInput `json:"checksum"`
	// Filter by file phash
	Phash *StringCriterionInput `json:"phash"`
	// Filter by phash distance
	PhashDistance *PhashDistanceCriterionInput `json:"phash_distance"`
	// Filter by path
	Path *StringCriterionInput `json:"path"`
	// Filter by file count
	FileCount *IntCriterionInput `json:"file_count"`
	// Filter by rating expressed as 1-100
	Rating100 *IntCriterionInput `json:"rating100"`
	// Filter by organized
	Organized *bool `json:"organized"`
	// Filter by o-counter
	OCounter *IntCriterionInput `json:"o_counter"`
	// Filter by word count
	WordCount *IntCriterionInput `json:"word_count"`
	// Filter to only include texts which have bookmarks. `true` or `false`
	HasBookmarks *string `json:"has_bookmarks"`
	// Filter to only include texts missing this property
	IsMissing *string `json:"is_missing"`
	// Filter to only include texts with this studio
	Studios *HierarchicalMultiCriterionInput `json:"studios"`
	// Filter to only include texts with these tags
	Tags *HierarchicalMultiCriterionInput `json:"tags"`
	// Filter by tag count
	TagCount *IntCriterionInput `json:"tag_count"`
	// Filter to only include texts with performers with these tags
	PerformerTags *HierarchicalMultiCriterionInput `json:"performer_tags"`
	// Filter texts that have performers that have been favorited
	PerformerFavorite *bool `json:"performer_favorite"`
	// Filter texts by performer age at time of text
	PerformerAge *IntCriterionInput `json:"performer_age"`
	// Filter to only include texts with these performers
	Performers *MultiCriterionInput `json:"performers"`
	// Filter by performer count
	PerformerCount *IntCriterionInput `json:"performer_count"`
	// Filter by url
	URL *StringCriterionInput `json:"url"`
	// Filter by resume location
	ResumeLocation *StringCriterionInput `json:"resume_location"`
	// Filter by read count
	ReadCount *IntCriterionInput `json:"read_count"`
	// Filter by read duration (in seconds)
	ReadDuration *IntCriterionInput `json:"read_duration"`
	// Filter by last read at
	LastReadAt *TimestampCriterionInput `json:"last_read_at"`
	// Filter by date
	Date *DateCriterionInput `json:"date"`
	// Filter by related performers that meet this criteria
	PerformersFilter *PerformerFilterType `json:"performers_filter"`
	// Filter by related studios that meet this criteria
	StudiosFilter *StudioFilterType `json:"studios_filter"`
	// Filter by related tags that meet this criteria
	TagsFilter *TagFilterType `json:"tags_filter"`
	// Filter by related bookmarks that meet this criteria
	BookmarksFilter *TextBookmarkFilterType `json:"bookmarks_filter"`
	// Filter by created at
	CreatedAt *TimestampCriterionInput `json:"created_at"`
	// Filter by updated at
	UpdatedAt *TimestampCriterionInput `json:"updated_at"`
}

type TextQueryOptions struct {
	QueryOptions
	TextFilter *TextFilterType

	TotalWordCount bool
	TotalSize      bool
}

type TextQueryResult struct {
	QueryResult
	TotalWordCount float64
	TotalSize      float64

	getter     TextGetter
	texts      []*Text
	resolveErr error
}

type TextCreateInput struct {
	Title        *string  `json:"title"`
	TagLine      *string  `json:"tag_line"`
	Code         *string  `json:"code"`
	Details      *string  `json:"details"`
	Author       *string  `json:"author"`
	LanguageCode *string  `json:"language_code"`
	URL          *string  `json:"url"`
	Urls         []string `json:"urls"`
	Date         *string  `json:"date"`
	Rating100    *int     `json:"rating100"`
	Organized    *bool    `json:"organized"`
	StudioID     *string  `json:"studio_id"`
	PerformerIds []string `json:"performer_ids"`
	TagIds       []string `json:"tag_ids"`
	// This should be a URL or a base64 encoded data URL
	CoverImage *string `json:"cover_image"`
	// The first id will be assigned as primary.
	// Files will be reassigned from existing texts if applicable.
	// Files must not already be primary for another text.
	FileIds []string `json:"file_ids"`
}

type TextUpdateInput struct {
	ClientMutationID *string  `json:"clientMutationId"`
	ID               string   `json:"id"`
	Title            *string  `json:"title"`
	TagLine          *string  `json:"tag_line"`
	Code             *string  `json:"code"`
	Details          *string  `json:"details"`
	Author           *string  `json:"author"`
	LanguageCode     *string  `json:"language_code"`
	URL              *string  `json:"url"`
	Urls             []string `json:"urls"`
	Date             *string  `json:"date"`
	Rating100        *int     `json:"rating100"`
	OCounter         *int     `json:"o_counter"`
	Organized        *bool    `json:"organized"`
	StudioID         *string  `json:"studio_id"`
	PerformerIds     []string `json:"performer_ids"`
	TagIds           []string `json:"tag_ids"`
	// This should be a URL or a base64 encoded data URL
	CoverImage     *string  `json:"cover_image"`
	ResumeLocation *string  `json:"resume_time"`
	ReadDuration   *float64 `json:"read_duration"`
	ReadCount      *int     `json:"read_count"`
	PrimaryFileID  *string  `json:"primary_file_id"`
}

type TextDestroyInput struct {
	ID              string `json:"id"`
	DeleteFile      *bool  `json:"delete_file"`
	DeleteGenerated *bool  `json:"delete_generated"`
}

type TextsDestroyInput struct {
	Ids             []string `json:"ids"`
	DeleteFile      *bool    `json:"delete_file"`
	DeleteGenerated *bool    `json:"delete_generated"`
}

func NewTextQueryResult(getter TextGetter) *TextQueryResult {
	return &TextQueryResult{
		getter: getter,
	}
}

func (r *TextQueryResult) Resolve(ctx context.Context) ([]*Text, error) {
	// cache results
	if r.texts == nil && r.resolveErr == nil {
		r.texts, r.resolveErr = r.getter.FindMany(ctx, r.IDs)
	}
	return r.texts, r.resolveErr
}
