package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/jmoiron/sqlx"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/sliceutil"
)

const (
	textsChaptersTable = "texts_chapters"
)

type textChapterRow struct {
	ID         int       `db:"id" goqu:"skipinsert"`
	Title      string    `db:"title"` // TODO: make db schema (and gql schema) nullable
	ImageIndex int       `db:"image_index"`
	TextID     int       `db:"text_id"`
	CreatedAt  Timestamp `db:"created_at"`
	UpdatedAt  Timestamp `db:"updated_at"`
}

func (r *textChapterRow) fromTextChapter(o models.TextChapter) {
	r.ID = o.ID
	r.Title = o.Title
	r.ImageIndex = o.ImageIndex
	r.TextID = o.TextID
	r.CreatedAt = Timestamp{Timestamp: o.CreatedAt}
	r.UpdatedAt = Timestamp{Timestamp: o.UpdatedAt}
}

func (r *textChapterRow) resolve() *models.TextChapter {
	ret := &models.TextChapter{
		ID:         r.ID,
		Title:      r.Title,
		ImageIndex: r.ImageIndex,
		TextID:     r.TextID,
		CreatedAt:  r.CreatedAt.Timestamp,
		UpdatedAt:  r.UpdatedAt.Timestamp,
	}

	return ret
}

type textChapterRowRecord struct {
	updateRecord
}

func (r *textChapterRowRecord) fromPartial(o models.TextChapterPartial) {
	// TODO: replace with setNullString after schema is made nullable
	// r.setNullString("title", o.Title)
	// saves a null input as the empty string
	if o.Title.Set {
		r.set("title", o.Title.Value)
	}
	r.setInt("image_index", o.ImageIndex)
	r.setInt("text_id", o.TextID)
	r.setTimestamp("created_at", o.CreatedAt)
	r.setTimestamp("updated_at", o.UpdatedAt)
}

type TextChapterStore struct {
	repository

	tableMgr *table
}

func NewTextChapterStore() *TextChapterStore {
	return &TextChapterStore{
		repository: repository{
			tableName: textsChaptersTable,
			idColumn:  idColumn,
		},
		tableMgr: textsChaptersTableMgr,
	}
}

func (qb *TextChapterStore) table() exp.IdentifierExpression {
	return qb.tableMgr.table
}

func (qb *TextChapterStore) selectDataset() *goqu.SelectDataset {
	return dialect.From(qb.table()).Select(qb.table().All())
}

func (qb *TextChapterStore) Create(ctx context.Context, newObject *models.TextChapter) error {
	var r textChapterRow
	r.fromTextChapter(*newObject)

	id, err := qb.tableMgr.insertID(ctx, r)
	if err != nil {
		return err
	}

	updated, err := qb.find(ctx, id)
	if err != nil {
		return fmt.Errorf("finding after create: %w", err)
	}

	*newObject = *updated

	return nil
}

func (qb *TextChapterStore) Update(ctx context.Context, updatedObject *models.TextChapter) error {
	var r textChapterRow
	r.fromTextChapter(*updatedObject)

	if err := qb.tableMgr.updateByID(ctx, updatedObject.ID, r); err != nil {
		return err
	}

	return nil
}

func (qb *TextChapterStore) UpdatePartial(ctx context.Context, id int, partial models.TextChapterPartial) (*models.TextChapter, error) {
	r := textChapterRowRecord{
		updateRecord{
			Record: make(exp.Record),
		},
	}

	r.fromPartial(partial)

	if len(r.Record) > 0 {
		if err := qb.tableMgr.updateByID(ctx, id, r.Record); err != nil {
			return nil, err
		}
	}

	return qb.find(ctx, id)
}

func (qb *TextChapterStore) Destroy(ctx context.Context, id int) error {
	return qb.destroyExisting(ctx, []int{id})
}

// returns nil, nil if not found
func (qb *TextChapterStore) Find(ctx context.Context, id int) (*models.TextChapter, error) {
	ret, err := qb.find(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return ret, err
}

func (qb *TextChapterStore) FindMany(ctx context.Context, ids []int) ([]*models.TextChapter, error) {
	ret := make([]*models.TextChapter, len(ids))

	table := qb.table()
	q := qb.selectDataset().Prepared(true).Where(table.Col(idColumn).In(ids))
	unsorted, err := qb.getMany(ctx, q)
	if err != nil {
		return nil, err
	}

	for _, s := range unsorted {
		i := sliceutil.Index(ids, s.ID)
		ret[i] = s
	}

	for i := range ret {
		if ret[i] == nil {
			return nil, fmt.Errorf("text chapter with id %d not found", ids[i])
		}
	}

	return ret, nil
}

// returns nil, sql.ErrNoRows if not found
func (qb *TextChapterStore) find(ctx context.Context, id int) (*models.TextChapter, error) {
	q := qb.selectDataset().Where(qb.tableMgr.byID(id))

	ret, err := qb.get(ctx, q)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

// returns nil, sql.ErrNoRows if not found
func (qb *TextChapterStore) get(ctx context.Context, q *goqu.SelectDataset) (*models.TextChapter, error) {
	ret, err := qb.getMany(ctx, q)
	if err != nil {
		return nil, err
	}

	if len(ret) == 0 {
		return nil, sql.ErrNoRows
	}

	return ret[0], nil
}

func (qb *TextChapterStore) getMany(ctx context.Context, q *goqu.SelectDataset) ([]*models.TextChapter, error) {
	const single = false
	var ret []*models.TextChapter
	if err := queryFunc(ctx, q, single, func(r *sqlx.Rows) error {
		var f textChapterRow
		if err := r.StructScan(&f); err != nil {
			return err
		}

		s := f.resolve()

		ret = append(ret, s)
		return nil
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (qb *TextChapterStore) FindByTextID(ctx context.Context, textID int) ([]*models.TextChapter, error) {
	query := `
		SELECT texts_chapters.* FROM texts_chapters
		WHERE texts_chapters.text_id = ?
		GROUP BY texts_chapters.id
		ORDER BY texts_chapters.image_index ASC
	`
	args := []interface{}{textID}
	return qb.queryTextChapters(ctx, query, args)
}

func (qb *TextChapterStore) queryTextChapters(ctx context.Context, query string, args []interface{}) ([]*models.TextChapter, error) {
	const single = false
	var ret []*models.TextChapter
	if err := qb.queryFunc(ctx, query, args, single, func(r *sqlx.Rows) error {
		var f textChapterRow
		if err := r.StructScan(&f); err != nil {
			return err
		}

		s := f.resolve()

		ret = append(ret, s)
		return nil
	}); err != nil {
		return nil, err
	}

	return ret, nil
}
