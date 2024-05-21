package scraper

import (
	"github.com/stashapp/stash/pkg/models"
)

type ScrapedStory struct {
	Title         *string  `json:"title"`
	TagLine       *string  `json:"tag_line"`
	Code          *string  `json:"code"`
	Content       *string  `json:"content"`
	Details       *string  `json:"details"`
	Author        *string  `json:"author"`
	URLs          []string `json:"urls"`
	DatePublished *string  `json:"date_published"`
	DateUpdated   *string  `json:"date_updated"`
	// This should be a base64 encoded data URL
	Image        *string                       `json:"image"`
	Studio       *models.ScrapedStudio         `json:"studio"`
	Tags         []*models.ScrapedTag          `json:"tags"`
	Performers   []*models.ScrapedPerformer    `json:"performers"`
	RemoteSiteID *string                       `json:"remote_site_id"`
	Fingerprints []*models.StashBoxFingerprint `json:"fingerprints"`
}

func (ScrapedStory) IsScrapedContent() {}

type ScrapedStoryInput struct {
	Title         *string  `json:"title"`
	TagLine       *string  `json:"tag_line"`
	Code          *string  `json:"code"`
	Content       *string  `json:"content"`
	Details       *string  `json:"details"`
	Author        *string  `json:"author"`
	URLs          []string `json:"urls"`
	DatePublished *string  `json:"date_published"`
	DateUpdated   *string  `json:"date_updated"`
	RemoteSiteID  *string  `json:"remote_site_id"`
}
