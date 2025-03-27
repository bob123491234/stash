package sqlite

import (
	"context"
	"fmt"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/utils"
)

type textFilterHandler struct {
	textFilter *models.TextFilterType
}

func (qb *textFilterHandler) validate() error {
	textFilter := qb.textFilter
	if textFilter == nil {
		return nil
	}

	if err := validateFilterCombination(textFilter.OperatorFilter); err != nil {
		return err
	}

	if subFilter := textFilter.SubFilter(); subFilter != nil {
		sqb := &textFilterHandler{textFilter: subFilter}
		if err := sqb.validate(); err != nil {
			return err
		}
	}

	return nil
}

func (qb *textFilterHandler) handle(ctx context.Context, f *filterBuilder) {
	textFilter := qb.textFilter
	if textFilter == nil {
		return
	}

	if err := qb.validate(); err != nil {
		f.setError(err)
		return
	}

	sf := textFilter.SubFilter()
	if sf != nil {
		sub := &textFilterHandler{sf}
		handleSubFilter(ctx, sub, f, textFilter.OperatorFilter)
	}

	f.handleCriterion(ctx, qb.criterionHandler())
}

func (qb *textFilterHandler) criterionHandler() criterionHandler {
	textFilter := qb.textFilter
	return compoundHandler{
		intCriterionHandler(textFilter.ID, "texts.id", nil),
		pathCriterionHandler(textFilter.Path, "folders.path", "files.basename", qb.addFoldersTable),
		qb.fileCountCriterionHandler(textFilter.FileCount),
		stringCriterionHandler(textFilter.Title, "texts.title"),
		stringCriterionHandler(textFilter.Code, "texts.code"),
		stringCriterionHandler(textFilter.Details, "texts.details"),
		stringCriterionHandler(textFilter.Director, "texts.director"),
		criterionHandlerFunc(func(ctx context.Context, f *filterBuilder) {
			if textFilter.Oshash != nil {
				qb.addTextFilesTable(f)
				f.addLeftJoin(fingerprintTable, "fingerprints_oshash", "texts_files.file_id = fingerprints_oshash.file_id AND fingerprints_oshash.type = 'oshash'")
			}

			stringCriterionHandler(textFilter.Oshash, "fingerprints_oshash.fingerprint")(ctx, f)
		}),

		criterionHandlerFunc(func(ctx context.Context, f *filterBuilder) {
			if textFilter.Checksum != nil {
				qb.addTextFilesTable(f)
				f.addLeftJoin(fingerprintTable, "fingerprints_md5", "texts_files.file_id = fingerprints_md5.file_id AND fingerprints_md5.type = 'md5'")
			}

			stringCriterionHandler(textFilter.Checksum, "fingerprints_md5.fingerprint")(ctx, f)
		}),

		criterionHandlerFunc(func(ctx context.Context, f *filterBuilder) {
			if textFilter.Phash != nil {
				// backwards compatibility
				qb.phashDistanceCriterionHandler(&models.PhashDistanceCriterionInput{
					Value:    textFilter.Phash.Value,
					Modifier: textFilter.Phash.Modifier,
				})(ctx, f)
			}
		}),

		qb.phashDistanceCriterionHandler(textFilter.PhashDistance),

		intCriterionHandler(textFilter.Rating100, "texts.rating", nil),
		qb.oCountCriterionHandler(textFilter.OCounter),
		boolCriterionHandler(textFilter.Organized, "texts.organized", nil),

		floatIntCriterionHandler(textFilter.Duration, "video_files.duration", qb.addVideoFilesTable),
		resolutionCriterionHandler(textFilter.Resolution, "video_files.height", "video_files.width", qb.addVideoFilesTable),
		orientationCriterionHandler(textFilter.Orientation, "video_files.height", "video_files.width", qb.addVideoFilesTable),
		floatIntCriterionHandler(textFilter.Framerate, "ROUND(video_files.frame_rate)", qb.addVideoFilesTable),
		intCriterionHandler(textFilter.Bitrate, "video_files.bit_rate", qb.addVideoFilesTable),
		qb.codecCriterionHandler(textFilter.VideoCodec, "video_files.video_codec", qb.addVideoFilesTable),
		qb.codecCriterionHandler(textFilter.AudioCodec, "video_files.audio_codec", qb.addVideoFilesTable),

		qb.hasBookmarksCriterionHandler(textFilter.HasBookmarks),
		qb.isMissingCriterionHandler(textFilter.IsMissing),
		qb.urlsCriterionHandler(textFilter.URL),

		criterionHandlerFunc(func(ctx context.Context, f *filterBuilder) {
			if textFilter.StashID != nil {
				textRepository.stashIDs.join(f, "text_stash_ids", "texts.id")
				stringCriterionHandler(textFilter.StashID, "text_stash_ids.stash_id")(ctx, f)
			}
		}),

		&stashIDCriterionHandler{
			c:                 textFilter.StashIDEndpoint,
			stashIDRepository: &textRepository.stashIDs,
			stashIDTableAs:    "text_stash_ids",
			parentIDCol:       "texts.id",
		},

		boolCriterionHandler(textFilter.Interactive, "video_files.interactive", qb.addVideoFilesTable),
		intCriterionHandler(textFilter.InteractiveSpeed, "video_files.interactive_speed", qb.addVideoFilesTable),

		qb.captionCriterionHandler(textFilter.Captions),

		floatIntCriterionHandler(textFilter.ResumeTime, "texts.resume_time", nil),
		floatIntCriterionHandler(textFilter.PlayDuration, "texts.play_duration", nil),
		qb.playCountCriterionHandler(textFilter.PlayCount),
		criterionHandlerFunc(func(ctx context.Context, f *filterBuilder) {
			if textFilter.LastPlayedAt != nil {
				f.addLeftJoin(
					fmt.Sprintf("(SELECT %s, MAX(%s) as last_played_at FROM %s GROUP BY %s)", textIDColumn, textReadDateColumn, textsReadDatesTable, textIDColumn),
					"text_last_view",
					fmt.Sprintf("text_last_view.%s = texts.id", textIDColumn),
				)
				h := timestampCriterionHandler{textFilter.LastPlayedAt, "IFNULL(last_played_at, datetime(0))", nil}
				h.handle(ctx, f)
			}
		}),

		qb.tagsCriterionHandler(textFilter.Tags),
		qb.tagCountCriterionHandler(textFilter.TagCount),
		qb.performersCriterionHandler(textFilter.Performers),
		qb.performerCountCriterionHandler(textFilter.PerformerCount),
		studioCriterionHandler(textTable, textFilter.Studios),

		qb.groupsCriterionHandler(textFilter.Groups),
		qb.groupsCriterionHandler(textFilter.Movies),

		qb.galleriesCriterionHandler(textFilter.Galleries),
		qb.performerTagsCriterionHandler(textFilter.PerformerTags),
		qb.performerFavoriteCriterionHandler(textFilter.PerformerFavorite),
		qb.performerAgeCriterionHandler(textFilter.PerformerAge),
		qb.phashDuplicatedCriterionHandler(textFilter.Duplicated, qb.addTextFilesTable),
		&dateCriterionHandler{textFilter.Date, "texts.date", nil},
		&timestampCriterionHandler{textFilter.CreatedAt, "texts.created_at", nil},
		&timestampCriterionHandler{textFilter.UpdatedAt, "texts.updated_at", nil},

		&relatedFilterHandler{
			relatedIDCol:   "texts_galleries.gallery_id",
			relatedRepo:    galleryRepository.repository,
			relatedHandler: &galleryFilterHandler{textFilter.GalleriesFilter},
			joinFn: func(f *filterBuilder) {
				textRepository.galleries.innerJoin(f, "", "texts.id")
			},
		},

		&relatedFilterHandler{
			relatedIDCol:   "performers_join.performer_id",
			relatedRepo:    performerRepository.repository,
			relatedHandler: &performerFilterHandler{textFilter.PerformersFilter},
			joinFn: func(f *filterBuilder) {
				textRepository.performers.innerJoin(f, "performers_join", "texts.id")
			},
		},

		&relatedFilterHandler{
			relatedIDCol:   "texts.studio_id",
			relatedRepo:    studioRepository.repository,
			relatedHandler: &studioFilterHandler{textFilter.StudiosFilter},
		},

		&relatedFilterHandler{
			relatedIDCol:   "text_tag.tag_id",
			relatedRepo:    tagRepository.repository,
			relatedHandler: &tagFilterHandler{textFilter.TagsFilter},
			joinFn: func(f *filterBuilder) {
				textRepository.tags.innerJoin(f, "text_tag", "texts.id")
			},
		},

		&relatedFilterHandler{
			relatedIDCol:   "movies_texts.movie_id",
			relatedRepo:    groupRepository.repository,
			relatedHandler: &groupFilterHandler{textFilter.MoviesFilter},
			joinFn: func(f *filterBuilder) {
				textRepository.groups.innerJoin(f, "", "texts.id")
			},
		},

		&relatedFilterHandler{
			relatedIDCol:   "text_bookmarks.id",
			relatedRepo:    textBookmarkRepository.repository,
			relatedHandler: &textBookmarkFilterHandler{textFilter.BookmarksFilter},
			joinFn: func(f *filterBuilder) {
				f.addInnerJoin("text_bookmarks", "", "texts.id")
			},
		},
	}
}

func (qb *textFilterHandler) addTextFilesTable(f *filterBuilder) {
	f.addLeftJoin(textsFilesTable, "", "texts_files.text_id = texts.id")
}

func (qb *textFilterHandler) addFilesTable(f *filterBuilder) {
	qb.addTextFilesTable(f)
	f.addLeftJoin(fileTable, "", "texts_files.file_id = files.id")
}

func (qb *textFilterHandler) addFoldersTable(f *filterBuilder) {
	qb.addFilesTable(f)
	f.addLeftJoin(folderTable, "", "files.parent_folder_id = folders.id")
}

func (qb *textFilterHandler) addVideoFilesTable(f *filterBuilder) {
	qb.addTextFilesTable(f)
	f.addLeftJoin(videoFileTable, "", "video_files.file_id = texts_files.file_id")
}

func (qb *textFilterHandler) playCountCriterionHandler(count *models.IntCriterionInput) criterionHandlerFunc {
	h := countCriterionHandlerBuilder{
		primaryTable: textTable,
		joinTable:    textsReadDatesTable,
		primaryFK:    textIDColumn,
	}

	return h.handler(count)
}

func (qb *textFilterHandler) oCountCriterionHandler(count *models.IntCriterionInput) criterionHandlerFunc {
	h := countCriterionHandlerBuilder{
		primaryTable: textTable,
		joinTable:    textsODatesTable,
		primaryFK:    textIDColumn,
	}

	return h.handler(count)
}

func (qb *textFilterHandler) fileCountCriterionHandler(fileCount *models.IntCriterionInput) criterionHandlerFunc {
	h := countCriterionHandlerBuilder{
		primaryTable: textTable,
		joinTable:    textsFilesTable,
		primaryFK:    textIDColumn,
	}

	return h.handler(fileCount)
}

func (qb *textFilterHandler) phashDuplicatedCriterionHandler(duplicatedFilter *models.PHashDuplicationCriterionInput, addJoinFn func(f *filterBuilder)) criterionHandlerFunc {
	return func(ctx context.Context, f *filterBuilder) {
		// TODO: Wishlist item: Implement Distance matching
		if duplicatedFilter != nil {
			if addJoinFn != nil {
				addJoinFn(f)
			}

			var v string
			if *duplicatedFilter.Duplicated {
				v = ">"
			} else {
				v = "="
			}

			f.addInnerJoin("(SELECT file_id FROM files_fingerprints INNER JOIN (SELECT fingerprint FROM files_fingerprints WHERE type = 'phash' GROUP BY fingerprint HAVING COUNT (fingerprint) "+v+" 1) dupes on files_fingerprints.fingerprint = dupes.fingerprint)", "scph", "texts_files.file_id = scph.file_id")
		}
	}
}

func (qb *textFilterHandler) codecCriterionHandler(codec *models.StringCriterionInput, codecColumn string, addJoinFn func(f *filterBuilder)) criterionHandlerFunc {
	return func(ctx context.Context, f *filterBuilder) {
		if codec != nil {
			if addJoinFn != nil {
				addJoinFn(f)
			}

			stringCriterionHandler(codec, codecColumn)(ctx, f)
		}
	}
}

func (qb *textFilterHandler) hasBookmarksCriterionHandler(hasBookmarks *string) criterionHandlerFunc {
	return func(ctx context.Context, f *filterBuilder) {
		if hasBookmarks != nil {
			f.addLeftJoin("text_bookmarks", "", "text_bookmarks.text_id = texts.id")
			if *hasBookmarks == "true" {
				f.addHaving("count(text_bookmarks.text_id) > 0")
			} else {
				f.addWhere("text_bookmarks.id IS NULL")
			}
		}
	}
}

func (qb *textFilterHandler) isMissingCriterionHandler(isMissing *string) criterionHandlerFunc {
	return func(ctx context.Context, f *filterBuilder) {
		if isMissing != nil && *isMissing != "" {
			switch *isMissing {
			case "url":
				textsURLsTableMgr.join(f, "", "texts.id")
				f.addWhere("text_urls.url IS NULL")
			case "galleries":
				textRepository.galleries.join(f, "galleries_join", "texts.id")
				f.addWhere("galleries_join.text_id IS NULL")
			case "studio":
				f.addWhere("texts.studio_id IS NULL")
			case "movie":
				textRepository.groups.join(f, "movies_join", "texts.id")
				f.addWhere("movies_join.text_id IS NULL")
			case "performers":
				textRepository.performers.join(f, "performers_join", "texts.id")
				f.addWhere("performers_join.text_id IS NULL")
			case "date":
				f.addWhere(`texts.date IS NULL OR texts.date IS ""`)
			case "tags":
				textRepository.tags.join(f, "tags_join", "texts.id")
				f.addWhere("tags_join.text_id IS NULL")
			case "stash_id":
				textRepository.stashIDs.join(f, "text_stash_ids", "texts.id")
				f.addWhere("text_stash_ids.text_id IS NULL")
			case "phash":
				qb.addTextFilesTable(f)
				f.addLeftJoin(fingerprintTable, "fingerprints_phash", "texts_files.file_id = fingerprints_phash.file_id AND fingerprints_phash.type = 'phash'")
				f.addWhere("fingerprints_phash.fingerprint IS NULL")
			case "cover":
				f.addWhere("texts.cover_blob IS NULL")
			default:
				f.addWhere("(texts." + *isMissing + " IS NULL OR TRIM(texts." + *isMissing + ") = '')")
			}
		}
	}
}

func (qb *textFilterHandler) urlsCriterionHandler(url *models.StringCriterionInput) criterionHandlerFunc {
	h := stringListCriterionHandlerBuilder{
		primaryTable: textTable,
		primaryFK:    textIDColumn,
		joinTable:    textsURLsTable,
		stringColumn: textURLColumn,
		addJoinTable: func(f *filterBuilder) {
			textsURLsTableMgr.join(f, "", "texts.id")
		},
	}

	return h.handler(url)
}

func (qb *textFilterHandler) getMultiCriterionHandlerBuilder(foreignTable, joinTable, foreignFK string, addJoinsFunc func(f *filterBuilder)) multiCriterionHandlerBuilder {
	return multiCriterionHandlerBuilder{
		primaryTable: textTable,
		foreignTable: foreignTable,
		joinTable:    joinTable,
		primaryFK:    textIDColumn,
		foreignFK:    foreignFK,
		addJoinsFunc: addJoinsFunc,
	}
}

func (qb *textFilterHandler) captionCriterionHandler(captions *models.StringCriterionInput) criterionHandlerFunc {
	h := stringListCriterionHandlerBuilder{
		primaryTable: textTable,
		primaryFK:    textIDColumn,
		joinTable:    videoCaptionsTable,
		stringColumn: captionCodeColumn,
		addJoinTable: func(f *filterBuilder) {
			qb.addTextFilesTable(f)
			f.addLeftJoin(videoCaptionsTable, "", "video_captions.file_id = texts_files.file_id")
		},
		excludeHandler: func(f *filterBuilder, criterion *models.StringCriterionInput) {
			excludeClause := `texts.id NOT IN (
				SELECT texts_files.text_id from texts_files 
				INNER JOIN video_captions on video_captions.file_id = texts_files.file_id 
				WHERE video_captions.language_code LIKE ?
			)`
			f.addWhere(excludeClause, criterion.Value)

			// TODO - should we also exclude null values?
		},
	}

	return h.handler(captions)
}

func (qb *textFilterHandler) tagsCriterionHandler(tags *models.HierarchicalMultiCriterionInput) criterionHandlerFunc {
	h := joinedHierarchicalMultiCriterionHandlerBuilder{
		primaryTable: textTable,
		foreignTable: tagTable,
		foreignFK:    "tag_id",

		relationsTable: "tags_relations",
		joinAs:         "text_tag",
		joinTable:      textsTagsTable,
		primaryFK:      textIDColumn,
	}

	return h.handler(tags)
}

func (qb *textFilterHandler) tagCountCriterionHandler(tagCount *models.IntCriterionInput) criterionHandlerFunc {
	h := countCriterionHandlerBuilder{
		primaryTable: textTable,
		joinTable:    textsTagsTable,
		primaryFK:    textIDColumn,
	}

	return h.handler(tagCount)
}

func (qb *textFilterHandler) performersCriterionHandler(performers *models.MultiCriterionInput) criterionHandlerFunc {
	h := joinedMultiCriterionHandlerBuilder{
		primaryTable: textTable,
		joinTable:    performersTextsTable,
		joinAs:       "performers_join",
		primaryFK:    textIDColumn,
		foreignFK:    performerIDColumn,

		addJoinTable: func(f *filterBuilder) {
			textRepository.performers.join(f, "performers_join", "texts.id")
		},
	}

	return h.handler(performers)
}

func (qb *textFilterHandler) performerCountCriterionHandler(performerCount *models.IntCriterionInput) criterionHandlerFunc {
	h := countCriterionHandlerBuilder{
		primaryTable: textTable,
		joinTable:    performersTextsTable,
		primaryFK:    textIDColumn,
	}

	return h.handler(performerCount)
}

func (qb *textFilterHandler) performerFavoriteCriterionHandler(performerfavorite *bool) criterionHandlerFunc {
	return func(ctx context.Context, f *filterBuilder) {
		if performerfavorite != nil {
			f.addLeftJoin("performers_texts", "", "texts.id = performers_texts.text_id")

			if *performerfavorite {
				// contains at least one favorite
				f.addLeftJoin("performers", "", "performers.id = performers_texts.performer_id")
				f.addWhere("performers.favorite = 1")
			} else {
				// contains zero favorites
				f.addLeftJoin(`(SELECT performers_texts.text_id as id FROM performers_texts
JOIN performers ON performers.id = performers_texts.performer_id
GROUP BY performers_texts.text_id HAVING SUM(performers.favorite) = 0)`, "nofaves", "texts.id = nofaves.id")
				f.addWhere("performers_texts.text_id IS NULL OR nofaves.id IS NOT NULL")
			}
		}
	}
}

func (qb *textFilterHandler) performerAgeCriterionHandler(performerAge *models.IntCriterionInput) criterionHandlerFunc {
	return func(ctx context.Context, f *filterBuilder) {
		if performerAge != nil {
			f.addInnerJoin("performers_texts", "", "texts.id = performers_texts.text_id")
			f.addInnerJoin("performers", "", "performers_texts.performer_id = performers.id")

			f.addWhere("texts.date != '' AND performers.birthdate != ''")
			f.addWhere("texts.date IS NOT NULL AND performers.birthdate IS NOT NULL")

			ageCalc := "cast(strftime('%Y.%m%d', texts.date) - strftime('%Y.%m%d', performers.birthdate) as int)"
			whereClause, args := getIntWhereClause(ageCalc, performerAge.Modifier, performerAge.Value, performerAge.Value2)
			f.addWhere(whereClause, args...)
		}
	}
}

func (qb *textFilterHandler) groupsCriterionHandler(movies *models.MultiCriterionInput) criterionHandlerFunc {
	addJoinsFunc := func(f *filterBuilder) {
		textRepository.groups.join(f, "", "texts.id")
		f.addLeftJoin("movies", "", "movies_texts.movie_id = movies.id")
	}
	h := qb.getMultiCriterionHandlerBuilder(groupTable, groupsTextsTable, "movie_id", addJoinsFunc)
	return h.handler(movies)
}

func (qb *textFilterHandler) galleriesCriterionHandler(galleries *models.MultiCriterionInput) criterionHandlerFunc {
	addJoinsFunc := func(f *filterBuilder) {
		textRepository.galleries.join(f, "", "texts.id")
		f.addLeftJoin("galleries", "", "texts_galleries.gallery_id = galleries.id")
	}
	h := qb.getMultiCriterionHandlerBuilder(galleryTable, textsGalleriesTable, "gallery_id", addJoinsFunc)
	return h.handler(galleries)
}

func (qb *textFilterHandler) performerTagsCriterionHandler(tags *models.HierarchicalMultiCriterionInput) criterionHandler {
	return &joinedPerformerTagsHandler{
		criterion:      tags,
		primaryTable:   textTable,
		joinTable:      performersTextsTable,
		joinPrimaryKey: textIDColumn,
	}
}

func (qb *textFilterHandler) phashDistanceCriterionHandler(phashDistance *models.PhashDistanceCriterionInput) criterionHandlerFunc {
	return func(ctx context.Context, f *filterBuilder) {
		if phashDistance != nil {
			qb.addTextFilesTable(f)
			f.addLeftJoin(fingerprintTable, "fingerprints_phash", "texts_files.file_id = fingerprints_phash.file_id AND fingerprints_phash.type = 'phash'")

			value, _ := utils.StringToPhash(phashDistance.Value)
			distance := 0
			if phashDistance.Distance != nil {
				distance = *phashDistance.Distance
			}

			if distance == 0 {
				// use the default handler
				intCriterionHandler(&models.IntCriterionInput{
					Value:    int(value),
					Modifier: phashDistance.Modifier,
				}, "fingerprints_phash.fingerprint", nil)(ctx, f)
			}

			switch {
			case phashDistance.Modifier == models.CriterionModifierEquals && distance > 0:
				// needed to avoid a type mismatch
				f.addWhere("typeof(fingerprints_phash.fingerprint) = 'integer'")
				f.addWhere("phash_distance(fingerprints_phash.fingerprint, ?) < ?", value, distance)
			case phashDistance.Modifier == models.CriterionModifierNotEquals && distance > 0:
				// needed to avoid a type mismatch
				f.addWhere("typeof(fingerprints_phash.fingerprint) = 'integer'")
				f.addWhere("phash_distance(fingerprints_phash.fingerprint, ?) > ?", value, distance)
			default:
				intCriterionHandler(&models.IntCriterionInput{
					Value:    int(value),
					Modifier: phashDistance.Modifier,
				}, "fingerprints_phash.fingerprint", nil)(ctx, f)
			}
		}
	}
}
