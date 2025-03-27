//go:build integration
// +build integration

package sqlite_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/sliceutil"
	"github.com/stashapp/stash/pkg/sliceutil/stringslice"
	"github.com/stretchr/testify/assert"
)

func TestBookmarkFindByTextID(t *testing.T) {
	withTxn(func(ctx context.Context) error {
		mqb := db.TextBookmark

		textID := textIDs[textIdxWithBookmarks]
		bookmarks, err := mqb.FindByTextID(ctx, textID)

		if err != nil {
			t.Errorf("Error finding bookmarks: %s", err.Error())
		}

		assert.Greater(t, len(bookmarks), 0)
		for _, bookmark := range bookmarks {
			assert.Equal(t, textIDs[textIdxWithBookmarks], bookmark.TextID)
		}

		bookmarks, err = mqb.FindByTextID(ctx, 0)

		if err != nil {
			t.Errorf("Error finding bookmark: %s", err.Error())
		}

		assert.Len(t, bookmarks, 0)

		return nil
	})
}

func TestBookmarkCountByTagID(t *testing.T) {
	withTxn(func(ctx context.Context) error {
		mqb := db.TextBookmark

		bookmarkCount, err := mqb.CountByTagID(ctx, tagIDs[tagIdxWithPrimaryBookmarks])

		if err != nil {
			t.Errorf("error calling CountByTagID: %s", err.Error())
		}

		assert.Equal(t, 6, bookmarkCount)

		bookmarkCount, err = mqb.CountByTagID(ctx, tagIDs[tagIdxWithBookmarks])

		if err != nil {
			t.Errorf("error calling CountByTagID: %s", err.Error())
		}

		assert.Equal(t, 2, bookmarkCount)

		bookmarkCount, err = mqb.CountByTagID(ctx, 0)

		if err != nil {
			t.Errorf("error calling CountByTagID: %s", err.Error())
		}

		assert.Equal(t, 0, bookmarkCount)

		return nil
	})
}

func TestBookmarkQueryQ(t *testing.T) {
	withTxn(func(ctx context.Context) error {
		q := getTextTitle(textIdxWithBookmarks)
		m, _, err := db.TextBookmark.Query(ctx, nil, &models.FindFilterType{
			Q: &q,
		})

		if err != nil {
			t.Errorf("Error querying text bookmarks: %s", err.Error())
		}

		if !assert.Greater(t, len(m), 0) {
			return nil
		}

		assert.Equal(t, textIDs[textIdxWithBookmarks], m[0].TextID)

		return nil
	})
}

func TestBookmarkQuerySortByTextUpdated(t *testing.T) {
	withTxn(func(ctx context.Context) error {
		sort := "texts_updated_at"
		_, _, err := db.TextBookmark.Query(ctx, nil, &models.FindFilterType{
			Sort: &sort,
		})

		if err != nil {
			t.Errorf("Error querying text bookmarks: %s", err.Error())
		}

		return nil
	})
}

func verifyIDs(t *testing.T, modifier models.CriterionModifier, values []int, results []int) {
	t.Helper()
	switch modifier {
	case models.CriterionModifierIsNull:
		assert.Len(t, results, 0)
	case models.CriterionModifierNotNull:
		assert.NotEqual(t, 0, len(results))
	case models.CriterionModifierIncludes:
		for _, v := range values {
			assert.Contains(t, results, v)
		}
	case models.CriterionModifierExcludes:
		for _, v := range values {
			assert.NotContains(t, results, v)
		}
	case models.CriterionModifierEquals:
		for _, v := range values {
			assert.Contains(t, results, v)
		}
		assert.Len(t, results, len(values))
	case models.CriterionModifierNotEquals:
		foundAll := true
		for _, v := range values {
			if !sliceutil.Contains(results, v) {
				foundAll = false
				break
			}
		}
		if foundAll && len(results) == len(values) {
			t.Errorf("expected ids not equal to %v - found %v", values, results)
		}
	}
}

func TestBookmarkQueryTags(t *testing.T) {
	type test struct {
		name           string
		bookmarkFilter *models.TextBookmarkFilterType
		findFilter     *models.FindFilterType
	}

	withTxn(func(ctx context.Context) error {
		testTags := func(t *testing.T, m *models.TextBookmark, bookmarkFilter *models.TextBookmarkFilterType) {
			tagIDs, err := db.TextBookmark.GetTagIDs(ctx, m.ID)
			if err != nil {
				t.Errorf("error getting bookmark tag ids: %v", err)
			}

			// HACK - if modifier isn't null/not null, then add the primary tag id
			if bookmarkFilter.Tags.Modifier != models.CriterionModifierIsNull && bookmarkFilter.Tags.Modifier != models.CriterionModifierNotNull {
				tagIDs = append(tagIDs, m.PrimaryTagID)
			}

			values, _ := stringslice.StringSliceToIntSlice(bookmarkFilter.Tags.Value)
			verifyIDs(t, bookmarkFilter.Tags.Modifier, values, tagIDs)
		}

		cases := []test{
			{
				"is null",
				&models.TextBookmarkFilterType{
					Tags: &models.HierarchicalMultiCriterionInput{
						Modifier: models.CriterionModifierIsNull,
					},
				},
				nil,
			},
			{
				"not null",
				&models.TextBookmarkFilterType{
					Tags: &models.HierarchicalMultiCriterionInput{
						Modifier: models.CriterionModifierNotNull,
					},
				},
				nil,
			},
			{
				"includes",
				&models.TextBookmarkFilterType{
					Tags: &models.HierarchicalMultiCriterionInput{
						Modifier: models.CriterionModifierIncludes,
						Value: []string{
							strconv.Itoa(tagIDs[tagIdxWithBookmarks]),
						},
					},
				},
				nil,
			},
			{
				"includes all",
				&models.TextBookmarkFilterType{
					Tags: &models.HierarchicalMultiCriterionInput{
						Modifier: models.CriterionModifierIncludesAll,
						Value: []string{
							strconv.Itoa(tagIDs[tagIdxWithBookmarks]),
							strconv.Itoa(tagIDs[tagIdx2WithBookmarks]),
						},
					},
				},
				nil,
			},
			{
				"equals",
				&models.TextBookmarkFilterType{
					Tags: &models.HierarchicalMultiCriterionInput{
						Modifier: models.CriterionModifierEquals,
						Value: []string{
							strconv.Itoa(tagIDs[tagIdxWithPrimaryBookmarks]),
							strconv.Itoa(tagIDs[tagIdxWithBookmarks]),
							strconv.Itoa(tagIDs[tagIdx2WithBookmarks]),
						},
					},
				},
				nil,
			},
			// not equals not supported
			// {
			// 	"not equals",
			// 	&models.TextBookmarkFilterType{
			// 		Tags: &models.HierarchicalMultiCriterionInput{
			// 			Modifier: models.CriterionModifierNotEquals,
			// 			Value: []string{
			// 				strconv.Itoa(tagIDs[tagIdx2WithText]),
			// 				strconv.Itoa(tagIDs[tagIdx3WithText]),
			// 			},
			// 		},
			// 	},
			// 	nil,
			// },
			{
				"excludes",
				&models.TextBookmarkFilterType{
					Tags: &models.HierarchicalMultiCriterionInput{
						Modifier: models.CriterionModifierIncludes,
						Value: []string{
							strconv.Itoa(tagIDs[tagIdx2WithBookmarks]),
						},
					},
				},
				nil,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				bookmarks := queryBookmarks(ctx, t, db.TextBookmark, tc.bookmarkFilter, tc.findFilter)
				assert.Greater(t, len(bookmarks), 0)
				for _, m := range bookmarks {
					testTags(t, m, tc.bookmarkFilter)
				}
			})
		}

		return nil
	})
}

func TestBookmarkQueryTextTags(t *testing.T) {
	type test struct {
		name           string
		bookmarkFilter *models.TextBookmarkFilterType
		findFilter     *models.FindFilterType
	}

	withTxn(func(ctx context.Context) error {
		testTags := func(t *testing.T, m *models.TextBookmark, bookmarkFilter *models.TextBookmarkFilterType) {
			s, err := db.Text.Find(ctx, m.TextID)
			if err != nil {
				t.Errorf("error getting bookmark tag ids: %v", err)
				return
			}

			if err := s.LoadTagIDs(ctx, db.Text); err != nil {
				t.Errorf("error getting bookmark tag ids: %v", err)
				return
			}

			tagIDs := s.TagIDs.List()
			values, _ := stringslice.StringSliceToIntSlice(bookmarkFilter.TextTags.Value)
			verifyIDs(t, bookmarkFilter.TextTags.Modifier, values, tagIDs)
		}

		cases := []test{
			{
				"is null",
				&models.TextBookmarkFilterType{
					TextTags: &models.HierarchicalMultiCriterionInput{
						Modifier: models.CriterionModifierIsNull,
					},
				},
				nil,
			},
			{
				"not null",
				&models.TextBookmarkFilterType{
					TextTags: &models.HierarchicalMultiCriterionInput{
						Modifier: models.CriterionModifierNotNull,
					},
				},
				nil,
			},
			{
				"includes",
				&models.TextBookmarkFilterType{
					TextTags: &models.HierarchicalMultiCriterionInput{
						Modifier: models.CriterionModifierIncludes,
						Value: []string{
							strconv.Itoa(tagIDs[tagIdx3WithText]),
						},
					},
				},
				nil,
			},
			{
				"includes all",
				&models.TextBookmarkFilterType{
					TextTags: &models.HierarchicalMultiCriterionInput{
						Modifier: models.CriterionModifierIncludesAll,
						Value: []string{
							strconv.Itoa(tagIDs[tagIdx2WithText]),
							strconv.Itoa(tagIDs[tagIdx3WithText]),
						},
					},
				},
				nil,
			},
			{
				"equals",
				&models.TextBookmarkFilterType{
					TextTags: &models.HierarchicalMultiCriterionInput{
						Modifier: models.CriterionModifierEquals,
						Value: []string{
							strconv.Itoa(tagIDs[tagIdx2WithText]),
							strconv.Itoa(tagIDs[tagIdx3WithText]),
						},
					},
				},
				nil,
			},
			// not equals not supported
			// {
			// 	"not equals",
			// 	&models.TextBookmarkFilterType{
			// 		TextTags: &models.HierarchicalMultiCriterionInput{
			// 			Modifier: models.CriterionModifierNotEquals,
			// 			Value: []string{
			// 				strconv.Itoa(tagIDs[tagIdx2WithText]),
			// 				strconv.Itoa(tagIDs[tagIdx3WithText]),
			// 			},
			// 		},
			// 	},
			// 	nil,
			// },
			{
				"excludes",
				&models.TextBookmarkFilterType{
					TextTags: &models.HierarchicalMultiCriterionInput{
						Modifier: models.CriterionModifierIncludes,
						Value: []string{
							strconv.Itoa(tagIDs[tagIdx2WithText]),
						},
					},
				},
				nil,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				bookmarks := queryBookmarks(ctx, t, db.TextBookmark, tc.bookmarkFilter, tc.findFilter)
				assert.Greater(t, len(bookmarks), 0)
				for _, m := range bookmarks {
					testTags(t, m, tc.bookmarkFilter)
				}
			})
		}

		return nil
	})
}

func queryBookmarks(ctx context.Context, t *testing.T, sqb models.TextBookmarkReader, bookmarkFilter *models.TextBookmarkFilterType, findFilter *models.FindFilterType) []*models.TextBookmark {
	t.Helper()
	result, _, err := sqb.Query(ctx, bookmarkFilter, findFilter)
	if err != nil {
		t.Errorf("Error querying bookmarks: %v", err)
	}

	return result
}

// TODO Update
// TODO Destroy
// TODO Find
// TODO GetBookmarkStrings
// TODO Wall
// TODO Count
// TODO All
// TODO Query
