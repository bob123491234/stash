package api

import (
	"context"
	"strconv"

	"github.com/99designs/gqlgen/graphql"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/sliceutil"
	"github.com/stashapp/stash/pkg/sliceutil/stringslice"
	"github.com/stashapp/stash/pkg/story"
)

func (r *queryResolver) FindStory(ctx context.Context, id *string, checksum *string) (*models.Story, error) {
	var story *models.Story
	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Story
		var err error
		if id != nil {
			idInt, err := strconv.Atoi(*id)
			if err != nil {
				return err
			}
			story, err = qb.Find(ctx, idInt)
			if err != nil {
				return err
			}
		} else if checksum != nil {
			var stories []*models.Story
			stories, err = qb.FindByChecksum(ctx, *checksum)
			if len(stories) > 0 {
				story = stories[0]
			}
		}

		return err
	}); err != nil {
		return nil, err
	}

	return story, nil
}

func (r *queryResolver) FindStoryByHash(ctx context.Context, input StoryHashInput) (*models.Story, error) {
	var story *models.Story

	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Story
		if input.Checksum != nil {
			stories, err := qb.FindByChecksum(ctx, *input.Checksum)
			if err != nil {
				return err
			}
			if len(stories) > 0 {
				story = stories[0]
			}
		}

		if story == nil && input.Oshash != nil {
			stories, err := qb.FindByOSHash(ctx, *input.Oshash)
			if err != nil {
				return err
			}
			if len(stories) > 0 {
				story = stories[0]
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return story, nil
}

func (r *queryResolver) FindStories(
	ctx context.Context,
	storyFilter *models.StoryFilterType,
	storyIDs []int,
	ids []string,
	filter *models.FindFilterType,
) (ret *FindStoriesResultType, err error) {
	if len(ids) > 0 {
		storyIDs, err = stringslice.StringSliceToIntSlice(ids)
		if err != nil {
			return nil, err
		}
	}

	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		var stories []*models.Story
		var err error

		fields := graphql.CollectAllFields(ctx)
		result := &models.StoryQueryResult{}

		if len(storyIDs) > 0 {
			stories, err = r.repository.Story.FindMany(ctx, storyIDs)
			if err == nil {
				result.Count = len(stories)
				for _, s := range stories {
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
			result, err = r.repository.Story.Query(ctx, models.StoryQueryOptions{
				QueryOptions: models.QueryOptions{
					FindFilter: filter,
					Count:      sliceutil.Contains(fields, "count"),
				},
				StoryFilter:   storyFilter,
				TotalDuration: sliceutil.Contains(fields, "duration"),
				TotalSize:     sliceutil.Contains(fields, "filesize"),
			})
			if err == nil {
				stories, err = result.Resolve(ctx)
			}
		}

		if err != nil {
			return err
		}

		ret = &FindStoriesResultType{
			Count:    result.Count,
			Stories:  stories,
			Duration: result.TotalDuration,
			Filesize: result.TotalSize,
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *queryResolver) FindStoriesByPathRegex(ctx context.Context, filter *models.FindFilterType) (ret *FindStoriesResultType, err error) {
	if err := r.withReadTxn(ctx, func(ctx context.Context) error {

		storyFilter := &models.StoryFilterType{}

		if filter != nil && filter.Q != nil {
			storyFilter.Path = &models.StringCriterionInput{
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

		result, err := r.repository.Story.Query(ctx, models.StoryQueryOptions{
			QueryOptions: models.QueryOptions{
				FindFilter: queryFilter,
				Count:      sliceutil.Contains(fields, "count"),
			},
			StoryFilter:   storyFilter,
			TotalDuration: sliceutil.Contains(fields, "duration"),
			TotalSize:     sliceutil.Contains(fields, "filesize"),
		})
		if err != nil {
			return err
		}

		stories, err := result.Resolve(ctx)
		if err != nil {
			return err
		}

		ret = &FindStoriesResultType{
			Count:    result.Count,
			Stories:  stories,
			Duration: result.TotalDuration,
			Filesize: result.TotalSize,
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *queryResolver) ParseStoryFilenames(ctx context.Context, filter *models.FindFilterType, config models.StoryParserInput) (ret *StoryParserResultType, err error) {
	repo := story.NewFilenameParserRepository(r.repository)
	parser := story.NewFilenameParser(filter, config, repo)

	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		result, count, err := parser.Parse(ctx)

		if err != nil {
			return err
		}

		ret = &StoryParserResultType{
			Count:   count,
			Results: result,
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *queryResolver) FindDuplicateStories(ctx context.Context, distance *int, durationDiff *float64) (ret [][]*models.Story, err error) {
	dist := 0
	durDiff := -1.
	if distance != nil {
		dist = *distance
	}
	if durationDiff != nil {
		durDiff = *durationDiff
	}
	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		ret, err = r.repository.Story.FindDuplicates(ctx, dist, durDiff)
		return err
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *queryResolver) AllStories(ctx context.Context) (ret []*models.Story, err error) {
	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		ret, err = r.repository.Story.All(ctx)
		return err
	}); err != nil {
		return nil, err
	}

	return ret, nil
}
