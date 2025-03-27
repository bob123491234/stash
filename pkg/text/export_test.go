package text

import (
	"errors"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/models/json"
	"github.com/stashapp/stash/pkg/models/jsonschema"
	"github.com/stashapp/stash/pkg/models/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"testing"
	"time"
)

const (
	textID     = 1
	noImageID  = 2
	errImageID = 3

	studioID        = 4
	missingStudioID = 5
	errStudioID     = 6

	noTagsID  = 11
	errTagsID = 12

	noGroupsID     = 13
	errFindGroupID = 15

	noBookmarksID       = 16
	errBookmarksID      = 17
	errFindPrimaryTagID = 18
	errFindByBookmarkID = 19
)

var (
	url        = "url"
	title      = "title"
	date       = "2001-01-01"
	dateObj, _ = models.ParseDate(date)
	rating     = 5
	organized  = true
	details    = "details"
)

var (
	studioName = "studioName"
	// galleryChecksum = "galleryChecksum"

	validGroup1  = 1
	validGroup2  = 2
	invalidGroup = 3

	group1Name = "group1Name"
	group2Name = "group2Name"

	group1Text = 1
	group2Text = 2
)

var names = []string{
	"name1",
	"name2",
}

var imageBytes = []byte("imageBytes")

var stashID = models.StashID{
	StashID:  "StashID",
	Endpoint: "Endpoint",
}

const (
	path        = "path"
	imageBase64 = "aW1hZ2VCeXRlcw=="
)

var (
	createTime = time.Date(2001, 01, 01, 0, 0, 0, 0, time.UTC)
	updateTime = time.Date(2002, 01, 01, 0, 0, 0, 0, time.UTC)
)

func createFullText(id int) models.Text {
	return models.Text{
		ID:        id,
		Title:     title,
		Date:      &dateObj,
		Details:   details,
		Rating:    &rating,
		Organized: organized,
		URLs:      models.NewRelatedStrings([]string{url}),
		Files: models.NewRelatedVideoFiles([]*models.VideoFile{
			{
				BaseFile: &models.BaseFile{
					Path: path,
				},
			},
		}),
		CreatedAt: createTime,
		UpdatedAt: updateTime,
	}
}

func createEmptyText(id int) models.Text {
	return models.Text{
		ID: id,
		Files: models.NewRelatedVideoFiles([]*models.VideoFile{
			{
				BaseFile: &models.BaseFile{
					Path: path,
				},
			},
		}),
		URLs:      models.NewRelatedStrings([]string{}),
		CreatedAt: createTime,
		UpdatedAt: updateTime,
	}
}

func createFullJSONText(image string) *jsonschema.Text {
	return &jsonschema.Text{
		Title:     title,
		Files:     []string{path},
		Date:      date,
		Details:   details,
		Rating:    rating,
		Organized: organized,
		URLs:      []string{url},
		CreatedAt: json.JSONTime{
			Time: createTime,
		},
		UpdatedAt: json.JSONTime{
			Time: updateTime,
		},
		Cover: image,
		StashIDs: []models.StashID{
			stashID,
		},
	}
}

func createEmptyJSONText() *jsonschema.Text {
	return &jsonschema.Text{
		URLs:  []string{},
		Files: []string{path},
		CreatedAt: json.JSONTime{
			Time: createTime,
		},
		UpdatedAt: json.JSONTime{
			Time: updateTime,
		},
	}
}

type basicTestScenario struct {
	input    models.Text
	expected *jsonschema.Text
	err      bool
}

var scenarios = []basicTestScenario{
	{
		createFullText(textID),
		createFullJSONText(imageBase64),
		false,
	},
	{
		createEmptyText(noImageID),
		createEmptyJSONText(),
		false,
	},
	{
		createFullText(errImageID),
		createFullJSONText(""),
		// failure to get image should not cause an error
		false,
	},
}

func TestToJSON(t *testing.T) {
	db := mocks.NewDatabase()

	imageErr := errors.New("error getting image")

	db.Text.On("GetCover", testCtx, textID).Return(imageBytes, nil).Once()
	db.Text.On("GetCover", testCtx, noImageID).Return(nil, nil).Once()
	db.Text.On("GetCover", testCtx, errImageID).Return(nil, imageErr).Once()
	db.Text.On("GetViewDates", testCtx, mock.Anything).Return(nil, nil)
	db.Text.On("GetODates", testCtx, mock.Anything).Return(nil, nil)

	for i, s := range scenarios {
		text := s.input
		json, err := ToBasicJSON(testCtx, db.Text, &text)

		switch {
		case !s.err && err != nil:
			t.Errorf("[%d] unexpected error: %s", i, err.Error())
		case s.err && err == nil:
			t.Errorf("[%d] expected error not returned", i)
		default:
			assert.Equal(t, s.expected, json, "[%d]", i)
		}
	}

	db.AssertExpectations(t)
}

func createStudioText(studioID int) models.Text {
	return models.Text{
		StudioID: &studioID,
	}
}

type stringTestScenario struct {
	input    models.Text
	expected string
	err      bool
}

var getStudioScenarios = []stringTestScenario{
	{
		createStudioText(studioID),
		studioName,
		false,
	},
	{
		createStudioText(missingStudioID),
		"",
		false,
	},
	{
		createStudioText(errStudioID),
		"",
		true,
	},
}

func TestGetStudioName(t *testing.T) {
	db := mocks.NewDatabase()

	studioErr := errors.New("error getting image")

	db.Studio.On("Find", testCtx, studioID).Return(&models.Studio{
		Name: studioName,
	}, nil).Once()
	db.Studio.On("Find", testCtx, missingStudioID).Return(nil, nil).Once()
	db.Studio.On("Find", testCtx, errStudioID).Return(nil, studioErr).Once()

	for i, s := range getStudioScenarios {
		text := s.input
		json, err := GetStudioName(testCtx, db.Studio, &text)

		switch {
		case !s.err && err != nil:
			t.Errorf("[%d] unexpected error: %s", i, err.Error())
		case s.err && err == nil:
			t.Errorf("[%d] expected error not returned", i)
		default:
			assert.Equal(t, s.expected, json, "[%d]", i)
		}
	}

	db.AssertExpectations(t)
}

type stringSliceTestScenario struct {
	input    models.Text
	expected []string
	err      bool
}

var getTagNamesScenarios = []stringSliceTestScenario{
	{
		createEmptyText(textID),
		names,
		false,
	},
	{
		createEmptyText(noTagsID),
		nil,
		false,
	},
	{
		createEmptyText(errTagsID),
		nil,
		true,
	},
}

func getTags(names []string) []*models.Tag {
	var ret []*models.Tag
	for _, n := range names {
		ret = append(ret, &models.Tag{
			Name: n,
		})
	}

	return ret
}

func TestGetTagNames(t *testing.T) {
	db := mocks.NewDatabase()

	tagErr := errors.New("error getting tag")

	db.Tag.On("FindByTextID", testCtx, textID).Return(getTags(names), nil).Once()
	db.Tag.On("FindByTextID", testCtx, noTagsID).Return(nil, nil).Once()
	db.Tag.On("FindByTextID", testCtx, errTagsID).Return(nil, tagErr).Once()

	for i, s := range getTagNamesScenarios {
		text := s.input
		json, err := GetTagNames(testCtx, db.Tag, &text)

		switch {
		case !s.err && err != nil:
			t.Errorf("[%d] unexpected error: %s", i, err.Error())
		case s.err && err == nil:
			t.Errorf("[%d] expected error not returned", i)
		default:
			assert.Equal(t, s.expected, json, "[%d]", i)
		}
	}

	db.AssertExpectations(t)
}

type textGroupsTestScenario struct {
	input    models.Text
	expected []jsonschema.TextGroup
	err      bool
}

var validGroups = models.NewRelatedGroups([]models.GroupsTexts{
	{
		GroupID:   validGroup1,
		TextIndex: &group1Text,
	},
	{
		GroupID:   validGroup2,
		TextIndex: &group2Text,
	},
})

var invalidGroups = models.NewRelatedGroups([]models.GroupsTexts{
	{
		GroupID:   invalidGroup,
		TextIndex: &group1Text,
	},
})

var getTextGroupsJSONScenarios = []textGroupsTestScenario{
	{
		models.Text{
			ID:     textID,
			Groups: validGroups,
		},
		[]jsonschema.TextGroup{
			{
				GroupName: group1Name,
				TextIndex: group1Text,
			},
			{
				GroupName: group2Name,
				TextIndex: group2Text,
			},
		},
		false,
	},
	{
		models.Text{
			ID:     noGroupsID,
			Groups: models.NewRelatedGroups([]models.GroupsTexts{}),
		},
		nil,
		false,
	},
	{
		models.Text{
			ID:     errFindGroupID,
			Groups: invalidGroups,
		},
		nil,
		true,
	},
}

func TestGetTextGroupsJSON(t *testing.T) {
	db := mocks.NewDatabase()

	groupErr := errors.New("error getting group")

	db.Group.On("Find", testCtx, validGroup1).Return(&models.Group{
		Name: group1Name,
	}, nil).Once()
	db.Group.On("Find", testCtx, validGroup2).Return(&models.Group{
		Name: group2Name,
	}, nil).Once()
	db.Group.On("Find", testCtx, invalidGroup).Return(nil, groupErr).Once()

	for i, s := range getTextGroupsJSONScenarios {
		text := s.input
		json, err := GetTextGroupsJSON(testCtx, db.Group, &text)

		switch {
		case !s.err && err != nil:
			t.Errorf("[%d] unexpected error: %s", i, err.Error())
		case s.err && err == nil:
			t.Errorf("[%d] expected error not returned", i)
		default:
			assert.Equal(t, s.expected, json, "[%d]", i)
		}
	}

	db.AssertExpectations(t)
}

const (
	validBookmarkID1 = 1
	validBookmarkID2 = 2

	invalidBookmarkID1 = 3
	invalidBookmarkID2 = 4

	validTagID1 = 1
	validTagID2 = 2

	validTagName1 = "validTagName1"
	validTagName2 = "validTagName2"

	invalidTagID = 3

	bookmarkTitle1 = "bookmarkTitle1"
	bookmarkTitle2 = "bookmarkTitle2"

	bookmarkSeconds1 = 1.0
	bookmarkSeconds2 = 2.3

	bookmarkSeconds1Str = "1.0"
	bookmarkSeconds2Str = "2.3"
)

type textBookmarksTestScenario struct {
	input    models.Text
	expected []jsonschema.TextBookmark
	err      bool
}

var getTextBookmarksJSONScenarios = []textBookmarksTestScenario{
	{
		createEmptyText(textID),
		[]jsonschema.TextBookmark{
			{
				Title:      bookmarkTitle1,
				PrimaryTag: validTagName1,
				Seconds:    bookmarkSeconds1Str,
				Tags: []string{
					validTagName1,
					validTagName2,
				},
				CreatedAt: json.JSONTime{
					Time: createTime,
				},
				UpdatedAt: json.JSONTime{
					Time: updateTime,
				},
			},
			{
				Title:      bookmarkTitle2,
				PrimaryTag: validTagName2,
				Seconds:    bookmarkSeconds2Str,
				Tags: []string{
					validTagName2,
				},
				CreatedAt: json.JSONTime{
					Time: createTime,
				},
				UpdatedAt: json.JSONTime{
					Time: updateTime,
				},
			},
		},
		false,
	},
	{
		createEmptyText(noBookmarksID),
		nil,
		false,
	},
	{
		createEmptyText(errBookmarksID),
		nil,
		true,
	},
	{
		createEmptyText(errFindPrimaryTagID),
		nil,
		true,
	},
	{
		createEmptyText(errFindByBookmarkID),
		nil,
		true,
	},
}

var validBookmarks = []*models.TextBookmark{
	{
		ID:           validBookmarkID1,
		Title:        bookmarkTitle1,
		PrimaryTagID: validTagID1,
		Seconds:      bookmarkSeconds1,
		CreatedAt:    createTime,
		UpdatedAt:    updateTime,
	},
	{
		ID:           validBookmarkID2,
		Title:        bookmarkTitle2,
		PrimaryTagID: validTagID2,
		Seconds:      bookmarkSeconds2,
		CreatedAt:    createTime,
		UpdatedAt:    updateTime,
	},
}

var invalidBookmarks1 = []*models.TextBookmark{
	{
		ID:           invalidBookmarkID1,
		PrimaryTagID: invalidTagID,
	},
}

var invalidBookmarks2 = []*models.TextBookmark{
	{
		ID:           invalidBookmarkID2,
		PrimaryTagID: validTagID1,
	},
}

func TestGetTextBookmarksJSON(t *testing.T) {
	db := mocks.NewDatabase()

	bookmarksErr := errors.New("error getting text bookmarks")
	tagErr := errors.New("error getting tags")

	db.TextBookmark.On("FindByTextID", testCtx, textID).Return(validBookmarks, nil).Once()
	db.TextBookmark.On("FindByTextID", testCtx, noBookmarksID).Return(nil, nil).Once()
	db.TextBookmark.On("FindByTextID", testCtx, errBookmarksID).Return(nil, bookmarksErr).Once()
	db.TextBookmark.On("FindByTextID", testCtx, errFindPrimaryTagID).Return(invalidBookmarks1, nil).Once()
	db.TextBookmark.On("FindByTextID", testCtx, errFindByBookmarkID).Return(invalidBookmarks2, nil).Once()

	db.Tag.On("Find", testCtx, validTagID1).Return(&models.Tag{
		Name: validTagName1,
	}, nil)
	db.Tag.On("Find", testCtx, validTagID2).Return(&models.Tag{
		Name: validTagName2,
	}, nil)
	db.Tag.On("Find", testCtx, invalidTagID).Return(nil, tagErr)

	db.Tag.On("FindByTextBookmarkID", testCtx, validBookmarkID1).Return([]*models.Tag{
		{
			Name: validTagName1,
		},
		{
			Name: validTagName2,
		},
	}, nil)
	db.Tag.On("FindByTextBookmarkID", testCtx, validBookmarkID2).Return([]*models.Tag{
		{
			Name: validTagName2,
		},
	}, nil)
	db.Tag.On("FindByTextBookmarkID", testCtx, invalidBookmarkID2).Return(nil, tagErr).Once()

	for i, s := range getTextBookmarksJSONScenarios {
		text := s.input
		json, err := GetTextBookmarksJSON(testCtx, db.TextBookmark, db.Tag, &text)

		switch {
		case !s.err && err != nil:
			t.Errorf("[%d] unexpected error: %s", i, err.Error())
		case s.err && err == nil:
			t.Errorf("[%d] expected error not returned", i)
		default:
			assert.Equal(t, s.expected, json, "[%d]", i)
		}
	}

	db.AssertExpectations(t)
}
