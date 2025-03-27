package api

import (
	"context"

	"github.com/stashapp/stash/pkg/models"
)

func (r *queryResolver) FindTextBookmarks(ctx context.Context, textBookmarkFilter *models.TextBookmarkFilterType, filter *models.FindFilterType) (ret *FindTextBookmarksResultType, err error) {
	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		textBookmarks, total, err := r.repository.TextBookmark.Query(ctx, textBookmarkFilter, filter)
		if err != nil {
			return err
		}
		ret = &FindTextBookmarksResultType{
			Count:         total,
			TextBookmarks: textBookmarks,
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *queryResolver) AllTextBookmarks(ctx context.Context) (ret []*models.TextBookmark, err error) {
	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		ret, err = r.repository.TextBookmark.All(ctx)
		return err
	}); err != nil {
		return nil, err
	}

	return ret, nil
}
