package text

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/stashapp/stash/pkg/fsutil"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/sliceutil"
	"github.com/stashapp/stash/pkg/txn"
)

type MergeOptions struct {
	TextPartial        models.TextPartial
	IncludeReadHistory bool
	IncludeOHistory    bool
}

func (s *Service) Merge(ctx context.Context, sourceIDs []int, destinationID int, fileDeleter *FileDeleter, options MergeOptions) error {
	textPartial := options.TextPartial

	// ensure source ids are unique
	sourceIDs = sliceutil.AppendUniques(nil, sourceIDs)

	// ensure destination is not in source list
	if sliceutil.Contains(sourceIDs, destinationID) {
		return errors.New("destination text cannot be in source list")
	}

	dest, err := s.Repository.Find(ctx, destinationID)
	if err != nil {
		return fmt.Errorf("finding destination text ID %d: %w", destinationID, err)
	}

	sources, err := s.Repository.FindMany(ctx, sourceIDs)
	if err != nil {
		return fmt.Errorf("finding source texts: %w", err)
	}

	var fileIDs []models.FileID

	for _, src := range sources {
		if err := src.LoadRelationships(ctx, s.Repository); err != nil {
			return fmt.Errorf("loading text relationships from %d: %w", src.ID, err)
		}

		for _, f := range src.Files.List() {
			fileIDs = append(fileIDs, f.Base().ID)
		}

		if err := s.mergeTextBookmarks(ctx, dest, src); err != nil {
			return err
		}
	}

	// move files to destination text
	if len(fileIDs) > 0 {
		if err := s.Repository.AssignFiles(ctx, destinationID, fileIDs); err != nil {
			return fmt.Errorf("moving files to destination text: %w", err)
		}

		// if text didn't already have a primary file, then set it now
		if dest.PrimaryFileID == nil {
			textPartial.PrimaryFileID = &fileIDs[0]
		} else {
			// don't allow changing primary file ID from the input values
			textPartial.PrimaryFileID = nil
		}
	}

	if _, err := s.Repository.UpdatePartial(ctx, destinationID, textPartial); err != nil {
		return fmt.Errorf("updating text: %w", err)
	}

	// merge read history
	if options.IncludeReadHistory {
		var allDates []time.Time
		for _, src := range sources {
			thisDates, err := s.Repository.GetReadDates(ctx, src.ID)
			if err != nil {
				return fmt.Errorf("getting read dates for text %d: %w", src.ID, err)
			}

			allDates = append(allDates, thisDates...)
		}

		if len(allDates) > 0 {
			if _, err := s.Repository.AddReads(ctx, destinationID, allDates); err != nil {
				return fmt.Errorf("adding read dates to text %d: %w", destinationID, err)
			}
		}
	}

	// merge o history
	if options.IncludeOHistory {
		var allDates []time.Time
		for _, src := range sources {
			thisDates, err := s.Repository.GetODates(ctx, src.ID)
			if err != nil {
				return fmt.Errorf("getting o dates for text %d: %w", src.ID, err)
			}

			allDates = append(allDates, thisDates...)
		}

		if len(allDates) > 0 {
			if _, err := s.Repository.AddO(ctx, destinationID, allDates); err != nil {
				return fmt.Errorf("adding o dates to text %d: %w", destinationID, err)
			}
		}
	}

	// delete old texts
	for _, src := range sources {
		const deleteGenerated = true
		const deleteFile = false
		if err := s.Destroy(ctx, src, fileDeleter, deleteGenerated, deleteFile); err != nil {
			return fmt.Errorf("deleting text %d: %w", src.ID, err)
		}
	}

	return nil
}

func (s *Service) mergeTextBookmarks(ctx context.Context, dest *models.Text, src *models.Text) error {
	bookmarks, err := s.BookmarkRepository.FindByTextID(ctx, src.ID)
	if err != nil {
		return fmt.Errorf("finding text bookmarks: %w", err)
	}

	type rename struct {
		src  string
		dest string
	}

	var toRename []rename

	for _, m := range bookmarks {
		// updated the text id
		m.TextID = dest.ID

		if err := s.BookmarkRepository.Update(ctx, m); err != nil {
			return fmt.Errorf("updating text bookmark %d: %w", m.ID, err)
		}
	}

	if len(toRename) > 0 {
		txn.AddPostCommitHook(ctx, func(ctx context.Context) {
			// rename the files if they exist
			for _, e := range toRename {
				srcExists, _ := fsutil.FileExists(e.src)
				destExists, _ := fsutil.FileExists(e.dest)

				if srcExists && !destExists {
					destDir := filepath.Dir(e.dest)
					if err := fsutil.EnsureDir(destDir); err != nil {
						logger.Errorf("Error creating generated bookmark folder %s: %v", destDir, err)
						continue
					}

					if err := os.Rename(e.src, e.dest); err != nil {
						logger.Errorf("Error renaming generated bookmark file from %s to %s: %v", e.src, e.dest, err)
					}
				}
			}
		})
	}

	return nil
}
