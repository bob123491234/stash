package models

import "context"

// TextBookmarkGetter provides methods to get text bookmarks by ID.
type TextBookmarkGetter interface {
	// TODO - rename this to Find and remove existing method
	FindMany(ctx context.Context, ids []int) ([]*TextBookmark, error)
	Find(ctx context.Context, id int) (*TextBookmark, error)
}

// TextBookmarkFinder provides methods to find text bookmarks.
type TextBookmarkFinder interface {
	TextBookmarkGetter
	FindByTextID(ctx context.Context, textID int) ([]*TextBookmark, error)
}

// TextBookmarkQueryer provides methods to query text bookmarks.
type TextBookmarkQueryer interface {
	Query(ctx context.Context, textBookmarkFilter *TextBookmarkFilterType, findFilter *FindFilterType) ([]*TextBookmark, int, error)
	QueryCount(ctx context.Context, textBookmarkFilter *TextBookmarkFilterType, findFilter *FindFilterType) (int, error)
}

// TextBookmarkCounter provides methods to count text bookmarks.
type TextBookmarkCounter interface {
	Count(ctx context.Context) (int, error)
	CountByTagID(ctx context.Context, tagID int) (int, error)
}

// TextBookmarkCreator provides methods to create text bookmarks.
type TextBookmarkCreator interface {
	Create(ctx context.Context, newTextBookmark *TextBookmark) error
}

// TextBookmarkUpdater provides methods to update text bookmarks.
type TextBookmarkUpdater interface {
	Update(ctx context.Context, updatedTextBookmark *TextBookmark) error
	UpdatePartial(ctx context.Context, id int, updatedTextBookmark TextBookmarkPartial) (*TextBookmark, error)
	UpdateTags(ctx context.Context, bookmarkID int, tagIDs []int) error
}

// TextBookmarkDestroyer provides methods to destroy text bookmarks.
type TextBookmarkDestroyer interface {
	Destroy(ctx context.Context, id int) error
}

type TextBookmarkCreatorUpdater interface {
	TextBookmarkCreator
	TextBookmarkUpdater
}

// TextBookmarkReader provides all methods to read text bookmarks.
type TextBookmarkReader interface {
	TextBookmarkFinder
	TextBookmarkQueryer
	TextBookmarkCounter

	TagIDLoader

	All(ctx context.Context) ([]*TextBookmark, error)
	Wall(ctx context.Context, q *string) ([]*TextBookmark, error)
	GetBookmarkStrings(ctx context.Context, q *string, sort *string) ([]*BookmarkStringsResultType, error)
}

// TextBookmarkWriter provides all methods to modify text bookmarks.
type TextBookmarkWriter interface {
	TextBookmarkCreator
	TextBookmarkUpdater
	TextBookmarkDestroyer
}

// TextBookmarkReaderWriter provides all text bookmark methods.
type TextBookmarkReaderWriter interface {
	TextBookmarkReader
	TextBookmarkWriter
}
