package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/jmoiron/sqlx"
	"gopkg.in/guregu/null.v4"
	"gopkg.in/guregu/null.v4/zero"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/sliceutil"
	"github.com/stashapp/stash/pkg/utils"
)

const (
	textTable            = "texts"
	textsFilesTable      = "texts_files"
	textIDColumn         = "text_id"
	performersTextsTable = "performers_texts"
	textsTagsTable       = "texts_tags"
	textsURLsTable       = "text_urls"
	textURLColumn        = "url"
	textsReadDatesTable  = "texts_read_dates"
	textReadDateColumn   = "read_date"
	textsODatesTable     = "texts_o_dates"
	textODateColumn      = "o_date"

	textCoverBlobColumn = "cover_blob"
)

type textRow struct {
	ID       int         `db:"id" goqu:"skipinsert"`
	Title    zero.String `db:"title"`
	Code     zero.String `db:"code"`
	Details  zero.String `db:"details"`
	Director zero.String `db:"director"`
	Date     NullDate    `db:"date"`
	// expressed as 1-100
	Rating       null.Int  `db:"rating"`
	Organized    bool      `db:"organized"`
	StudioID     null.Int  `db:"studio_id,omitempty"`
	CreatedAt    Timestamp `db:"created_at"`
	UpdatedAt    Timestamp `db:"updated_at"`
	ResumeTime   float64   `db:"resume_time"`
	ReadDuration float64   `db:"read_duration"`

	// not used in resolutions or updates
	CoverBlob zero.String `db:"cover_blob"`
}

func (r *textRow) fromText(o models.Text) {
	r.ID = o.ID
	r.Title = zero.StringFrom(o.Title)
	r.Code = zero.StringFrom(o.Code)
	r.Details = zero.StringFrom(o.Details)
	r.Director = zero.StringFrom(o.Director)
	r.Date = NullDateFromDatePtr(o.Date)
	r.Rating = intFromPtr(o.Rating)
	r.Organized = o.Organized
	r.StudioID = intFromPtr(o.StudioID)
	r.CreatedAt = Timestamp{Timestamp: o.CreatedAt}
	r.UpdatedAt = Timestamp{Timestamp: o.UpdatedAt}
	r.ResumeTime = o.ResumeTime
	r.ReadDuration = o.ReadDuration
}

type textQueryRow struct {
	textRow
	PrimaryFileID         null.Int    `db:"primary_file_id"`
	PrimaryFileFolderPath zero.String `db:"primary_file_folder_path"`
	PrimaryFileBasename   zero.String `db:"primary_file_basename"`
	PrimaryFileChecksum   zero.String `db:"primary_file_checksum"`
}

func (r *textQueryRow) resolve() *models.Text {
	ret := &models.Text{
		ID:        r.ID,
		Title:     r.Title.String,
		Code:      r.Code.String,
		Details:   r.Details.String,
		Director:  r.Director.String,
		Date:      r.Date.DatePtr(),
		Rating:    nullIntPtr(r.Rating),
		Organized: r.Organized,
		StudioID:  nullIntPtr(r.StudioID),

		PrimaryFileID: nullIntFileIDPtr(r.PrimaryFileID),
		OSHash:        r.PrimaryFileOshash.String,
		Checksum:      r.PrimaryFileChecksum.String,

		CreatedAt: r.CreatedAt.Timestamp,
		UpdatedAt: r.UpdatedAt.Timestamp,

		ResumeTime:   r.ResumeTime,
		ReadDuration: r.ReadDuration,
	}

	if r.PrimaryFileFolderPath.Valid && r.PrimaryFileBasename.Valid {
		ret.Path = filepath.Join(r.PrimaryFileFolderPath.String, r.PrimaryFileBasename.String)
	}

	return ret
}

type textRowRecord struct {
	updateRecord
}

func (r *textRowRecord) fromPartial(o models.TextPartial) {
	r.setNullString("title", o.Title)
	r.setNullString("code", o.Code)
	r.setNullString("details", o.Details)
	r.setNullString("director", o.Director)
	r.setNullDate("date", o.Date)
	r.setNullInt("rating", o.Rating)
	r.setBool("organized", o.Organized)
	r.setNullInt("studio_id", o.StudioID)
	r.setTimestamp("created_at", o.CreatedAt)
	r.setTimestamp("updated_at", o.UpdatedAt)
	r.setFloat64("resume_time", o.ResumeTime)
	r.setFloat64("read_duration", o.ReadDuration)
}

type textRepositoryType struct {
	repository
	galleries  joinRepository
	tags       joinRepository
	performers joinRepository

	files filesRepository

	stashIDs stashIDRepository
}

var (
	textRepository = textRepositoryType{
		repository: repository{
			tableName: textTable,
			idColumn:  idColumn,
		},
		galleries: joinRepository{
			repository: repository{
				tableName: textsGalleriesTable,
				idColumn:  textIDColumn,
			},
			fkColumn: galleryIDColumn,
		},
		tags: joinRepository{
			repository: repository{
				tableName: textsTagsTable,
				idColumn:  textIDColumn,
			},
			fkColumn:     tagIDColumn,
			foreignTable: tagTable,
			orderBy:      "tags.name ASC",
		},
		performers: joinRepository{
			repository: repository{
				tableName: performersTextsTable,
				idColumn:  textIDColumn,
			},
			fkColumn: performerIDColumn,
		},
		files: filesRepository{
			repository: repository{
				tableName: textsFilesTable,
				idColumn:  textIDColumn,
			},
		},
		stashIDs: stashIDRepository{
			repository{
				tableName: "text_stash_ids",
				idColumn:  textIDColumn,
			},
		},
	}
)

type TextStore struct {
	blobJoinQueryBuilder

	tableMgr *table
	oDateManager
	readDateManager

	repo *storeRepository
}

func NewTextStore(r *storeRepository, blobStore *BlobStore) *TextStore {
	return &TextStore{
		blobJoinQueryBuilder: blobJoinQueryBuilder{
			blobStore: blobStore,
			joinTable: textTable,
		},

		tableMgr:        textTableMgr,
		readDateManager: readDateManager{textsReadTableMgr},
		oDateManager:    oDateManager{textsOTableMgr},
		repo:            r,
	}
}

func (qb *TextStore) table() exp.IdentifierExpression {
	return qb.tableMgr.table
}

func (qb *TextStore) selectDataset() *goqu.SelectDataset {
	table := qb.table()
	files := fileTableMgr.table
	folders := folderTableMgr.table
	checksum := fingerprintTableMgr.table.As("fingerprint_md5")
	oshash := fingerprintTableMgr.table.As("fingerprint_oshash")

	return dialect.From(table).LeftJoin(
		textsFilesJoinTable,
		goqu.On(
			textsFilesJoinTable.Col(textIDColumn).Eq(table.Col(idColumn)),
			textsFilesJoinTable.Col("primary").Eq(1),
		),
	).LeftJoin(
		files,
		goqu.On(files.Col(idColumn).Eq(textsFilesJoinTable.Col(fileIDColumn))),
	).LeftJoin(
		folders,
		goqu.On(folders.Col(idColumn).Eq(files.Col("parent_folder_id"))),
	).LeftJoin(
		checksum,
		goqu.On(
			checksum.Col(fileIDColumn).Eq(textsFilesJoinTable.Col(fileIDColumn)),
			checksum.Col("type").Eq(models.FingerprintTypeMD5),
		),
	).LeftJoin(
		oshash,
		goqu.On(
			oshash.Col(fileIDColumn).Eq(textsFilesJoinTable.Col(fileIDColumn)),
			oshash.Col("type").Eq(models.FingerprintTypeOshash),
		),
	).Select(
		qb.table().All(),
		textsFilesJoinTable.Col(fileIDColumn).As("primary_file_id"),
		folders.Col("path").As("primary_file_folder_path"),
		files.Col("basename").As("primary_file_basename"),
		checksum.Col("fingerprint").As("primary_file_checksum"),
		oshash.Col("fingerprint").As("primary_file_oshash"),
	)
}

func (qb *TextStore) Create(ctx context.Context, newObject *models.Text, fileIDs []models.FileID) error {
	var r textRow
	r.fromText(*newObject)

	id, err := qb.tableMgr.insertID(ctx, r)
	if err != nil {
		return err
	}

	if len(fileIDs) > 0 {
		const firstPrimary = true
		if err := textsFilesTableMgr.insertJoins(ctx, id, firstPrimary, fileIDs); err != nil {
			return err
		}
	}

	if newObject.URLs.Loaded() {
		const startPos = 0
		if err := textsURLsTableMgr.insertJoins(ctx, id, startPos, newObject.URLs.List()); err != nil {
			return err
		}
	}

	if newObject.PerformerIDs.Loaded() {
		if err := textsPerformersTableMgr.insertJoins(ctx, id, newObject.PerformerIDs.List()); err != nil {
			return err
		}
	}
	if newObject.TagIDs.Loaded() {
		if err := textsTagsTableMgr.insertJoins(ctx, id, newObject.TagIDs.List()); err != nil {
			return err
		}
	}

	if newObject.GalleryIDs.Loaded() {
		if err := textsGalleriesTableMgr.insertJoins(ctx, id, newObject.GalleryIDs.List()); err != nil {
			return err
		}
	}

	updated, err := qb.find(ctx, id)
	if err != nil {
		return fmt.Errorf("finding after create: %w", err)
	}

	*newObject = *updated

	return nil
}

func (qb *TextStore) UpdatePartial(ctx context.Context, id int, partial models.TextPartial) (*models.Text, error) {
	r := textRowRecord{
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

	if partial.URLs != nil {
		if err := textsURLsTableMgr.modifyJoins(ctx, id, partial.URLs.Values, partial.URLs.Mode); err != nil {
			return nil, err
		}
	}
	if partial.PerformerIDs != nil {
		if err := textsPerformersTableMgr.modifyJoins(ctx, id, partial.PerformerIDs.IDs, partial.PerformerIDs.Mode); err != nil {
			return nil, err
		}
	}
	if partial.TagIDs != nil {
		if err := textsTagsTableMgr.modifyJoins(ctx, id, partial.TagIDs.IDs, partial.TagIDs.Mode); err != nil {
			return nil, err
		}
	}
	if partial.GalleryIDs != nil {
		if err := textsGalleriesTableMgr.modifyJoins(ctx, id, partial.GalleryIDs.IDs, partial.GalleryIDs.Mode); err != nil {
			return nil, err
		}
	}
	if partial.PrimaryFileID != nil {
		if err := textsFilesTableMgr.setPrimary(ctx, id, *partial.PrimaryFileID); err != nil {
			return nil, err
		}
	}

	return qb.find(ctx, id)
}

func (qb *TextStore) Update(ctx context.Context, updatedObject *models.Text) error {
	var r textRow
	r.fromText(*updatedObject)

	if err := qb.tableMgr.updateByID(ctx, updatedObject.ID, r); err != nil {
		return err
	}

	if updatedObject.URLs.Loaded() {
		if err := textsURLsTableMgr.replaceJoins(ctx, updatedObject.ID, updatedObject.URLs.List()); err != nil {
			return err
		}
	}

	if updatedObject.PerformerIDs.Loaded() {
		if err := textsPerformersTableMgr.replaceJoins(ctx, updatedObject.ID, updatedObject.PerformerIDs.List()); err != nil {
			return err
		}
	}

	if updatedObject.TagIDs.Loaded() {
		if err := textsTagsTableMgr.replaceJoins(ctx, updatedObject.ID, updatedObject.TagIDs.List()); err != nil {
			return err
		}
	}

	if updatedObject.GalleryIDs.Loaded() {
		if err := textsGalleriesTableMgr.replaceJoins(ctx, updatedObject.ID, updatedObject.GalleryIDs.List()); err != nil {
			return err
		}
	}

	if updatedObject.Files.Loaded() {
		fileIDs := make([]models.FileID, len(updatedObject.Files.List()))
		for i, f := range updatedObject.Files.List() {
			fileIDs[i] = f.ID
		}

		if err := textsFilesTableMgr.replaceJoins(ctx, updatedObject.ID, fileIDs); err != nil {
			return err
		}
	}

	return nil
}

func (qb *TextStore) Destroy(ctx context.Context, id int) error {
	// must handle image checksums manually
	if err := qb.destroyCover(ctx, id); err != nil {
		return err
	}

	// text bookmarks should be handled prior to calling destroy
	// galleries should be handled prior to calling destroy

	return qb.tableMgr.destroyExisting(ctx, []int{id})
}

// returns nil, nil if not found
func (qb *TextStore) Find(ctx context.Context, id int) (*models.Text, error) {
	ret, err := qb.find(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return ret, err
}

func (qb *TextStore) FindMany(ctx context.Context, ids []int) ([]*models.Text, error) {
	texts := make([]*models.Text, len(ids))

	table := qb.table()
	if err := batchExec(ids, defaultBatchSize, func(batch []int) error {
		q := qb.selectDataset().Prepared(true).Where(table.Col(idColumn).In(batch))
		unsorted, err := qb.getMany(ctx, q)
		if err != nil {
			return err
		}

		for _, s := range unsorted {
			i := sliceutil.Index(ids, s.ID)
			texts[i] = s
		}

		return nil
	}); err != nil {
		return nil, err
	}

	for i := range texts {
		if texts[i] == nil {
			return nil, fmt.Errorf("text with id %d not found", ids[i])
		}
	}

	return texts, nil
}

// returns nil, sql.ErrNoRows if not found
func (qb *TextStore) find(ctx context.Context, id int) (*models.Text, error) {
	q := qb.selectDataset().Where(qb.tableMgr.byID(id))

	ret, err := qb.get(ctx, q)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (qb *TextStore) findBySubquery(ctx context.Context, sq *goqu.SelectDataset) ([]*models.Text, error) {
	table := qb.table()

	q := qb.selectDataset().Where(
		table.Col(idColumn).Eq(
			sq,
		),
	)

	return qb.getMany(ctx, q)
}

// returns nil, sql.ErrNoRows if not found
func (qb *TextStore) get(ctx context.Context, q *goqu.SelectDataset) (*models.Text, error) {
	ret, err := qb.getMany(ctx, q)
	if err != nil {
		return nil, err
	}

	if len(ret) == 0 {
		return nil, sql.ErrNoRows
	}

	return ret[0], nil
}

func (qb *TextStore) getMany(ctx context.Context, q *goqu.SelectDataset) ([]*models.Text, error) {
	const single = false
	var ret []*models.Text
	var lastID int
	if err := queryFunc(ctx, q, single, func(r *sqlx.Rows) error {
		var f textQueryRow
		if err := r.StructScan(&f); err != nil {
			return err
		}

		s := f.resolve()
		if s.ID == lastID {
			return fmt.Errorf("internal error: multiple rows returned for single text id %d", s.ID)
		}
		lastID = s.ID

		ret = append(ret, s)
		return nil
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (qb *TextStore) GetFiles(ctx context.Context, id int) ([]*models.VideoFile, error) {
	fileIDs, err := textRepository.files.get(ctx, id)
	if err != nil {
		return nil, err
	}

	// use fileStore to load files
	files, err := qb.repo.File.Find(ctx, fileIDs...)
	if err != nil {
		return nil, err
	}

	ret := make([]*models.VideoFile, len(files))
	for i, f := range files {
		var ok bool
		ret[i], ok = f.(*models.VideoFile)
		if !ok {
			return nil, fmt.Errorf("expected file to be *file.VideoFile not %T", f)
		}
	}

	return ret, nil
}

func (qb *TextStore) GetManyFileIDs(ctx context.Context, ids []int) ([][]models.FileID, error) {
	const primaryOnly = false
	return textRepository.files.getMany(ctx, ids, primaryOnly)
}

func (qb *TextStore) FindByFileID(ctx context.Context, fileID models.FileID) ([]*models.Text, error) {
	sq := dialect.From(textsFilesJoinTable).Select(textsFilesJoinTable.Col(textIDColumn)).Where(
		textsFilesJoinTable.Col(fileIDColumn).Eq(fileID),
	)

	ret, err := qb.findBySubquery(ctx, sq)
	if err != nil {
		return nil, fmt.Errorf("getting texts by file id %d: %w", fileID, err)
	}

	return ret, nil
}

func (qb *TextStore) FindByPrimaryFileID(ctx context.Context, fileID models.FileID) ([]*models.Text, error) {
	sq := dialect.From(textsFilesJoinTable).Select(textsFilesJoinTable.Col(textIDColumn)).Where(
		textsFilesJoinTable.Col(fileIDColumn).Eq(fileID),
		textsFilesJoinTable.Col("primary").Eq(1),
	)

	ret, err := qb.findBySubquery(ctx, sq)
	if err != nil {
		return nil, fmt.Errorf("getting texts by primary file id %d: %w", fileID, err)
	}

	return ret, nil
}

func (qb *TextStore) CountByFileID(ctx context.Context, fileID models.FileID) (int, error) {
	joinTable := textsFilesJoinTable

	q := dialect.Select(goqu.COUNT("*")).From(joinTable).Where(joinTable.Col(fileIDColumn).Eq(fileID))
	return count(ctx, q)
}

func (qb *TextStore) FindByFingerprints(ctx context.Context, fp []models.Fingerprint) ([]*models.Text, error) {
	fingerprintTable := fingerprintTableMgr.table

	var ex []exp.Expression

	for _, v := range fp {
		ex = append(ex, goqu.And(
			fingerprintTable.Col("type").Eq(v.Type),
			fingerprintTable.Col("fingerprint").Eq(v.Fingerprint),
		))
	}

	sq := dialect.From(textsFilesJoinTable).
		InnerJoin(
			fingerprintTable,
			goqu.On(fingerprintTable.Col(fileIDColumn).Eq(textsFilesJoinTable.Col(fileIDColumn))),
		).
		Select(textsFilesJoinTable.Col(textIDColumn)).Where(goqu.Or(ex...))

	ret, err := qb.findBySubquery(ctx, sq)
	if err != nil {
		return nil, fmt.Errorf("getting texts by fingerprints: %w", err)
	}

	return ret, nil
}

func (qb *TextStore) FindByChecksum(ctx context.Context, checksum string) ([]*models.Text, error) {
	return qb.FindByFingerprints(ctx, []models.Fingerprint{
		{
			Type:        models.FingerprintTypeMD5,
			Fingerprint: checksum,
		},
	})
}

func (qb *TextStore) FindByOSHash(ctx context.Context, oshash string) ([]*models.Text, error) {
	return qb.FindByFingerprints(ctx, []models.Fingerprint{
		{
			Type:        models.FingerprintTypeOshash,
			Fingerprint: oshash,
		},
	})
}

func (qb *TextStore) FindByPath(ctx context.Context, p string) ([]*models.Text, error) {
	filesTable := fileTableMgr.table
	foldersTable := folderTableMgr.table
	basename := filepath.Base(p)
	dir := filepath.Dir(p)

	// replace wildcards
	basename = strings.ReplaceAll(basename, "*", "%")
	dir = strings.ReplaceAll(dir, "*", "%")

	sq := dialect.From(textsFilesJoinTable).InnerJoin(
		filesTable,
		goqu.On(filesTable.Col(idColumn).Eq(textsFilesJoinTable.Col(fileIDColumn))),
	).InnerJoin(
		foldersTable,
		goqu.On(foldersTable.Col(idColumn).Eq(filesTable.Col("parent_folder_id"))),
	).Select(textsFilesJoinTable.Col(textIDColumn)).Where(
		foldersTable.Col("path").Like(dir),
		filesTable.Col("basename").Like(basename),
	)

	ret, err := qb.findBySubquery(ctx, sq)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("getting text by path %s: %w", p, err)
	}

	return ret, nil
}

func (qb *TextStore) FindByPerformerID(ctx context.Context, performerID int) ([]*models.Text, error) {
	sq := dialect.From(textsPerformersJoinTable).Select(textsPerformersJoinTable.Col(textIDColumn)).Where(
		textsPerformersJoinTable.Col(performerIDColumn).Eq(performerID),
	)
	ret, err := qb.findBySubquery(ctx, sq)

	if err != nil {
		return nil, fmt.Errorf("getting texts for performer %d: %w", performerID, err)
	}

	return ret, nil
}

func (qb *TextStore) FindByGalleryID(ctx context.Context, galleryID int) ([]*models.Text, error) {
	sq := dialect.From(galleriesTextsJoinTable).Select(galleriesTextsJoinTable.Col(textIDColumn)).Where(
		galleriesTextsJoinTable.Col(galleryIDColumn).Eq(galleryID),
	)
	ret, err := qb.findBySubquery(ctx, sq)

	if err != nil {
		return nil, fmt.Errorf("getting texts for gallery %d: %w", galleryID, err)
	}

	return ret, nil
}

func (qb *TextStore) CountByPerformerID(ctx context.Context, performerID int) (int, error) {
	joinTable := textsPerformersJoinTable

	q := dialect.Select(goqu.COUNT("*")).From(joinTable).Where(joinTable.Col(performerIDColumn).Eq(performerID))
	return count(ctx, q)
}

func (qb *TextStore) OCountByPerformerID(ctx context.Context, performerID int) (int, error) {
	table := qb.table()
	joinTable := textsPerformersJoinTable
	oHistoryTable := goqu.T(textsODatesTable)

	q := dialect.Select(goqu.COUNT("*")).From(table).InnerJoin(
		oHistoryTable,
		goqu.On(table.Col(idColumn).Eq(oHistoryTable.Col(textIDColumn))),
	).InnerJoin(
		joinTable,
		goqu.On(
			table.Col(idColumn).Eq(joinTable.Col(textIDColumn)),
		),
	).Where(joinTable.Col(performerIDColumn).Eq(performerID))

	var ret int
	if err := querySimple(ctx, q, &ret); err != nil {
		return 0, err
	}

	return ret, nil
}

func (qb *TextStore) Size(ctx context.Context) (float64, error) {
	table := qb.table()
	fileTable := fileTableMgr.table
	q := dialect.Select(
		goqu.COALESCE(goqu.SUM(fileTableMgr.table.Col("size")), 0),
	).From(table).InnerJoin(
		textsFilesJoinTable,
		goqu.On(table.Col(idColumn).Eq(textsFilesJoinTable.Col(textIDColumn))),
	).InnerJoin(
		fileTable,
		goqu.On(textsFilesJoinTable.Col(fileIDColumn).Eq(fileTable.Col(idColumn))),
	)
	var ret float64
	if err := querySimple(ctx, q, &ret); err != nil {
		return 0, err
	}

	return ret, nil
}

func (qb *TextStore) Duration(ctx context.Context) (float64, error) {
	table := qb.table()
	videoFileTable := videoFileTableMgr.table

	q := dialect.Select(
		goqu.COALESCE(goqu.SUM(videoFileTable.Col("duration")), 0),
	).From(table).InnerJoin(
		textsFilesJoinTable,
		goqu.On(textsFilesJoinTable.Col("text_id").Eq(table.Col(idColumn))),
	).InnerJoin(
		videoFileTable,
		goqu.On(videoFileTable.Col("file_id").Eq(textsFilesJoinTable.Col("file_id"))),
	)

	var ret float64
	if err := querySimple(ctx, q, &ret); err != nil {
		return 0, err
	}

	return ret, nil
}

func (qb *TextStore) ReadDuration(ctx context.Context) (float64, error) {
	table := qb.table()

	q := dialect.Select(goqu.COALESCE(goqu.SUM("read_duration"), 0)).From(table)

	var ret float64
	if err := querySimple(ctx, q, &ret); err != nil {
		return 0, err
	}

	return ret, nil
}

func (qb *TextStore) CountByStudioID(ctx context.Context, studioID int) (int, error) {
	table := qb.table()

	q := dialect.Select(goqu.COUNT("*")).From(table).Where(table.Col(studioIDColumn).Eq(studioID))
	return count(ctx, q)
}

func (qb *TextStore) CountByTagID(ctx context.Context, tagID int) (int, error) {
	joinTable := textsTagsJoinTable

	q := dialect.Select(goqu.COUNT("*")).From(joinTable).Where(joinTable.Col(tagIDColumn).Eq(tagID))
	return count(ctx, q)
}

func (qb *TextStore) countMissingFingerprints(ctx context.Context, fpType string) (int, error) {
	fpTable := fingerprintTableMgr.table.As("fingerprints_temp")

	q := dialect.From(textsFilesJoinTable).LeftJoin(
		fpTable,
		goqu.On(
			textsFilesJoinTable.Col(fileIDColumn).Eq(fpTable.Col(fileIDColumn)),
			fpTable.Col("type").Eq(fpType),
		),
	).Select(goqu.COUNT(goqu.DISTINCT(textsFilesJoinTable.Col(textIDColumn)))).Where(fpTable.Col("fingerprint").IsNull())

	return count(ctx, q)
}

// CountMissingChecksum returns the number of texts missing a checksum value.
func (qb *TextStore) CountMissingChecksum(ctx context.Context) (int, error) {
	return qb.countMissingFingerprints(ctx, "md5")
}

// CountMissingOSHash returns the number of texts missing an oshash value.
func (qb *TextStore) CountMissingOSHash(ctx context.Context) (int, error) {
	return qb.countMissingFingerprints(ctx, "oshash")
}

func (qb *TextStore) Wall(ctx context.Context, q *string) ([]*models.Text, error) {
	s := ""
	if q != nil {
		s = *q
	}

	table := qb.table()
	qq := qb.selectDataset().Prepared(true).Where(table.Col("details").Like("%" + s + "%")).Order(goqu.L("RANDOM()").Asc()).Limit(80)
	return qb.getMany(ctx, qq)
}

func (qb *TextStore) All(ctx context.Context) ([]*models.Text, error) {
	table := qb.table()
	fileTable := fileTableMgr.table
	folderTable := folderTableMgr.table

	return qb.getMany(ctx, qb.selectDataset().Order(
		folderTable.Col("path").Asc(),
		fileTable.Col("basename").Asc(),
		table.Col("date").Asc(),
	))
}

func (qb *TextStore) makeQuery(ctx context.Context, textFilter *models.TextFilterType, findFilter *models.FindFilterType) (*queryBuilder, error) {
	if textFilter == nil {
		textFilter = &models.TextFilterType{}
	}
	if findFilter == nil {
		findFilter = &models.FindFilterType{}
	}

	query := textRepository.newQuery()
	distinctIDs(&query, textTable)

	if q := findFilter.Q; q != nil && *q != "" {
		query.addJoins(
			join{
				table:    textsFilesTable,
				onClause: "texts_files.text_id = texts.id",
			},
			join{
				table:    fileTable,
				onClause: "texts_files.file_id = files.id",
			},
			join{
				table:    folderTable,
				onClause: "files.parent_folder_id = folders.id",
			},
			join{
				table:    fingerprintTable,
				onClause: "files_fingerprints.file_id = texts_files.file_id",
			},
			join{
				table:    textBookmarkTable,
				onClause: "text_bookmarks.text_id = texts.id",
			},
		)

		filepathColumn := "folders.path || '" + string(filepath.Separator) + "' || files.basename"
		searchColumns := []string{"texts.title", "texts.details", filepathColumn, "files_fingerprints.fingerprint", "text_bookmarks.title"}
		query.parseQueryString(searchColumns, *q)
	}

	filter := filterBuilderFromHandler(ctx, &textFilterHandler{
		textFilter: textFilter,
	})

	if err := query.addFilter(filter); err != nil {
		return nil, err
	}

	if err := qb.setTextSort(&query, findFilter); err != nil {
		return nil, err
	}
	query.sortAndPagination += getPagination(findFilter)

	return &query, nil
}

func (qb *TextStore) Query(ctx context.Context, options models.TextQueryOptions) (*models.TextQueryResult, error) {
	query, err := qb.makeQuery(ctx, options.TextFilter, options.FindFilter)
	if err != nil {
		return nil, err
	}

	result, err := qb.queryGroupedFields(ctx, options, *query)
	if err != nil {
		return nil, fmt.Errorf("error querying aggregate fields: %w", err)
	}

	idsResult, err := query.findIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("error finding IDs: %w", err)
	}

	result.IDs = idsResult
	return result, nil
}

func (qb *TextStore) queryGroupedFields(ctx context.Context, options models.TextQueryOptions, query queryBuilder) (*models.TextQueryResult, error) {
	if !options.Count && !options.TotalDuration && !options.TotalSize {
		// nothing to do - return empty result
		return models.NewTextQueryResult(qb), nil
	}

	aggregateQuery := textRepository.newQuery()

	if options.Count {
		aggregateQuery.addColumn("COUNT(DISTINCT temp.id) as total")
	}

	if options.TotalDuration {
		query.addJoins(
			join{
				table:    textsFilesTable,
				onClause: "texts_files.text_id = texts.id",
			},
			join{
				table:    videoFileTable,
				onClause: "texts_files.file_id = video_files.file_id",
			},
		)
		query.addColumn("COALESCE(video_files.duration, 0) as duration")
		aggregateQuery.addColumn("SUM(temp.duration) as duration")
	}

	if options.TotalSize {
		query.addJoins(
			join{
				table:    textsFilesTable,
				onClause: "texts_files.text_id = texts.id",
			},
			join{
				table:    fileTable,
				onClause: "texts_files.file_id = files.id",
			},
		)
		query.addColumn("COALESCE(files.size, 0) as size")
		aggregateQuery.addColumn("SUM(temp.size) as size")
	}

	const includeSortPagination = false
	aggregateQuery.from = fmt.Sprintf("(%s) as temp", query.toSQL(includeSortPagination))

	out := struct {
		Total    int
		Duration null.Float
		Size     null.Float
	}{}
	if err := textRepository.queryStruct(ctx, aggregateQuery.toSQL(includeSortPagination), query.args, &out); err != nil {
		return nil, err
	}

	ret := models.NewTextQueryResult(qb)
	ret.Count = out.Total
	ret.TotalDuration = out.Duration.Float64
	ret.TotalSize = out.Size.Float64
	return ret, nil
}

func (qb *TextStore) QueryCount(ctx context.Context, textFilter *models.TextFilterType, findFilter *models.FindFilterType) (int, error) {
	query, err := qb.makeQuery(ctx, textFilter, findFilter)
	if err != nil {
		return 0, err
	}

	return query.executeCount(ctx)
}

var textSortOptions = sortOptions{
	"bitrate",
	"created_at",
	"date",
	"file_count",
	"filesize",
	"duration",
	"file_mod_time",
	"framerate",
	"id",
	"interactive",
	"interactive_speed",
	"last_o_at",
	"last_read_at",
	"o_counter",
	"organized",
	"performer_count",
	"read_count",
	"read_duration",
	"resume_time",
	"path",
	"perceptual_similarity",
	"random",
	"rating",
	"tag_count",
	"title",
	"updated_at",
}

func (qb *TextStore) setTextSort(query *queryBuilder, findFilter *models.FindFilterType) error {
	if findFilter == nil || findFilter.Sort == nil || *findFilter.Sort == "" {
		return nil
	}
	sort := findFilter.GetSort("title")

	// CVE-2024-32231 - ensure sort is in the list of allowed sorts
	if err := textSortOptions.validateSort(sort); err != nil {
		return err
	}

	addFileTable := func() {
		query.addJoins(
			join{
				table:    textsFilesTable,
				onClause: "texts_files.text_id = texts.id",
			},
			join{
				table:    fileTable,
				onClause: "texts_files.file_id = files.id",
			},
		)
	}

	addVideoFileTable := func() {
		addFileTable()
		query.addJoins(
			join{
				table:    videoFileTable,
				onClause: "video_files.file_id = texts_files.file_id",
			},
		)
	}

	addFolderTable := func() {
		query.addJoins(
			join{
				table:    folderTable,
				onClause: "files.parent_folder_id = folders.id",
			},
		)
	}

	direction := findFilter.GetDirection()
	switch sort {
	case "tag_count":
		query.sortAndPagination += getCountSort(textTable, textsTagsTable, textIDColumn, direction)
	case "performer_count":
		query.sortAndPagination += getCountSort(textTable, performersTextsTable, textIDColumn, direction)
	case "file_count":
		query.sortAndPagination += getCountSort(textTable, textsFilesTable, textIDColumn, direction)
	case "path":
		// special handling for path
		addFileTable()
		addFolderTable()
		query.sortAndPagination += fmt.Sprintf(" ORDER BY COALESCE(folders.path, '') || COALESCE(files.basename, '') COLLATE NATURAL_CI %s", direction)
	case "perceptual_similarity":
		// special handling for phash
		addFileTable()
		query.addJoins(
			join{
				table:    fingerprintTable,
				as:       "fingerprints_phash",
				onClause: "texts_files.file_id = fingerprints_phash.file_id AND fingerprints_phash.type = 'phash'",
			},
		)

		query.sortAndPagination += " ORDER BY fingerprints_phash.fingerprint " + direction + ", files.size DESC"
	case "bitrate":
		sort = "bit_rate"
		addVideoFileTable()
		query.sortAndPagination += getSort(sort, direction, videoFileTable)
	case "file_mod_time":
		sort = "mod_time"
		addFileTable()
		query.sortAndPagination += getSort(sort, direction, fileTable)
	case "framerate":
		sort = "frame_rate"
		addVideoFileTable()
		query.sortAndPagination += getSort(sort, direction, videoFileTable)
	case "filesize":
		addFileTable()
		query.sortAndPagination += getSort(sort, direction, fileTable)
	case "duration":
		addVideoFileTable()
		query.sortAndPagination += getSort(sort, direction, videoFileTable)
	case "interactive", "interactive_speed":
		addVideoFileTable()
		query.sortAndPagination += getSort(sort, direction, videoFileTable)
	case "title":
		addFileTable()
		addFolderTable()
		query.sortAndPagination += " ORDER BY COALESCE(texts.title, files.basename) COLLATE NATURAL_CI " + direction + ", folders.path COLLATE NATURAL_CI " + direction
	case "read_count":
		query.sortAndPagination += getCountSort(textTable, textsReadDatesTable, textIDColumn, direction)
	case "last_read_at":
		query.sortAndPagination += fmt.Sprintf(" ORDER BY (SELECT MAX(read_date) FROM %s AS sort WHERE sort.%s = %s.id) %s", textsReadDatesTable, textIDColumn, textTable, getSortDirection(direction))
	case "last_o_at":
		query.sortAndPagination += fmt.Sprintf(" ORDER BY (SELECT MAX(o_date) FROM %s AS sort WHERE sort.%s = %s.id) %s", textsODatesTable, textIDColumn, textTable, getSortDirection(direction))
	case "o_counter":
		query.sortAndPagination += getCountSort(textTable, textsODatesTable, textIDColumn, direction)
	default:
		query.sortAndPagination += getSort(sort, direction, "texts")
	}

	// Whatever the sorting, always use title/id as a final sort
	query.sortAndPagination += ", COALESCE(texts.title, texts.id) COLLATE NATURAL_CI ASC"

	return nil
}

func (qb *TextStore) SaveActivity(ctx context.Context, id int, resumeTime *float64, readDuration *float64) (bool, error) {
	if err := qb.tableMgr.checkIDExists(ctx, id); err != nil {
		return false, err
	}

	record := goqu.Record{}

	if resumeTime != nil {
		record["resume_time"] = resumeTime
	}

	if readDuration != nil {
		record["read_duration"] = goqu.L("read_duration + ?", readDuration)
	}

	if len(record) > 0 {
		if err := qb.tableMgr.updateByID(ctx, id, record); err != nil {
			return false, err
		}
	}

	return true, nil
}

func (qb *TextStore) GetURLs(ctx context.Context, textID int) ([]string, error) {
	return textsURLsTableMgr.get(ctx, textID)
}

func (qb *TextStore) GetCover(ctx context.Context, textID int) ([]byte, error) {
	return qb.GetImage(ctx, textID, textCoverBlobColumn)
}

func (qb *TextStore) HasCover(ctx context.Context, textID int) (bool, error) {
	return qb.HasImage(ctx, textID, textCoverBlobColumn)
}

func (qb *TextStore) UpdateCover(ctx context.Context, textID int, image []byte) error {
	return qb.UpdateImage(ctx, textID, textCoverBlobColumn, image)
}

func (qb *TextStore) destroyCover(ctx context.Context, textID int) error {
	return qb.DestroyImage(ctx, textID, textCoverBlobColumn)
}

func (qb *TextStore) AssignFiles(ctx context.Context, textID int, fileIDs []models.FileID) error {
	// assuming a file can only be assigned to a single text
	if err := textsFilesTableMgr.destroyJoins(ctx, fileIDs); err != nil {
		return err
	}

	// assign primary only if destination has no files
	existingFileIDs, err := textRepository.files.get(ctx, textID)
	if err != nil {
		return err
	}

	firstPrimary := len(existingFileIDs) == 0
	return textsFilesTableMgr.insertJoins(ctx, textID, firstPrimary, fileIDs)
}

func (qb *TextStore) AddFileID(ctx context.Context, id int, fileID models.FileID) error {
	const firstPrimary = false
	return textsFilesTableMgr.insertJoins(ctx, id, firstPrimary, []models.FileID{fileID})
}

func (qb *TextStore) GetPerformerIDs(ctx context.Context, id int) ([]int, error) {
	return textRepository.performers.getIDs(ctx, id)
}

func (qb *TextStore) GetTagIDs(ctx context.Context, id int) ([]int, error) {
	return textRepository.tags.getIDs(ctx, id)
}

func (qb *TextStore) GetGalleryIDs(ctx context.Context, id int) ([]int, error) {
	return textRepository.galleries.getIDs(ctx, id)
}

func (qb *TextStore) AddGalleryIDs(ctx context.Context, textID int, galleryIDs []int) error {
	return textsGalleriesTableMgr.addJoins(ctx, textID, galleryIDs)
}

func (qb *TextStore) GetStashIDs(ctx context.Context, textID int) ([]models.StashID, error) {
	return textRepository.stashIDs.get(ctx, textID)
}

func (qb *TextStore) FindDuplicates(ctx context.Context, distance int, durationDiff float64) ([][]*models.Text, error) {
	var dupeIds [][]int
	if distance == 0 {
		var ids []string
		if err := dbWrapper.Select(ctx, &ids, findExactDuplicateQuery, durationDiff); err != nil {
			return nil, err
		}

		for _, id := range ids {
			strIds := strings.Split(id, ",")
			var textIds []int
			for _, strId := range strIds {
				if intId, err := strconv.Atoi(strId); err == nil {
					textIds = sliceutil.AppendUnique(textIds, intId)
				}
			}
			// filter out
			if len(textIds) > 1 {
				dupeIds = append(dupeIds, textIds)
			}
		}
	} else {
		var hashes []*utils.Phash

		if err := textRepository.queryFunc(ctx, findAllPhashesQuery, nil, false, func(rows *sqlx.Rows) error {
			phash := utils.Phash{
				Bucket:   -1,
				Duration: -1,
			}
			if err := rows.StructScan(&phash); err != nil {
				return err
			}

			hashes = append(hashes, &phash)
			return nil
		}); err != nil {
			return nil, err
		}

		dupeIds = utils.FindDuplicates(hashes, distance, durationDiff)
	}

	var duplicates [][]*models.Text
	for _, textIds := range dupeIds {
		if texts, err := qb.FindMany(ctx, textIds); err == nil {
			duplicates = append(duplicates, texts)
		}
	}

	sortByPath(duplicates)

	return duplicates, nil
}

func sortByPath(texts [][]*models.Text) {
	lessFunc := func(i int, j int) bool {
		firstPathI := getFirstPath(texts[i])
		firstPathJ := getFirstPath(texts[j])
		return firstPathI < firstPathJ
	}
	sort.SliceStable(texts, lessFunc)
}

func getFirstPath(texts []*models.Text) string {
	var firstPath string
	for i, text := range texts {
		if i == 0 || text.Path < firstPath {
			firstPath = text.Path
		}
	}
	return firstPath
}
