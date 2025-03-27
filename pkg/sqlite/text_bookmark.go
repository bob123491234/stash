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

const textBookmarkTable = "text_bookmarks"

const countTextBookmarksForTagQuery = `
SELECT text_bookmarks.id FROM text_bookmarks
LEFT JOIN text_bookmarks_tags as tags_join on tags_join.text_bookmark_id = text_bookmarks.id
WHERE tags_join.tag_id = ? OR text_bookmarks.primary_tag_id = ?
GROUP BY text_bookmarks.id
`

type textBookmarkRow struct {
	ID           int       `db:"id" goqu:"skipinsert"`
	Title        string    `db:"title"` // TODO: make db schema (and gql schema) nullable
	Seconds      float64   `db:"seconds"`
	PrimaryTagID int       `db:"primary_tag_id"`
	TextID       int       `db:"text_id"`
	CreatedAt    Timestamp `db:"created_at"`
	UpdatedAt    Timestamp `db:"updated_at"`
}

func (r *textBookmarkRow) fromTextBookmark(o models.TextBookmark) {
	r.ID = o.ID
	r.Title = o.Title
	r.Seconds = o.Seconds
	r.PrimaryTagID = o.PrimaryTagID
	r.TextID = o.TextID
	r.CreatedAt = Timestamp{Timestamp: o.CreatedAt}
	r.UpdatedAt = Timestamp{Timestamp: o.UpdatedAt}
}

func (r *textBookmarkRow) resolve() *models.TextBookmark {
	ret := &models.TextBookmark{
		ID:           r.ID,
		Title:        r.Title,
		Seconds:      r.Seconds,
		PrimaryTagID: r.PrimaryTagID,
		TextID:       r.TextID,
		CreatedAt:    r.CreatedAt.Timestamp,
		UpdatedAt:    r.UpdatedAt.Timestamp,
	}

	return ret
}

type textBookmarkRowRecord struct {
	updateRecord
}

func (r *textBookmarkRowRecord) fromPartial(o models.TextBookmarkPartial) {
	// TODO: replace with setNullString after schema is made nullable
	// r.setNullString("title", o.Title)
	// saves a null input as the empty string
	if o.Title.Set {
		r.set("title", o.Title.Value)
	}
	r.setFloat64("seconds", o.Seconds)
	r.setInt("primary_tag_id", o.PrimaryTagID)
	r.setInt("text_id", o.TextID)
	r.setTimestamp("created_at", o.CreatedAt)
	r.setTimestamp("updated_at", o.UpdatedAt)
}

type textBookmarkRepositoryType struct {
	repository

	texts repository
	tags  joinRepository
}

var (
	textBookmarkRepository = textBookmarkRepositoryType{
		repository: repository{
			tableName: textBookmarkTable,
			idColumn:  idColumn,
		},
		texts: repository{
			tableName: textTable,
			idColumn:  idColumn,
		},
		tags: joinRepository{
			repository: repository{
				tableName: "text_bookmarks_tags",
				idColumn:  "text_bookmark_id",
			},
			fkColumn: tagIDColumn,
		},
	}
)

type TextBookmarkStore struct{}

func NewTextBookmarkStore() *TextBookmarkStore {
	return &TextBookmarkStore{}
}

func (qb *TextBookmarkStore) table() exp.IdentifierExpression {
	return textBookmarkTableMgr.table
}

func (qb *TextBookmarkStore) selectDataset() *goqu.SelectDataset {
	return dialect.From(qb.table()).Select(qb.table().All())
}

func (qb *TextBookmarkStore) Create(ctx context.Context, newObject *models.TextBookmark) error {
	var r textBookmarkRow
	r.fromTextBookmark(*newObject)

	id, err := textBookmarkTableMgr.insertID(ctx, r)
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

func (qb *TextBookmarkStore) UpdatePartial(ctx context.Context, id int, partial models.TextBookmarkPartial) (*models.TextBookmark, error) {
	r := textBookmarkRowRecord{
		updateRecord{
			Record: make(exp.Record),
		},
	}

	r.fromPartial(partial)

	if len(r.Record) > 0 {
		if err := textBookmarkTableMgr.updateByID(ctx, id, r.Record); err != nil {
			return nil, err
		}
	}

	return qb.find(ctx, id)
}

func (qb *TextBookmarkStore) Update(ctx context.Context, updatedObject *models.TextBookmark) error {
	var r textBookmarkRow
	r.fromTextBookmark(*updatedObject)

	if err := textBookmarkTableMgr.updateByID(ctx, updatedObject.ID, r); err != nil {
		return err
	}

	return nil
}

func (qb *TextBookmarkStore) Destroy(ctx context.Context, id int) error {
	return textBookmarkRepository.destroyExisting(ctx, []int{id})
}

// returns nil, nil if not found
func (qb *TextBookmarkStore) Find(ctx context.Context, id int) (*models.TextBookmark, error) {
	ret, err := qb.find(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return ret, err
}

func (qb *TextBookmarkStore) FindMany(ctx context.Context, ids []int) ([]*models.TextBookmark, error) {
	ret := make([]*models.TextBookmark, len(ids))

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
			return nil, fmt.Errorf("text bookmark with id %d not found", ids[i])
		}
	}

	return ret, nil
}

// returns nil, sql.ErrNoRows if not found
func (qb *TextBookmarkStore) find(ctx context.Context, id int) (*models.TextBookmark, error) {
	q := qb.selectDataset().Where(textBookmarkTableMgr.byID(id))

	ret, err := qb.get(ctx, q)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

// returns nil, sql.ErrNoRows if not found
func (qb *TextBookmarkStore) get(ctx context.Context, q *goqu.SelectDataset) (*models.TextBookmark, error) {
	ret, err := qb.getMany(ctx, q)
	if err != nil {
		return nil, err
	}

	if len(ret) == 0 {
		return nil, sql.ErrNoRows
	}

	return ret[0], nil
}

func (qb *TextBookmarkStore) getMany(ctx context.Context, q *goqu.SelectDataset) ([]*models.TextBookmark, error) {
	const single = false
	var ret []*models.TextBookmark
	if err := queryFunc(ctx, q, single, func(r *sqlx.Rows) error {
		var f textBookmarkRow
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

func (qb *TextBookmarkStore) FindByTextID(ctx context.Context, textID int) ([]*models.TextBookmark, error) {
	query := `
		SELECT text_bookmarks.* FROM text_bookmarks
		WHERE text_bookmarks.text_id = ?
		GROUP BY text_bookmarks.id
		ORDER BY text_bookmarks.seconds ASC
	`
	args := []interface{}{textID}
	return qb.queryTextBookmarks(ctx, query, args)
}

func (qb *TextBookmarkStore) CountByTagID(ctx context.Context, tagID int) (int, error) {
	args := []interface{}{tagID, tagID}
	return textBookmarkRepository.runCountQuery(ctx, textBookmarkRepository.buildCountQuery(countTextBookmarksForTagQuery), args)
}

func (qb *TextBookmarkStore) GetBookmarkStrings(ctx context.Context, q *string, sort *string) ([]*models.BookmarkStringsResultType, error) {
	query := "SELECT count(*) as `count`, text_bookmarks.id as id, text_bookmarks.title as title FROM text_bookmarks"
	if q != nil {
		query += " WHERE title LIKE '%" + *q + "%'"
	}
	query += " GROUP BY title"
	if sort != nil && *sort == "count" {
		query += " ORDER BY `count` DESC"
	} else {
		query += " ORDER BY title ASC"
	}
	var args []interface{}
	return qb.queryBookmarkStringsResultType(ctx, query, args)
}

func (qb *TextBookmarkStore) Wall(ctx context.Context, q *string) ([]*models.TextBookmark, error) {
	s := ""
	if q != nil {
		s = *q
	}

	table := qb.table()
	qq := qb.selectDataset().Prepared(true).Where(table.Col("title").Like("%" + s + "%")).Order(goqu.L("RANDOM()").Asc()).Limit(80)
	return qb.getMany(ctx, qq)
}

func (qb *TextBookmarkStore) makeQuery(ctx context.Context, textBookmarkFilter *models.TextBookmarkFilterType, findFilter *models.FindFilterType) (*queryBuilder, error) {
	if textBookmarkFilter == nil {
		textBookmarkFilter = &models.TextBookmarkFilterType{}
	}
	if findFilter == nil {
		findFilter = &models.FindFilterType{}
	}

	query := textBookmarkRepository.newQuery()
	distinctIDs(&query, textBookmarkTable)

	if q := findFilter.Q; q != nil && *q != "" {
		query.join(textTable, "", "texts.id = text_bookmarks.text_id")
		query.join(tagTable, "", "text_bookmarks.primary_tag_id = tags.id")
		searchColumns := []string{"text_bookmarks.title", "texts.title", "tags.name"}
		query.parseQueryString(searchColumns, *q)
	}

	filter := filterBuilderFromHandler(ctx, &textBookmarkFilterHandler{
		textBookmarkFilter: textBookmarkFilter,
	})

	if err := query.addFilter(filter); err != nil {
		return nil, err
	}

	if err := qb.setTextBookmarkSort(&query, findFilter); err != nil {
		return nil, err
	}
	query.sortAndPagination += getPagination(findFilter)

	return &query, nil
}

func (qb *TextBookmarkStore) Query(ctx context.Context, textBookmarkFilter *models.TextBookmarkFilterType, findFilter *models.FindFilterType) ([]*models.TextBookmark, int, error) {
	query, err := qb.makeQuery(ctx, textBookmarkFilter, findFilter)
	if err != nil {
		return nil, 0, err
	}

	idsResult, countResult, err := query.executeFind(ctx)
	if err != nil {
		return nil, 0, err
	}

	textBookmarks, err := qb.FindMany(ctx, idsResult)
	if err != nil {
		return nil, 0, err
	}

	return textBookmarks, countResult, nil
}

func (qb *TextBookmarkStore) QueryCount(ctx context.Context, textBookmarkFilter *models.TextBookmarkFilterType, findFilter *models.FindFilterType) (int, error) {
	query, err := qb.makeQuery(ctx, textBookmarkFilter, findFilter)
	if err != nil {
		return 0, err
	}

	return query.executeCount(ctx)
}

var textBookmarkSortOptions = sortOptions{
	"created_at",
	"id",
	"title",
	"random",
	"text_id",
	"texts_updated_at",
	"seconds",
	"updated_at",
}

func (qb *TextBookmarkStore) setTextBookmarkSort(query *queryBuilder, findFilter *models.FindFilterType) error {
	sort := findFilter.GetSort("title")
	direction := findFilter.GetDirection()

	// CVE-2024-32231 - ensure sort is in the list of allowed sorts
	if err := textBookmarkSortOptions.validateSort(sort); err != nil {
		return err
	}

	switch sort {
	case "texts_updated_at":
		sort = "updated_at"
		query.join(textTable, "", "texts.id = text_bookmarks.text_id")
		query.sortAndPagination += getSort(sort, direction, textTable)
	case "title":
		query.join(tagTable, "", "text_bookmarks.primary_tag_id = tags.id")
		query.sortAndPagination += " ORDER BY COALESCE(NULLIF(text_bookmarks.title,''), tags.name) COLLATE NATURAL_CI " + direction
	default:
		query.sortAndPagination += getSort(sort, direction, textBookmarkTable)
	}

	query.sortAndPagination += ", text_bookmarks.text_id ASC, text_bookmarks.seconds ASC"
	return nil
}

func (qb *TextBookmarkStore) queryTextBookmarks(ctx context.Context, query string, args []interface{}) ([]*models.TextBookmark, error) {
	const single = false
	var ret []*models.TextBookmark
	if err := textBookmarkRepository.queryFunc(ctx, query, args, single, func(r *sqlx.Rows) error {
		var f textBookmarkRow
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

func (qb *TextBookmarkStore) queryBookmarkStringsResultType(ctx context.Context, query string, args []interface{}) ([]*models.BookmarkStringsResultType, error) {
	rows, err := dbWrapper.Queryx(ctx, query, args...)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	defer rows.Close()

	bookmarkStrings := make([]*models.BookmarkStringsResultType, 0)
	for rows.Next() {
		bookmarkString := models.BookmarkStringsResultType{}
		if err := rows.StructScan(&bookmarkString); err != nil {
			return nil, err
		}
		bookmarkStrings = append(bookmarkStrings, &bookmarkString)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return bookmarkStrings, nil
}

func (qb *TextBookmarkStore) GetTagIDs(ctx context.Context, id int) ([]int, error) {
	return textBookmarkRepository.tags.getIDs(ctx, id)
}

func (qb *TextBookmarkStore) UpdateTags(ctx context.Context, id int, tagIDs []int) error {
	// Delete the existing joins and then create new ones
	return textBookmarkRepository.tags.replace(ctx, id, tagIDs)
}

func (qb *TextBookmarkStore) Count(ctx context.Context) (int, error) {
	q := dialect.Select(goqu.COUNT("*")).From(qb.table())
	return count(ctx, q)
}

func (qb *TextBookmarkStore) All(ctx context.Context) ([]*models.TextBookmark, error) {
	return qb.getMany(ctx, qb.selectDataset())
}
