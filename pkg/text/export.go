package text

import (
	"context"
	"fmt"
	"math"
	"strconv"

	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/models/json"
	"github.com/stashapp/stash/pkg/models/jsonschema"
	"github.com/stashapp/stash/pkg/sliceutil"
	"github.com/stashapp/stash/pkg/utils"
)

type ExportGetter interface {
	models.ViewDateReader
	models.ODateReader
	GetCover(ctx context.Context, textID int) ([]byte, error)
}

type TagFinder interface {
	models.TagGetter
	FindByTextID(ctx context.Context, textID int) ([]*models.Tag, error)
	FindByTextBookmarkID(ctx context.Context, textBookmarkID int) ([]*models.Tag, error)
}

// ToBasicJSON converts a text object into its JSON object equivalent. It
// does not convert the relationships to other objects, with the exception
// of cover image.
func ToBasicJSON(ctx context.Context, reader ExportGetter, text *models.Text) (*jsonschema.Text, error) {
	newTextJSON := jsonschema.Text{
		Title:        text.Title,
		TagLine:      text.TagLine,
		Code:         text.Code,
		URLs:         text.URLs.List(),
		Details:      text.Details,
		Author:       text.Author,
		LanguageCode: text.LanguageCode,
		CreatedAt:    json.JSONTime{Time: text.CreatedAt},
		UpdatedAt:    json.JSONTime{Time: text.UpdatedAt},
	}

	if text.Date != nil {
		newTextJSON.Date = text.Date.String()
	}

	if text.Rating != nil {
		newTextJSON.Rating = *text.Rating
	}

	newTextJSON.Organized = text.Organized

	for _, f := range text.Files.List() {
		newTextJSON.Files = append(newTextJSON.Files, f.Base().Path)
	}

	cover, err := reader.GetCover(ctx, text.ID)
	if err != nil {
		logger.Errorf("Error getting text cover: %v", err)
	}

	if len(cover) > 0 {
		newTextJSON.Cover = utils.GetBase64StringFromData(cover)
	}

	dates, err := reader.GetViewDates(ctx, text.ID)
	if err != nil {
		return nil, fmt.Errorf("error getting view dates: %v", err)
	}

	for _, date := range dates {
		newTextJSON.ReadHistory = append(newTextJSON.ReadHistory, json.JSONTime{Time: date})
	}

	odates, err := reader.GetODates(ctx, text.ID)
	if err != nil {
		return nil, fmt.Errorf("error getting o dates: %v", err)
	}

	for _, date := range odates {
		newTextJSON.OHistory = append(newTextJSON.OHistory, json.JSONTime{Time: date})
	}

	return &newTextJSON, nil
}

// GetStudioName returns the name of the provided text's studio. It returns an
// empty string if there is no studio assigned to the text.
func GetStudioName(ctx context.Context, reader models.StudioGetter, text *models.Text) (string, error) {
	if text.StudioID != nil {
		studio, err := reader.Find(ctx, *text.StudioID)
		if err != nil {
			return "", err
		}

		if studio != nil {
			return studio.Name, nil
		}
	}

	return "", nil
}

// GetTagNames returns a slice of tag names corresponding to the provided
// text's tags.
func GetTagNames(ctx context.Context, reader TagFinder, text *models.Text) ([]string, error) {
	tags, err := reader.FindByTextID(ctx, text.ID)
	if err != nil {
		return nil, fmt.Errorf("error getting text tags: %v", err)
	}

	return getTagNames(tags), nil
}

func getTagNames(tags []*models.Tag) []string {
	var results []string
	for _, tag := range tags {
		if tag.Name != "" {
			results = append(results, tag.Name)
		}
	}

	return results
}

// GetDependentTagIDs returns a slice of unique tag IDs that this text references.
func GetDependentTagIDs(ctx context.Context, tags TagFinder, bookmarkReader models.TextBookmarkFinder, text *models.Text) ([]int, error) {
	var ret []int

	t, err := tags.FindByTextID(ctx, text.ID)
	if err != nil {
		return nil, err
	}

	for _, tt := range t {
		ret = sliceutil.AppendUnique(ret, tt.ID)
	}

	sm, err := bookmarkReader.FindByTextID(ctx, text.ID)
	if err != nil {
		return nil, err
	}

	for _, smm := range sm {
		ret = sliceutil.AppendUnique(ret, smm.PrimaryTagID)
		smmt, err := tags.FindByTextBookmarkID(ctx, smm.ID)
		if err != nil {
			return nil, fmt.Errorf("invalid tags for text bookmark: %v", err)
		}

		for _, smmtt := range smmt {
			ret = sliceutil.AppendUnique(ret, smmtt.ID)
		}
	}

	return ret, nil
}

// GetTextBookmarksJSON returns a slice of TextBookmark JSON representation
// objects corresponding to the provided text's bookmarks.
func GetTextBookmarksJSON(ctx context.Context, bookmarkReader models.TextBookmarkFinder, tagReader TagFinder, text *models.Text) ([]jsonschema.TextBookmark, error) {
	textBookmarks, err := bookmarkReader.FindByTextID(ctx, text.ID)
	if err != nil {
		return nil, fmt.Errorf("error getting text bookmarks: %v", err)
	}

	var results []jsonschema.TextBookmark

	for _, textBookmark := range textBookmarks {
		primaryTag, err := tagReader.Find(ctx, textBookmark.PrimaryTagID)
		if err != nil {
			return nil, fmt.Errorf("invalid primary tag for text bookmark: %v", err)
		}

		textBookmarkTags, err := tagReader.FindByTextBookmarkID(ctx, textBookmark.ID)
		if err != nil {
			return nil, fmt.Errorf("invalid tags for text bookmark: %v", err)
		}

		textBookmarkJSON := jsonschema.TextBookmark{
			Title:      textBookmark.Title,
			Location:   textBookmark.Location,
			PrimaryTag: primaryTag.Name,
			Tags:       getTagNames(textBookmarkTags),
			CreatedAt:  json.JSONTime{Time: textBookmark.CreatedAt},
			UpdatedAt:  json.JSONTime{Time: textBookmark.UpdatedAt},
		}

		results = append(results, textBookmarkJSON)
	}

	return results, nil
}

func getDecimalString(num float64) string {
	if num == 0 {
		return ""
	}

	precision := getPrecision(num)
	if precision == 0 {
		precision = 1
	}
	return fmt.Sprintf("%."+strconv.Itoa(precision)+"f", num)
}

func getPrecision(num float64) int {
	if num == 0 {
		return 0
	}

	e := 1.0
	p := 0
	for (math.Round(num*e) / e) != num {
		e *= 10
		p++
	}
	return p
}
