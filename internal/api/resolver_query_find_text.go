package api

import (
	"context"
	"strconv"

	"github.com/99designs/gqlgen/graphql"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/sliceutil"
	"github.com/stashapp/stash/pkg/sliceutil/stringslice"
	"github.com/stashapp/stash/pkg/text"
)

func (r *queryResolver) FindText(ctx context.Context, id *string, checksum *string) (*models.Text, error) {
	var text *models.Text
	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Text
		var err error
		if id != nil {
			idInt, err := strconv.Atoi(*id)
			if err != nil {
				return err
			}
			text, err = qb.Find(ctx, idInt)
			if err != nil {
				return err
			}
		} else if checksum != nil {
			var texts []*models.Text
			texts, err = qb.FindByChecksum(ctx, *checksum)
			if len(texts) > 0 {
				text = texts[0]
			}
		}

		return err
	}); err != nil {
		return nil, err
	}

	return text, nil
}

func (r *queryResolver) FindTextByHash(ctx context.Context, input TextHashInput) (*models.Text, error) {
	var text *models.Text

	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Text
		if input.Checksum != nil {
			texts, err := qb.FindByChecksum(ctx, *input.Checksum)
			if err != nil {
				return err
			}
			if len(texts) > 0 {
				text = texts[0]
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return text, nil
}

func (r *queryResolver) FindTexts(
	ctx context.Context,
	textFilter *models.TextFilterType,
	textIDs []int,
	ids []string,
	filter *models.FindFilterType,
) (ret *FindTextsResultType, err error) {
	if len(ids) > 0 {
		textIDs, err = stringslice.StringSliceToIntSlice(ids)
		if err != nil {
			return nil, err
		}
	}

	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		var texts []*models.Text
		var err error

		fields := graphql.CollectAllFields(ctx)
		result := &models.TextQueryResult{}

		if len(textIDs) > 0 {
			texts, err = r.repository.Text.FindMany(ctx, textIDs)
			if err == nil {
				result.Count = len(texts)
				for _, s := range texts {
					if err = s.LoadPrimaryFile(ctx, r.repository.File); err != nil {
						break
					}

					f := s.Files.Primary()
					if f == nil {
						continue
					}

					result.TotalDuration += f.Duration

					result.TotalSize += float64(f.Size)
				}
			}
		} else {
			result, err = r.repository.Text.Query(ctx, models.TextQueryOptions{
				QueryOptions: models.QueryOptions{
					FindFilter: filter,
					Count:      sliceutil.Contains(fields, "count"),
				},
				TextFilter:    textFilter,
				TotalDuration: sliceutil.Contains(fields, "duration"),
				TotalSize:     sliceutil.Contains(fields, "filesize"),
			})
			if err == nil {
				texts, err = result.Resolve(ctx)
			}
		}

		if err != nil {
			return err
		}

		ret = &FindTextsResultType{
			Count:    result.Count,
			Texts:    texts,
			Duration: result.TotalDuration,
			Filesize: result.TotalSize,
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *queryResolver) FindTextsByPathRegex(ctx context.Context, filter *models.FindFilterType) (ret *FindTextsResultType, err error) {
	if err := r.withReadTxn(ctx, func(ctx context.Context) error {

		textFilter := &models.TextFilterType{}

		if filter != nil && filter.Q != nil {
			textFilter.Path = &models.StringCriterionInput{
				Modifier: models.CriterionModifierMatchesRegex,
				Value:    "(?i)" + *filter.Q,
			}
		}

		// make a copy of the filter if provided, nilling out Q
		var queryFilter *models.FindFilterType
		if filter != nil {
			f := *filter
			queryFilter = &f
			queryFilter.Q = nil
		}

		fields := graphql.CollectAllFields(ctx)

		result, err := r.repository.Text.Query(ctx, models.TextQueryOptions{
			QueryOptions: models.QueryOptions{
				FindFilter: queryFilter,
				Count:      sliceutil.Contains(fields, "count"),
			},
			TextFilter:    textFilter,
			TotalDuration: sliceutil.Contains(fields, "duration"),
			TotalSize:     sliceutil.Contains(fields, "filesize"),
		})
		if err != nil {
			return err
		}

		texts, err := result.Resolve(ctx)
		if err != nil {
			return err
		}

		ret = &FindTextsResultType{
			Count:    result.Count,
			Texts:    texts,
			Duration: result.TotalDuration,
			Filesize: result.TotalSize,
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *queryResolver) ParseTextFilenames(ctx context.Context, filter *models.FindFilterType, config models.TextParserInput) (ret *TextParserResultType, err error) {
	repo := text.NewFilenameParserRepository(r.repository)
	parser := text.NewFilenameParser(filter, config, repo)

	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		result, count, err := parser.Parse(ctx)

		if err != nil {
			return err
		}

		ret = &TextParserResultType{
			Count:   count,
			Results: result,
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *queryResolver) FindDuplicateTexts(ctx context.Context, distance *int, durationDiff *float64) (ret [][]*models.Text, err error) {
	dist := 0
	durDiff := -1.
	if distance != nil {
		dist = *distance
	}
	if durationDiff != nil {
		durDiff = *durationDiff
	}
	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		ret, err = r.repository.Text.FindDuplicates(ctx, dist, durDiff)
		return err
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *queryResolver) AllTexts(ctx context.Context) (ret []*models.Text, err error) {
	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		ret, err = r.repository.Text.All(ctx)
		return err
	}); err != nil {
		return nil, err
	}

	return ret, nil
}
