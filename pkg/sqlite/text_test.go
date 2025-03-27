//go:build integration
// +build integration

package sqlite_test

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/sliceutil"
	"github.com/stretchr/testify/assert"
)

func loadTextRelationships(ctx context.Context, expected models.Text, actual *models.Text) error {
	if expected.URLs.Loaded() {
		if err := actual.LoadURLs(ctx, db.Text); err != nil {
			return err
		}
	}

	if expected.GalleryIDs.Loaded() {
		if err := actual.LoadGalleryIDs(ctx, db.Text); err != nil {
			return err
		}
	}
	if expected.TagIDs.Loaded() {
		if err := actual.LoadTagIDs(ctx, db.Text); err != nil {
			return err
		}
	}
	if expected.PerformerIDs.Loaded() {
		if err := actual.LoadPerformerIDs(ctx, db.Text); err != nil {
			return err
		}
	}
	if expected.Groups.Loaded() {
		if err := actual.LoadGroups(ctx, db.Text); err != nil {
			return err
		}
	}
	if expected.StashIDs.Loaded() {
		if err := actual.LoadStashIDs(ctx, db.Text); err != nil {
			return err
		}
	}
	if expected.Files.Loaded() {
		if err := actual.LoadFiles(ctx, db.Text); err != nil {
			return err
		}
	}

	// clear Path, Checksum, PrimaryFileID
	if expected.Path == "" {
		actual.Path = ""
	}
	if expected.Checksum == "" {
		actual.Checksum = ""
	}
	if expected.OSHash == "" {
		actual.OSHash = ""
	}
	if expected.PrimaryFileID == nil {
		actual.PrimaryFileID = nil
	}

	return nil
}

func Test_textQueryBuilder_Create(t *testing.T) {
	var (
		title        = "title"
		code         = "1337"
		details      = "details"
		director     = "director"
		url          = "url"
		rating       = 60
		resumeTime   = 10.0
		playDuration = 34.0
		createdAt    = time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
		updatedAt    = time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
		textIndex   = 123
		textIndex2  = 234
		endpoint1    = "endpoint1"
		endpoint2    = "endpoint2"
		stashID1     = "stashid1"
		stashID2     = "stashid2"

		date, _ = models.ParseDate("2003-02-01")

		videoFile = makeFileWithID(fileIdxStartVideoFiles)
	)

	tests := []struct {
		name      string
		newObject models.Text
		wantErr   bool
	}{
		{
			"full",
			models.Text{
				Title:        title,
				Code:         code,
				Details:      details,
				Director:     director,
				URLs:         models.NewRelatedStrings([]string{url}),
				Date:         &date,
				Rating:       &rating,
				Organized:    true,
				StudioID:     &studioIDs[studioIdxWithText],
				CreatedAt:    createdAt,
				UpdatedAt:    updatedAt,
				GalleryIDs:   models.NewRelatedIDs([]int{galleryIDs[galleryIdxWithText]}),
				TagIDs:       models.NewRelatedIDs([]int{tagIDs[tagIdx1WithDupName], tagIDs[tagIdx1WithText]}),
				PerformerIDs: models.NewRelatedIDs([]int{performerIDs[performerIdx1WithText], performerIDs[performerIdx1WithDupName]}),
				Groups: models.NewRelatedGroups([]models.GroupsTexts{
					{
						GroupID:    groupIDs[groupIdxWithText],
						TextIndex: &textIndex,
					},
					{
						GroupID:    groupIDs[groupIdxWithStudio],
						TextIndex: &textIndex2,
					},
				}),
				StashIDs: models.NewRelatedStashIDs([]models.StashID{
					{
						StashID:  stashID1,
						Endpoint: endpoint1,
					},
					{
						StashID:  stashID2,
						Endpoint: endpoint2,
					},
				}),
				ResumeTime:   float64(resumeTime),
				PlayDuration: playDuration,
			},
			false,
		},
		{
			"with file",
			models.Text{
				Title:     title,
				Code:      code,
				Details:   details,
				Director:  director,
				URLs:      models.NewRelatedStrings([]string{url}),
				Date:      &date,
				Rating:    &rating,
				Organized: true,
				StudioID:  &studioIDs[studioIdxWithText],
				Files: models.NewRelatedVideoFiles([]*models.VideoFile{
					videoFile.(*models.VideoFile),
				}),
				CreatedAt:    createdAt,
				UpdatedAt:    updatedAt,
				GalleryIDs:   models.NewRelatedIDs([]int{galleryIDs[galleryIdxWithText]}),
				TagIDs:       models.NewRelatedIDs([]int{tagIDs[tagIdx1WithDupName], tagIDs[tagIdx1WithText]}),
				PerformerIDs: models.NewRelatedIDs([]int{performerIDs[performerIdx1WithText], performerIDs[performerIdx1WithDupName]}),
				Groups: models.NewRelatedGroups([]models.GroupsTexts{
					{
						GroupID:    groupIDs[groupIdxWithText],
						TextIndex: &textIndex,
					},
					{
						GroupID:    groupIDs[groupIdxWithStudio],
						TextIndex: &textIndex2,
					},
				}),
				StashIDs: models.NewRelatedStashIDs([]models.StashID{
					{
						StashID:  stashID1,
						Endpoint: endpoint1,
					},
					{
						StashID:  stashID2,
						Endpoint: endpoint2,
					},
				}),
				ResumeTime:   resumeTime,
				PlayDuration: playDuration,
			},
			false,
		},
		{
			"invalid studio id",
			models.Text{
				StudioID: &invalidID,
			},
			true,
		},
		{
			"invalid gallery id",
			models.Text{
				GalleryIDs: models.NewRelatedIDs([]int{invalidID}),
			},
			true,
		},
		{
			"invalid tag id",
			models.Text{
				TagIDs: models.NewRelatedIDs([]int{invalidID}),
			},
			true,
		},
		{
			"invalid performer id",
			models.Text{
				PerformerIDs: models.NewRelatedIDs([]int{invalidID}),
			},
			true,
		},
		{
			"invalid group id",
			models.Text{
				Groups: models.NewRelatedGroups([]models.GroupsTexts{
					{
						GroupID:    invalidID,
						TextIndex: &textIndex,
					},
				}),
			},
			true,
		},
	}

	qb := db.Text

	for _, tt := range tests {
		runWithRollbackTxn(t, tt.name, func(t *testing.T, ctx context.Context) {
			assert := assert.New(t)

			var fileIDs []models.FileID
			if tt.newObject.Files.Loaded() {
				for _, f := range tt.newObject.Files.List() {
					fileIDs = append(fileIDs, f.ID)
				}
			}

			s := tt.newObject
			if err := qb.Create(ctx, &s, fileIDs); (err != nil) != tt.wantErr {
				t.Errorf("textQueryBuilder.Create() error = %v, wantErr = %v", err, tt.wantErr)
			}

			if tt.wantErr {
				assert.Zero(s.ID)
				return
			}

			assert.NotZero(s.ID)

			copy := tt.newObject
			copy.ID = s.ID

			// load relationships
			if err := loadTextRelationships(ctx, copy, &s); err != nil {
				t.Errorf("loadTextRelationships() error = %v", err)
				return
			}

			assert.Equal(copy, s)

			// ensure can find the text
			found, err := qb.Find(ctx, s.ID)
			if err != nil {
				t.Errorf("textQueryBuilder.Find() error = %v", err)
			}

			if !assert.NotNil(found) {
				return
			}

			// load relationships
			if err := loadTextRelationships(ctx, copy, found); err != nil {
				t.Errorf("loadTextRelationships() error = %v", err)
				return
			}
			assert.Equal(copy, *found)

			return
		})
	}
}

func clearTextFileIDs(text *models.Text) {
	if text.Files.Loaded() {
		for _, f := range text.Files.List() {
			f.Base().ID = 0
		}
	}
}

func makeTextFileWithID(i int) *models.VideoFile {
	ret := makeTextFile(i)
	ret.ID = textFileIDs[i]
	return ret
}

func Test_textQueryBuilder_Update(t *testing.T) {
	var (
		title        = "title"
		code         = "1337"
		details      = "details"
		director     = "director"
		url          = "url"
		rating       = 60
		resumeTime   = 10.0
		playDuration = 34.0
		createdAt    = time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
		updatedAt    = time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
		textIndex   = 123
		textIndex2  = 234
		endpoint1    = "endpoint1"
		endpoint2    = "endpoint2"
		stashID1     = "stashid1"
		stashID2     = "stashid2"

		date, _ = models.ParseDate("2003-02-01")
	)

	tests := []struct {
		name          string
		updatedObject *models.Text
		wantErr       bool
	}{
		{
			"full",
			&models.Text{
				ID:           textIDs[textIdxWithGallery],
				Title:        title,
				Code:         code,
				Details:      details,
				Director:     director,
				URLs:         models.NewRelatedStrings([]string{url}),
				Date:         &date,
				Rating:       &rating,
				Organized:    true,
				StudioID:     &studioIDs[studioIdxWithText],
				CreatedAt:    createdAt,
				UpdatedAt:    updatedAt,
				GalleryIDs:   models.NewRelatedIDs([]int{galleryIDs[galleryIdxWithText]}),
				TagIDs:       models.NewRelatedIDs([]int{tagIDs[tagIdx1WithDupName], tagIDs[tagIdx1WithText]}),
				PerformerIDs: models.NewRelatedIDs([]int{performerIDs[performerIdx1WithText], performerIDs[performerIdx1WithDupName]}),
				Groups: models.NewRelatedGroups([]models.GroupsTexts{
					{
						GroupID:    groupIDs[groupIdxWithText],
						TextIndex: &textIndex,
					},
					{
						GroupID:    groupIDs[groupIdxWithStudio],
						TextIndex: &textIndex2,
					},
				}),
				StashIDs: models.NewRelatedStashIDs([]models.StashID{
					{
						StashID:  stashID1,
						Endpoint: endpoint1,
					},
					{
						StashID:  stashID2,
						Endpoint: endpoint2,
					},
				}),
				ResumeTime:   resumeTime,
				PlayDuration: playDuration,
			},
			false,
		},
		{
			"clear nullables",
			&models.Text{
				ID:           textIDs[textIdxWithSpacedName],
				GalleryIDs:   models.NewRelatedIDs([]int{}),
				TagIDs:       models.NewRelatedIDs([]int{}),
				PerformerIDs: models.NewRelatedIDs([]int{}),
				Groups:       models.NewRelatedGroups([]models.GroupsTexts{}),
				StashIDs:     models.NewRelatedStashIDs([]models.StashID{}),
			},
			false,
		},
		{
			"clear gallery ids",
			&models.Text{
				ID:         textIDs[textIdxWithGallery],
				GalleryIDs: models.NewRelatedIDs([]int{}),
			},
			false,
		},
		{
			"clear tag ids",
			&models.Text{
				ID:     textIDs[textIdxWithTag],
				TagIDs: models.NewRelatedIDs([]int{}),
			},
			false,
		},
		{
			"clear performer ids",
			&models.Text{
				ID:           textIDs[textIdxWithPerformer],
				PerformerIDs: models.NewRelatedIDs([]int{}),
			},
			false,
		},
		{
			"clear groups",
			&models.Text{
				ID:     textIDs[textIdxWithGroup],
				Groups: models.NewRelatedGroups([]models.GroupsTexts{}),
			},
			false,
		},
		{
			"invalid studio id",
			&models.Text{
				ID:       textIDs[textIdxWithGallery],
				StudioID: &invalidID,
			},
			true,
		},
		{
			"invalid gallery id",
			&models.Text{
				ID:         textIDs[textIdxWithGallery],
				GalleryIDs: models.NewRelatedIDs([]int{invalidID}),
			},
			true,
		},
		{
			"invalid tag id",
			&models.Text{
				ID:     textIDs[textIdxWithGallery],
				TagIDs: models.NewRelatedIDs([]int{invalidID}),
			},
			true,
		},
		{
			"invalid performer id",
			&models.Text{
				ID:           textIDs[textIdxWithGallery],
				PerformerIDs: models.NewRelatedIDs([]int{invalidID}),
			},
			true,
		},
		{
			"invalid group id",
			&models.Text{
				ID: textIDs[textIdxWithSpacedName],
				Groups: models.NewRelatedGroups([]models.GroupsTexts{
					{
						GroupID:    invalidID,
						TextIndex: &textIndex,
					},
				}),
			},
			true,
		},
	}

	qb := db.Text
	for _, tt := range tests {
		runWithRollbackTxn(t, tt.name, func(t *testing.T, ctx context.Context) {
			assert := assert.New(t)

			copy := *tt.updatedObject

			if err := qb.Update(ctx, tt.updatedObject); (err != nil) != tt.wantErr {
				t.Errorf("textQueryBuilder.Update() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			s, err := qb.Find(ctx, tt.updatedObject.ID)
			if err != nil {
				t.Errorf("textQueryBuilder.Find() error = %v", err)
			}

			// load relationships
			if err := loadTextRelationships(ctx, copy, s); err != nil {
				t.Errorf("loadTextRelationships() error = %v", err)
				return
			}

			assert.Equal(copy, *s)
		})
	}
}

func clearTextPartial() models.TextPartial {
	// leave mandatory fields
	return models.TextPartial{
		Title:        models.OptionalString{Set: true, Null: true},
		Code:         models.OptionalString{Set: true, Null: true},
		Details:      models.OptionalString{Set: true, Null: true},
		Director:     models.OptionalString{Set: true, Null: true},
		URLs:         &models.UpdateStrings{Mode: models.RelationshipUpdateModeSet},
		Date:         models.OptionalDate{Set: true, Null: true},
		Rating:       models.OptionalInt{Set: true, Null: true},
		StudioID:     models.OptionalInt{Set: true, Null: true},
		GalleryIDs:   &models.UpdateIDs{Mode: models.RelationshipUpdateModeSet},
		TagIDs:       &models.UpdateIDs{Mode: models.RelationshipUpdateModeSet},
		PerformerIDs: &models.UpdateIDs{Mode: models.RelationshipUpdateModeSet},
		StashIDs:     &models.UpdateStashIDs{Mode: models.RelationshipUpdateModeSet},
	}
}

func Test_textQueryBuilder_UpdatePartial(t *testing.T) {
	var (
		title        = "title"
		code         = "1337"
		details      = "details"
		director     = "director"
		url          = "url"
		rating       = 60
		resumeTime   = 10.0
		playDuration = 34.0
		createdAt    = time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
		updatedAt    = time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
		textIndex   = 123
		textIndex2  = 234
		endpoint1    = "endpoint1"
		endpoint2    = "endpoint2"
		stashID1     = "stashid1"
		stashID2     = "stashid2"

		date, _ = models.ParseDate("2003-02-01")
	)

	tests := []struct {
		name    string
		id      int
		partial models.TextPartial
		want    models.Text
		wantErr bool
	}{
		{
			"full",
			textIDs[textIdxWithSpacedName],
			models.TextPartial{
				Title:    models.NewOptionalString(title),
				Code:     models.NewOptionalString(code),
				Details:  models.NewOptionalString(details),
				Director: models.NewOptionalString(director),
				URLs: &models.UpdateStrings{
					Values: []string{url},
					Mode:   models.RelationshipUpdateModeSet,
				},
				Date:      models.NewOptionalDate(date),
				Rating:    models.NewOptionalInt(rating),
				Organized: models.NewOptionalBool(true),
				StudioID:  models.NewOptionalInt(studioIDs[studioIdxWithText]),
				CreatedAt: models.NewOptionalTime(createdAt),
				UpdatedAt: models.NewOptionalTime(updatedAt),
				GalleryIDs: &models.UpdateIDs{
					IDs:  []int{galleryIDs[galleryIdxWithText]},
					Mode: models.RelationshipUpdateModeSet,
				},
				TagIDs: &models.UpdateIDs{
					IDs:  []int{tagIDs[tagIdx1WithText], tagIDs[tagIdx1WithDupName]},
					Mode: models.RelationshipUpdateModeSet,
				},
				PerformerIDs: &models.UpdateIDs{
					IDs:  []int{performerIDs[performerIdx1WithText], performerIDs[performerIdx1WithDupName]},
					Mode: models.RelationshipUpdateModeSet,
				},
				GroupIDs: &models.UpdateGroupIDs{
					Groups: []models.GroupsTexts{
						{
							GroupID:    groupIDs[groupIdxWithText],
							TextIndex: &textIndex,
						},
						{
							GroupID:    groupIDs[groupIdxWithStudio],
							TextIndex: &textIndex2,
						},
					},
					Mode: models.RelationshipUpdateModeSet,
				},
				StashIDs: &models.UpdateStashIDs{
					StashIDs: []models.StashID{
						{
							StashID:  stashID1,
							Endpoint: endpoint1,
						},
						{
							StashID:  stashID2,
							Endpoint: endpoint2,
						},
					},
					Mode: models.RelationshipUpdateModeSet,
				},
				ResumeTime:   models.NewOptionalFloat64(resumeTime),
				PlayDuration: models.NewOptionalFloat64(playDuration),
			},
			models.Text{
				ID: textIDs[textIdxWithSpacedName],
				Files: models.NewRelatedVideoFiles([]*models.VideoFile{
					makeTextFile(textIdxWithSpacedName),
				}),
				Title:        title,
				Code:         code,
				Details:      details,
				Director:     director,
				URLs:         models.NewRelatedStrings([]string{url}),
				Date:         &date,
				Rating:       &rating,
				Organized:    true,
				StudioID:     &studioIDs[studioIdxWithText],
				CreatedAt:    createdAt,
				UpdatedAt:    updatedAt,
				GalleryIDs:   models.NewRelatedIDs([]int{galleryIDs[galleryIdxWithText]}),
				TagIDs:       models.NewRelatedIDs([]int{tagIDs[tagIdx1WithDupName], tagIDs[tagIdx1WithText]}),
				PerformerIDs: models.NewRelatedIDs([]int{performerIDs[performerIdx1WithText], performerIDs[performerIdx1WithDupName]}),
				Groups: models.NewRelatedGroups([]models.GroupsTexts{
					{
						GroupID:    groupIDs[groupIdxWithText],
						TextIndex: &textIndex,
					},
					{
						GroupID:    groupIDs[groupIdxWithStudio],
						TextIndex: &textIndex2,
					},
				}),
				StashIDs: models.NewRelatedStashIDs([]models.StashID{
					{
						StashID:  stashID1,
						Endpoint: endpoint1,
					},
					{
						StashID:  stashID2,
						Endpoint: endpoint2,
					},
				}),
				ResumeTime:   resumeTime,
				PlayDuration: playDuration,
			},
			false,
		},
		{
			"clear all",
			textIDs[textIdxWithSpacedName],
			clearTextPartial(),
			models.Text{
				ID: textIDs[textIdxWithSpacedName],
				Files: models.NewRelatedVideoFiles([]*models.VideoFile{
					makeTextFile(textIdxWithSpacedName),
				}),
				GalleryIDs:   models.NewRelatedIDs([]int{}),
				TagIDs:       models.NewRelatedIDs([]int{}),
				PerformerIDs: models.NewRelatedIDs([]int{}),
				Groups:       models.NewRelatedGroups([]models.GroupsTexts{}),
				StashIDs:     models.NewRelatedStashIDs([]models.StashID{}),
				PlayDuration: getTextPlayDuration(textIdxWithSpacedName),
				ResumeTime:   getTextResumeTime(textIdxWithSpacedName),
			},
			false,
		},
		{
			"invalid id",
			invalidID,
			models.TextPartial{},
			models.Text{},
			true,
		},
	}
	for _, tt := range tests {
		qb := db.Text

		runWithRollbackTxn(t, tt.name, func(t *testing.T, ctx context.Context) {
			assert := assert.New(t)

			got, err := qb.UpdatePartial(ctx, tt.id, tt.partial)
			if (err != nil) != tt.wantErr {
				t.Errorf("textQueryBuilder.UpdatePartial() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// load relationships
			if err := loadTextRelationships(ctx, tt.want, got); err != nil {
				t.Errorf("loadTextRelationships() error = %v", err)
				return
			}

			// ignore file ids
			clearTextFileIDs(got)

			assert.Equal(tt.want, *got)

			s, err := qb.Find(ctx, tt.id)
			if err != nil {
				t.Errorf("textQueryBuilder.Find() error = %v", err)
			}

			// load relationships
			if err := loadTextRelationships(ctx, tt.want, s); err != nil {
				t.Errorf("loadTextRelationships() error = %v", err)
				return
			}
			// ignore file ids
			clearTextFileIDs(s)

			assert.Equal(tt.want, *s)
		})
	}
}

func Test_textQueryBuilder_UpdatePartialRelationships(t *testing.T) {
	var (
		textIndex  = 123
		textIndex2 = 234
		endpoint1   = "endpoint1"
		endpoint2   = "endpoint2"
		stashID1    = "stashid1"
		stashID2    = "stashid2"

		groupTexts = []models.GroupsTexts{
			{
				GroupID:    groupIDs[groupIdxWithDupName],
				TextIndex: &textIndex,
			},
			{
				GroupID:    groupIDs[groupIdxWithStudio],
				TextIndex: &textIndex2,
			},
		}

		stashIDs = []models.StashID{
			{
				StashID:  stashID1,
				Endpoint: endpoint1,
			},
			{
				StashID:  stashID2,
				Endpoint: endpoint2,
			},
		}
	)

	tests := []struct {
		name    string
		id      int
		partial models.TextPartial
		want    models.Text
		wantErr bool
	}{
		{
			"add galleries",
			textIDs[textIdxWithGallery],
			models.TextPartial{
				GalleryIDs: &models.UpdateIDs{
					IDs:  []int{galleryIDs[galleryIdx1WithImage], galleryIDs[galleryIdx1WithPerformer]},
					Mode: models.RelationshipUpdateModeAdd,
				},
			},
			models.Text{
				GalleryIDs: models.NewRelatedIDs(append(indexesToIDs(galleryIDs, textGalleries[textIdxWithGallery]),
					galleryIDs[galleryIdx1WithImage],
					galleryIDs[galleryIdx1WithPerformer],
				)),
			},
			false,
		},
		{
			"add identical galleries",
			textIDs[textIdxWithGallery],
			models.TextPartial{
				GalleryIDs: &models.UpdateIDs{
					IDs:  []int{galleryIDs[galleryIdx1WithImage], galleryIDs[galleryIdx1WithImage]},
					Mode: models.RelationshipUpdateModeAdd,
				},
			},
			models.Text{
				GalleryIDs: models.NewRelatedIDs(append(indexesToIDs(galleryIDs, textGalleries[textIdxWithGallery]),
					galleryIDs[galleryIdx1WithImage],
				)),
			},
			false,
		},
		{
			"add tags",
			textIDs[textIdxWithTwoTags],
			models.TextPartial{
				TagIDs: &models.UpdateIDs{
					IDs:  []int{tagIDs[tagIdx1WithDupName], tagIDs[tagIdx1WithGallery]},
					Mode: models.RelationshipUpdateModeAdd,
				},
			},
			models.Text{
				TagIDs: models.NewRelatedIDs(append(
					[]int{
						tagIDs[tagIdx1WithGallery],
						tagIDs[tagIdx1WithDupName],
					},
					indexesToIDs(tagIDs, textTags[textIdxWithTwoTags])...,
				)),
			},
			false,
		},
		{
			"add identical tags",
			textIDs[textIdxWithTwoTags],
			models.TextPartial{
				TagIDs: &models.UpdateIDs{
					IDs:  []int{tagIDs[tagIdx1WithDupName], tagIDs[tagIdx1WithDupName]},
					Mode: models.RelationshipUpdateModeAdd,
				},
			},
			models.Text{
				TagIDs: models.NewRelatedIDs(append(
					[]int{
						tagIDs[tagIdx1WithDupName],
					},
					indexesToIDs(tagIDs, textTags[textIdxWithTwoTags])...,
				)),
			},
			false,
		},
		{
			"add performers",
			textIDs[textIdxWithTwoPerformers],
			models.TextPartial{
				PerformerIDs: &models.UpdateIDs{
					IDs:  []int{performerIDs[performerIdx1WithDupName], performerIDs[performerIdx1WithGallery]},
					Mode: models.RelationshipUpdateModeAdd,
				},
			},
			models.Text{
				PerformerIDs: models.NewRelatedIDs(append(indexesToIDs(performerIDs, textPerformers[textIdxWithTwoPerformers]),
					performerIDs[performerIdx1WithDupName],
					performerIDs[performerIdx1WithGallery],
				)),
			},
			false,
		},
		{
			"add identical performers",
			textIDs[textIdxWithTwoPerformers],
			models.TextPartial{
				PerformerIDs: &models.UpdateIDs{
					IDs:  []int{performerIDs[performerIdx1WithDupName], performerIDs[performerIdx1WithDupName]},
					Mode: models.RelationshipUpdateModeAdd,
				},
			},
			models.Text{
				PerformerIDs: models.NewRelatedIDs(append(indexesToIDs(performerIDs, textPerformers[textIdxWithTwoPerformers]),
					performerIDs[performerIdx1WithDupName],
				)),
			},
			false,
		},
		{
			"add groups",
			textIDs[textIdxWithGroup],
			models.TextPartial{
				GroupIDs: &models.UpdateGroupIDs{
					Groups: groupTexts,
					Mode:   models.RelationshipUpdateModeAdd,
				},
			},
			models.Text{
				Groups: models.NewRelatedGroups(append([]models.GroupsTexts{
					{
						GroupID: indexesToIDs(groupIDs, textGroups[textIdxWithGroup])[0],
					},
				}, groupTexts...)),
			},
			false,
		},
		{
			"add groups to empty",
			textIDs[textIdx1WithPerformer],
			models.TextPartial{
				GroupIDs: &models.UpdateGroupIDs{
					Groups: groupTexts,
					Mode:   models.RelationshipUpdateModeAdd,
				},
			},
			models.Text{
				Groups: models.NewRelatedGroups([]models.GroupsTexts{
					{
						GroupID:    groupIDs[groupIdxWithDupName],
						TextIndex: &textIndex,
					},
					{
						GroupID:    groupIDs[groupIdxWithStudio],
						TextIndex: &textIndex2,
					},
				}),
			},
			false,
		},
		{
			"add stash ids",
			textIDs[textIdxWithSpacedName],
			models.TextPartial{
				StashIDs: &models.UpdateStashIDs{
					StashIDs: stashIDs,
					Mode:     models.RelationshipUpdateModeAdd,
				},
			},
			models.Text{
				StashIDs: models.NewRelatedStashIDs(append([]models.StashID{textStashID(textIdxWithSpacedName)}, stashIDs...)),
			},
			false,
		},
		{
			"add duplicate galleries",
			textIDs[textIdxWithGallery],
			models.TextPartial{
				GalleryIDs: &models.UpdateIDs{
					IDs:  []int{galleryIDs[galleryIdxWithText], galleryIDs[galleryIdx1WithPerformer]},
					Mode: models.RelationshipUpdateModeAdd,
				},
			},
			models.Text{
				GalleryIDs: models.NewRelatedIDs(append(indexesToIDs(galleryIDs, textGalleries[textIdxWithGallery]),
					galleryIDs[galleryIdx1WithPerformer],
				)),
			},
			false,
		},
		{
			"add duplicate tags",
			textIDs[textIdxWithTwoTags],
			models.TextPartial{
				TagIDs: &models.UpdateIDs{
					IDs:  []int{tagIDs[tagIdx1WithText], tagIDs[tagIdx1WithGallery]},
					Mode: models.RelationshipUpdateModeAdd,
				},
			},
			models.Text{
				TagIDs: models.NewRelatedIDs(append(
					[]int{tagIDs[tagIdx1WithGallery]},
					indexesToIDs(tagIDs, textTags[textIdxWithTwoTags])...,
				)),
			},
			false,
		},
		{
			"add duplicate performers",
			textIDs[textIdxWithTwoPerformers],
			models.TextPartial{
				PerformerIDs: &models.UpdateIDs{
					IDs:  []int{performerIDs[performerIdx1WithText], performerIDs[performerIdx1WithGallery]},
					Mode: models.RelationshipUpdateModeAdd,
				},
			},
			models.Text{
				PerformerIDs: models.NewRelatedIDs(append(indexesToIDs(performerIDs, textPerformers[textIdxWithTwoPerformers]),
					performerIDs[performerIdx1WithGallery],
				)),
			},
			false,
		},
		{
			"add duplicate groups",
			textIDs[textIdxWithGroup],
			models.TextPartial{
				GroupIDs: &models.UpdateGroupIDs{
					Groups: append([]models.GroupsTexts{
						{
							GroupID:    groupIDs[groupIdxWithText],
							TextIndex: &textIndex,
						},
					},
						groupTexts...,
					),
					Mode: models.RelationshipUpdateModeAdd,
				},
			},
			models.Text{
				Groups: models.NewRelatedGroups(append([]models.GroupsTexts{
					{
						GroupID: indexesToIDs(groupIDs, textGroups[textIdxWithGroup])[0],
					},
				}, groupTexts...)),
			},
			false,
		},
		{
			"add duplicate stash ids",
			textIDs[textIdxWithSpacedName],
			models.TextPartial{
				StashIDs: &models.UpdateStashIDs{
					StashIDs: []models.StashID{
						textStashID(textIdxWithSpacedName),
					},
					Mode: models.RelationshipUpdateModeAdd,
				},
			},
			models.Text{
				StashIDs: models.NewRelatedStashIDs([]models.StashID{textStashID(textIdxWithSpacedName)}),
			},
			false,
		},
		{
			"add invalid galleries",
			textIDs[textIdxWithGallery],
			models.TextPartial{
				GalleryIDs: &models.UpdateIDs{
					IDs:  []int{invalidID},
					Mode: models.RelationshipUpdateModeAdd,
				},
			},
			models.Text{},
			true,
		},
		{
			"add invalid tags",
			textIDs[textIdxWithTwoTags],
			models.TextPartial{
				TagIDs: &models.UpdateIDs{
					IDs:  []int{invalidID},
					Mode: models.RelationshipUpdateModeAdd,
				},
			},
			models.Text{},
			true,
		},
		{
			"add invalid performers",
			textIDs[textIdxWithTwoPerformers],
			models.TextPartial{
				PerformerIDs: &models.UpdateIDs{
					IDs:  []int{invalidID},
					Mode: models.RelationshipUpdateModeAdd,
				},
			},
			models.Text{},
			true,
		},
		{
			"add invalid groups",
			textIDs[textIdxWithGroup],
			models.TextPartial{
				GroupIDs: &models.UpdateGroupIDs{
					Groups: []models.GroupsTexts{
						{
							GroupID: invalidID,
						},
					},
					Mode: models.RelationshipUpdateModeAdd,
				},
			},
			models.Text{},
			true,
		},
		{
			"remove galleries",
			textIDs[textIdxWithGallery],
			models.TextPartial{
				GalleryIDs: &models.UpdateIDs{
					IDs:  []int{galleryIDs[galleryIdxWithText]},
					Mode: models.RelationshipUpdateModeRemove,
				},
			},
			models.Text{
				GalleryIDs: models.NewRelatedIDs([]int{}),
			},
			false,
		},
		{
			"remove tags",
			textIDs[textIdxWithTwoTags],
			models.TextPartial{
				TagIDs: &models.UpdateIDs{
					IDs:  []int{tagIDs[tagIdx1WithText]},
					Mode: models.RelationshipUpdateModeRemove,
				},
			},
			models.Text{
				TagIDs: models.NewRelatedIDs([]int{tagIDs[tagIdx2WithText]}),
			},
			false,
		},
		{
			"remove performers",
			textIDs[textIdxWithTwoPerformers],
			models.TextPartial{
				PerformerIDs: &models.UpdateIDs{
					IDs:  []int{performerIDs[performerIdx1WithText]},
					Mode: models.RelationshipUpdateModeRemove,
				},
			},
			models.Text{
				PerformerIDs: models.NewRelatedIDs([]int{performerIDs[performerIdx2WithText]}),
			},
			false,
		},
		{
			"remove groups",
			textIDs[textIdxWithGroup],
			models.TextPartial{
				GroupIDs: &models.UpdateGroupIDs{
					Groups: []models.GroupsTexts{
						{
							GroupID: groupIDs[groupIdxWithText],
						},
					},
					Mode: models.RelationshipUpdateModeRemove,
				},
			},
			models.Text{
				Groups: models.NewRelatedGroups([]models.GroupsTexts{}),
			},
			false,
		},
		{
			"remove stash ids",
			textIDs[textIdxWithSpacedName],
			models.TextPartial{
				StashIDs: &models.UpdateStashIDs{
					StashIDs: []models.StashID{textStashID(textIdxWithSpacedName)},
					Mode:     models.RelationshipUpdateModeRemove,
				},
			},
			models.Text{
				StashIDs: models.NewRelatedStashIDs([]models.StashID{}),
			},
			false,
		},
		{
			"remove unrelated galleries",
			textIDs[textIdxWithGallery],
			models.TextPartial{
				GalleryIDs: &models.UpdateIDs{
					IDs:  []int{galleryIDs[galleryIdx1WithImage]},
					Mode: models.RelationshipUpdateModeRemove,
				},
			},
			models.Text{
				GalleryIDs: models.NewRelatedIDs([]int{galleryIDs[galleryIdxWithText]}),
			},
			false,
		},
		{
			"remove unrelated tags",
			textIDs[textIdxWithTwoTags],
			models.TextPartial{
				TagIDs: &models.UpdateIDs{
					IDs:  []int{tagIDs[tagIdx1WithPerformer]},
					Mode: models.RelationshipUpdateModeRemove,
				},
			},
			models.Text{
				TagIDs: models.NewRelatedIDs(indexesToIDs(tagIDs, textTags[textIdxWithTwoTags])),
			},
			false,
		},
		{
			"remove unrelated performers",
			textIDs[textIdxWithTwoPerformers],
			models.TextPartial{
				PerformerIDs: &models.UpdateIDs{
					IDs:  []int{performerIDs[performerIdx1WithDupName]},
					Mode: models.RelationshipUpdateModeRemove,
				},
			},
			models.Text{
				PerformerIDs: models.NewRelatedIDs(indexesToIDs(performerIDs, textPerformers[textIdxWithTwoPerformers])),
			},
			false,
		},
		{
			"remove unrelated groups",
			textIDs[textIdxWithGroup],
			models.TextPartial{
				GroupIDs: &models.UpdateGroupIDs{
					Groups: []models.GroupsTexts{
						{
							GroupID: groupIDs[groupIdxWithDupName],
						},
					},
					Mode: models.RelationshipUpdateModeRemove,
				},
			},
			models.Text{
				Groups: models.NewRelatedGroups([]models.GroupsTexts{
					{
						GroupID: indexesToIDs(groupIDs, textGroups[textIdxWithGroup])[0],
					},
				}),
			},
			false,
		},
		{
			"remove unrelated stash ids",
			textIDs[textIdxWithGallery],
			models.TextPartial{
				StashIDs: &models.UpdateStashIDs{
					StashIDs: stashIDs,
					Mode:     models.RelationshipUpdateModeRemove,
				},
			},
			models.Text{
				StashIDs: models.NewRelatedStashIDs([]models.StashID{textStashID(textIdxWithGallery)}),
			},
			false,
		},
	}

	for _, tt := range tests {
		qb := db.Text

		runWithRollbackTxn(t, tt.name, func(t *testing.T, ctx context.Context) {
			assert := assert.New(t)

			got, err := qb.UpdatePartial(ctx, tt.id, tt.partial)
			if (err != nil) != tt.wantErr {
				t.Errorf("textQueryBuilder.UpdatePartial() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			s, err := qb.Find(ctx, tt.id)
			if err != nil {
				t.Errorf("textQueryBuilder.Find() error = %v", err)
			}

			// load relationships
			if err := loadTextRelationships(ctx, tt.want, got); err != nil {
				t.Errorf("loadTextRelationships() error = %v", err)
				return
			}
			if err := loadTextRelationships(ctx, tt.want, s); err != nil {
				t.Errorf("loadTextRelationships() error = %v", err)
				return
			}

			// only compare fields that were in the partial
			if tt.partial.PerformerIDs != nil {
				assert.ElementsMatch(tt.want.PerformerIDs.List(), got.PerformerIDs.List())
				assert.ElementsMatch(tt.want.PerformerIDs.List(), s.PerformerIDs.List())
			}
			if tt.partial.TagIDs != nil {
				assert.ElementsMatch(tt.want.TagIDs.List(), got.TagIDs.List())
				assert.ElementsMatch(tt.want.TagIDs.List(), s.TagIDs.List())
			}
			if tt.partial.GalleryIDs != nil {
				assert.ElementsMatch(tt.want.GalleryIDs.List(), got.GalleryIDs.List())
				assert.ElementsMatch(tt.want.GalleryIDs.List(), s.GalleryIDs.List())
			}
			if tt.partial.GroupIDs != nil {
				assert.ElementsMatch(tt.want.Groups.List(), got.Groups.List())
				assert.ElementsMatch(tt.want.Groups.List(), s.Groups.List())
			}
			if tt.partial.StashIDs != nil {
				assert.ElementsMatch(tt.want.StashIDs.List(), got.StashIDs.List())
				assert.ElementsMatch(tt.want.StashIDs.List(), s.StashIDs.List())
			}
		})
	}
}

func Test_textQueryBuilder_AddO(t *testing.T) {
	tests := []struct {
		name    string
		id      int
		want    int
		wantErr bool
	}{
		{
			"increment",
			textIDs[1],
			1,
			false,
		},
		{
			"invalid",
			invalidID,
			0,
			true,
		},
	}

	qb := db.Text

	for _, tt := range tests {
		runWithRollbackTxn(t, tt.name, func(t *testing.T, ctx context.Context) {
			got, err := qb.AddO(ctx, tt.id, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("textQueryBuilder.AddO() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != tt.want {
				t.Errorf("textQueryBuilder.AddO() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_textQueryBuilder_DeleteO(t *testing.T) {
	tests := []struct {
		name    string
		id      int
		want    int
		wantErr bool
	}{
		{
			"decrement",
			textIDs[2],
			0,
			false,
		},
		{
			"zero",
			textIDs[0],
			0,
			false,
		},
	}

	qb := db.Text

	for _, tt := range tests {
		runWithRollbackTxn(t, tt.name, func(t *testing.T, ctx context.Context) {
			got, err := qb.DeleteO(ctx, tt.id, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("textQueryBuilder.DeleteO() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != tt.want {
				t.Errorf("textQueryBuilder.DeleteO() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_textQueryBuilder_ResetO(t *testing.T) {
	tests := []struct {
		name    string
		id      int
		want    int
		wantErr bool
	}{
		{
			"decrement",
			textIDs[2],
			0,
			false,
		},
		{
			"zero",
			textIDs[0],
			0,
			false,
		},
	}

	qb := db.Text

	for _, tt := range tests {
		runWithRollbackTxn(t, tt.name, func(t *testing.T, ctx context.Context) {
			got, err := qb.ResetO(ctx, tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("textQueryBuilder.ResetO() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("textQueryBuilder.ResetOCounter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_textQueryBuilder_ResetWatchCount(t *testing.T) {
	return
}

func Test_textQueryBuilder_Destroy(t *testing.T) {
	tests := []struct {
		name    string
		id      int
		wantErr bool
	}{
		{
			"valid",
			textIDs[textIdxWithGallery],
			false,
		},
		{
			"invalid",
			invalidID,
			true,
		},
	}

	qb := db.Text

	for _, tt := range tests {
		runWithRollbackTxn(t, tt.name, func(t *testing.T, ctx context.Context) {
			assert := assert.New(t)
			if err := qb.Destroy(ctx, tt.id); (err != nil) != tt.wantErr {
				t.Errorf("textQueryBuilder.Destroy() error = %v, wantErr %v", err, tt.wantErr)
			}

			// ensure cannot be found
			i, err := qb.Find(ctx, tt.id)

			assert.Nil(err)
			assert.Nil(i)
		})
	}
}

func makeTextWithID(index int) *models.Text {
	ret := makeText(index)
	ret.ID = textIDs[index]

	ret.Files = models.NewRelatedVideoFiles([]*models.VideoFile{makeTextFile(index)})

	return ret
}

func Test_textQueryBuilder_Find(t *testing.T) {
	tests := []struct {
		name    string
		id      int
		want    *models.Text
		wantErr bool
	}{
		{
			"valid",
			textIDs[textIdxWithSpacedName],
			makeTextWithID(textIdxWithSpacedName),
			false,
		},
		{
			"invalid",
			invalidID,
			nil,
			false,
		},
		{
			"with galleries",
			textIDs[textIdxWithGallery],
			makeTextWithID(textIdxWithGallery),
			false,
		},
		{
			"with performers",
			textIDs[textIdxWithTwoPerformers],
			makeTextWithID(textIdxWithTwoPerformers),
			false,
		},
		{
			"with tags",
			textIDs[textIdxWithTwoTags],
			makeTextWithID(textIdxWithTwoTags),
			false,
		},
		{
			"with groups",
			textIDs[textIdxWithGroup],
			makeTextWithID(textIdxWithGroup),
			false,
		},
	}

	qb := db.Text

	for _, tt := range tests {
		runWithRollbackTxn(t, tt.name, func(t *testing.T, ctx context.Context) {
			assert := assert.New(t)
			got, err := qb.Find(ctx, tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("textQueryBuilder.Find() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != nil {
				// load relationships
				if err := loadTextRelationships(ctx, *tt.want, got); err != nil {
					t.Errorf("loadTextRelationships() error = %v", err)
					return
				}

				clearTextFileIDs(got)
			}

			assert.Equal(tt.want, got)
		})
	}
}

func postFindTexts(ctx context.Context, want []*models.Text, got []*models.Text) error {
	for i, s := range got {
		// load relationships
		if i < len(want) {
			if err := loadTextRelationships(ctx, *want[i], s); err != nil {
				return err
			}
		}
		clearTextFileIDs(s)
	}

	return nil
}

func Test_textQueryBuilder_FindMany(t *testing.T) {
	tests := []struct {
		name    string
		ids     []int
		want    []*models.Text
		wantErr bool
	}{
		{
			"valid with relationships",
			[]int{
				textIDs[textIdxWithGallery],
				textIDs[textIdxWithTwoPerformers],
				textIDs[textIdxWithTwoTags],
				textIDs[textIdxWithGroup],
			},
			[]*models.Text{
				makeTextWithID(textIdxWithGallery),
				makeTextWithID(textIdxWithTwoPerformers),
				makeTextWithID(textIdxWithTwoTags),
				makeTextWithID(textIdxWithGroup),
			},
			false,
		},
		{
			"invalid",
			[]int{textIDs[textIdxWithGallery], textIDs[textIdxWithTwoPerformers], invalidID},
			nil,
			true,
		},
	}

	qb := db.Text

	for _, tt := range tests {
		runWithRollbackTxn(t, tt.name, func(t *testing.T, ctx context.Context) {
			assert := assert.New(t)
			got, err := qb.FindMany(ctx, tt.ids)
			if (err != nil) != tt.wantErr {
				t.Errorf("textQueryBuilder.FindMany() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err := postFindTexts(ctx, tt.want, got); err != nil {
				t.Errorf("loadTextRelationships() error = %v", err)
				return
			}

			assert.Equal(tt.want, got)
		})
	}
}

func Test_textQueryBuilder_FindByChecksum(t *testing.T) {
	getChecksum := func(index int) string {
		return getTextStringValue(index, checksumField)
	}

	tests := []struct {
		name     string
		checksum string
		want     []*models.Text
		wantErr  bool
	}{
		{
			"valid",
			getChecksum(textIdxWithSpacedName),
			[]*models.Text{makeTextWithID(textIdxWithSpacedName)},
			false,
		},
		{
			"invalid",
			"invalid checksum",
			nil,
			false,
		},
		{
			"with galleries",
			getChecksum(textIdxWithGallery),
			[]*models.Text{makeTextWithID(textIdxWithGallery)},
			false,
		},
		{
			"with performers",
			getChecksum(textIdxWithTwoPerformers),
			[]*models.Text{makeTextWithID(textIdxWithTwoPerformers)},
			false,
		},
		{
			"with tags",
			getChecksum(textIdxWithTwoTags),
			[]*models.Text{makeTextWithID(textIdxWithTwoTags)},
			false,
		},
		{
			"with groups",
			getChecksum(textIdxWithGroup),
			[]*models.Text{makeTextWithID(textIdxWithGroup)},
			false,
		},
	}

	qb := db.Text

	for _, tt := range tests {
		runWithRollbackTxn(t, tt.name, func(t *testing.T, ctx context.Context) {
			assert := assert.New(t)
			got, err := qb.FindByChecksum(ctx, tt.checksum)
			if (err != nil) != tt.wantErr {
				t.Errorf("textQueryBuilder.FindByChecksum() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err := postFindTexts(ctx, tt.want, got); err != nil {
				t.Errorf("loadTextRelationships() error = %v", err)
				return
			}

			assert.Equal(tt.want, got)
		})
	}
}

func Test_textQueryBuilder_FindByOSHash(t *testing.T) {
	getOSHash := func(index int) string {
		return getTextStringValue(index, "oshash")
	}

	tests := []struct {
		name    string
		oshash  string
		want    []*models.Text
		wantErr bool
	}{
		{
			"valid",
			getOSHash(textIdxWithSpacedName),
			[]*models.Text{makeTextWithID(textIdxWithSpacedName)},
			false,
		},
		{
			"invalid",
			"invalid oshash",
			nil,
			false,
		},
		{
			"with galleries",
			getOSHash(textIdxWithGallery),
			[]*models.Text{makeTextWithID(textIdxWithGallery)},
			false,
		},
		{
			"with performers",
			getOSHash(textIdxWithTwoPerformers),
			[]*models.Text{makeTextWithID(textIdxWithTwoPerformers)},
			false,
		},
		{
			"with tags",
			getOSHash(textIdxWithTwoTags),
			[]*models.Text{makeTextWithID(textIdxWithTwoTags)},
			false,
		},
		{
			"with groups",
			getOSHash(textIdxWithGroup),
			[]*models.Text{makeTextWithID(textIdxWithGroup)},
			false,
		},
	}

	qb := db.Text

	for _, tt := range tests {
		runWithRollbackTxn(t, tt.name, func(t *testing.T, ctx context.Context) {
			got, err := qb.FindByOSHash(ctx, tt.oshash)
			if (err != nil) != tt.wantErr {
				t.Errorf("textQueryBuilder.FindByOSHash() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err := postFindTexts(ctx, tt.want, got); err != nil {
				t.Errorf("loadTextRelationships() error = %v", err)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("textQueryBuilder.FindByOSHash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_textQueryBuilder_FindByPath(t *testing.T) {
	getPath := func(index int) string {
		return getFilePath(folderIdxWithTextFiles, getTextBasename(index))
	}

	tests := []struct {
		name    string
		path    string
		want    []*models.Text
		wantErr bool
	}{
		{
			"valid",
			getPath(textIdxWithSpacedName),
			[]*models.Text{makeTextWithID(textIdxWithSpacedName)},
			false,
		},
		{
			"invalid",
			"invalid path",
			nil,
			false,
		},
		{
			"with galleries",
			getPath(textIdxWithGallery),
			[]*models.Text{makeTextWithID(textIdxWithGallery)},
			false,
		},
		{
			"with performers",
			getPath(textIdxWithTwoPerformers),
			[]*models.Text{makeTextWithID(textIdxWithTwoPerformers)},
			false,
		},
		{
			"with tags",
			getPath(textIdxWithTwoTags),
			[]*models.Text{makeTextWithID(textIdxWithTwoTags)},
			false,
		},
		{
			"with groups",
			getPath(textIdxWithGroup),
			[]*models.Text{makeTextWithID(textIdxWithGroup)},
			false,
		},
	}

	qb := db.Text

	for _, tt := range tests {
		runWithRollbackTxn(t, tt.name, func(t *testing.T, ctx context.Context) {
			assert := assert.New(t)
			got, err := qb.FindByPath(ctx, tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("textQueryBuilder.FindByPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err := postFindTexts(ctx, tt.want, got); err != nil {
				t.Errorf("loadTextRelationships() error = %v", err)
				return
			}

			assert.Equal(tt.want, got)
		})
	}
}

func Test_textQueryBuilder_FindByGalleryID(t *testing.T) {
	tests := []struct {
		name      string
		galleryID int
		want      []*models.Text
		wantErr   bool
	}{
		{
			"valid",
			galleryIDs[galleryIdxWithText],
			[]*models.Text{makeTextWithID(textIdxWithGallery)},
			false,
		},
		{
			"none",
			galleryIDs[galleryIdx1WithPerformer],
			nil,
			false,
		},
	}

	qb := db.Text

	for _, tt := range tests {
		runWithRollbackTxn(t, tt.name, func(t *testing.T, ctx context.Context) {
			assert := assert.New(t)
			got, err := qb.FindByGalleryID(ctx, tt.galleryID)
			if (err != nil) != tt.wantErr {
				t.Errorf("textQueryBuilder.FindByGalleryID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err := postFindTexts(ctx, tt.want, got); err != nil {
				t.Errorf("loadTextRelationships() error = %v", err)
				return
			}

			assert.Equal(tt.want, got)
			return
		})
	}
}

func TestTextCountByPerformerID(t *testing.T) {
	withTxn(func(ctx context.Context) error {
		sqb := db.Text
		count, err := sqb.CountByPerformerID(ctx, performerIDs[performerIdxWithText])

		if err != nil {
			t.Errorf("Error counting texts: %s", err.Error())
		}

		assert.Equal(t, 1, count)

		count, err = sqb.CountByPerformerID(ctx, 0)

		if err != nil {
			t.Errorf("Error counting texts: %s", err.Error())
		}

		assert.Equal(t, 0, count)

		return nil
	})
}

func textsToIDs(i []*models.Text) []int {
	ret := make([]int, len(i))
	for i, v := range i {
		ret[i] = v.ID
	}

	return ret
}

func Test_textStore_FindByFileID(t *testing.T) {
	tests := []struct {
		name    string
		fileID  models.FileID
		include []int
		exclude []int
	}{
		{
			"valid",
			textFileIDs[textIdx1WithPerformer],
			[]int{textIdx1WithPerformer},
			nil,
		},
		{
			"invalid",
			invalidFileID,
			nil,
			[]int{textIdx1WithPerformer},
		},
	}

	qb := db.Text

	for _, tt := range tests {
		runWithRollbackTxn(t, tt.name, func(t *testing.T, ctx context.Context) {
			assert := assert.New(t)
			got, err := qb.FindByFileID(ctx, tt.fileID)
			if err != nil {
				t.Errorf("TextStore.FindByFileID() error = %v", err)
				return
			}
			for _, f := range got {
				clearTextFileIDs(f)
			}

			ids := textsToIDs(got)
			include := indexesToIDs(galleryIDs, tt.include)
			exclude := indexesToIDs(galleryIDs, tt.exclude)

			for _, i := range include {
				assert.Contains(ids, i)
			}
			for _, e := range exclude {
				assert.NotContains(ids, e)
			}
		})
	}
}

func Test_textStore_CountByFileID(t *testing.T) {
	tests := []struct {
		name   string
		fileID models.FileID
		want   int
	}{
		{
			"valid",
			textFileIDs[textIdxWithTwoPerformers],
			1,
		},
		{
			"invalid",
			invalidFileID,
			0,
		},
	}

	qb := db.Text

	for _, tt := range tests {
		runWithRollbackTxn(t, tt.name, func(t *testing.T, ctx context.Context) {
			assert := assert.New(t)
			got, err := qb.CountByFileID(ctx, tt.fileID)
			if err != nil {
				t.Errorf("TextStore.CountByFileID() error = %v", err)
				return
			}

			assert.Equal(tt.want, got)
		})
	}
}

func Test_textStore_CountMissingChecksum(t *testing.T) {
	tests := []struct {
		name string
		want int
	}{
		{
			"valid",
			0,
		},
	}

	qb := db.Text

	for _, tt := range tests {
		runWithRollbackTxn(t, tt.name, func(t *testing.T, ctx context.Context) {
			assert := assert.New(t)
			got, err := qb.CountMissingChecksum(ctx)
			if err != nil {
				t.Errorf("TextStore.CountMissingChecksum() error = %v", err)
				return
			}

			assert.Equal(tt.want, got)
		})
	}
}

func Test_textStore_CountMissingOshash(t *testing.T) {
	tests := []struct {
		name string
		want int
	}{
		{
			"valid",
			0,
		},
	}

	qb := db.Text

	for _, tt := range tests {
		runWithRollbackTxn(t, tt.name, func(t *testing.T, ctx context.Context) {
			assert := assert.New(t)
			got, err := qb.CountMissingOSHash(ctx)
			if err != nil {
				t.Errorf("TextStore.CountMissingOSHash() error = %v", err)
				return
			}

			assert.Equal(tt.want, got)
		})
	}
}

func TestTextWall(t *testing.T) {
	withTxn(func(ctx context.Context) error {
		sqb := db.Text

		const textIdx = 2
		wallQuery := getTextStringValue(textIdx, "Details")
		texts, err := sqb.Wall(ctx, &wallQuery)

		if err != nil {
			t.Errorf("Error finding texts: %s", err.Error())
			return nil
		}

		assert.Len(t, texts, 1)
		text := texts[0]
		assert.Equal(t, textIDs[textIdx], text.ID)
		textPath := getFilePath(folderIdxWithTextFiles, getTextBasename(textIdx))
		assert.Equal(t, textPath, text.Path)

		wallQuery = "not exist"
		texts, err = sqb.Wall(ctx, &wallQuery)

		if err != nil {
			t.Errorf("Error finding text: %s", err.Error())
			return nil
		}

		assert.Len(t, texts, 0)

		return nil
	})
}

func TestTextQueryQ(t *testing.T) {
	const textIdx = 2

	q := getTextStringValue(textIdx, titleField)

	withTxn(func(ctx context.Context) error {
		sqb := db.Text

		textQueryQ(ctx, t, sqb, q, textIdx)

		return nil
	})
}

func queryText(ctx context.Context, t *testing.T, sqb models.TextReader, textFilter *models.TextFilterType, findFilter *models.FindFilterType) []*models.Text {
	t.Helper()
	result, err := sqb.Query(ctx, models.TextQueryOptions{
		QueryOptions: models.QueryOptions{
			FindFilter: findFilter,
			Count:      true,
		},
		TextFilter:   textFilter,
		TotalDuration: true,
		TotalSize:     true,
	})
	if err != nil {
		t.Errorf("Error querying text: %v", err)
		return nil
	}

	texts, err := result.Resolve(ctx)
	if err != nil {
		t.Errorf("Error resolving texts: %v", err)
	}

	return texts
}

func textQueryQ(ctx context.Context, t *testing.T, sqb models.TextReader, q string, expectedTextIdx int) {
	filter := models.FindFilterType{
		Q: &q,
	}
	texts := queryText(ctx, t, sqb, nil, &filter)

	if !assert.Len(t, texts, 1) {
		return
	}
	text := texts[0]
	assert.Equal(t, textIDs[expectedTextIdx], text.ID)

	// no Q should return all results
	filter.Q = nil
	pp := totalTexts
	filter.PerPage = &pp
	texts = queryText(ctx, t, sqb, nil, &filter)

	assert.Len(t, texts, totalTexts)
}

func TestTextQuery(t *testing.T) {
	var (
		endpoint = textStashID(textIdxWithGallery).Endpoint
		stashID  = textStashID(textIdxWithGallery).StashID

		depth = -1
	)

	tests := []struct {
		name        string
		findFilter  *models.FindFilterType
		filter      *models.TextFilterType
		includeIdxs []int
		excludeIdxs []int
		wantErr     bool
	}{
		{
			"specific resume time",
			nil,
			&models.TextFilterType{
				ResumeTime: &models.IntCriterionInput{
					Modifier: models.CriterionModifierEquals,
					Value:    int(getTextResumeTime(textIdxWithGallery)),
				},
			},
			[]int{textIdxWithGallery},
			[]int{textIdxWithGroup},
			false,
		},
		{
			"specific play duration",
			nil,
			&models.TextFilterType{
				PlayDuration: &models.IntCriterionInput{
					Modifier: models.CriterionModifierEquals,
					Value:    int(getTextPlayDuration(textIdxWithGallery)),
				},
			},
			[]int{textIdxWithGallery},
			[]int{textIdxWithGroup},
			false,
		},
		// {
		// 	"specific play count",
		// 	nil,
		// 	&models.TextFilterType{
		// 		PlayCount: &models.IntCriterionInput{
		// 			Modifier: models.CriterionModifierEquals,
		// 			Value:    getTextPlayCount(textIdxWithGallery),
		// 		},
		// 	},
		// 	[]int{textIdxWithGallery},
		// 	[]int{textIdxWithGroup},
		// 	false,
		// },
		{
			"stash id with endpoint",
			nil,
			&models.TextFilterType{
				StashIDEndpoint: &models.StashIDCriterionInput{
					Endpoint: &endpoint,
					StashID:  &stashID,
					Modifier: models.CriterionModifierEquals,
				},
			},
			[]int{textIdxWithGallery},
			nil,
			false,
		},
		{
			"exclude stash id with endpoint",
			nil,
			&models.TextFilterType{
				StashIDEndpoint: &models.StashIDCriterionInput{
					Endpoint: &endpoint,
					StashID:  &stashID,
					Modifier: models.CriterionModifierNotEquals,
				},
			},
			nil,
			[]int{textIdxWithGallery},
			false,
		},
		{
			"null stash id with endpoint",
			nil,
			&models.TextFilterType{
				StashIDEndpoint: &models.StashIDCriterionInput{
					Endpoint: &endpoint,
					Modifier: models.CriterionModifierIsNull,
				},
			},
			nil,
			[]int{textIdxWithGallery},
			false,
		},
		{
			"not null stash id with endpoint",
			nil,
			&models.TextFilterType{
				StashIDEndpoint: &models.StashIDCriterionInput{
					Endpoint: &endpoint,
					Modifier: models.CriterionModifierNotNull,
				},
			},
			[]int{textIdxWithGallery},
			nil,
			false,
		},
		{
			"with studio id 0 including child studios",
			nil,
			&models.TextFilterType{
				Studios: &models.HierarchicalMultiCriterionInput{
					Value:    []string{"0"},
					Modifier: models.CriterionModifierIncludes,
					Depth:    &depth,
				},
			},
			nil,
			nil,
			false,
		},
	}

	for _, tt := range tests {
		runWithRollbackTxn(t, tt.name, func(t *testing.T, ctx context.Context) {
			assert := assert.New(t)

			results, err := db.Text.Query(ctx, models.TextQueryOptions{
				TextFilter: tt.filter,
				QueryOptions: models.QueryOptions{
					FindFilter: tt.findFilter,
				},
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("PerformerStore.Query() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			include := indexesToIDs(textIDs, tt.includeIdxs)
			exclude := indexesToIDs(textIDs, tt.excludeIdxs)

			for _, i := range include {
				assert.Contains(results.IDs, i)
			}
			for _, e := range exclude {
				assert.NotContains(results.IDs, e)
			}
		})
	}
}

func TestTextQueryPath(t *testing.T) {
	const (
		textIdx      = 1
		otherTextIdx = 2
	)
	folder := folderPaths[folderIdxWithTextFiles]
	basename := getTextBasename(textIdx)
	textPath := getFilePath(folderIdxWithTextFiles, getTextBasename(textIdx))

	tests := []struct {
		name        string
		input       models.StringCriterionInput
		mustInclude []int
		mustExclude []int
	}{
		{
			"equals full path",
			models.StringCriterionInput{
				Value:    textPath,
				Modifier: models.CriterionModifierEquals,
			},
			[]int{textIdx},
			[]int{otherTextIdx},
		},
		{
			"equals full path wildcard",
			models.StringCriterionInput{
				Value:    filepath.Join(folder, "text_0001_%"),
				Modifier: models.CriterionModifierEquals,
			},
			[]int{textIdx},
			[]int{otherTextIdx},
		},
		{
			"not equals full path",
			models.StringCriterionInput{
				Value:    textPath,
				Modifier: models.CriterionModifierNotEquals,
			},
			[]int{otherTextIdx},
			[]int{textIdx},
		},
		{
			"includes folder name",
			models.StringCriterionInput{
				Value:    folder,
				Modifier: models.CriterionModifierIncludes,
			},
			[]int{textIdx},
			nil,
		},
		{
			"includes base name",
			models.StringCriterionInput{
				Value:    basename,
				Modifier: models.CriterionModifierIncludes,
			},
			[]int{textIdx},
			nil,
		},
		{
			"includes full path",
			models.StringCriterionInput{
				Value:    textPath,
				Modifier: models.CriterionModifierIncludes,
			},
			[]int{textIdx},
			[]int{otherTextIdx},
		},
		{
			"matches regex",
			models.StringCriterionInput{
				Value:    "text_.*1_Path",
				Modifier: models.CriterionModifierMatchesRegex,
			},
			[]int{textIdx},
			nil,
		},
		{
			"not matches regex",
			models.StringCriterionInput{
				Value:    "text_.*1_Path",
				Modifier: models.CriterionModifierNotMatchesRegex,
			},
			nil,
			[]int{textIdx},
		},
	}

	qb := db.Text

	for _, tt := range tests {
		runWithRollbackTxn(t, tt.name, func(t *testing.T, ctx context.Context) {
			got, err := qb.Query(ctx, models.TextQueryOptions{
				TextFilter: &models.TextFilterType{
					Path: &tt.input,
				},
			})

			if err != nil {
				t.Errorf("textQueryBuilder.TestTextQueryPath() error = %v", err)
				return
			}

			mustInclude := indexesToIDs(textIDs, tt.mustInclude)
			mustExclude := indexesToIDs(textIDs, tt.mustExclude)

			missing := sliceutil.Exclude(mustInclude, got.IDs)
			if len(missing) > 0 {
				t.Errorf("TextStore.TestTextQueryPath() missing expected IDs: %v", missing)
			}

			notExcluded := sliceutil.Intersect(mustExclude, got.IDs)
			if len(notExcluded) > 0 {
				t.Errorf("TextStore.TestTextQueryPath() expected IDs to be excluded: %v", notExcluded)
			}
		})
	}
}

func TestTextQueryURL(t *testing.T) {
	const textIdx = 1
	textURL := getTextStringValue(textIdx, urlField)

	urlCriterion := models.StringCriterionInput{
		Value:    textURL,
		Modifier: models.CriterionModifierEquals,
	}

	filter := models.TextFilterType{
		URL: &urlCriterion,
	}

	verifyFn := func(s *models.Text) {
		t.Helper()

		urls := s.URLs.List()
		var url string
		if len(urls) > 0 {
			url = urls[0]
		}

		verifyString(t, url, urlCriterion)
	}

	verifyTextQuery(t, filter, verifyFn)

	urlCriterion.Modifier = models.CriterionModifierNotEquals
	verifyTextQuery(t, filter, verifyFn)

	urlCriterion.Modifier = models.CriterionModifierMatchesRegex
	urlCriterion.Value = "text_.*1_URL"
	verifyTextQuery(t, filter, verifyFn)

	urlCriterion.Modifier = models.CriterionModifierNotMatchesRegex
	verifyTextQuery(t, filter, verifyFn)

	urlCriterion.Modifier = models.CriterionModifierIsNull
	urlCriterion.Value = ""
	verifyTextQuery(t, filter, verifyFn)

	urlCriterion.Modifier = models.CriterionModifierNotNull
	verifyTextQuery(t, filter, verifyFn)
}

func TestTextQueryPathOr(t *testing.T) {
	const text1Idx = 1
	const text2Idx = 2

	text1Path := getFilePath(folderIdxWithTextFiles, getTextBasename(text1Idx))
	text2Path := getFilePath(folderIdxWithTextFiles, getTextBasename(text2Idx))

	textFilter := models.TextFilterType{
		Path: &models.StringCriterionInput{
			Value:    text1Path,
			Modifier: models.CriterionModifierEquals,
		},
		OperatorFilter: models.OperatorFilter[models.TextFilterType]{
			Or: &models.TextFilterType{
				Path: &models.StringCriterionInput{
					Value:    text2Path,
					Modifier: models.CriterionModifierEquals,
				},
			},
		},
	}

	withTxn(func(ctx context.Context) error {
		sqb := db.Text

		texts := queryText(ctx, t, sqb, &textFilter, nil)

		if !assert.Len(t, texts, 2) {
			return nil
		}
		assert.Equal(t, text1Path, texts[0].Path)
		assert.Equal(t, text2Path, texts[1].Path)

		return nil
	})
}

func TestTextQueryPathAndRating(t *testing.T) {
	const textIdx = 1
	textPath := getFilePath(folderIdxWithTextFiles, getTextBasename(textIdx))
	textRating := int(getRating(textIdx).Int64)

	textFilter := models.TextFilterType{
		Path: &models.StringCriterionInput{
			Value:    textPath,
			Modifier: models.CriterionModifierEquals,
		},
		OperatorFilter: models.OperatorFilter[models.TextFilterType]{
			And: &models.TextFilterType{
				Rating100: &models.IntCriterionInput{
					Value:    textRating,
					Modifier: models.CriterionModifierEquals,
				},
			},
		},
	}

	withTxn(func(ctx context.Context) error {
		sqb := db.Text

		texts := queryText(ctx, t, sqb, &textFilter, nil)

		if !assert.Len(t, texts, 1) {
			return nil
		}
		assert.Equal(t, textPath, texts[0].Path)
		assert.Equal(t, textRating, *texts[0].Rating)

		return nil
	})
}

func TestTextQueryPathNotRating(t *testing.T) {
	const textIdx = 1

	textRating := getRating(textIdx)

	pathCriterion := models.StringCriterionInput{
		Value:    "text_.*1_Path",
		Modifier: models.CriterionModifierMatchesRegex,
	}

	ratingCriterion := models.IntCriterionInput{
		Value:    int(textRating.Int64),
		Modifier: models.CriterionModifierEquals,
	}

	textFilter := models.TextFilterType{
		Path: &pathCriterion,
		OperatorFilter: models.OperatorFilter[models.TextFilterType]{
			Not: &models.TextFilterType{
				Rating100: &ratingCriterion,
			},
		},
	}

	withTxn(func(ctx context.Context) error {
		sqb := db.Text

		texts := queryText(ctx, t, sqb, &textFilter, nil)

		for _, text := range texts {
			verifyString(t, text.Path, pathCriterion)
			ratingCriterion.Modifier = models.CriterionModifierNotEquals
			verifyIntPtr(t, text.Rating, ratingCriterion)
		}

		return nil
	})
}

func TestTextIllegalQuery(t *testing.T) {
	assert := assert.New(t)

	const textIdx = 1
	subFilter := models.TextFilterType{
		Path: &models.StringCriterionInput{
			Value:    getTextStringValue(textIdx, "Path"),
			Modifier: models.CriterionModifierEquals,
		},
	}

	textFilter := &models.TextFilterType{
		OperatorFilter: models.OperatorFilter[models.TextFilterType]{
			And: &subFilter,
			Or:  &subFilter,
		},
	}

	withTxn(func(ctx context.Context) error {
		sqb := db.Text

		queryOptions := models.TextQueryOptions{
			TextFilter: textFilter,
		}

		_, err := sqb.Query(ctx, queryOptions)
		assert.NotNil(err)

		textFilter.Or = nil
		textFilter.Not = &subFilter
		_, err = sqb.Query(ctx, queryOptions)
		assert.NotNil(err)

		textFilter.And = nil
		textFilter.Or = &subFilter
		_, err = sqb.Query(ctx, queryOptions)
		assert.NotNil(err)

		return nil
	})
}

func verifyTextQuery(t *testing.T, filter models.TextFilterType, verifyFn func(s *models.Text)) {
	t.Helper()
	withTxn(func(ctx context.Context) error {
		t.Helper()
		sqb := db.Text

		texts := queryText(ctx, t, sqb, &filter, nil)

		for _, text := range texts {
			if err := text.LoadRelationships(ctx, sqb); err != nil {
				t.Errorf("Error loading text relationships: %v", err)
			}
		}

		// assume it should find at least one
		assert.Greater(t, len(texts), 0)

		for _, text := range texts {
			verifyFn(text)
		}

		return nil
	})
}

func verifyTextsPath(t *testing.T, pathCriterion models.StringCriterionInput) {
	withTxn(func(ctx context.Context) error {
		sqb := db.Text
		textFilter := models.TextFilterType{
			Path: &pathCriterion,
		}

		texts := queryText(ctx, t, sqb, &textFilter, nil)

		for _, text := range texts {
			verifyString(t, text.Path, pathCriterion)
		}

		return nil
	})
}

func verifyStringPtr(t *testing.T, value *string, criterion models.StringCriterionInput) {
	t.Helper()
	assert := assert.New(t)
	if criterion.Modifier == models.CriterionModifierIsNull {
		if value != nil && *value == "" {
			// correct
			return
		}
		assert.Nil(value, "expect is null values to be null")
	}
	if criterion.Modifier == models.CriterionModifierNotNull {
		assert.NotNil(value, "expect is null values to be null")
		assert.Greater(len(*value), 0)
	}
	if criterion.Modifier == models.CriterionModifierEquals {
		assert.Equal(criterion.Value, *value)
	}
	if criterion.Modifier == models.CriterionModifierNotEquals {
		assert.NotEqual(criterion.Value, *value)
	}
	if criterion.Modifier == models.CriterionModifierMatchesRegex {
		assert.NotNil(value)
		assert.Regexp(regexp.MustCompile(criterion.Value), *value)
	}
	if criterion.Modifier == models.CriterionModifierNotMatchesRegex {
		if value == nil {
			// correct
			return
		}
		assert.NotRegexp(regexp.MustCompile(criterion.Value), value)
	}
}

func verifyString(t *testing.T, value string, criterion models.StringCriterionInput) {
	t.Helper()
	assert := assert.New(t)
	switch criterion.Modifier {
	case models.CriterionModifierEquals:
		assert.Equal(criterion.Value, value)
	case models.CriterionModifierNotEquals:
		assert.NotEqual(criterion.Value, value)
	case models.CriterionModifierMatchesRegex:
		assert.Regexp(regexp.MustCompile(criterion.Value), value)
	case models.CriterionModifierNotMatchesRegex:
		assert.NotRegexp(regexp.MustCompile(criterion.Value), value)
	case models.CriterionModifierIsNull:
		assert.Equal("", value)
	case models.CriterionModifierNotNull:
		assert.NotEqual("", value)
	}
}

func TestTextQueryRating100(t *testing.T) {
	const rating = 60
	ratingCriterion := models.IntCriterionInput{
		Value:    rating,
		Modifier: models.CriterionModifierEquals,
	}

	verifyTextsRating100(t, ratingCriterion)

	ratingCriterion.Modifier = models.CriterionModifierNotEquals
	verifyTextsRating100(t, ratingCriterion)

	ratingCriterion.Modifier = models.CriterionModifierGreaterThan
	verifyTextsRating100(t, ratingCriterion)

	ratingCriterion.Modifier = models.CriterionModifierLessThan
	verifyTextsRating100(t, ratingCriterion)

	ratingCriterion.Modifier = models.CriterionModifierIsNull
	verifyTextsRating100(t, ratingCriterion)

	ratingCriterion.Modifier = models.CriterionModifierNotNull
	verifyTextsRating100(t, ratingCriterion)
}

func verifyTextsRating100(t *testing.T, ratingCriterion models.IntCriterionInput) {
	withTxn(func(ctx context.Context) error {
		sqb := db.Text
		textFilter := models.TextFilterType{
			Rating100: &ratingCriterion,
		}

		texts := queryText(ctx, t, sqb, &textFilter, nil)

		for _, text := range texts {
			verifyIntPtr(t, text.Rating, ratingCriterion)
		}

		return nil
	})
}

func verifyIntPtr(t *testing.T, value *int, criterion models.IntCriterionInput) {
	t.Helper()
	assert := assert.New(t)
	if criterion.Modifier == models.CriterionModifierIsNull {
		assert.Nil(value, "expect is null values to be null")
	}
	if criterion.Modifier == models.CriterionModifierNotNull {
		assert.NotNil(value, "expect is null values to be null")
	}
	if criterion.Modifier == models.CriterionModifierEquals {
		assert.Equal(criterion.Value, *value)
	}
	if criterion.Modifier == models.CriterionModifierNotEquals {
		assert.NotEqual(criterion.Value, *value)
	}
	if criterion.Modifier == models.CriterionModifierGreaterThan {
		assert.True(*value > criterion.Value)
	}
	if criterion.Modifier == models.CriterionModifierLessThan {
		assert.True(*value < criterion.Value)
	}
}

func TestTextQueryOCounter(t *testing.T) {
	const oCounter = 1
	oCounterCriterion := models.IntCriterionInput{
		Value:    oCounter,
		Modifier: models.CriterionModifierEquals,
	}

	verifyTextsOCounter(t, oCounterCriterion)

	oCounterCriterion.Modifier = models.CriterionModifierNotEquals
	verifyTextsOCounter(t, oCounterCriterion)

	oCounterCriterion.Modifier = models.CriterionModifierGreaterThan
	verifyTextsOCounter(t, oCounterCriterion)

	oCounterCriterion.Modifier = models.CriterionModifierLessThan
	verifyTextsOCounter(t, oCounterCriterion)
}

func verifyTextsOCounter(t *testing.T, oCounterCriterion models.IntCriterionInput) {
	withTxn(func(ctx context.Context) error {
		sqb := db.Text
		textFilter := models.TextFilterType{
			OCounter: &oCounterCriterion,
		}

		texts := queryText(ctx, t, sqb, &textFilter, nil)

		for _, text := range texts {
			count, err := sqb.GetOCount(ctx, text.ID)
			if err != nil {
				t.Errorf("Error getting ocounter: %v", err)
			}
			verifyInt(t, count, oCounterCriterion)
		}

		return nil
	})
}

func verifyInt(t *testing.T, value int, criterion models.IntCriterionInput) bool {
	t.Helper()
	assert := assert.New(t)
	if criterion.Modifier == models.CriterionModifierEquals {
		return assert.Equal(criterion.Value, value)
	}
	if criterion.Modifier == models.CriterionModifierNotEquals {
		return assert.NotEqual(criterion.Value, value)
	}
	if criterion.Modifier == models.CriterionModifierGreaterThan {
		return assert.Greater(value, criterion.Value)
	}
	if criterion.Modifier == models.CriterionModifierLessThan {
		return assert.Less(value, criterion.Value)
	}

	return true
}

func TestTextQueryDuration(t *testing.T) {
	duration := 200.432

	durationCriterion := models.IntCriterionInput{
		Value:    int(duration),
		Modifier: models.CriterionModifierEquals,
	}
	verifyTextsDuration(t, durationCriterion)

	durationCriterion.Modifier = models.CriterionModifierNotEquals
	verifyTextsDuration(t, durationCriterion)

	durationCriterion.Modifier = models.CriterionModifierGreaterThan
	verifyTextsDuration(t, durationCriterion)

	durationCriterion.Modifier = models.CriterionModifierLessThan
	verifyTextsDuration(t, durationCriterion)

	durationCriterion.Modifier = models.CriterionModifierIsNull
	verifyTextsDuration(t, durationCriterion)

	durationCriterion.Modifier = models.CriterionModifierNotNull
	verifyTextsDuration(t, durationCriterion)
}

func verifyTextsDuration(t *testing.T, durationCriterion models.IntCriterionInput) {
	withTxn(func(ctx context.Context) error {
		sqb := db.Text
		textFilter := models.TextFilterType{
			Duration: &durationCriterion,
		}

		texts := queryText(ctx, t, sqb, &textFilter, nil)

		for _, text := range texts {
			if err := text.LoadPrimaryFile(ctx, db.File); err != nil {
				t.Errorf("Error querying text files: %v", err)
				return nil
			}

			duration := text.Files.Primary().Duration
			if durationCriterion.Modifier == models.CriterionModifierEquals {
				assert.True(t, duration >= float64(durationCriterion.Value) && duration < float64(durationCriterion.Value+1))
			} else if durationCriterion.Modifier == models.CriterionModifierNotEquals {
				assert.True(t, duration < float64(durationCriterion.Value) || duration >= float64(durationCriterion.Value+1))
			} else {
				verifyFloat64(t, duration, durationCriterion)
			}
		}

		return nil
	})
}

func verifyFloat64(t *testing.T, value float64, criterion models.IntCriterionInput) {
	assert := assert.New(t)
	if criterion.Modifier == models.CriterionModifierEquals {
		assert.Equal(float64(criterion.Value), value)
	}
	if criterion.Modifier == models.CriterionModifierNotEquals {
		assert.NotEqual(float64(criterion.Value), value)
	}
	if criterion.Modifier == models.CriterionModifierGreaterThan {
		assert.True(value > float64(criterion.Value))
	}
	if criterion.Modifier == models.CriterionModifierLessThan {
		assert.True(value < float64(criterion.Value))
	}
}

func verifyFloat64Ptr(t *testing.T, value *float64, criterion models.IntCriterionInput) {
	assert := assert.New(t)
	switch criterion.Modifier {
	case models.CriterionModifierIsNull:
		assert.Nil(value, "expect is null values to be null")
	case models.CriterionModifierNotNull:
		assert.NotNil(value, "expect is not null values to not be null")
	case models.CriterionModifierEquals:
		assert.EqualValues(float64(criterion.Value), value)
	case models.CriterionModifierNotEquals:
		assert.NotEqualValues(float64(criterion.Value), value)
	case models.CriterionModifierGreaterThan:
		assert.True(value != nil && *value > float64(criterion.Value))
	case models.CriterionModifierLessThan:
		assert.True(value != nil && *value < float64(criterion.Value))
	}
}

func TestTextQueryResolution(t *testing.T) {
	verifyTextsResolution(t, models.ResolutionEnumLow)
	verifyTextsResolution(t, models.ResolutionEnumStandard)
	verifyTextsResolution(t, models.ResolutionEnumStandardHd)
	verifyTextsResolution(t, models.ResolutionEnumFullHd)
	verifyTextsResolution(t, models.ResolutionEnumFourK)
	verifyTextsResolution(t, models.ResolutionEnum("unknown"))
}

func verifyTextsResolution(t *testing.T, resolution models.ResolutionEnum) {
	withTxn(func(ctx context.Context) error {
		sqb := db.Text
		textFilter := models.TextFilterType{
			Resolution: &models.ResolutionCriterionInput{
				Value:    resolution,
				Modifier: models.CriterionModifierEquals,
			},
		}

		texts := queryText(ctx, t, sqb, &textFilter, nil)

		for _, text := range texts {
			if err := text.LoadPrimaryFile(ctx, db.File); err != nil {
				t.Errorf("Error querying text files: %v", err)
				return nil
			}
			f := text.Files.Primary()
			height := 0
			if f != nil {
				height = f.Height
			}
			verifyTextResolution(t, &height, resolution)
		}

		return nil
	})
}

func verifyTextResolution(t *testing.T, height *int, resolution models.ResolutionEnum) {
	if !resolution.IsValid() {
		return
	}

	assert := assert.New(t)
	assert.NotNil(height)
	if t.Failed() {
		return
	}

	h := *height

	switch resolution {
	case models.ResolutionEnumLow:
		assert.True(h < 480)
	case models.ResolutionEnumStandard:
		assert.True(h >= 480 && h < 720)
	case models.ResolutionEnumStandardHd:
		assert.True(h >= 720 && h < 1080)
	case models.ResolutionEnumFullHd:
		assert.True(h >= 1080 && h < 2160)
	case models.ResolutionEnumFourK:
		assert.True(h >= 2160)
	}
}

func TestAllResolutionsHaveResolutionRange(t *testing.T) {
	for _, resolution := range models.AllResolutionEnum {
		assert.NotZero(t, resolution.GetMinResolution(), "Define resolution range for %s in extension_resolution.go", resolution)
		assert.NotZero(t, resolution.GetMaxResolution(), "Define resolution range for %s in extension_resolution.go", resolution)
	}
}

func TestTextQueryResolutionModifiers(t *testing.T) {
	if err := withRollbackTxn(func(ctx context.Context) error {
		qb := db.Text
		textNoResolution, _ := createText(ctx, 0, 0)
		firstText540P, _ := createText(ctx, 960, 540)
		secondText540P, _ := createText(ctx, 1280, 719)
		firstText720P, _ := createText(ctx, 1280, 720)
		secondText720P, _ := createText(ctx, 1280, 721)
		thirdText720P, _ := createText(ctx, 1920, 1079)
		text1080P, _ := createText(ctx, 1920, 1080)

		textsEqualTo720P := queryTexts(ctx, t, qb, models.ResolutionEnumStandardHd, models.CriterionModifierEquals)
		textsNotEqualTo720P := queryTexts(ctx, t, qb, models.ResolutionEnumStandardHd, models.CriterionModifierNotEquals)
		textsGreaterThan720P := queryTexts(ctx, t, qb, models.ResolutionEnumStandardHd, models.CriterionModifierGreaterThan)
		textsLessThan720P := queryTexts(ctx, t, qb, models.ResolutionEnumStandardHd, models.CriterionModifierLessThan)

		assert.Subset(t, textsEqualTo720P, []*models.Text{firstText720P, secondText720P, thirdText720P})
		assert.NotSubset(t, textsEqualTo720P, []*models.Text{textNoResolution, firstText540P, secondText540P, text1080P})

		assert.Subset(t, textsNotEqualTo720P, []*models.Text{textNoResolution, firstText540P, secondText540P, text1080P})
		assert.NotSubset(t, textsNotEqualTo720P, []*models.Text{firstText720P, secondText720P, thirdText720P})

		assert.Subset(t, textsGreaterThan720P, []*models.Text{text1080P})
		assert.NotSubset(t, textsGreaterThan720P, []*models.Text{textNoResolution, firstText540P, secondText540P, firstText720P, secondText720P, thirdText720P})

		assert.Subset(t, textsLessThan720P, []*models.Text{textNoResolution, firstText540P, secondText540P})
		assert.NotSubset(t, textsLessThan720P, []*models.Text{text1080P, firstText720P, secondText720P, thirdText720P})

		return nil
	}); err != nil {
		t.Error(err.Error())
	}
}

func queryTexts(ctx context.Context, t *testing.T, queryBuilder models.TextReaderWriter, resolution models.ResolutionEnum, modifier models.CriterionModifier) []*models.Text {
	textFilter := models.TextFilterType{
		Resolution: &models.ResolutionCriterionInput{
			Value:    resolution,
			Modifier: modifier,
		},
	}

	// needed so that we don't hit the default limit of 25 texts
	pp := 1000
	findFilter := &models.FindFilterType{
		PerPage: &pp,
	}

	return queryText(ctx, t, queryBuilder, &textFilter, findFilter)
}

func createText(ctx context.Context, width int, height int) (*models.Text, error) {
	name := fmt.Sprintf("TestTextQueryResolutionModifiers %d %d", width, height)

	textFile := &models.VideoFile{
		BaseFile: &models.BaseFile{
			Basename:       name,
			ParentFolderID: folderIDs[folderIdxWithTextFiles],
		},
		Width:  width,
		Height: height,
	}

	if err := db.File.Create(ctx, textFile); err != nil {
		return nil, err
	}

	text := &models.Text{}

	if err := db.Text.Create(ctx, text, []models.FileID{textFile.ID}); err != nil {
		return nil, err
	}

	return text, nil
}

func TestTextQueryHasBookmarks(t *testing.T) {
	withTxn(func(ctx context.Context) error {
		sqb := db.Text
		hasBookmarks := "true"
		textFilter := models.TextFilterType{
			HasBookmarks: &hasBookmarks,
		}

		q := getTextStringValue(textIdxWithBookmarks, titleField)
		findFilter := models.FindFilterType{
			Q: &q,
		}

		texts := queryText(ctx, t, sqb, &textFilter, &findFilter)

		assert.Len(t, texts, 1)
		assert.Equal(t, textIDs[textIdxWithBookmarks], texts[0].ID)

		hasBookmarks = "false"
		texts = queryText(ctx, t, sqb, &textFilter, &findFilter)
		assert.Len(t, texts, 0)

		findFilter.Q = nil
		texts = queryText(ctx, t, sqb, &textFilter, &findFilter)

		assert.NotEqual(t, 0, len(texts))

		// ensure non of the ids equal the one with gallery
		for _, text := range texts {
			assert.NotEqual(t, textIDs[textIdxWithBookmarks], text.ID)
		}

		return nil
	})
}

func TestTextQueryIsMissingGallery(t *testing.T) {
	withTxn(func(ctx context.Context) error {
		sqb := db.Text
		isMissing := "galleries"
		textFilter := models.TextFilterType{
			IsMissing: &isMissing,
		}

		q := getTextStringValue(textIdxWithGallery, titleField)
		findFilter := models.FindFilterType{
			Q: &q,
		}

		texts := queryText(ctx, t, sqb, &textFilter, &findFilter)

		assert.Len(t, texts, 0)

		findFilter.Q = nil
		texts = queryText(ctx, t, sqb, &textFilter, &findFilter)

		// ensure non of the ids equal the one with gallery
		for _, text := range texts {
			assert.NotEqual(t, textIDs[textIdxWithGallery], text.ID)
		}

		return nil
	})
}

func TestTextQueryIsMissingStudio(t *testing.T) {
	withTxn(func(ctx context.Context) error {
		sqb := db.Text
		isMissing := "studio"
		textFilter := models.TextFilterType{
			IsMissing: &isMissing,
		}

		q := getTextStringValue(textIdxWithStudio, titleField)
		findFilter := models.FindFilterType{
			Q: &q,
		}

		texts := queryText(ctx, t, sqb, &textFilter, &findFilter)

		assert.Len(t, texts, 0)

		findFilter.Q = nil
		texts = queryText(ctx, t, sqb, &textFilter, &findFilter)

		// ensure non of the ids equal the one with studio
		for _, text := range texts {
			assert.NotEqual(t, textIDs[textIdxWithStudio], text.ID)
		}

		return nil
	})
}

func TestTextQueryIsMissingMovies(t *testing.T) {
	withTxn(func(ctx context.Context) error {
		sqb := db.Text
		isMissing := "movie"
		textFilter := models.TextFilterType{
			IsMissing: &isMissing,
		}

		q := getTextStringValue(textIdxWithGroup, titleField)
		findFilter := models.FindFilterType{
			Q: &q,
		}

		texts := queryText(ctx, t, sqb, &textFilter, &findFilter)

		assert.Len(t, texts, 0)

		findFilter.Q = nil
		texts = queryText(ctx, t, sqb, &textFilter, &findFilter)

		// ensure non of the ids equal the one with movies
		for _, text := range texts {
			assert.NotEqual(t, textIDs[textIdxWithGroup], text.ID)
		}

		return nil
	})
}

func TestTextQueryIsMissingPerformers(t *testing.T) {
	withTxn(func(ctx context.Context) error {
		sqb := db.Text
		isMissing := "performers"
		textFilter := models.TextFilterType{
			IsMissing: &isMissing,
		}

		q := getTextStringValue(textIdxWithPerformer, titleField)
		findFilter := models.FindFilterType{
			Q: &q,
		}

		texts := queryText(ctx, t, sqb, &textFilter, &findFilter)

		assert.Len(t, texts, 0)

		findFilter.Q = nil
		texts = queryText(ctx, t, sqb, &textFilter, &findFilter)

		assert.True(t, len(texts) > 0)

		// ensure non of the ids equal the one with movies
		for _, text := range texts {
			assert.NotEqual(t, textIDs[textIdxWithPerformer], text.ID)
		}

		return nil
	})
}

func TestTextQueryIsMissingDate(t *testing.T) {
	withTxn(func(ctx context.Context) error {
		sqb := db.Text
		isMissing := "date"
		textFilter := models.TextFilterType{
			IsMissing: &isMissing,
		}

		texts := queryText(ctx, t, sqb, &textFilter, nil)

		// one in four texts have no date
		assert.Len(t, texts, int(math.Ceil(float64(totalTexts)/4)))

		// ensure date is null
		for _, text := range texts {
			assert.Nil(t, text.Date)
		}

		return nil
	})
}

func TestTextQueryIsMissingTags(t *testing.T) {
	withTxn(func(ctx context.Context) error {
		sqb := db.Text
		isMissing := "tags"
		textFilter := models.TextFilterType{
			IsMissing: &isMissing,
		}

		q := getTextStringValue(textIdxWithTwoTags, titleField)
		findFilter := models.FindFilterType{
			Q: &q,
		}

		texts := queryText(ctx, t, sqb, &textFilter, &findFilter)

		assert.Len(t, texts, 0)

		findFilter.Q = nil
		texts = queryText(ctx, t, sqb, &textFilter, &findFilter)

		assert.True(t, len(texts) > 0)

		return nil
	})
}

func TestTextQueryIsMissingRating(t *testing.T) {
	withTxn(func(ctx context.Context) error {
		sqb := db.Text
		isMissing := "rating"
		textFilter := models.TextFilterType{
			IsMissing: &isMissing,
		}

		texts := queryText(ctx, t, sqb, &textFilter, nil)

		assert.True(t, len(texts) > 0)

		// ensure rating is null
		for _, text := range texts {
			assert.Nil(t, text.Rating)
		}

		return nil
	})
}

func TestTextQueryIsMissingPhash(t *testing.T) {
	withTxn(func(ctx context.Context) error {
		sqb := db.Text
		isMissing := "phash"
		textFilter := models.TextFilterType{
			IsMissing: &isMissing,
		}

		texts := queryText(ctx, t, sqb, &textFilter, nil)

		if !assert.Len(t, texts, 1) {
			return nil
		}

		assert.Equal(t, textIDs[textIdxMissingPhash], texts[0].ID)

		return nil
	})
}

func TestTextQueryPerformers(t *testing.T) {
	tests := []struct {
		name        string
		filter      models.MultiCriterionInput
		includeIdxs []int
		excludeIdxs []int
		wantErr     bool
	}{
		{
			"includes",
			models.MultiCriterionInput{
				Value: []string{
					strconv.Itoa(performerIDs[performerIdxWithText]),
					strconv.Itoa(performerIDs[performerIdx1WithText]),
				},
				Modifier: models.CriterionModifierIncludes,
			},
			[]int{
				textIdxWithPerformer,
				textIdxWithTwoPerformers,
			},
			[]int{
				textIdxWithGallery,
			},
			false,
		},
		{
			"includes all",
			models.MultiCriterionInput{
				Value: []string{
					strconv.Itoa(performerIDs[performerIdx1WithText]),
					strconv.Itoa(performerIDs[performerIdx2WithText]),
				},
				Modifier: models.CriterionModifierIncludesAll,
			},
			[]int{
				textIdxWithTwoPerformers,
			},
			[]int{
				textIdxWithPerformer,
			},
			false,
		},
		{
			"excludes",
			models.MultiCriterionInput{
				Modifier: models.CriterionModifierExcludes,
				Value:    []string{strconv.Itoa(tagIDs[performerIdx1WithText])},
			},
			nil,
			[]int{textIdxWithTwoPerformers},
			false,
		},
		{
			"is null",
			models.MultiCriterionInput{
				Modifier: models.CriterionModifierIsNull,
			},
			[]int{textIdxWithTag},
			[]int{
				textIdxWithPerformer,
				textIdxWithTwoPerformers,
				textIdxWithPerformerTwoTags,
			},
			false,
		},
		{
			"not null",
			models.MultiCriterionInput{
				Modifier: models.CriterionModifierNotNull,
			},
			[]int{
				textIdxWithPerformer,
				textIdxWithTwoPerformers,
				textIdxWithPerformerTwoTags,
			},
			[]int{textIdxWithTag},
			false,
		},
		{
			"equals",
			models.MultiCriterionInput{
				Modifier: models.CriterionModifierEquals,
				Value: []string{
					strconv.Itoa(tagIDs[performerIdx1WithText]),
					strconv.Itoa(tagIDs[performerIdx2WithText]),
				},
			},
			[]int{textIdxWithTwoPerformers},
			[]int{
				textIdxWithThreePerformers,
			},
			false,
		},
		{
			"not equals",
			models.MultiCriterionInput{
				Modifier: models.CriterionModifierNotEquals,
				Value: []string{
					strconv.Itoa(tagIDs[performerIdx1WithText]),
					strconv.Itoa(tagIDs[performerIdx2WithText]),
				},
			},
			nil,
			nil,
			true,
		},
	}

	for _, tt := range tests {
		runWithRollbackTxn(t, tt.name, func(t *testing.T, ctx context.Context) {
			assert := assert.New(t)

			results, err := db.Text.Query(ctx, models.TextQueryOptions{
				TextFilter: &models.TextFilterType{
					Performers: &tt.filter,
				},
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("TextStore.Query() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			include := indexesToIDs(textIDs, tt.includeIdxs)
			exclude := indexesToIDs(textIDs, tt.excludeIdxs)

			for _, i := range include {
				assert.Contains(results.IDs, i)
			}
			for _, e := range exclude {
				assert.NotContains(results.IDs, e)
			}
		})
	}
}

func TestTextQueryTags(t *testing.T) {
	tests := []struct {
		name        string
		filter      models.HierarchicalMultiCriterionInput
		includeIdxs []int
		excludeIdxs []int
		wantErr     bool
	}{
		{
			"includes",
			models.HierarchicalMultiCriterionInput{
				Value: []string{
					strconv.Itoa(tagIDs[tagIdxWithText]),
					strconv.Itoa(tagIDs[tagIdx1WithText]),
				},
				Modifier: models.CriterionModifierIncludes,
			},
			[]int{
				textIdxWithTag,
				textIdxWithTwoTags,
			},
			[]int{
				textIdxWithGallery,
			},
			false,
		},
		{
			"includes all",
			models.HierarchicalMultiCriterionInput{
				Value: []string{
					strconv.Itoa(tagIDs[tagIdx1WithText]),
					strconv.Itoa(tagIDs[tagIdx2WithText]),
				},
				Modifier: models.CriterionModifierIncludesAll,
			},
			[]int{
				textIdxWithTwoTags,
			},
			[]int{
				textIdxWithTag,
			},
			false,
		},
		{
			"excludes",
			models.HierarchicalMultiCriterionInput{
				Modifier: models.CriterionModifierExcludes,
				Value:    []string{strconv.Itoa(tagIDs[tagIdx1WithText])},
			},
			nil,
			[]int{textIdxWithTwoTags},
			false,
		},
		{
			"is null",
			models.HierarchicalMultiCriterionInput{
				Modifier: models.CriterionModifierIsNull,
			},
			[]int{textIdx1WithPerformer},
			[]int{
				textIdxWithTag,
				textIdxWithTwoTags,
				textIdxWithBookmarkAndTag,
			},
			false,
		},
		{
			"not null",
			models.HierarchicalMultiCriterionInput{
				Modifier: models.CriterionModifierNotNull,
			},
			[]int{
				textIdxWithTag,
				textIdxWithTwoTags,
				textIdxWithBookmarkAndTag,
			},
			[]int{textIdx1WithPerformer},
			false,
		},
		{
			"equals",
			models.HierarchicalMultiCriterionInput{
				Modifier: models.CriterionModifierEquals,
				Value: []string{
					strconv.Itoa(tagIDs[tagIdx1WithText]),
					strconv.Itoa(tagIDs[tagIdx2WithText]),
				},
			},
			[]int{textIdxWithTwoTags},
			[]int{
				textIdxWithThreeTags,
			},
			false,
		},
		{
			"not equals",
			models.HierarchicalMultiCriterionInput{
				Modifier: models.CriterionModifierNotEquals,
				Value: []string{
					strconv.Itoa(tagIDs[tagIdx1WithText]),
					strconv.Itoa(tagIDs[tagIdx2WithText]),
				},
			},
			nil,
			nil,
			true,
		},
	}

	for _, tt := range tests {
		runWithRollbackTxn(t, tt.name, func(t *testing.T, ctx context.Context) {
			assert := assert.New(t)

			results, err := db.Text.Query(ctx, models.TextQueryOptions{
				TextFilter: &models.TextFilterType{
					Tags: &tt.filter,
				},
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("TextStore.Query() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			include := indexesToIDs(textIDs, tt.includeIdxs)
			exclude := indexesToIDs(textIDs, tt.excludeIdxs)

			for _, i := range include {
				assert.Contains(results.IDs, i)
			}
			for _, e := range exclude {
				assert.NotContains(results.IDs, e)
			}
		})
	}
}

func TestTextQueryPerformerTags(t *testing.T) {
	allDepth := -1

	tests := []struct {
		name        string
		findFilter  *models.FindFilterType
		filter      *models.TextFilterType
		includeIdxs []int
		excludeIdxs []int
		wantErr     bool
	}{
		{
			"includes",
			nil,
			&models.TextFilterType{
				PerformerTags: &models.HierarchicalMultiCriterionInput{
					Value: []string{
						strconv.Itoa(tagIDs[tagIdxWithPerformer]),
						strconv.Itoa(tagIDs[tagIdx1WithPerformer]),
					},
					Modifier: models.CriterionModifierIncludes,
				},
			},
			[]int{
				textIdxWithPerformerTag,
				textIdxWithPerformerTwoTags,
				textIdxWithTwoPerformerTag,
			},
			[]int{
				textIdxWithPerformer,
			},
			false,
		},
		{
			"includes sub-tags",
			nil,
			&models.TextFilterType{
				PerformerTags: &models.HierarchicalMultiCriterionInput{
					Value: []string{
						strconv.Itoa(tagIDs[tagIdxWithParentAndChild]),
					},
					Depth:    &allDepth,
					Modifier: models.CriterionModifierIncludes,
				},
			},
			[]int{
				textIdxWithPerformerParentTag,
			},
			[]int{
				textIdxWithPerformer,
				textIdxWithPerformerTag,
				textIdxWithPerformerTwoTags,
				textIdxWithTwoPerformerTag,
			},
			false,
		},
		{
			"includes all",
			nil,
			&models.TextFilterType{
				PerformerTags: &models.HierarchicalMultiCriterionInput{
					Value: []string{
						strconv.Itoa(tagIDs[tagIdx1WithPerformer]),
						strconv.Itoa(tagIDs[tagIdx2WithPerformer]),
					},
					Modifier: models.CriterionModifierIncludesAll,
				},
			},
			[]int{
				textIdxWithPerformerTwoTags,
			},
			[]int{
				textIdxWithPerformer,
				textIdxWithPerformerTag,
				textIdxWithTwoPerformerTag,
			},
			false,
		},
		{
			"excludes performer tag tagIdx2WithPerformer",
			nil,
			&models.TextFilterType{
				PerformerTags: &models.HierarchicalMultiCriterionInput{
					Modifier: models.CriterionModifierExcludes,
					Value:    []string{strconv.Itoa(tagIDs[tagIdx2WithPerformer])},
				},
			},
			nil,
			[]int{textIdxWithTwoPerformerTag},
			false,
		},
		{
			"excludes sub-tags",
			nil,
			&models.TextFilterType{
				PerformerTags: &models.HierarchicalMultiCriterionInput{
					Value: []string{
						strconv.Itoa(tagIDs[tagIdxWithParentAndChild]),
					},
					Depth:    &allDepth,
					Modifier: models.CriterionModifierExcludes,
				},
			},
			[]int{
				textIdxWithPerformer,
				textIdxWithPerformerTag,
				textIdxWithPerformerTwoTags,
				textIdxWithTwoPerformerTag,
			},
			[]int{
				textIdxWithPerformerParentTag,
			},
			false,
		},
		{
			"is null",
			nil,
			&models.TextFilterType{
				PerformerTags: &models.HierarchicalMultiCriterionInput{
					Modifier: models.CriterionModifierIsNull,
				},
			},
			[]int{textIdx1WithPerformer},
			[]int{textIdxWithPerformerTag},
			false,
		},
		{
			"not null",
			nil,
			&models.TextFilterType{
				PerformerTags: &models.HierarchicalMultiCriterionInput{
					Modifier: models.CriterionModifierNotNull,
				},
			},
			[]int{textIdxWithPerformerTag},
			[]int{textIdx1WithPerformer},
			false,
		},
		{
			"equals",
			nil,
			&models.TextFilterType{
				PerformerTags: &models.HierarchicalMultiCriterionInput{
					Modifier: models.CriterionModifierEquals,
					Value: []string{
						strconv.Itoa(tagIDs[tagIdx2WithPerformer]),
					},
				},
			},
			nil,
			nil,
			true,
		},
		{
			"not equals",
			nil,
			&models.TextFilterType{
				PerformerTags: &models.HierarchicalMultiCriterionInput{
					Modifier: models.CriterionModifierNotEquals,
					Value: []string{
						strconv.Itoa(tagIDs[tagIdx2WithPerformer]),
					},
				},
			},
			nil,
			nil,
			true,
		},
	}

	for _, tt := range tests {
		runWithRollbackTxn(t, tt.name, func(t *testing.T, ctx context.Context) {
			assert := assert.New(t)

			results, err := db.Text.Query(ctx, models.TextQueryOptions{
				TextFilter: tt.filter,
				QueryOptions: models.QueryOptions{
					FindFilter: tt.findFilter,
				},
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("TextStore.Query() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			include := indexesToIDs(textIDs, tt.includeIdxs)
			exclude := indexesToIDs(textIDs, tt.excludeIdxs)

			for _, i := range include {
				assert.Contains(results.IDs, i)
			}
			for _, e := range exclude {
				assert.NotContains(results.IDs, e)
			}
		})
	}
}

func TestTextQueryStudio(t *testing.T) {
	tests := []struct {
		name            string
		q               string
		studioCriterion models.HierarchicalMultiCriterionInput
		expectedIDs     []int
		wantErr         bool
	}{
		{
			"includes",
			"",
			models.HierarchicalMultiCriterionInput{
				Value: []string{
					strconv.Itoa(studioIDs[studioIdxWithText]),
				},
				Modifier: models.CriterionModifierIncludes,
			},
			[]int{textIDs[textIdxWithStudio]},
			false,
		},
		{
			"excludes",
			getTextStringValue(textIdxWithStudio, titleField),
			models.HierarchicalMultiCriterionInput{
				Value: []string{
					strconv.Itoa(studioIDs[studioIdxWithText]),
				},
				Modifier: models.CriterionModifierExcludes,
			},
			[]int{},
			false,
		},
		{
			"excludes includes null",
			getTextStringValue(textIdxWithGallery, titleField),
			models.HierarchicalMultiCriterionInput{
				Value: []string{
					strconv.Itoa(studioIDs[studioIdxWithText]),
				},
				Modifier: models.CriterionModifierExcludes,
			},
			[]int{textIDs[textIdxWithGallery]},
			false,
		},
		{
			"equals",
			"",
			models.HierarchicalMultiCriterionInput{
				Value: []string{
					strconv.Itoa(studioIDs[studioIdxWithText]),
				},
				Modifier: models.CriterionModifierEquals,
			},
			[]int{textIDs[textIdxWithStudio]},
			false,
		},
		{
			"not equals",
			getTextStringValue(textIdxWithStudio, titleField),
			models.HierarchicalMultiCriterionInput{
				Value: []string{
					strconv.Itoa(studioIDs[studioIdxWithText]),
				},
				Modifier: models.CriterionModifierNotEquals,
			},
			[]int{},
			false,
		},
	}

	qb := db.Text

	for _, tt := range tests {
		runWithRollbackTxn(t, tt.name, func(t *testing.T, ctx context.Context) {
			studioCriterion := tt.studioCriterion

			textFilter := models.TextFilterType{
				Studios: &studioCriterion,
			}

			var findFilter *models.FindFilterType
			if tt.q != "" {
				findFilter = &models.FindFilterType{
					Q: &tt.q,
				}
			}

			texts := queryText(ctx, t, qb, &textFilter, findFilter)

			assert.ElementsMatch(t, textsToIDs(texts), tt.expectedIDs)
		})
	}
}

func TestTextQueryStudioDepth(t *testing.T) {
	withTxn(func(ctx context.Context) error {
		sqb := db.Text
		depth := 2
		studioCriterion := models.HierarchicalMultiCriterionInput{
			Value: []string{
				strconv.Itoa(studioIDs[studioIdxWithGrandChild]),
			},
			Modifier: models.CriterionModifierIncludes,
			Depth:    &depth,
		}

		textFilter := models.TextFilterType{
			Studios: &studioCriterion,
		}

		texts := queryText(ctx, t, sqb, &textFilter, nil)
		assert.Len(t, texts, 1)

		depth = 1

		texts = queryText(ctx, t, sqb, &textFilter, nil)
		assert.Len(t, texts, 0)

		studioCriterion.Value = []string{strconv.Itoa(studioIDs[studioIdxWithParentAndChild])}
		texts = queryText(ctx, t, sqb, &textFilter, nil)
		assert.Len(t, texts, 1)

		// ensure id is correct
		assert.Equal(t, textIDs[textIdxWithGrandChildStudio], texts[0].ID)
		depth = 2

		studioCriterion = models.HierarchicalMultiCriterionInput{
			Value: []string{
				strconv.Itoa(studioIDs[studioIdxWithGrandChild]),
			},
			Modifier: models.CriterionModifierExcludes,
			Depth:    &depth,
		}

		q := getTextStringValue(textIdxWithGrandChildStudio, titleField)
		findFilter := models.FindFilterType{
			Q: &q,
		}

		texts = queryText(ctx, t, sqb, &textFilter, &findFilter)
		assert.Len(t, texts, 0)

		depth = 1
		texts = queryText(ctx, t, sqb, &textFilter, &findFilter)
		assert.Len(t, texts, 1)

		studioCriterion.Value = []string{strconv.Itoa(studioIDs[studioIdxWithParentAndChild])}
		texts = queryText(ctx, t, sqb, &textFilter, &findFilter)
		assert.Len(t, texts, 0)

		return nil
	})
}

func TestTextQueryMovies(t *testing.T) {
	withTxn(func(ctx context.Context) error {
		sqb := db.Text
		movieCriterion := models.MultiCriterionInput{
			Value: []string{
				strconv.Itoa(groupIDs[groupIdxWithText]),
			},
			Modifier: models.CriterionModifierIncludes,
		}

		textFilter := models.TextFilterType{
			Movies: &movieCriterion,
		}

		texts := queryText(ctx, t, sqb, &textFilter, nil)

		assert.Len(t, texts, 1)

		// ensure id is correct
		assert.Equal(t, textIDs[textIdxWithGroup], texts[0].ID)

		movieCriterion = models.MultiCriterionInput{
			Value: []string{
				strconv.Itoa(groupIDs[groupIdxWithText]),
			},
			Modifier: models.CriterionModifierExcludes,
		}

		q := getTextStringValue(textIdxWithGroup, titleField)
		findFilter := models.FindFilterType{
			Q: &q,
		}

		texts = queryText(ctx, t, sqb, &textFilter, &findFilter)
		assert.Len(t, texts, 0)

		return nil
	})
}

func TestTextQueryPhashDuplicated(t *testing.T) {
	withTxn(func(ctx context.Context) error {
		sqb := db.Text
		duplicated := true
		phashCriterion := models.PHashDuplicationCriterionInput{
			Duplicated: &duplicated,
		}

		textFilter := models.TextFilterType{
			Duplicated: &phashCriterion,
		}

		texts := queryText(ctx, t, sqb, &textFilter, nil)

		assert.Len(t, texts, dupeTextPhashes*2)

		duplicated = false

		texts = queryText(ctx, t, sqb, &textFilter, nil)
		// -1 for missing phash
		assert.Len(t, texts, totalTexts-(dupeTextPhashes*2)-1)

		return nil
	})
}

func TestTextQuerySorting(t *testing.T) {
	tests := []struct {
		name          string
		sortBy        string
		dir           models.SortDirectionEnum
		firstTextIdx int // -1 to ignore
		lastTextIdx  int
	}{
		{
			"bitrate",
			"bitrate",
			models.SortDirectionEnumAsc,
			-1,
			-1,
		},
		{
			"duration",
			"duration",
			models.SortDirectionEnumDesc,
			-1,
			-1,
		},
		{
			"file mod time",
			"file_mod_time",
			models.SortDirectionEnumDesc,
			-1,
			-1,
		},
		{
			"file size",
			"filesize",
			models.SortDirectionEnumDesc,
			-1,
			-1,
		},
		{
			"frame rate",
			"framerate",
			models.SortDirectionEnumDesc,
			-1,
			-1,
		},
		{
			"path",
			"path",
			models.SortDirectionEnumDesc,
			-1,
			-1,
		},
		{
			"perceptual_similarity",
			"perceptual_similarity",
			models.SortDirectionEnumDesc,
			-1,
			-1,
		},
		{
			"play_count",
			"play_count",
			models.SortDirectionEnumDesc,
			-1,
			-1,
		},
		{
			"last_played_at",
			"last_played_at",
			models.SortDirectionEnumDesc,
			-1,
			-1,
		},
		{
			"resume_time",
			"resume_time",
			models.SortDirectionEnumDesc,
			textIDs[textIdx1WithPerformer],
			-1,
		},
		{
			"play_duration",
			"play_duration",
			models.SortDirectionEnumDesc,
			textIDs[textIdx1WithPerformer],
			-1,
		},
	}

	qb := db.Text

	for _, tt := range tests {
		runWithRollbackTxn(t, tt.name, func(t *testing.T, ctx context.Context) {
			assert := assert.New(t)
			got, err := qb.Query(ctx, models.TextQueryOptions{
				QueryOptions: models.QueryOptions{
					FindFilter: &models.FindFilterType{
						Sort:      &tt.sortBy,
						Direction: &tt.dir,
					},
				},
			})

			if err != nil {
				t.Errorf("textQueryBuilder.TestTextQuerySorting() error = %v", err)
				return
			}

			texts, err := got.Resolve(ctx)
			if err != nil {
				t.Errorf("textQueryBuilder.TestTextQuerySorting() error = %v", err)
				return
			}

			if !assert.Greater(len(texts), 0) {
				return
			}

			// texts should be in same order as indexes
			firstText := texts[0]
			lastText := texts[len(texts)-1]

			if tt.firstTextIdx != -1 {
				firstTextID := textIDs[tt.firstTextIdx]
				assert.Equal(firstTextID, firstText.ID)
			}
			if tt.lastTextIdx != -1 {
				lastTextID := textIDs[tt.lastTextIdx]
				assert.Equal(lastTextID, lastText.ID)
			}
		})
	}
}

func TestTextQueryPagination(t *testing.T) {
	perPage := 1
	findFilter := models.FindFilterType{
		PerPage: &perPage,
	}

	withTxn(func(ctx context.Context) error {
		sqb := db.Text
		texts := queryText(ctx, t, sqb, nil, &findFilter)

		assert.Len(t, texts, 1)

		firstID := texts[0].ID

		page := 2
		findFilter.Page = &page
		texts = queryText(ctx, t, sqb, nil, &findFilter)

		assert.Len(t, texts, 1)
		secondID := texts[0].ID
		assert.NotEqual(t, firstID, secondID)

		perPage = 2
		page = 1

		texts = queryText(ctx, t, sqb, nil, &findFilter)
		assert.Len(t, texts, 2)
		assert.Equal(t, firstID, texts[0].ID)
		assert.Equal(t, secondID, texts[1].ID)

		return nil
	})
}

func TestTextQueryTagCount(t *testing.T) {
	const tagCount = 1
	tagCountCriterion := models.IntCriterionInput{
		Value:    tagCount,
		Modifier: models.CriterionModifierEquals,
	}

	verifyTextsTagCount(t, tagCountCriterion)

	tagCountCriterion.Modifier = models.CriterionModifierNotEquals
	verifyTextsTagCount(t, tagCountCriterion)

	tagCountCriterion.Modifier = models.CriterionModifierGreaterThan
	verifyTextsTagCount(t, tagCountCriterion)

	tagCountCriterion.Modifier = models.CriterionModifierLessThan
	verifyTextsTagCount(t, tagCountCriterion)
}

func verifyTextsTagCount(t *testing.T, tagCountCriterion models.IntCriterionInput) {
	withTxn(func(ctx context.Context) error {
		sqb := db.Text
		textFilter := models.TextFilterType{
			TagCount: &tagCountCriterion,
		}

		texts := queryText(ctx, t, sqb, &textFilter, nil)
		assert.Greater(t, len(texts), 0)

		for _, text := range texts {
			if err := text.LoadTagIDs(ctx, sqb); err != nil {
				t.Errorf("text.LoadTagIDs() error = %v", err)
				return nil
			}
			verifyInt(t, len(text.TagIDs.List()), tagCountCriterion)
		}

		return nil
	})
}

func TestTextQueryPerformerCount(t *testing.T) {
	const performerCount = 1
	performerCountCriterion := models.IntCriterionInput{
		Value:    performerCount,
		Modifier: models.CriterionModifierEquals,
	}

	verifyTextsPerformerCount(t, performerCountCriterion)

	performerCountCriterion.Modifier = models.CriterionModifierNotEquals
	verifyTextsPerformerCount(t, performerCountCriterion)

	performerCountCriterion.Modifier = models.CriterionModifierGreaterThan
	verifyTextsPerformerCount(t, performerCountCriterion)

	performerCountCriterion.Modifier = models.CriterionModifierLessThan
	verifyTextsPerformerCount(t, performerCountCriterion)
}

func verifyTextsPerformerCount(t *testing.T, performerCountCriterion models.IntCriterionInput) {
	withTxn(func(ctx context.Context) error {
		sqb := db.Text
		textFilter := models.TextFilterType{
			PerformerCount: &performerCountCriterion,
		}

		texts := queryText(ctx, t, sqb, &textFilter, nil)
		assert.Greater(t, len(texts), 0)

		for _, text := range texts {
			if err := text.LoadPerformerIDs(ctx, sqb); err != nil {
				t.Errorf("text.LoadPerformerIDs() error = %v", err)
				return nil
			}

			verifyInt(t, len(text.PerformerIDs.List()), performerCountCriterion)
		}

		return nil
	})
}

func TestTextCountByTagID(t *testing.T) {
	withTxn(func(ctx context.Context) error {
		sqb := db.Text

		textCount, err := sqb.CountByTagID(ctx, tagIDs[tagIdxWithText])

		if err != nil {
			t.Errorf("error calling CountByTagID: %s", err.Error())
		}

		assert.Equal(t, 1, textCount)

		textCount, err = sqb.CountByTagID(ctx, 0)

		if err != nil {
			t.Errorf("error calling CountByTagID: %s", err.Error())
		}

		assert.Equal(t, 0, textCount)

		return nil
	})
}

func TestTextCountByGroupID(t *testing.T) {
	withTxn(func(ctx context.Context) error {
		sqb := db.Text

		textCount, err := sqb.CountByGroupID(ctx, groupIDs[groupIdxWithText])

		if err != nil {
			t.Errorf("error calling CountByGroupID: %s", err.Error())
		}

		assert.Equal(t, 1, textCount)

		textCount, err = sqb.CountByGroupID(ctx, 0)

		if err != nil {
			t.Errorf("error calling CountByGroupID: %s", err.Error())
		}

		assert.Equal(t, 0, textCount)

		return nil
	})
}

func TestTextCountByStudioID(t *testing.T) {
	withTxn(func(ctx context.Context) error {
		sqb := db.Text

		textCount, err := sqb.CountByStudioID(ctx, studioIDs[studioIdxWithText])

		if err != nil {
			t.Errorf("error calling CountByStudioID: %s", err.Error())
		}

		assert.Equal(t, 1, textCount)

		textCount, err = sqb.CountByStudioID(ctx, 0)

		if err != nil {
			t.Errorf("error calling CountByStudioID: %s", err.Error())
		}

		assert.Equal(t, 0, textCount)

		return nil
	})
}

func TestFindByMovieID(t *testing.T) {
	withTxn(func(ctx context.Context) error {
		sqb := db.Text

		texts, err := sqb.FindByGroupID(ctx, groupIDs[groupIdxWithText])

		if err != nil {
			t.Errorf("error calling FindByMovieID: %s", err.Error())
		}

		assert.Len(t, texts, 1)
		assert.Equal(t, textIDs[textIdxWithGroup], texts[0].ID)

		texts, err = sqb.FindByGroupID(ctx, 0)

		if err != nil {
			t.Errorf("error calling FindByMovieID: %s", err.Error())
		}

		assert.Len(t, texts, 0)

		return nil
	})
}

func TestFindByPerformerID(t *testing.T) {
	withTxn(func(ctx context.Context) error {
		sqb := db.Text

		texts, err := sqb.FindByPerformerID(ctx, performerIDs[performerIdxWithText])

		if err != nil {
			t.Errorf("error calling FindByPerformerID: %s", err.Error())
		}

		assert.Len(t, texts, 1)
		assert.Equal(t, textIDs[textIdxWithPerformer], texts[0].ID)

		texts, err = sqb.FindByPerformerID(ctx, 0)

		if err != nil {
			t.Errorf("error calling FindByPerformerID: %s", err.Error())
		}

		assert.Len(t, texts, 0)

		return nil
	})
}

func TestTextUpdateTextCover(t *testing.T) {
	if err := withTxn(func(ctx context.Context) error {
		qb := db.Text

		textID := textIDs[textIdxWithGallery]

		return testUpdateImage(t, ctx, textID, qb.UpdateCover, qb.GetCover)
	}); err != nil {
		t.Error(err.Error())
	}
}

func TestTextStashIDs(t *testing.T) {
	if err := withTxn(func(ctx context.Context) error {
		qb := db.Text

		// create text to test against
		const name = "TestTextStashIDs"
		text := &models.Text{
			Title: name,
		}
		if err := qb.Create(ctx, text, nil); err != nil {
			return fmt.Errorf("Error creating text: %s", err.Error())
		}

		if err := text.LoadStashIDs(ctx, qb); err != nil {
			return err
		}

		testTextStashIDs(ctx, t, text)
		return nil
	}); err != nil {
		t.Error(err.Error())
	}
}

func testTextStashIDs(ctx context.Context, t *testing.T, s *models.Text) {
	// ensure no stash IDs to begin with
	assert.Len(t, s.StashIDs.List(), 0)

	// add stash ids
	const stashIDStr = "stashID"
	const endpoint = "endpoint"
	stashID := models.StashID{
		StashID:  stashIDStr,
		Endpoint: endpoint,
	}

	qb := db.Text

	// update stash ids and ensure was updated
	var err error
	s, err = qb.UpdatePartial(ctx, s.ID, models.TextPartial{
		StashIDs: &models.UpdateStashIDs{
			StashIDs: []models.StashID{stashID},
			Mode:     models.RelationshipUpdateModeSet,
		},
	})
	if err != nil {
		t.Error(err.Error())
	}

	if err := s.LoadStashIDs(ctx, qb); err != nil {
		t.Error(err.Error())
		return
	}

	assert.Equal(t, []models.StashID{stashID}, s.StashIDs.List())

	// remove stash ids and ensure was updated
	s, err = qb.UpdatePartial(ctx, s.ID, models.TextPartial{
		StashIDs: &models.UpdateStashIDs{
			StashIDs: []models.StashID{stashID},
			Mode:     models.RelationshipUpdateModeRemove,
		},
	})
	if err != nil {
		t.Error(err.Error())
	}

	if err := s.LoadStashIDs(ctx, qb); err != nil {
		t.Error(err.Error())
		return
	}

	assert.Len(t, s.StashIDs.List(), 0)
}

func TestTextQueryQTrim(t *testing.T) {
	if err := withTxn(func(ctx context.Context) error {
		qb := db.Text

		expectedID := textIDs[textIdxWithSpacedName]

		type test struct {
			query string
			id    int
			count int
		}
		tests := []test{
			{query: " zzz    yyy    ", id: expectedID, count: 1},
			{query: "   \"zzz yyy xxx\" ", id: expectedID, count: 1},
			{query: "zzz", id: expectedID, count: 1},
			{query: "\" zzz    yyy    \"", count: 0},
			{query: "\"zzz    yyy\"", count: 0},
			{query: "\" zzz yyy\"", count: 0},
			{query: "\"zzz yyy  \"", count: 0},
		}

		for _, tst := range tests {
			f := models.FindFilterType{
				Q: &tst.query,
			}
			texts := queryText(ctx, t, qb, nil, &f)

			assert.Len(t, texts, tst.count)
			if len(texts) > 0 {
				assert.Equal(t, tst.id, texts[0].ID)
			}
		}

		findFilter := models.FindFilterType{}
		texts := queryText(ctx, t, qb, nil, &findFilter)
		assert.NotEqual(t, 0, len(texts))

		return nil
	}); err != nil {
		t.Error(err.Error())
	}
}

func TestTextStore_All(t *testing.T) {
	qb := db.Text

	withRollbackTxn(func(ctx context.Context) error {
		got, err := qb.All(ctx)
		if err != nil {
			t.Errorf("TextStore.All() error = %v", err)
			return nil
		}

		// it's possible that other tests have created texts
		assert.GreaterOrEqual(t, len(got), len(textIDs))

		return nil
	})
}

func TestTextStore_FindDuplicates(t *testing.T) {
	qb := db.Text

	withRollbackTxn(func(ctx context.Context) error {
		distance := 0
		durationDiff := -1.
		got, err := qb.FindDuplicates(ctx, distance, durationDiff)
		if err != nil {
			t.Errorf("TextStore.FindDuplicates() error = %v", err)
			return nil
		}

		assert.Len(t, got, dupeTextPhashes)

		distance = 1
		durationDiff = -1.
		got, err = qb.FindDuplicates(ctx, distance, durationDiff)
		if err != nil {
			t.Errorf("TextStore.FindDuplicates() error = %v", err)
			return nil
		}

		assert.Len(t, got, dupeTextPhashes)

		return nil
	})
}

func TestTextStore_AssignFiles(t *testing.T) {
	tests := []struct {
		name    string
		textID int
		fileID  models.FileID
		wantErr bool
	}{
		{
			"valid",
			textIDs[textIdx1WithPerformer],
			textFileIDs[textIdx1WithStudio],
			false,
		},
		{
			"invalid file id",
			textIDs[textIdx1WithPerformer],
			invalidFileID,
			true,
		},
		{
			"invalid text id",
			invalidID,
			textFileIDs[textIdx1WithStudio],
			true,
		},
	}

	qb := db.Text

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withRollbackTxn(func(ctx context.Context) error {
				if err := qb.AssignFiles(ctx, tt.textID, []models.FileID{tt.fileID}); (err != nil) != tt.wantErr {
					t.Errorf("TextStore.AssignFiles() error = %v, wantErr %v", err, tt.wantErr)
				}

				return nil
			})
		})
	}
}

func TestTextStore_AddView(t *testing.T) {
	tests := []struct {
		name          string
		textID       int
		expectedCount int
		wantErr       bool
	}{
		{
			"valid",
			textIDs[textIdx1WithPerformer],
			1, //getTextPlayCount(textIdx1WithPerformer) + 1,
			false,
		},
		{
			"invalid text id",
			invalidID,
			0,
			true,
		},
	}

	qb := db.Text

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withRollbackTxn(func(ctx context.Context) error {
				views, err := qb.AddViews(ctx, tt.textID, nil)
				if (err != nil) != tt.wantErr {
					t.Errorf("TextStore.AddView() error = %v, wantErr %v", err, tt.wantErr)
				}

				if err != nil {
					return nil
				}

				assert := assert.New(t)
				assert.Equal(tt.expectedCount, len(views))

				// find the text and check the count
				count, err := qb.CountViews(ctx, tt.textID)
				if err != nil {
					t.Errorf("TextStore.CountViews() error = %v", err)
				}

				lastView, err := qb.LastView(ctx, tt.textID)
				if err != nil {
					t.Errorf("TextStore.LastView() error = %v", err)
				}

				assert.Equal(tt.expectedCount, count)
				assert.True(lastView.After(time.Now().Add(-1 * time.Minute)))

				return nil
			})
		})
	}
}

func TestTextStore_DecrementWatchCount(t *testing.T) {
	return
}

func TestTextStore_SaveActivity(t *testing.T) {
	var (
		resumeTime   = 111.2
		playDuration = 98.7
	)

	tests := []struct {
		name         string
		textIdx     int
		resumeTime   *float64
		playDuration *float64
		wantErr      bool
	}{
		{
			"both",
			textIdx1WithPerformer,
			&resumeTime,
			&playDuration,
			false,
		},
		{
			"resumeTime only",
			textIdx1WithPerformer,
			&resumeTime,
			nil,
			false,
		},
		{
			"playDuration only",
			textIdx1WithPerformer,
			nil,
			&playDuration,
			false,
		},
		{
			"none",
			textIdx1WithPerformer,
			nil,
			nil,
			false,
		},
		{
			"invalid text id",
			-1,
			&resumeTime,
			&playDuration,
			true,
		},
	}

	qb := db.Text

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withRollbackTxn(func(ctx context.Context) error {
				id := -1
				if tt.textIdx != -1 {
					id = textIDs[tt.textIdx]
				}

				_, err := qb.SaveActivity(ctx, id, tt.resumeTime, tt.playDuration)
				if (err != nil) != tt.wantErr {
					t.Errorf("TextStore.SaveActivity() error = %v, wantErr %v", err, tt.wantErr)
				}

				if err != nil {
					return nil
				}

				assert := assert.New(t)

				// find the text and check the values
				text, err := qb.Find(ctx, id)
				if err != nil {
					t.Errorf("TextStore.Find() error = %v", err)
				}

				expectedResumeTime := getTextResumeTime(tt.textIdx)
				expectedPlayDuration := getTextPlayDuration(tt.textIdx)

				if tt.resumeTime != nil {
					expectedResumeTime = *tt.resumeTime
				}
				if tt.playDuration != nil {
					expectedPlayDuration += *tt.playDuration
				}

				assert.Equal(expectedResumeTime, text.ResumeTime)
				assert.Equal(expectedPlayDuration, text.PlayDuration)

				return nil
			})
		})
	}
}

// TODO Count
// TODO SizeCount

// TODO - this should be in history_test and generalised
func TestTextStore_CountAllViews(t *testing.T) {
	withRollbackTxn(func(ctx context.Context) error {
		qb := db.Text

		textID := textIDs[textIdx1WithPerformer]

		// get the current play count
		currentCount, err := qb.CountAllViews(ctx)
		if err != nil {
			t.Errorf("TextStore.CountAllViews() error = %v", err)
			return nil
		}

		// add a view
		_, err = qb.AddViews(ctx, textID, nil)
		if err != nil {
			t.Errorf("TextStore.AddViews() error = %v", err)
			return nil
		}

		// get the new play count
		newCount, err := qb.CountAllViews(ctx)
		if err != nil {
			t.Errorf("TextStore.CountAllViews() error = %v", err)
			return nil
		}

		assert.Equal(t, currentCount+1, newCount)

		return nil
	})
}

func TestTextStore_CountUniqueViews(t *testing.T) {
	withRollbackTxn(func(ctx context.Context) error {
		qb := db.Text

		textID := textIDs[textIdx1WithPerformer]

		// get the current play count
		currentCount, err := qb.CountUniqueViews(ctx)
		if err != nil {
			t.Errorf("TextStore.CountUniqueViews() error = %v", err)
			return nil
		}

		// add a view
		_, err = qb.AddViews(ctx, textID, nil)
		if err != nil {
			t.Errorf("TextStore.AddViews() error = %v", err)
			return nil
		}

		// add a second view
		_, err = qb.AddViews(ctx, textID, nil)
		if err != nil {
			t.Errorf("TextStore.AddViews() error = %v", err)
			return nil
		}

		// get the new play count
		newCount, err := qb.CountUniqueViews(ctx)
		if err != nil {
			t.Errorf("TextStore.CountUniqueViews() error = %v", err)
			return nil
		}

		assert.Equal(t, currentCount+1, newCount)

		return nil
	})
}
