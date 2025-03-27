//go:build integration
// +build integration

package sqlite_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChapterFindByTextID(t *testing.T) {
	withTxn(func(ctx context.Context) error {
		mqb := db.TextChapter

		textID := textIDs[textIdxWithChapters]
		chapters, err := mqb.FindByTextID(ctx, textID)

		if err != nil {
			t.Errorf("Error finding chapters: %s", err.Error())
		}

		assert.Greater(t, len(chapters), 0)
		for _, chapter := range chapters {
			assert.Equal(t, textIDs[textIdxWithChapters], chapter.TextID)
		}

		chapters, err = mqb.FindByTextID(ctx, 0)

		if err != nil {
			t.Errorf("Error finding chapter: %s", err.Error())
		}

		assert.Len(t, chapters, 0)

		return nil
	})
}

// TODO Update
// TODO Destroy
// TODO Find
