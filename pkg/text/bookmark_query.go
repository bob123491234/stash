package text

import (
	"context"
	"strconv"

	"github.com/stashapp/stash/pkg/models"
)

func BookmarkCountByTagID(ctx context.Context, r models.TextBookmarkQueryer, id int, depth *int) (int, error) {
	filter := &models.TextBookmarkFilterType{
		Tags: &models.HierarchicalMultiCriterionInput{
			Value:    []string{strconv.Itoa(id)},
			Modifier: models.CriterionModifierIncludes,
			Depth:    depth,
		},
	}

	return r.QueryCount(ctx, filter, nil)
}
