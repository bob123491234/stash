package text

import (
	"context"
	"fmt"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/models/jsonschema"
)

type BookmarkCreatorUpdater interface {
	models.TextBookmarkCreatorUpdater
	FindByTextID(ctx context.Context, textID int) ([]*models.TextBookmark, error)
}

type BookmarkImporter struct {
	TextID              int
	ReaderWriter        BookmarkCreatorUpdater
	TagWriter           models.TagFinderCreator
	Input               jsonschema.TextBookmark
	MissingRefBehaviour models.ImportMissingRefEnum

	tags     []*models.Tag
	bookmark models.TextBookmark
}

func (i *BookmarkImporter) PreImport(ctx context.Context) error {
	i.bookmark = models.TextBookmark{
		Title:     i.Input.Title,
		Location:  i.Input.Location,
		TextID:    i.TextID,
		CreatedAt: i.Input.CreatedAt.GetTime(),
		UpdatedAt: i.Input.UpdatedAt.GetTime(),
	}

	if err := i.populateTags(ctx); err != nil {
		return err
	}

	return nil
}

func (i *BookmarkImporter) populateTags(ctx context.Context) error {
	// primary tag cannot be ignored
	mrb := i.MissingRefBehaviour
	if mrb == models.ImportMissingRefEnumIgnore {
		mrb = models.ImportMissingRefEnumFail
	}

	primaryTag, err := importTags(ctx, i.TagWriter, []string{i.Input.PrimaryTag}, mrb)
	if err != nil {
		return err
	}

	i.bookmark.PrimaryTagID = primaryTag[0].ID

	if len(i.Input.Tags) > 0 {
		tags, err := importTags(ctx, i.TagWriter, i.Input.Tags, i.MissingRefBehaviour)
		if err != nil {
			return err
		}

		i.tags = tags
	}

	return nil
}

func (i *BookmarkImporter) PostImport(ctx context.Context, id int) error {
	if len(i.tags) > 0 {
		var tagIDs []int
		for _, t := range i.tags {
			tagIDs = append(tagIDs, t.ID)
		}
		if err := i.ReaderWriter.UpdateTags(ctx, id, tagIDs); err != nil {
			return fmt.Errorf("failed to associate tags: %v", err)
		}
	}

	return nil
}

func (i *BookmarkImporter) Name() string {
	return fmt.Sprintf("%s (%s)", i.Input.Title, i.Input.Location)
}

func (i *BookmarkImporter) FindExistingID(ctx context.Context) (*int, error) {
	existingBookmarks, err := i.ReaderWriter.FindByTextID(ctx, i.TextID)

	if err != nil {
		return nil, err
	}

	for _, b := range existingBookmarks {
		if b.Location == i.bookmark.Location {
			id := b.ID
			return &id, nil
		}
	}

	return nil, nil
}

func (i *BookmarkImporter) Create(ctx context.Context) (*int, error) {
	err := i.ReaderWriter.Create(ctx, &i.bookmark)
	if err != nil {
		return nil, fmt.Errorf("error creating bookmark: %v", err)
	}

	id := i.bookmark.ID
	return &id, nil
}

func (i *BookmarkImporter) Update(ctx context.Context, id int) error {
	bookmark := i.bookmark
	bookmark.ID = id
	err := i.ReaderWriter.Update(ctx, &bookmark)
	if err != nil {
		return fmt.Errorf("error updating existing bookmark: %v", err)
	}

	return nil
}
