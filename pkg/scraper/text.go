package scraper

import (
	"github.com/stashapp/stash/pkg/models"
)

type ScrapedText struct {
	Title        *string  `json:"title"`
	TagLine      string   `json:"tag_line"`
	Code         *string  `json:"code"`
	Details      *string  `json:"details"`
	Author       *string  `json:"author"`
	LanguageCode string   `json:"language_code"`
	URL          *string  `json:"url"`
	URLs         []string `json:"urls"`
	Date         *string  `json:"date"`
	// This should be a base64 encoded data URL
	Image      *string                    `json:"image"`
	File       *models.SceneFileType      `json:"file"`
	Studio     *models.ScrapedStudio      `json:"studio"`
	Tags       []*models.ScrapedTag       `json:"tags"`
	Performers []*models.ScrapedPerformer `json:"performers"`
}

func (ScrapedText) IsScrapedContent() {}

type ScrapedTextInput struct {
	Title        *string  `json:"title"`
	TagLine      string   `json:"tag_line"`
	Code         *string  `json:"code"`
	Details      *string  `json:"details"`
	Author       *string  `json:"author"`
	LanguageCode string   `json:"language_code"`
	URL          *string  `json:"url"`
	URLs         []string `json:"urls"`
	Date         *string  `json:"date"`
	RemoteSiteID *string  `json:"remote_site_id"`
}
