package text

import (
	"context"
	"errors"
	"fmt"

	"github.com/stashapp/stash/pkg/file/video"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/models/paths"
	"github.com/stashapp/stash/pkg/plugin"
	"github.com/stashapp/stash/pkg/plugin/hook"
	"github.com/stashapp/stash/pkg/txn"
)

var (
	ErrNotTextFile = errors.New("not a text file")
)

type ScanCreatorUpdater interface {
	FindByFileID(ctx context.Context, fileID models.FileID) ([]*models.Text, error)
	FindByFingerprints(ctx context.Context, fp []models.Fingerprint) ([]*models.Text, error)
	GetFiles(ctx context.Context, relatedID int) ([]*models.BaseFile, error)

	Create(ctx context.Context, newText *models.Text, fileIDs []models.FileID) error
	UpdatePartial(ctx context.Context, id int, updatedText models.TextPartial) (*models.Text, error)
	AddFileID(ctx context.Context, id int, fileID models.FileID) error
}

type ScanGenerator interface {
	Generate(ctx context.Context, s *models.Text, f *models.BaseFile) error
}

type ScanHandler struct {
	CreatorUpdater ScanCreatorUpdater

	ScanGenerator ScanGenerator
	PluginCache   *plugin.Cache

	Paths *paths.Paths
}

func (h *ScanHandler) validate() error {
	if h.CreatorUpdater == nil {
		return errors.New("CreatorUpdater is required")
	}
	if h.ScanGenerator == nil {
		return errors.New("ScanGenerator is required")
	}
	if h.Paths == nil {
		return errors.New("Paths is required")
	}

	return nil
}

func (h *ScanHandler) Handle(ctx context.Context, f models.File, oldFile models.File) error {
	if err := h.validate(); err != nil {
		return err
	}

	videoFile, ok := f.(*models.BaseFile)
	if !ok {
		return ErrNotTextFile
	}

	// try to match the file to a text
	existing, err := h.CreatorUpdater.FindByFileID(ctx, f.Base().ID)
	if err != nil {
		return fmt.Errorf("finding existing text: %w", err)
	}

	if len(existing) == 0 {
		// try also to match file by fingerprints
		existing, err = h.CreatorUpdater.FindByFingerprints(ctx, videoFile.Fingerprints)
		if err != nil {
			return fmt.Errorf("finding existing text by fingerprints: %w", err)
		}
	}

	if len(existing) > 0 {
		updateExisting := oldFile != nil
		if err := h.associateExisting(ctx, existing, videoFile, updateExisting); err != nil {
			return err
		}
	} else {
		// create a new text
		newText := models.NewText()

		logger.Infof("%s doesn't exist. Creating new text...", f.Base().Path)

		if err := h.CreatorUpdater.Create(ctx, &newText, []models.FileID{videoFile.ID}); err != nil {
			return fmt.Errorf("creating new text: %w", err)
		}

		h.PluginCache.RegisterPostHooks(ctx, newText.ID, hook.TextCreatePost, nil, nil)

		existing = []*models.Text{&newText}
	}

	if oldFile != nil {
		oldHash := oldFile.Base().Fingerprints.GetString(models.FingerprintTypeMD5)
		newHash := f.Base().Fingerprints.GetString(models.FingerprintTypeMD5)

		if oldHash != "" && newHash != "" && oldHash != newHash {
			// remove cache dir of gallery
			_ = os.Remove(h.Paths.Generated.GetThumbnailPath(oldHash, models.DefaultGthumbWidth))
		}
	}

	// do this after the commit so that cover generation doesn't hold up the transaction
	txn.AddPostCommitHook(ctx, func(ctx context.Context) {
		for _, s := range existing {
			if err := h.ScanGenerator.Generate(ctx, s, videoFile); err != nil {
				// just log if cover generation fails. We can try again on rescan
				logger.Errorf("Error generating content for %s: %v", videoFile.Path, err)
			}
		}
	})

	return nil
}

func (h *ScanHandler) associateExisting(ctx context.Context, existing []*models.Text, f *models.BaseFile, updateExisting bool) error {
	for _, s := range existing {
		if err := s.LoadFiles(ctx, h.CreatorUpdater); err != nil {
			return err
		}

		found := false
		for _, sf := range s.Files.List() {
			if sf.ID == f.ID {
				found = true
				break
			}
		}

		if !found {
			logger.Infof("Adding %s to text %s", f.Path, s.DisplayName())

			if err := h.CreatorUpdater.AddFileID(ctx, s.ID, f.ID); err != nil {
				return fmt.Errorf("adding file to text: %w", err)
			}

			// update updated_at time
			textPartial := models.NewTextPartial()
			if _, err := h.CreatorUpdater.UpdatePartial(ctx, s.ID, textPartial); err != nil {
				return fmt.Errorf("updating text: %w", err)
			}
		}

		if !found || updateExisting {
			h.PluginCache.RegisterPostHooks(ctx, s.ID, hook.TextUpdatePost, nil, nil)
		}
	}

	return nil
}
