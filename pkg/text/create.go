package text

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/plugin/hook"
)

func (t *Service) Create(ctx context.Context, input *models.Text, fileIDs []models.FileID, coverImage []byte) (*models.Text, error) {
	// title must be set if no files are provided
	if input.Title == "" && len(fileIDs) == 0 {
		return nil, errors.New("title must be set if text has no files")
	}

	now := time.Now()
	newText := *input
	newText.CreatedAt = now
	newText.UpdatedAt = now

	// don't pass the file ids since they may be already assigned
	// assign them afterwards
	if err := t.Repository.Create(ctx, &newText, nil); err != nil {
		return nil, fmt.Errorf("creating new text: %w", err)
	}

	for _, f := range fileIDs {
		if err := t.AssignFile(ctx, newText.ID, f); err != nil {
			return nil, fmt.Errorf("assigning file %d to new text: %w", f, err)
		}
	}

	if len(fileIDs) > 0 {
		// assign the primary to the first
		if _, err := t.Repository.UpdatePartial(ctx, newText.ID, models.TextPartial{
			PrimaryFileID: &fileIDs[0],
		}); err != nil {
			return nil, fmt.Errorf("setting primary file on new text: %w", err)
		}
	}

	// re-find the text so that it correctly returns file-related fields
	ret, err := t.Repository.Find(ctx, newText.ID)
	if err != nil {
		return nil, err
	}

	if len(coverImage) > 0 {
		if err := t.Repository.UpdateCover(ctx, ret.ID, coverImage); err != nil {
			return nil, fmt.Errorf("setting cover on new text: %w", err)
		}
	}

	t.PluginCache.RegisterPostHooks(ctx, ret.ID, hook.TextCreatePost, nil, nil)

	// re-find the text so that it correctly returns file-related fields
	return ret, nil
}
