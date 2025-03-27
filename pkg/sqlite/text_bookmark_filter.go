package sqlite

import (
	"context"
	"fmt"

	"github.com/stashapp/stash/pkg/models"
)

type textBookmarkFilterHandler struct {
	textBookmarkFilter *models.TextBookmarkFilterType
}

func (qb *textBookmarkFilterHandler) validate() error {
	return nil
}

func (qb *textBookmarkFilterHandler) handle(ctx context.Context, f *filterBuilder) {
	textBookmarkFilter := qb.textBookmarkFilter
	if textBookmarkFilter == nil {
		return
	}

	if err := qb.validate(); err != nil {
		f.setError(err)
		return
	}

	f.handleCriterion(ctx, qb.criterionHandler())
}

func (qb *textBookmarkFilterHandler) joinTexts(f *filterBuilder) {
	textBookmarkRepository.texts.innerJoin(f, "", "text_bookmarks.text_id")
}

func (qb *textBookmarkFilterHandler) criterionHandler() criterionHandler {
	textBookmarkFilter := qb.textBookmarkFilter
	return compoundHandler{
		qb.tagIDCriterionHandler(textBookmarkFilter.TagID),
		qb.tagsCriterionHandler(textBookmarkFilter.Tags),
		qb.textTagsCriterionHandler(textBookmarkFilter.TextTags),
		qb.performersCriterionHandler(textBookmarkFilter.Performers),
		&timestampCriterionHandler{textBookmarkFilter.CreatedAt, "text_bookmarks.created_at", nil},
		&timestampCriterionHandler{textBookmarkFilter.UpdatedAt, "text_bookmarks.updated_at", nil},
		&dateCriterionHandler{textBookmarkFilter.TextDate, "texts.date", qb.joinTexts},
		&timestampCriterionHandler{textBookmarkFilter.TextCreatedAt, "texts.created_at", qb.joinTexts},
		&timestampCriterionHandler{textBookmarkFilter.TextUpdatedAt, "texts.updated_at", qb.joinTexts},

		&relatedFilterHandler{
			relatedIDCol:   "texts.id",
			relatedRepo:    textRepository.repository,
			relatedHandler: &textFilterHandler{textBookmarkFilter.TextFilter},
			joinFn: func(f *filterBuilder) {
				qb.joinTexts(f)
			},
		},
	}
}

func (qb *textBookmarkFilterHandler) tagIDCriterionHandler(tagID *string) criterionHandlerFunc {
	return func(ctx context.Context, f *filterBuilder) {
		if tagID != nil {
			f.addLeftJoin("text_bookmarks_tags", "", "text_bookmarks_tags.text_bookmark_id = text_bookmarks.id")

			f.addWhere("(text_bookmarks.primary_tag_id = ? OR text_bookmarks_tags.tag_id = ?)", *tagID, *tagID)
		}
	}
}

func (qb *textBookmarkFilterHandler) tagsCriterionHandler(criterion *models.HierarchicalMultiCriterionInput) criterionHandlerFunc {
	return func(ctx context.Context, f *filterBuilder) {
		if criterion != nil {
			tags := criterion.CombineExcludes()

			if tags.Modifier == models.CriterionModifierIsNull || tags.Modifier == models.CriterionModifierNotNull {
				var notClause string
				if tags.Modifier == models.CriterionModifierNotNull {
					notClause = "NOT"
				}

				f.addLeftJoin("text_bookmarks_tags", "", "text_bookmarks.id = text_bookmarks_tags.text_bookmark_id")

				f.addWhere(fmt.Sprintf("%s text_bookmarks_tags.tag_id IS NULL", notClause))
				return
			}

			if tags.Modifier == models.CriterionModifierEquals && tags.Depth != nil && *tags.Depth != 0 {
				f.setError(fmt.Errorf("depth is not supported for equals modifier for bookmark tag filtering"))
				return
			}

			if len(tags.Value) == 0 && len(tags.Excludes) == 0 {
				return
			}

			if len(tags.Value) > 0 {
				valuesClause, err := getHierarchicalValues(ctx, tags.Value, tagTable, "tags_relations", "parent_id", "child_id", tags.Depth)
				if err != nil {
					f.setError(err)
					return
				}

				f.addWith(`bookmark_tags AS (
	SELECT mt.text_bookmark_id, t.column1 AS root_tag_id FROM text_bookmarks_tags mt
	INNER JOIN (` + valuesClause + `) t ON t.column2 = mt.tag_id
	UNION
	SELECT m.id, t.column1 FROM text_bookmarks m
	INNER JOIN (` + valuesClause + `) t ON t.column2 = m.primary_tag_id
	)`)

				f.addLeftJoin("bookmark_tags", "", "bookmark_tags.text_bookmark_id = text_bookmarks.id")

				switch tags.Modifier {
				case models.CriterionModifierEquals:
					// includes only the provided ids
					f.addWhere("bookmark_tags.root_tag_id IS NOT NULL")
					tagsLen := len(tags.Value)
					f.addHaving(fmt.Sprintf("count(distinct bookmark_tags.root_tag_id) IS %d", tagsLen))
					// decrement by one to account for primary tag id
					f.addWhere("(SELECT COUNT(*) FROM text_bookmarks_tags s WHERE s.text_bookmark_id = text_bookmarks.id) = ?", tagsLen-1)
				case models.CriterionModifierNotEquals:
					f.setError(fmt.Errorf("not equals modifier is not supported for text bookmark tags"))
				default:
					addHierarchicalConditionClauses(f, tags, "bookmark_tags", "root_tag_id")
				}
			}

			if len(criterion.Excludes) > 0 {
				valuesClause, err := getHierarchicalValues(ctx, tags.Excludes, tagTable, "tags_relations", "parent_id", "child_id", tags.Depth)
				if err != nil {
					f.setError(err)
					return
				}

				clause := "text_bookmarks.id NOT IN (SELECT text_bookmarks_tags.text_bookmark_id FROM text_bookmarks_tags WHERE text_bookmarks_tags.tag_id IN (SELECT column2 FROM (%s)))"
				f.addWhere(fmt.Sprintf(clause, valuesClause))

				f.addWhere(fmt.Sprintf("text_bookmarks.primary_tag_id NOT IN (SELECT column2 FROM (%s))", valuesClause))
			}
		}
	}
}

func (qb *textBookmarkFilterHandler) textTagsCriterionHandler(tags *models.HierarchicalMultiCriterionInput) criterionHandlerFunc {
	return func(ctx context.Context, f *filterBuilder) {
		if tags != nil {
			f.addLeftJoin("texts_tags", "", "text_bookmarks.text_id = texts_tags.text_id")

			h := joinedHierarchicalMultiCriterionHandlerBuilder{
				primaryTable: "text_bookmarks",
				primaryKey:   textIDColumn,
				foreignTable: tagTable,
				foreignFK:    tagIDColumn,

				relationsTable: "tags_relations",
				joinTable:      "texts_tags",
				joinAs:         "bookmark_texts_tags",
				primaryFK:      textIDColumn,
			}

			h.handler(tags).handle(ctx, f)
		}
	}
}

func (qb *textBookmarkFilterHandler) performersCriterionHandler(performers *models.MultiCriterionInput) criterionHandlerFunc {
	h := joinedMultiCriterionHandlerBuilder{
		primaryTable: textTable,
		joinTable:    performersTextsTable,
		joinAs:       "performers_join",
		primaryFK:    textIDColumn,
		foreignFK:    performerIDColumn,

		addJoinTable: func(f *filterBuilder) {
			f.addLeftJoin(performersTextsTable, "performers_join", "performers_join.text_id = text_bookmarks.text_id")
		},
	}

	handler := h.handler(performers)
	return func(ctx context.Context, f *filterBuilder) {
		if performers == nil {
			return
		}

		// Make sure texts is included, otherwise excludes filter fails
		qb.joinTexts(f)
		handler(ctx, f)
	}
}
