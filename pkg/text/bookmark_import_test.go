package text

// import (
// 	"context"
// 	"errors"
// 	"testing"

// 	"github.com/stashapp/stash/pkg/models"
// 	"github.com/stashapp/stash/pkg/models/jsonschema"
// 	"github.com/stashapp/stash/pkg/models/mocks"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/mock"
// )

// const (
// 	seconds      = "5"
// 	secondsFloat = 5.0
// 	errTextID   = 999
// )

// func TestBookmarkImporterName(t *testing.T) {
// 	i := BookmarkImporter{
// 		Input: jsonschema.TextBookmark{
// 			Title:   title,
// 			Seconds: seconds,
// 		},
// 	}

// 	assert.Equal(t, title+" (5)", i.Name())
// }

// func TestBookmarkImporterPreImportWithTag(t *testing.T) {
// 	tagReaderWriter := &mocks.TagReaderWriter{}
// 	ctx := context.Background()

// 	i := BookmarkImporter{
// 		TagWriter:           tagReaderWriter,
// 		MissingRefBehaviour: models.ImportMissingRefEnumFail,
// 		Input: jsonschema.TextBookmark{
// 			PrimaryTag: existingTagName,
// 		},
// 	}

// 	tagReaderWriter.On("FindByNames", ctx, []string{existingTagName}, false).Return([]*models.Tag{
// 		{
// 			ID:   existingTagID,
// 			Name: existingTagName,
// 		},
// 	}, nil).Times(4)
// 	tagReaderWriter.On("FindByNames", ctx, []string{existingTagErr}, false).Return(nil, errors.New("FindByNames error")).Times(2)

// 	err := i.PreImport(ctx)
// 	assert.Nil(t, err)
// 	assert.Equal(t, existingTagID, i.bookmark.PrimaryTagID)

// 	i.Input.PrimaryTag = existingTagErr
// 	err = i.PreImport(ctx)
// 	assert.NotNil(t, err)

// 	i.Input.PrimaryTag = existingTagName
// 	i.Input.Tags = []string{
// 		existingTagName,
// 	}
// 	err = i.PreImport(ctx)
// 	assert.Nil(t, err)
// 	assert.Equal(t, existingTagID, i.tags[0].ID)

// 	i.Input.Tags[0] = existingTagErr
// 	err = i.PreImport(ctx)
// 	assert.NotNil(t, err)

// 	tagReaderWriter.AssertExpectations(t)
// }

// func TestBookmarkImporterPostImportUpdateTags(t *testing.T) {
// 	textBookmarkReaderWriter := &mocks.TextBookmarkReaderWriter{}
// 	ctx := context.Background()

// 	i := BookmarkImporter{
// 		ReaderWriter: textBookmarkReaderWriter,
// 		tags: []*models.Tag{
// 			{
// 				ID: existingTagID,
// 			},
// 		},
// 	}

// 	updateErr := errors.New("UpdateTags error")

// 	textBookmarkReaderWriter.On("UpdateTags", ctx, textID, []int{existingTagID}).Return(nil).Once()
// 	textBookmarkReaderWriter.On("UpdateTags", ctx, errTagsID, mock.AnythingOfType("[]int")).Return(updateErr).Once()

// 	err := i.PostImport(ctx, textID)
// 	assert.Nil(t, err)

// 	err = i.PostImport(ctx, errTagsID)
// 	assert.NotNil(t, err)

// 	textBookmarkReaderWriter.AssertExpectations(t)
// }

// func TestBookmarkImporterFindExistingID(t *testing.T) {
// 	readerWriter := &mocks.TextBookmarkReaderWriter{}
// 	ctx := context.Background()

// 	i := BookmarkImporter{
// 		ReaderWriter: readerWriter,
// 		TextID:      textID,
// 		bookmark: models.TextBookmark{
// 			Seconds: secondsFloat,
// 		},
// 	}

// 	expectedErr := errors.New("FindBy* error")
// 	readerWriter.On("FindByTextID", ctx, textID).Return([]*models.TextBookmark{
// 		{
// 			ID:      existingTextID,
// 			Seconds: secondsFloat,
// 		},
// 	}, nil).Times(2)
// 	readerWriter.On("FindByTextID", ctx, errTextID).Return(nil, expectedErr).Once()

// 	id, err := i.FindExistingID(ctx)
// 	assert.Equal(t, existingTextID, *id)
// 	assert.Nil(t, err)

// 	i.bookmark.Seconds++
// 	id, err = i.FindExistingID(ctx)
// 	assert.Nil(t, id)
// 	assert.Nil(t, err)

// 	i.TextID = errTextID
// 	id, err = i.FindExistingID(ctx)
// 	assert.Nil(t, id)
// 	assert.NotNil(t, err)

// 	readerWriter.AssertExpectations(t)
// }

// func TestBookmarkImporterCreate(t *testing.T) {
// 	readerWriter := &mocks.TextBookmarkReaderWriter{}
// 	ctx := context.Background()

// 	text := models.TextBookmark{
// 		Title: title,
// 	}

// 	textErr := models.TextBookmark{
// 		Title: textNameErr,
// 	}

// 	i := BookmarkImporter{
// 		ReaderWriter: readerWriter,
// 		bookmark:       text,
// 	}

// 	errCreate := errors.New("Create error")
// 	readerWriter.On("Create", ctx, text).Return(&models.TextBookmark{
// 		ID: textID,
// 	}, nil).Once()
// 	readerWriter.On("Create", ctx, textErr).Return(nil, errCreate).Once()

// 	id, err := i.Create(ctx)
// 	assert.Equal(t, textID, *id)
// 	assert.Nil(t, err)

// 	i.bookmark = textErr
// 	id, err = i.Create(ctx)
// 	assert.Nil(t, id)
// 	assert.NotNil(t, err)

// 	readerWriter.AssertExpectations(t)
// }

// func TestBookmarkImporterUpdate(t *testing.T) {
// 	readerWriter := &mocks.TextBookmarkReaderWriter{}
// 	ctx := context.Background()

// 	text := models.TextBookmark{
// 		Title: title,
// 	}

// 	textErr := models.TextBookmark{
// 		Title: textNameErr,
// 	}

// 	i := BookmarkImporter{
// 		ReaderWriter: readerWriter,
// 		bookmark:       text,
// 	}

// 	errUpdate := errors.New("Update error")

// 	// id needs to be set for the mock input
// 	text.ID = textID
// 	readerWriter.On("Update", ctx, text).Return(nil, nil).Once()

// 	err := i.Update(ctx, textID)
// 	assert.Nil(t, err)

// 	i.bookmark = textErr

// 	// need to set id separately
// 	textErr.ID = errImageID
// 	readerWriter.On("Update", ctx, textErr).Return(nil, errUpdate).Once()

// 	err = i.Update(ctx, errImageID)
// 	assert.NotNil(t, err)

// 	readerWriter.AssertExpectations(t)
// }
