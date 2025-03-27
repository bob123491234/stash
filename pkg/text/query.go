package text

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/stashapp/stash/pkg/job"
	"github.com/stashapp/stash/pkg/models"
)

// QueryOptions returns a TextQueryOptions populated with the provided filters.
func QueryOptions(textFilter *models.TextFilterType, findFilter *models.FindFilterType, count bool) models.TextQueryOptions {
	return models.TextQueryOptions{
		QueryOptions: models.QueryOptions{
			FindFilter: findFilter,
			Count:      count,
		},
		TextFilter: textFilter,
	}
}

// QueryWithCount queries for texts, returning the text objects and the total count.
func QueryWithCount(ctx context.Context, qb models.TextQueryer, textFilter *models.TextFilterType, findFilter *models.FindFilterType) ([]*models.Text, int, error) {
	// this was moved from the queryBuilder code
	// left here so that calling functions can reference this instead
	result, err := qb.Query(ctx, QueryOptions(textFilter, findFilter, true))
	if err != nil {
		return nil, 0, err
	}

	texts, err := result.Resolve(ctx)
	if err != nil {
		return nil, 0, err
	}

	return texts, result.Count, nil
}

// Query queries for texts using the provided filters.
func Query(ctx context.Context, qb models.TextQueryer, textFilter *models.TextFilterType, findFilter *models.FindFilterType) ([]*models.Text, error) {
	result, err := qb.Query(ctx, QueryOptions(textFilter, findFilter, false))
	if err != nil {
		return nil, err
	}

	texts, err := result.Resolve(ctx)
	if err != nil {
		return nil, err
	}

	return texts, nil
}

func BatchProcess(ctx context.Context, reader models.TextQueryer, textFilter *models.TextFilterType, findFilter *models.FindFilterType, fn func(text *models.Text) error) error {
	const batchSize = 1000

	if findFilter == nil {
		findFilter = &models.FindFilterType{}
	}

	page := 1
	perPage := batchSize
	findFilter.Page = &page
	findFilter.PerPage = &perPage

	for more := true; more; {
		if job.IsCancelled(ctx) {
			return nil
		}

		texts, err := Query(ctx, reader, textFilter, findFilter)
		if err != nil {
			return fmt.Errorf("error querying for texts: %w", err)
		}

		for _, text := range texts {
			if err := fn(text); err != nil {
				return err
			}
		}

		if len(texts) != batchSize {
			more = false
		} else {
			*findFilter.Page++
		}
	}

	return nil
}

// FilterFromPaths creates a TextFilterType that filters using the provided
// paths.
func FilterFromPaths(paths []string) *models.TextFilterType {
	ret := &models.TextFilterType{}
	or := ret
	sep := string(filepath.Separator)

	for _, p := range paths {
		if !strings.HasSuffix(p, sep) {
			p += sep
		}

		if ret.Path == nil {
			or = ret
		} else {
			newOr := &models.TextFilterType{}
			or.Or = newOr
			or = newOr
		}

		or.Path = &models.StringCriterionInput{
			Modifier: models.CriterionModifierEquals,
			Value:    p + "%",
		}
	}

	return ret
}

func CountByStudioID(ctx context.Context, r models.TextQueryer, id int, depth *int) (int, error) {
	filter := &models.TextFilterType{
		Studios: &models.HierarchicalMultiCriterionInput{
			Value:    []string{strconv.Itoa(id)},
			Modifier: models.CriterionModifierIncludes,
			Depth:    depth,
		},
	}

	return r.QueryCount(ctx, filter, nil)
}

func CountByTagID(ctx context.Context, r models.TextQueryer, id int, depth *int) (int, error) {
	filter := &models.TextFilterType{
		Tags: &models.HierarchicalMultiCriterionInput{
			Value:    []string{strconv.Itoa(id)},
			Modifier: models.CriterionModifierIncludes,
			Depth:    depth,
		},
	}

	return r.QueryCount(ctx, filter, nil)
}
