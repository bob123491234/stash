package models

import (
	"context"
	"path/filepath"
	"strconv"
	"time"
)

// Text stores the metadata for a single video text.
type Text struct {
	ID           int    `json:"id"`
	Title        string `json:"title"`
	TagLine      string `json:"tag_line"`
	Code         string `json:"code"`
	Details      string `json:"details"`
	Author       string `json:"author"`
	LanguageCode string `json:"language_code"`
	Date         *Date  `json:"date"`
	// Rating expressed in 1-100 scale
	Rating    *int `json:"rating"`
	Organized bool `json:"organized"`
	StudioID  *int `json:"studio_id"`

	// transient - not persisted
	Files         RelatedFiles
	PrimaryFileID *FileID
	// transient - path of primary file - empty if no files
	Path string
	// transient - checksum of primary file - empty if no files
	Checksum string

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	ResumeLocation string  `json:"resume_location"`
	ReadDuration   float64 `json:"read_duration"`

	URLs         RelatedStrings `json:"urls"`
	TagIDs       RelatedIDs     `json:"tag_ids"`
	PerformerIDs RelatedIDs     `json:"performer_ids"`
}

func NewText() Text {
	currentTime := time.Now()
	return Text{
		CreatedAt: currentTime,
		UpdatedAt: currentTime,
	}
}

// TextPartial represents part of a Text object. It is used to update
// the database entry.
type TextPartial struct {
	Title        OptionalString
	TagLine      OptionalString
	Code         OptionalString
	Details      OptionalString
	Author       OptionalString
	LanguageCode OptionalString
	Date         OptionalDate
	// Rating expressed in 1-100 scale
	Rating         OptionalInt
	Organized      OptionalBool
	StudioID       OptionalInt
	CreatedAt      OptionalTime
	UpdatedAt      OptionalTime
	ResumeLocation OptionalString
	ReadDuration   OptionalFloat64

	URLs          *UpdateStrings
	GalleryIDs    *UpdateIDs
	TagIDs        *UpdateIDs
	PerformerIDs  *UpdateIDs
	GroupIDs      *UpdateGroupIDs
	StashIDs      *UpdateStashIDs
	PrimaryFileID *FileID
}

func NewTextPartial() TextPartial {
	currentTime := time.Now()
	return TextPartial{
		UpdatedAt: NewOptionalTime(currentTime),
	}
}

func (t *Text) LoadURLs(ctx context.Context, l URLLoader) error {
	return t.URLs.load(func() ([]string, error) {
		return l.GetURLs(ctx, t.ID)
	})
}

func (t *Text) LoadFiles(ctx context.Context, l FileLoader) error {
	return t.Files.load(func() ([]File, error) {
		return l.GetFiles(ctx, t.ID)
	})
}

func (t *Text) LoadPrimaryFile(ctx context.Context, l FileGetter) error {
	return t.Files.loadPrimary(func() (File, error) {
		if t.PrimaryFileID == nil {
			return nil, nil
		}

		f, err := l.Find(ctx, *t.PrimaryFileID)
		if err != nil {
			return nil, err
		}

		if len(f) > 0 {
			return f[0], nil
		}

		return nil, nil
	})
}

func (t *Text) LoadPerformerIDs(ctx context.Context, l PerformerIDLoader) error {
	return t.PerformerIDs.load(func() ([]int, error) {
		return l.GetPerformerIDs(ctx, t.ID)
	})
}

func (t *Text) LoadTagIDs(ctx context.Context, l TagIDLoader) error {
	return t.TagIDs.load(func() ([]int, error) {
		return l.GetTagIDs(ctx, t.ID)
	})
}

func (t *Text) LoadRelationships(ctx context.Context, l TextReader) error {
	if err := t.LoadURLs(ctx, l); err != nil {
		return err
	}

	if err := t.LoadPerformerIDs(ctx, l); err != nil {
		return err
	}

	if err := t.LoadTagIDs(ctx, l); err != nil {
		return err
	}

	// if err := t.LoadFiles(ctx, l); err != nil {
	// 	return err
	// }

	return nil
}

// UpdateInput constructs a TextUpdateInput using the populated fields in the TextPartial object.
func (t TextPartial) UpdateInput(id int) TextUpdateInput {
	var dateStr *string
	if t.Date.Set {
		d := t.Date.Value
		v := d.String()
		dateStr = &v
	}

	ret := TextUpdateInput{
		ID:           strconv.Itoa(id),
		Title:        t.Title.Ptr(),
		TagLine:      t.TagLine.Ptr(),
		Code:         t.Code.Ptr(),
		Details:      t.Details.Ptr(),
		Author:       t.Author.Ptr(),
		LanguageCode: t.LanguageCode.Ptr(),
		Urls:         t.URLs.Strings(),
		Date:         dateStr,
		Rating100:    t.Rating.Ptr(),
		Organized:    t.Organized.Ptr(),
		StudioID:     t.StudioID.StringPtr(),
		PerformerIds: t.PerformerIDs.IDStrings(),
		TagIds:       t.TagIDs.IDStrings(),
	}

	return ret
}

// GetTitle returns the title of the text. If the Title field is empty,
// then the base filename is returned.
func (t Text) GetTitle() string {
	if t.Title != "" {
		return t.Title
	}

	return filepath.Base(t.Path)
}

// DisplayName returns a display name for the text for logging purposes.
// It returns Path if not empty, otherwise it returns the ID.
func (t Text) DisplayName() string {
	if t.Path != "" {
		return t.Path
	}

	return strconv.Itoa(t.ID)
}

// TextFileType represents the file metadata for a text.
type TextFileType struct {
	Size *string `graphql:"size" json:"size"`
}
