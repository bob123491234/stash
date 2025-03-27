package text

import (
	"errors"
	"strconv"
	"testing"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/models/mocks"
	"github.com/stashapp/stash/pkg/sliceutil/intslice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestUpdater_IsEmpty(t *testing.T) {
	organized := true
	ids := []int{1}
	stashIDs := []models.StashID{
		{},
	}
	cover := []byte{1}

	tests := []struct {
		name string
		u    *UpdateSet
		want bool
	}{
		{
			"empty",
			&UpdateSet{},
			true,
		},
		{
			"partial set",
			&UpdateSet{
				Partial: models.TextPartial{
					Organized: models.NewOptionalBool(organized),
				},
			},
			false,
		},
		{
			"performer set",
			&UpdateSet{
				Partial: models.TextPartial{
					PerformerIDs: &models.UpdateIDs{
						IDs:  ids,
						Mode: models.RelationshipUpdateModeSet,
					},
				},
			},
			false,
		},
		{
			"tags set",
			&UpdateSet{
				Partial: models.TextPartial{
					TagIDs: &models.UpdateIDs{
						IDs:  ids,
						Mode: models.RelationshipUpdateModeSet,
					},
				},
			},
			false,
		},
		{
			"performer set",
			&UpdateSet{
				Partial: models.TextPartial{
					StashIDs: &models.UpdateStashIDs{
						StashIDs: stashIDs,
						Mode:     models.RelationshipUpdateModeSet,
					},
				},
			},
			false,
		},
		{
			"cover set",
			&UpdateSet{
				CoverImage: cover,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.u.IsEmpty(); got != tt.want {
				t.Errorf("Updater.IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUpdater_Update(t *testing.T) {
	const (
		textID = iota + 1
		badUpdateID
		badPerformersID
		badTagsID
		badStashIDsID
		badCoverID
		performerID
		tagID
	)

	performerIDs := []int{performerID}
	tagIDs := []int{tagID}
	stashID := "stashID"
	endpoint := "endpoint"

	title := "title"
	cover := []byte("cover")

	validText := &models.Text{}

	updateErr := errors.New("error updating")

	db := mocks.NewDatabase()

	db.Text.On("UpdatePartial", testCtx, mock.MatchedBy(func(id int) bool {
		return id != badUpdateID
	}), mock.Anything).Return(validText, nil)
	db.Text.On("UpdatePartial", testCtx, badUpdateID, mock.Anything).Return(nil, updateErr)

	db.Text.On("UpdateCover", testCtx, textID, cover).Return(nil).Once()
	db.Text.On("UpdateCover", testCtx, badCoverID, cover).Return(updateErr).Once()

	tests := []struct {
		name    string
		u       *UpdateSet
		wantNil bool
		wantErr bool
	}{
		{
			"empty",
			&UpdateSet{
				ID: textID,
			},
			true,
			true,
		},
		{
			"update all",
			&UpdateSet{
				ID: textID,
				Partial: models.TextPartial{
					PerformerIDs: &models.UpdateIDs{
						IDs:  performerIDs,
						Mode: models.RelationshipUpdateModeSet,
					},
					TagIDs: &models.UpdateIDs{
						IDs:  tagIDs,
						Mode: models.RelationshipUpdateModeSet,
					},
					StashIDs: &models.UpdateStashIDs{
						StashIDs: []models.StashID{
							{
								StashID:  stashID,
								Endpoint: endpoint,
							},
						},
						Mode: models.RelationshipUpdateModeSet,
					},
				},
				CoverImage: cover,
			},
			false,
			false,
		},
		{
			"update fields only",
			&UpdateSet{
				ID: textID,
				Partial: models.TextPartial{
					Title: models.NewOptionalString(title),
				},
			},
			false,
			false,
		},
		{
			"error updating text",
			&UpdateSet{
				ID: badUpdateID,
				Partial: models.TextPartial{
					Title: models.NewOptionalString(title),
				},
			},
			true,
			true,
		},
		{
			"error updating cover",
			&UpdateSet{
				ID:         badCoverID,
				CoverImage: cover,
			},
			true,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.u.Update(testCtx, db.Text)
			if (err != nil) != tt.wantErr {
				t.Errorf("Updater.Update() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (got == nil) != tt.wantNil {
				t.Errorf("Updater.Update() = %v, want %v", got, tt.wantNil)
			}
		})
	}

	db.AssertExpectations(t)
}

func TestUpdateSet_UpdateInput(t *testing.T) {
	const (
		textID = iota + 1
		badUpdateID
		badPerformersID
		badTagsID
		badStashIDsID
		badCoverID
		performerID
		tagID
	)

	textIDStr := strconv.Itoa(textID)

	performerIDs := []int{performerID}
	performerIDStrs := intslice.IntSliceToStringSlice(performerIDs)
	tagIDs := []int{tagID}
	tagIDStrs := intslice.IntSliceToStringSlice(tagIDs)
	stashID := "stashID"
	endpoint := "endpoint"
	stashIDs := []models.StashID{
		{
			StashID:  stashID,
			Endpoint: endpoint,
		},
	}
	stashIDInputs := []models.StashID{
		{
			StashID:  stashID,
			Endpoint: endpoint,
		},
	}

	title := "title"
	cover := []byte("cover")
	coverB64 := "Y292ZXI="

	tests := []struct {
		name string
		u    UpdateSet
		want models.TextUpdateInput
	}{
		{
			"empty",
			UpdateSet{
				ID: textID,
			},
			models.TextUpdateInput{
				ID: textIDStr,
			},
		},
		{
			"update all",
			UpdateSet{
				ID: textID,
				Partial: models.TextPartial{
					PerformerIDs: &models.UpdateIDs{
						IDs:  performerIDs,
						Mode: models.RelationshipUpdateModeSet,
					},
					TagIDs: &models.UpdateIDs{
						IDs:  tagIDs,
						Mode: models.RelationshipUpdateModeSet,
					},
					StashIDs: &models.UpdateStashIDs{
						StashIDs: stashIDs,
						Mode:     models.RelationshipUpdateModeSet,
					},
				},
				CoverImage: cover,
			},
			models.TextUpdateInput{
				ID:           textIDStr,
				PerformerIds: performerIDStrs,
				TagIds:       tagIDStrs,
				StashIds:     stashIDInputs,
				CoverImage:   &coverB64,
			},
		},
		{
			"update fields only",
			UpdateSet{
				ID: textID,
				Partial: models.TextPartial{
					Title: models.NewOptionalString(title),
				},
			},
			models.TextUpdateInput{
				ID:    textIDStr,
				Title: &title,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.u.UpdateInput()
			assert.Equal(t, tt.want, got)
		})
	}
}
