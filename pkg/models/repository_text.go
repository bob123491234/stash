package models

import (
	"context"
	"time"
)

// TextGetter provides methods to get texts by ID.
type TextGetter interface {
	// TODO - rename this to Find and remove existing method
	FindMany(ctx context.Context, ids []int) ([]*Text, error)
	Find(ctx context.Context, id int) (*Text, error)
}

// TextFinder provides methods to find texts.
type TextFinder interface {
	TextGetter
	FindByFingerprints(ctx context.Context, fp []Fingerprint) ([]*Text, error)
	FindByChecksum(ctx context.Context, checksum string) ([]*Text, error)
	FindByPath(ctx context.Context, path string) ([]*Text, error)
	FindByFileID(ctx context.Context, fileID FileID) ([]*Text, error)
	FindByPrimaryFileID(ctx context.Context, fileID FileID) ([]*Text, error)
	FindByPerformerID(ctx context.Context, performerID int) ([]*Text, error)
}

// TextQueryer provides methods to query texts.
type TextQueryer interface {
	Query(ctx context.Context, options TextQueryOptions) (*TextQueryResult, error)
	QueryCount(ctx context.Context, textFilter *TextFilterType, findFilter *FindFilterType) (int, error)
}

// TextCounter provides methods to count texts.
type TextCounter interface {
	Count(ctx context.Context) (int, error)
	CountByPerformerID(ctx context.Context, performerID int) (int, error)
	CountByFileID(ctx context.Context, fileID FileID) (int, error)
	CountByStudioID(ctx context.Context, studioID int) (int, error)
	CountByTagID(ctx context.Context, tagID int) (int, error)
	CountMissingChecksum(ctx context.Context) (int, error)
	OCountByPerformerID(ctx context.Context, performerID int) (int, error)
}

// TextCreator provides methods to create texts.
type TextCreator interface {
	Create(ctx context.Context, newText *Text, fileIDs []FileID) error
}

// TextUpdater provides methods to update texts.
type TextUpdater interface {
	Update(ctx context.Context, updatedText *Text) error
	UpdatePartial(ctx context.Context, id int, updatedText TextPartial) (*Text, error)
	UpdateCover(ctx context.Context, textID int, cover []byte) error
}

// TextDestroyer provides methods to destroy texts.
type TextDestroyer interface {
	Destroy(ctx context.Context, id int) error
}

type TextCreatorUpdater interface {
	TextCreator
	TextUpdater
}

type ReadDateReader interface {
	CountReads(ctx context.Context, id int) (int, error)
	CountAllReads(ctx context.Context) (int, error)
	CountUniqueReads(ctx context.Context) (int, error)
	GetManyReadCount(ctx context.Context, ids []int) ([]int, error)
	GetReadDates(ctx context.Context, relatedID int) ([]time.Time, error)
	GetManyReadDates(ctx context.Context, ids []int) ([][]time.Time, error)
	GetManyLastRead(ctx context.Context, ids []int) ([]*time.Time, error)
}

// type ODateReader interface {
// 	GetOCount(ctx context.Context, id int) (int, error)
// 	GetManyOCount(ctx context.Context, ids []int) ([]int, error)
// 	GetAllOCount(ctx context.Context) (int, error)
// 	GetODates(ctx context.Context, relatedID int) ([]time.Time, error)
// 	GetManyODates(ctx context.Context, ids []int) ([][]time.Time, error)
// }

// TextReader provides all methods to read texts.
type TextReader interface {
	TextFinder
	TextQueryer
	TextCounter

	URLLoader
	ReadDateReader
	ODateReader
	FileIDLoader
	GalleryIDLoader
	PerformerIDLoader
	TagIDLoader

	All(ctx context.Context) ([]*Text, error)
	Wall(ctx context.Context, q *string) ([]*Text, error)
	Size(ctx context.Context) (float64, error)
	Duration(ctx context.Context) (float64, error)
	ReadDuration(ctx context.Context) (float64, error)
	GetCover(ctx context.Context, textID int) ([]byte, error)
	HasCover(ctx context.Context, textID int) (bool, error)
}

// type OHistoryWriter interface {
// 	AddO(ctx context.Context, id int, dates []time.Time) ([]time.Time, error)
// 	DeleteO(ctx context.Context, id int, dates []time.Time) ([]time.Time, error)
// 	ResetO(ctx context.Context, id int) (int, error)
// }

type ReadHistoryWriter interface {
	AddReads(ctx context.Context, textID int, dates []time.Time) ([]time.Time, error)
	DeleteReads(ctx context.Context, id int, dates []time.Time) ([]time.Time, error)
	DeleteAllReads(ctx context.Context, id int) (int, error)
}

// TextWriter provides all methods to modify texts.
type TextWriter interface {
	TextCreator
	TextUpdater
	TextDestroyer

	AddFileID(ctx context.Context, id int, fileID FileID) error
	AssignFiles(ctx context.Context, textID int, fileID []FileID) error

	OHistoryWriter
	ReadHistoryWriter
	SaveActivity(ctx context.Context, textID int, resumeLocation *float64, readDuration *float64) (bool, error)
}

// TextReaderWriter provides all text methods.
type TextReaderWriter interface {
	TextReader
	TextWriter
}
