package api

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/pkg/file"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/plugin"
	"github.com/stashapp/stash/pkg/plugin/hook"
	"github.com/stashapp/stash/pkg/sliceutil"
	"github.com/stashapp/stash/pkg/sliceutil/stringslice"
	"github.com/stashapp/stash/pkg/text"
	"github.com/stashapp/stash/pkg/utils"
)

// used to refetch text after hooks run
func (r *mutationResolver) getText(ctx context.Context, id int) (ret *models.Text, err error) {
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		ret, err = r.repository.Text.Find(ctx, id)
		return err
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *mutationResolver) TextCreate(ctx context.Context, input models.TextCreateInput) (ret *models.Text, err error) {
	translator := changesetTranslator{
		inputMap: getUpdateInputMap(ctx),
	}

	fileIDs, err := translator.fileIDSliceFromStringSlice(input.FileIds)
	if err != nil {
		return nil, fmt.Errorf("converting file ids: %w", err)
	}

	// Populate a new text from the input
	newText := models.NewText()

	newText.Title = translator.string(input.Title)
	newText.TagLine = translator.string(input.TagLine)
	newText.Code = translator.string(input.Code)
	newText.Details = translator.string(input.Details)
	newText.Author = translator.string(input.Author)
	newText.LanguageCode = translator.string(input.LanguageCode)
	newText.Rating = input.Rating100
	newText.Organized = translator.bool(input.Organized)

	newText.Date, err = translator.datePtr(input.Date)
	if err != nil {
		return nil, fmt.Errorf("converting date: %w", err)
	}
	newText.StudioID, err = translator.intPtrFromString(input.StudioID)
	if err != nil {
		return nil, fmt.Errorf("converting studio id: %w", err)
	}

	if input.Urls != nil {
		newText.URLs = models.NewRelatedStrings(input.Urls)
	} else if input.URL != nil {
		newText.URLs = models.NewRelatedStrings([]string{*input.URL})
	}

	newText.PerformerIDs, err = translator.relatedIds(input.PerformerIds)
	if err != nil {
		return nil, fmt.Errorf("converting performer ids: %w", err)
	}
	newText.TagIDs, err = translator.relatedIds(input.TagIds)
	if err != nil {
		return nil, fmt.Errorf("converting tag ids: %w", err)
	}

	var coverImageData []byte
	if input.CoverImage != nil {
		var err error
		coverImageData, err = utils.ProcessImageInput(ctx, *input.CoverImage)
		if err != nil {
			return nil, fmt.Errorf("processing cover image: %w", err)
		}
	}

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		ret, err = r.Resolver.textService.Create(ctx, &newText, fileIDs, coverImageData)
		return err
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *mutationResolver) TextUpdate(ctx context.Context, input models.TextUpdateInput) (ret *models.Text, err error) {
	translator := changesetTranslator{
		inputMap: getUpdateInputMap(ctx),
	}

	// Start the transaction and save the text
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		ret, err = r.textUpdate(ctx, input, translator)
		return err
	}); err != nil {
		return nil, err
	}

	r.hookExecutor.ExecutePostHooks(ctx, ret.ID, hook.TextUpdatePost, input, translator.getFields())
	return r.getText(ctx, ret.ID)
}

func (r *mutationResolver) TextsUpdate(ctx context.Context, input []*models.TextUpdateInput) (ret []*models.Text, err error) {
	inputMaps := getUpdateInputMaps(ctx)

	// Start the transaction and save the texts
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		for i, text := range input {
			translator := changesetTranslator{
				inputMap: inputMaps[i],
			}

			thisText, err := r.textUpdate(ctx, *text, translator)
			if err != nil {
				return err
			}

			ret = append(ret, thisText)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	// execute post hooks outside of txn
	var newRet []*models.Text
	for i, text := range ret {
		translator := changesetTranslator{
			inputMap: inputMaps[i],
		}

		r.hookExecutor.ExecutePostHooks(ctx, text.ID, hook.TextUpdatePost, input, translator.getFields())

		text, err = r.getText(ctx, text.ID)
		if err != nil {
			return nil, err
		}

		newRet = append(newRet, text)
	}

	return newRet, nil
}

func textPartialFromInput(input models.TextUpdateInput, translator changesetTranslator) (*models.TextPartial, error) {
	updatedText := models.NewTextPartial()

	updatedText.Title = translator.optionalString(input.Title, "title")
	updatedText.TagLine = translator.optionalString(input.TagLine, "tag_line")
	updatedText.Code = translator.optionalString(input.Code, "code")
	updatedText.Details = translator.optionalString(input.Details, "details")
	updatedText.Author = translator.optionalString(input.Author, "author")
	updatedText.LanguageCode = translator.optionalString(input.LanguageCode, "language_code")
	updatedText.Rating = translator.optionalInt(input.Rating100, "rating100")

	updatedText.ReadDuration = translator.optionalFloat64(input.ReadDuration, "read_duration")
	updatedText.Organized = translator.optionalBool(input.Organized, "organized")

	var err error

	updatedText.Date, err = translator.optionalDate(input.Date, "date")
	if err != nil {
		return nil, fmt.Errorf("converting date: %w", err)
	}
	updatedText.StudioID, err = translator.optionalIntFromString(input.StudioID, "studio_id")
	if err != nil {
		return nil, fmt.Errorf("converting studio id: %w", err)
	}

	updatedText.URLs = translator.optionalURLs(input.Urls, input.URL)

	updatedText.PrimaryFileID, err = translator.fileIDPtrFromString(input.PrimaryFileID)
	if err != nil {
		return nil, fmt.Errorf("converting primary file id: %w", err)
	}

	updatedText.PerformerIDs, err = translator.updateIds(input.PerformerIds, "performer_ids")
	if err != nil {
		return nil, fmt.Errorf("converting performer ids: %w", err)
	}
	updatedText.TagIDs, err = translator.updateIds(input.TagIds, "tag_ids")
	if err != nil {
		return nil, fmt.Errorf("converting tag ids: %w", err)
	}

	return &updatedText, nil
}

func (r *mutationResolver) textUpdate(ctx context.Context, input models.TextUpdateInput, translator changesetTranslator) (*models.Text, error) {
	textID, err := strconv.Atoi(input.ID)
	if err != nil {
		return nil, fmt.Errorf("converting id: %w", err)
	}

	qb := r.repository.Text

	originalText, err := qb.Find(ctx, textID)
	if err != nil {
		return nil, err
	}

	if originalText == nil {
		return nil, fmt.Errorf("text with id %d not found", textID)
	}

	// Populate text from the input
	updatedText, err := textPartialFromInput(input, translator)
	if err != nil {
		return nil, err
	}

	// ensure that title is set where text has no file
	if updatedText.Title.Set && updatedText.Title.Value == "" {
		if err := originalText.LoadFiles(ctx, r.repository.Text); err != nil {
			return nil, err
		}

		if len(originalText.Files.List()) == 0 {
			return nil, errors.New("title must be set if text has no files")
		}
	}

	if updatedText.PrimaryFileID != nil {
		newPrimaryFileID := *updatedText.PrimaryFileID

		// if file hash has changed, we should migrate generated files
		// after commit
		if err := originalText.LoadFiles(ctx, r.repository.Text); err != nil {
			return nil, err
		}

		// ensure that new primary file is associated with text
		var f *models.BaseFile
		for _, ff := range originalText.Files.List() {
			if ff.ID == newPrimaryFileID {
				f = ff
			}
		}

		if f == nil {
			return nil, fmt.Errorf("file with id %d not associated with text", newPrimaryFileID)
		}
	}

	var coverImageData []byte
	if input.CoverImage != nil {
		var err error
		coverImageData, err = utils.ProcessImageInput(ctx, *input.CoverImage)
		if err != nil {
			return nil, fmt.Errorf("processing cover image: %w", err)
		}
	}

	text, err := qb.UpdatePartial(ctx, textID, *updatedText)
	if err != nil {
		return nil, err
	}

	if err := r.textUpdateCoverImage(ctx, text, coverImageData); err != nil {
		return nil, err
	}

	return text, nil
}

func (r *mutationResolver) textUpdateCoverImage(ctx context.Context, s *models.Text, coverImageData []byte) error {
	if len(coverImageData) > 0 {
		qb := r.repository.Text

		// update cover table
		if err := qb.UpdateCover(ctx, s.ID, coverImageData); err != nil {
			return err
		}
	}

	return nil
}

func (r *mutationResolver) BulkTextUpdate(ctx context.Context, input BulkTextUpdateInput) ([]*models.Text, error) {
	textIDs, err := stringslice.StringSliceToIntSlice(input.Ids)
	if err != nil {
		return nil, fmt.Errorf("converting ids: %w", err)
	}

	translator := changesetTranslator{
		inputMap: getUpdateInputMap(ctx),
	}

	// Populate text from the input
	updatedText := models.NewTextPartial()

	updatedText.Title = translator.optionalString(input.Title, "title")
	updatedText.TagLine = translator.optionalString(input.TagLine, "tag_line")
	updatedText.Code = translator.optionalString(input.Code, "code")
	updatedText.Details = translator.optionalString(input.Details, "details")
	updatedText.Author = translator.optionalString(input.Author, "author")
	updatedText.LanguageCode = translator.optionalString(input.LanguageCode, "language_code")
	updatedText.Rating = translator.optionalInt(input.Rating100, "rating100")
	updatedText.Organized = translator.optionalBool(input.Organized, "organized")

	updatedText.Date, err = translator.optionalDate(input.Date, "date")
	if err != nil {
		return nil, fmt.Errorf("converting date: %w", err)
	}
	updatedText.StudioID, err = translator.optionalIntFromString(input.StudioID, "studio_id")
	if err != nil {
		return nil, fmt.Errorf("converting studio id: %w", err)
	}

	updatedText.URLs = translator.optionalURLsBulk(input.Urls, input.URL)

	updatedText.PerformerIDs, err = translator.updateIdsBulk(input.PerformerIds, "performer_ids")
	if err != nil {
		return nil, fmt.Errorf("converting performer ids: %w", err)
	}
	updatedText.TagIDs, err = translator.updateIdsBulk(input.TagIds, "tag_ids")
	if err != nil {
		return nil, fmt.Errorf("converting tag ids: %w", err)
	}

	ret := []*models.Text{}

	// Start the transaction and save the texts
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Text

		for _, textID := range textIDs {
			text, err := qb.UpdatePartial(ctx, textID, updatedText)
			if err != nil {
				return err
			}

			ret = append(ret, text)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	// execute post hooks outside of txn
	var newRet []*models.Text
	for _, text := range ret {
		r.hookExecutor.ExecutePostHooks(ctx, text.ID, hook.TextUpdatePost, input, translator.getFields())

		text, err = r.getText(ctx, text.ID)
		if err != nil {
			return nil, err
		}

		newRet = append(newRet, text)
	}

	return newRet, nil
}

func (r *mutationResolver) TextDestroy(ctx context.Context, input models.TextDestroyInput) (bool, error) {
	textID, err := strconv.Atoi(input.ID)
	if err != nil {
		return false, fmt.Errorf("converting id: %w", err)
	}

	fileNamingAlgo := manager.GetInstance().Config.GetVideoFileNamingAlgorithm()

	var s *models.Text
	fileDeleter := &text.FileDeleter{
		Deleter:        file.NewDeleter(),
		FileNamingAlgo: fileNamingAlgo,
		Paths:          manager.GetInstance().Paths,
	}

	deleteGenerated := utils.IsTrue(input.DeleteGenerated)
	deleteFile := utils.IsTrue(input.DeleteFile)

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Text
		var err error
		s, err = qb.Find(ctx, textID)
		if err != nil {
			return err
		}

		if s == nil {
			return fmt.Errorf("text with id %d not found", textID)
		}

		return r.textService.Destroy(ctx, s, fileDeleter, deleteGenerated, deleteFile)
	}); err != nil {
		fileDeleter.Rollback()
		return false, err
	}

	// perform the post-commit actions
	fileDeleter.Commit()

	// call post hook after performing the other actions
	r.hookExecutor.ExecutePostHooks(ctx, s.ID, hook.TextDestroyPost, plugin.TextDestroyInput{
		TextDestroyInput: input,
		Checksum:         s.Checksum,
		Path:             s.Path,
	}, nil)

	return true, nil
}

func (r *mutationResolver) TextsDestroy(ctx context.Context, input models.TextsDestroyInput) (bool, error) {
	textIDs, err := stringslice.StringSliceToIntSlice(input.Ids)
	if err != nil {
		return false, fmt.Errorf("converting ids: %w", err)
	}

	var texts []*models.Text
	fileNamingAlgo := manager.GetInstance().Config.GetVideoFileNamingAlgorithm()

	fileDeleter := &text.FileDeleter{
		Deleter:        file.NewDeleter(),
		FileNamingAlgo: fileNamingAlgo,
		Paths:          manager.GetInstance().Paths,
	}

	deleteGenerated := utils.IsTrue(input.DeleteGenerated)
	deleteFile := utils.IsTrue(input.DeleteFile)

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Text

		for _, id := range textIDs {
			text, err := qb.Find(ctx, id)
			if err != nil {
				return err
			}
			if text == nil {
				return fmt.Errorf("text with id %d not found", id)
			}

			texts = append(texts, text)

			if err := r.textService.Destroy(ctx, text, fileDeleter, deleteGenerated, deleteFile); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		fileDeleter.Rollback()
		return false, err
	}

	// perform the post-commit actions
	fileDeleter.Commit()

	for _, text := range texts {
		// call post hook after performing the other actions
		r.hookExecutor.ExecutePostHooks(ctx, text.ID, hook.TextDestroyPost, plugin.TextsDestroyInput{
			TextsDestroyInput: input,
			Checksum:          text.Checksum,
			Path:              text.Path,
		}, nil)
	}

	return true, nil
}

func (r *mutationResolver) TextAssignFile(ctx context.Context, input AssignTextFileInput) (bool, error) {
	textID, err := strconv.Atoi(input.TextID)
	if err != nil {
		return false, fmt.Errorf("converting text id: %w", err)
	}

	fileID, err := strconv.Atoi(input.FileID)
	if err != nil {
		return false, fmt.Errorf("converting file id: %w", err)
	}

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		return r.Resolver.textService.AssignFile(ctx, textID, models.FileID(fileID))
	}); err != nil {
		return false, fmt.Errorf("assigning file to text: %w", err)
	}

	return true, nil
}

func (r *mutationResolver) TextMerge(ctx context.Context, input TextMergeInput) (*models.Text, error) {
	srcIDs, err := stringslice.StringSliceToIntSlice(input.Source)
	if err != nil {
		return nil, fmt.Errorf("converting source ids: %w", err)
	}

	destID, err := strconv.Atoi(input.Destination)
	if err != nil {
		return nil, fmt.Errorf("converting destination id: %w", err)
	}

	var values *models.TextPartial
	var coverImageData []byte

	if input.Values != nil {
		translator := changesetTranslator{
			inputMap: getNamedUpdateInputMap(ctx, "input.values"),
		}

		values, err = textPartialFromInput(*input.Values, translator)
		if err != nil {
			return nil, err
		}

		if input.Values.CoverImage != nil {
			var err error
			coverImageData, err = utils.ProcessImageInput(ctx, *input.Values.CoverImage)
			if err != nil {
				return nil, fmt.Errorf("processing cover image: %w", err)
			}
		}
	} else {
		v := models.NewTextPartial()
		values = &v
	}

	mgr := manager.GetInstance()
	fileDeleter := &text.FileDeleter{
		Deleter:        file.NewDeleter(),
		FileNamingAlgo: mgr.Config.GetVideoFileNamingAlgorithm(),
		Paths:          mgr.Paths,
	}

	var ret *models.Text
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		if err := r.Resolver.textService.Merge(ctx, srcIDs, destID, fileDeleter, text.MergeOptions{
			TextPartial:        *values,
			IncludeReadHistory: utils.IsTrue(input.ReadHistory),
			IncludeOHistory:    utils.IsTrue(input.OHistory),
		}); err != nil {
			return err
		}

		ret, err = r.Resolver.repository.Text.Find(ctx, destID)
		if err != nil {
			return err
		}
		if ret == nil {
			return fmt.Errorf("text with id %d not found", destID)
		}

		return r.textUpdateCoverImage(ctx, ret, coverImageData)
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *mutationResolver) getTextBookmark(ctx context.Context, id int) (ret *models.TextBookmark, err error) {
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		ret, err = r.repository.TextBookmark.Find(ctx, id)
		return err
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *mutationResolver) TextBookmarkCreate(ctx context.Context, input TextBookmarkCreateInput) (*models.TextBookmark, error) {
	textID, err := strconv.Atoi(input.TextID)
	if err != nil {
		return nil, fmt.Errorf("converting text id: %w", err)
	}

	primaryTagID, err := strconv.Atoi(input.PrimaryTagID)
	if err != nil {
		return nil, fmt.Errorf("converting primary tag id: %w", err)
	}

	// Populate a new text bookmark from the input
	newBookmark := models.NewTextBookmark()

	newBookmark.Title = input.Title
	newBookmark.Location = input.Location
	newBookmark.PrimaryTagID = primaryTagID
	newBookmark.TextID = textID

	tagIDs, err := stringslice.StringSliceToIntSlice(input.TagIds)
	if err != nil {
		return nil, fmt.Errorf("converting tag ids: %w", err)
	}

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.TextBookmark

		err := qb.Create(ctx, &newBookmark)
		if err != nil {
			return err
		}

		// Save the bookmark tags
		// If this tag is the primary tag, then let's not add it.
		tagIDs = sliceutil.Exclude(tagIDs, []int{newBookmark.PrimaryTagID})
		return qb.UpdateTags(ctx, newBookmark.ID, tagIDs)
	}); err != nil {
		return nil, err
	}

	r.hookExecutor.ExecutePostHooks(ctx, newBookmark.ID, hook.TextBookmarkCreatePost, input, nil)
	return r.getTextBookmark(ctx, newBookmark.ID)
}

func (r *mutationResolver) TextBookmarkUpdate(ctx context.Context, input TextBookmarkUpdateInput) (*models.TextBookmark, error) {
	bookmarkID, err := strconv.Atoi(input.ID)
	if err != nil {
		return nil, fmt.Errorf("converting id: %w", err)
	}

	translator := changesetTranslator{
		inputMap: getUpdateInputMap(ctx),
	}

	// Populate text bookmark from the input
	updatedBookmark := models.NewTextBookmarkPartial()

	updatedBookmark.Title = translator.optionalString(input.Title, "title")
	updatedBookmark.Location = translator.optionalString(input.Location, "location")
	updatedBookmark.TextID, err = translator.optionalIntFromString(input.TextID, "text_id")
	if err != nil {
		return nil, fmt.Errorf("converting text id: %w", err)
	}
	updatedBookmark.PrimaryTagID, err = translator.optionalIntFromString(input.PrimaryTagID, "primary_tag_id")
	if err != nil {
		return nil, fmt.Errorf("converting primary tag id: %w", err)
	}

	var tagIDs []int
	tagIdsIncluded := translator.hasField("tag_ids")
	if input.TagIds != nil {
		tagIDs, err = stringslice.StringSliceToIntSlice(input.TagIds)
		if err != nil {
			return nil, fmt.Errorf("converting tag ids: %w", err)
		}
	}

	mgr := manager.GetInstance()

	fileDeleter := &text.FileDeleter{
		Deleter:        file.NewDeleter(),
		FileNamingAlgo: mgr.Config.GetVideoFileNamingAlgorithm(),
		Paths:          mgr.Paths,
	}

	// Start the transaction and save the text bookmark
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.TextBookmark
		sqb := r.repository.Text

		// check to see if timestamp was changed
		existingBookmark, err := qb.Find(ctx, bookmarkID)
		if err != nil {
			return err
		}
		if existingBookmark == nil {
			return fmt.Errorf("text bookmark with id %d not found", bookmarkID)
		}

		newBookmark, err := qb.UpdatePartial(ctx, bookmarkID, updatedBookmark)
		if err != nil {
			return err
		}

		existingText, err := sqb.Find(ctx, existingBookmark.TextID)
		if err != nil {
			return err
		}
		if existingText == nil {
			return fmt.Errorf("text with id %d not found", existingBookmark.TextID)
		}

		if tagIdsIncluded {
			// Save the bookmark tags
			// If this tag is the primary tag, then let's not add it.
			tagIDs = sliceutil.Exclude(tagIDs, []int{newBookmark.PrimaryTagID})
			if err := qb.UpdateTags(ctx, bookmarkID, tagIDs); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		fileDeleter.Rollback()
		return nil, err
	}

	// perform the post-commit actions
	fileDeleter.Commit()

	r.hookExecutor.ExecutePostHooks(ctx, bookmarkID, hook.TextBookmarkUpdatePost, input, translator.getFields())
	return r.getTextBookmark(ctx, bookmarkID)
}

func (r *mutationResolver) TextBookmarkDestroy(ctx context.Context, id string) (bool, error) {
	bookmarkID, err := strconv.Atoi(id)
	if err != nil {
		return false, fmt.Errorf("converting id: %w", err)
	}

	fileNamingAlgo := manager.GetInstance().Config.GetVideoFileNamingAlgorithm()

	fileDeleter := &text.FileDeleter{
		Deleter:        file.NewDeleter(),
		FileNamingAlgo: fileNamingAlgo,
		Paths:          manager.GetInstance().Paths,
	}

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.TextBookmark
		sqb := r.repository.Text

		bookmark, err := qb.Find(ctx, bookmarkID)

		if err != nil {
			return err
		}

		if bookmark == nil {
			return fmt.Errorf("text bookmark with id %d not found", bookmarkID)
		}

		s, err := sqb.Find(ctx, bookmark.TextID)
		if err != nil {
			return err
		}

		if s == nil {
			return fmt.Errorf("text with id %d not found", bookmark.TextID)
		}

		return text.DestroyBookmark(ctx, s, bookmark, qb, fileDeleter)
	}); err != nil {
		fileDeleter.Rollback()
		return false, err
	}

	// perform the post-commit actions
	fileDeleter.Commit()

	r.hookExecutor.ExecutePostHooks(ctx, bookmarkID, hook.TextBookmarkDestroyPost, id, nil)

	return true, nil
}

func (r *mutationResolver) TextSaveActivity(ctx context.Context, id string, resumeTime *float64, playDuration *float64) (ret bool, err error) {
	textID, err := strconv.Atoi(id)
	if err != nil {
		return false, fmt.Errorf("converting id: %w", err)
	}

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Text

		ret, err = qb.SaveActivity(ctx, textID, resumeTime, playDuration)
		return err
	}); err != nil {
		return false, err
	}

	return ret, nil
}

func (r *mutationResolver) TextAddRead(ctx context.Context, id string, t []*time.Time) (*HistoryMutationResult, error) {
	textID, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("converting id: %w", err)
	}

	var times []time.Time

	// convert time to local time, so that sorting is consistent
	for _, tt := range t {
		times = append(times, tt.Local())
	}

	var updatedTimes []time.Time

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Text

		updatedTimes, err = qb.AddReads(ctx, textID, times)
		return err
	}); err != nil {
		return nil, err
	}

	return &HistoryMutationResult{
		Count:   len(updatedTimes),
		History: sliceutil.ValuesToPtrs(updatedTimes),
	}, nil
}

func (r *mutationResolver) TextDeleteRead(ctx context.Context, id string, t []*time.Time) (*HistoryMutationResult, error) {
	textID, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}

	var times []time.Time

	for _, tt := range t {
		times = append(times, *tt)
	}

	var updatedTimes []time.Time

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Text

		updatedTimes, err = qb.DeleteReads(ctx, textID, times)
		return err
	}); err != nil {
		return nil, err
	}

	return &HistoryMutationResult{
		Count:   len(updatedTimes),
		History: sliceutil.ValuesToPtrs(updatedTimes),
	}, nil
}

func (r *mutationResolver) TextResetReadCount(ctx context.Context, id string) (ret int, err error) {
	textID, err := strconv.Atoi(id)
	if err != nil {
		return 0, err
	}

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Text

		ret, err = qb.DeleteAllReads(ctx, textID)
		return err
	}); err != nil {
		return 0, err
	}

	return ret, nil
}

func (r *mutationResolver) TextResetO(ctx context.Context, id string) (ret int, err error) {
	textID, err := strconv.Atoi(id)
	if err != nil {
		return 0, fmt.Errorf("converting id: %w", err)
	}

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Text

		ret, err = qb.ResetO(ctx, textID)
		return err
	}); err != nil {
		return 0, err
	}

	return ret, nil
}

func (r *mutationResolver) TextAddO(ctx context.Context, id string, t []*time.Time) (*HistoryMutationResult, error) {
	textID, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("converting id: %w", err)
	}

	var times []time.Time

	// convert time to local time, so that sorting is consistent
	for _, tt := range t {
		times = append(times, tt.Local())
	}

	var updatedTimes []time.Time

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Text

		updatedTimes, err = qb.AddO(ctx, textID, times)
		return err
	}); err != nil {
		return nil, err
	}

	return &HistoryMutationResult{
		Count:   len(updatedTimes),
		History: sliceutil.ValuesToPtrs(updatedTimes),
	}, nil
}

func (r *mutationResolver) TextDeleteO(ctx context.Context, id string, t []*time.Time) (*HistoryMutationResult, error) {
	textID, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("converting id: %w", err)
	}

	var times []time.Time

	for _, tt := range t {
		times = append(times, *tt)
	}

	var updatedTimes []time.Time

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Text

		updatedTimes, err = qb.DeleteO(ctx, textID, times)
		return err
	}); err != nil {
		return nil, err
	}

	return &HistoryMutationResult{
		Count:   len(updatedTimes),
		History: sliceutil.ValuesToPtrs(updatedTimes),
	}, nil
}
