package text

import (
	"context"
	"path/filepath"

	"github.com/stashapp/stash/pkg/file"
	"github.com/stashapp/stash/pkg/file/video"
	"github.com/stashapp/stash/pkg/fsutil"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/models/paths"
)

// FileDeleter is an extension of file.Deleter that handles deletion of text files.
type FileDeleter struct {
	*file.Deleter

	FileNamingAlgo models.HashAlgorithm
	Paths          *paths.Paths
}

// Destroy deletes a text and its associated relationships from the
// database.
func (s *Service) Destroy(ctx context.Context, text *models.Text, fileDeleter *FileDeleter, deleteGenerated, deleteFile bool) error {
	mqb := s.BookmarkRepository
	bookmarks, err := mqb.FindByTextID(ctx, text.ID)
	if err != nil {
		return err
	}

	for _, m := range bookmarks {
		if err := DestroyBookmark(ctx, text, m, mqb, fileDeleter); err != nil {
			return err
		}
	}

	if deleteFile {
		if err := s.deleteFiles(ctx, text, fileDeleter); err != nil {
			return err
		}
	}

	if err := s.Repository.Destroy(ctx, text.ID); err != nil {
		return err
	}

	return nil
}

// deleteFiles deletes files from the database and file system
func (s *Service) deleteFiles(ctx context.Context, text *models.Text, fileDeleter *FileDeleter) error {
	if err := text.LoadFiles(ctx, s.Repository); err != nil {
		return err
	}

	for _, f := range text.Files.List() {
		// only delete files where there is no other associated text
		otherTexts, err := s.Repository.FindByFileID(ctx, f.ID)
		if err != nil {
			return err
		}

		if len(otherTexts) > 1 {
			// other texts associated, don't remove
			continue
		}

		const deleteFile = true
		logger.Info("Deleting text file: ", f.Path)
		if err := file.Destroy(ctx, s.File, f, fileDeleter.Deleter, deleteFile); err != nil {
			return err
		}
	}

	return nil
}

// DestroyBookmark deletes the text bookmark from the database
func DestroyBookmark(ctx context.Context, text *models.Text, textBookmark *models.TextBookmark, qb models.TextBookmarkDestroyer, fileDeleter *FileDeleter) error {
	if err := qb.Destroy(ctx, textBookmark.ID); err != nil {
		return err
	}

	return nil
}
