package models

import (
	"time"
)

type TextBookmark struct {
	ID           int       `json:"id"`
	Title        string    `json:"title"`
	Location     string    `json:"location"`
	PrimaryTagID int       `json:"primary_tag_id"`
	TextID       int       `json:"text_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func NewTextBookmark() TextBookmark {
	currentTime := time.Now()
	return TextBookmark{
		CreatedAt: currentTime,
		UpdatedAt: currentTime,
	}
}

// TextBookmarkPartial represents part of a TextBookmark object.
// It is used to update the database entry.
type TextBookmarkPartial struct {
	Title        OptionalString
	Location     OptionalString
	PrimaryTagID OptionalInt
	TextID       OptionalInt
	CreatedAt    OptionalTime
	UpdatedAt    OptionalTime
}

func NewTextBookmarkPartial() TextBookmarkPartial {
	currentTime := time.Now()
	return TextBookmarkPartial{
		UpdatedAt: NewOptionalTime(currentTime),
	}
}
