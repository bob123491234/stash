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
	"github.com/stashapp/stash/pkg/story"
	"github.com/stashapp/stash/pkg/utils"
)

// used to refetch story after hooks run
func (r *mutationResolver) getStory(ctx context.Context, id int) (ret *models.Story, err error) {
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		ret, err = r.repository.Story.Find(ctx, id)
		return err
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *mutationResolver) StoryCreate(ctx context.Context, input models.StoryCreateInput) (ret *models.Story, err error) {
	translator := changesetTranslator{
		inputMap: getUpdateInputMap(ctx),
	}

	fileIDs, err := translator.fileIDSliceFromStringSlice(input.FileIds)
	if err != nil {
		return nil, fmt.Errorf("converting file ids: %w", err)
	}

	// Populate a new story from the input
	newStory := models.NewStory()

	newStory.Title = translator.string(input.Title)
	newStory.TagLine = translator.string(input.TagLine)
	newStory.Code = translator.string(input.Code)
	newStory.Content = translator.string(input.Content)
	newStory.Details = translator.string(input.Details)
	newStory.Author = translator.string(input.Author)
	newStory.Rating = input.Rating100
	newStory.Organized = translator.bool(input.Organized)
	newStory.StashIDs = models.NewRelatedStashIDs(input.StashIds)

	newStory.DatePublished, err = translator.datePtr(input.DatePublished)
	if err != nil {
		return nil, fmt.Errorf("converting date: %w", err)
	}
	newStory.DateUpdated, err = translator.datePtr(input.DateUpdated)
	if err != nil {
		return nil, fmt.Errorf("converting date: %w", err)
	}
	newStory.StudioID, err = translator.intPtrFromString(input.StudioID)
	if err != nil {
		return nil, fmt.Errorf("converting studio id: %w", err)
	}

	if input.Urls != nil {
		newStory.URLs = models.NewRelatedStrings(input.Urls)
	}

	newStory.PerformerIDs, err = translator.relatedIds(input.PerformerIds)
	if err != nil {
		return nil, fmt.Errorf("converting performer ids: %w", err)
	}
	newStory.TagIDs, err = translator.relatedIds(input.TagIds)
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
		ret, err = r.Resolver.storyService.Create(ctx, &newStory, fileIDs, coverImageData)
		return err
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *mutationResolver) StoryUpdate(ctx context.Context, input models.StoryUpdateInput) (ret *models.Story, err error) {
	translator := changesetTranslator{
		inputMap: getUpdateInputMap(ctx),
	}

	// Start the transaction and save the story
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		ret, err = r.storyUpdate(ctx, input, translator)
		return err
	}); err != nil {
		return nil, err
	}

	r.hookExecutor.ExecutePostHooks(ctx, ret.ID, hook.StoryUpdatePost, input, translator.getFields())
	return r.getStory(ctx, ret.ID)
}

func (r *mutationResolver) StoriesUpdate(ctx context.Context, input []*models.StoryUpdateInput) (ret []*models.Story, err error) {
	inputMaps := getUpdateInputMaps(ctx)

	// Start the transaction and save the stories
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		for i, story := range input {
			translator := changesetTranslator{
				inputMap: inputMaps[i],
			}

			thisStory, err := r.storyUpdate(ctx, *story, translator)
			if err != nil {
				return err
			}

			ret = append(ret, thisStory)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	// execute post hooks outside of txn
	var newRet []*models.Story
	for i, story := range ret {
		translator := changesetTranslator{
			inputMap: inputMaps[i],
		}

		r.hookExecutor.ExecutePostHooks(ctx, story.ID, hook.StoryUpdatePost, input, translator.getFields())

		story, err = r.getStory(ctx, story.ID)
		if err != nil {
			return nil, err
		}

		newRet = append(newRet, story)
	}

	return newRet, nil
}

func storyPartialFromInput(input models.StoryUpdateInput, translator changesetTranslator) (*models.StoryPartial, error) {
	updatedStory := models.NewStoryPartial()

	updatedStory.Title = translator.optionalString(input.Title, "title")
	updatedStory.Code = translator.optionalString(input.Code, "code")
	updatedStory.Details = translator.optionalString(input.Details, "details")
	updatedStory.Director = translator.optionalString(input.Director, "director")
	updatedStory.Rating = translator.optionalInt(input.Rating100, "rating100")

	if input.OCounter != nil {
		logger.Warnf("o_counter is deprecated and no longer supported, use storyIncrementO/storyDecrementO instead")
	}

	if input.PlayCount != nil {
		logger.Warnf("play_count is deprecated and no longer supported, use storyIncrementPlayCount/storyDecrementPlayCount instead")
	}

	updatedStory.PlayDuration = translator.optionalFloat64(input.PlayDuration, "play_duration")
	updatedStory.Organized = translator.optionalBool(input.Organized, "organized")
	updatedStory.StashIDs = translator.updateStashIDs(input.StashIds, "stash_ids")

	var err error

	updatedStory.Date, err = translator.optionalDate(input.Date, "date")
	if err != nil {
		return nil, fmt.Errorf("converting date: %w", err)
	}
	updatedStory.StudioID, err = translator.optionalIntFromString(input.StudioID, "studio_id")
	if err != nil {
		return nil, fmt.Errorf("converting studio id: %w", err)
	}

	updatedStory.URLs = translator.optionalURLs(input.Urls, input.URL)

	updatedStory.PrimaryFileID, err = translator.fileIDPtrFromString(input.PrimaryFileID)
	if err != nil {
		return nil, fmt.Errorf("converting primary file id: %w", err)
	}

	updatedStory.PerformerIDs, err = translator.updateIds(input.PerformerIds, "performer_ids")
	if err != nil {
		return nil, fmt.Errorf("converting performer ids: %w", err)
	}
	updatedStory.TagIDs, err = translator.updateIds(input.TagIds, "tag_ids")
	if err != nil {
		return nil, fmt.Errorf("converting tag ids: %w", err)
	}
	updatedStory.GalleryIDs, err = translator.updateIds(input.GalleryIds, "gallery_ids")
	if err != nil {
		return nil, fmt.Errorf("converting gallery ids: %w", err)
	}

	updatedStory.MovieIDs, err = translator.updateMovieIDs(input.Movies, "movies")
	if err != nil {
		return nil, fmt.Errorf("converting movies: %w", err)
	}

	return &updatedStory, nil
}

func (r *mutationResolver) storyUpdate(ctx context.Context, input models.StoryUpdateInput, translator changesetTranslator) (*models.Story, error) {
	storyID, err := strconv.Atoi(input.ID)
	if err != nil {
		return nil, fmt.Errorf("converting id: %w", err)
	}

	qb := r.repository.Story

	originalStory, err := qb.Find(ctx, storyID)
	if err != nil {
		return nil, err
	}

	if originalStory == nil {
		return nil, fmt.Errorf("story with id %d not found", storyID)
	}

	// Populate story from the input
	updatedStory, err := storyPartialFromInput(input, translator)
	if err != nil {
		return nil, err
	}

	// ensure that title is set where story has no file
	if updatedStory.Title.Set && updatedStory.Title.Value == "" {
		if err := originalStory.LoadFiles(ctx, r.repository.Story); err != nil {
			return nil, err
		}

		if len(originalStory.Files.List()) == 0 {
			return nil, errors.New("title must be set if story has no files")
		}
	}

	if updatedStory.PrimaryFileID != nil {
		newPrimaryFileID := *updatedStory.PrimaryFileID

		// if file hash has changed, we should migrate generated files
		// after commit
		if err := originalStory.LoadFiles(ctx, r.repository.Story); err != nil {
			return nil, err
		}

		// ensure that new primary file is associated with story
		var f *models.VideoFile
		for _, ff := range originalStory.Files.List() {
			if ff.ID == newPrimaryFileID {
				f = ff
			}
		}

		if f == nil {
			return nil, fmt.Errorf("file with id %d not associated with story", newPrimaryFileID)
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

	story, err := qb.UpdatePartial(ctx, storyID, *updatedStory)
	if err != nil {
		return nil, err
	}

	if err := r.storyUpdateCoverImage(ctx, story, coverImageData); err != nil {
		return nil, err
	}

	return story, nil
}

func (r *mutationResolver) storyUpdateCoverImage(ctx context.Context, s *models.Story, coverImageData []byte) error {
	if len(coverImageData) > 0 {
		qb := r.repository.Story

		// update cover table
		if err := qb.UpdateCover(ctx, s.ID, coverImageData); err != nil {
			return err
		}
	}

	return nil
}

func (r *mutationResolver) BulkStoryUpdate(ctx context.Context, input BulkStoryUpdateInput) ([]*models.Story, error) {
	storyIDs, err := stringslice.StringSliceToIntSlice(input.Ids)
	if err != nil {
		return nil, fmt.Errorf("converting ids: %w", err)
	}

	translator := changesetTranslator{
		inputMap: getUpdateInputMap(ctx),
	}

	// Populate story from the input
	updatedStory := models.NewStoryPartial()

	updatedStory.Title = translator.optionalString(input.Title, "title")
	updatedStory.Code = translator.optionalString(input.Code, "code")
	updatedStory.Details = translator.optionalString(input.Details, "details")
	updatedStory.Director = translator.optionalString(input.Director, "director")
	updatedStory.Rating = translator.optionalInt(input.Rating100, "rating100")
	updatedStory.Organized = translator.optionalBool(input.Organized, "organized")

	updatedStory.Date, err = translator.optionalDate(input.Date, "date")
	if err != nil {
		return nil, fmt.Errorf("converting date: %w", err)
	}
	updatedStory.StudioID, err = translator.optionalIntFromString(input.StudioID, "studio_id")
	if err != nil {
		return nil, fmt.Errorf("converting studio id: %w", err)
	}

	updatedStory.URLs = translator.optionalURLsBulk(input.Urls, input.URL)

	updatedStory.PerformerIDs, err = translator.updateIdsBulk(input.PerformerIds, "performer_ids")
	if err != nil {
		return nil, fmt.Errorf("converting performer ids: %w", err)
	}
	updatedStory.TagIDs, err = translator.updateIdsBulk(input.TagIds, "tag_ids")
	if err != nil {
		return nil, fmt.Errorf("converting tag ids: %w", err)
	}
	updatedStory.GalleryIDs, err = translator.updateIdsBulk(input.GalleryIds, "gallery_ids")
	if err != nil {
		return nil, fmt.Errorf("converting gallery ids: %w", err)
	}

	updatedStory.MovieIDs, err = translator.updateMovieIDsBulk(input.MovieIds, "movie_ids")
	if err != nil {
		return nil, fmt.Errorf("converting movie ids: %w", err)
	}

	ret := []*models.Story{}

	// Start the transaction and save the stories
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Story

		for _, storyID := range storyIDs {
			story, err := qb.UpdatePartial(ctx, storyID, updatedStory)
			if err != nil {
				return err
			}

			ret = append(ret, story)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	// execute post hooks outside of txn
	var newRet []*models.Story
	for _, story := range ret {
		r.hookExecutor.ExecutePostHooks(ctx, story.ID, hook.StoryUpdatePost, input, translator.getFields())

		story, err = r.getStory(ctx, story.ID)
		if err != nil {
			return nil, err
		}

		newRet = append(newRet, story)
	}

	return newRet, nil
}

func (r *mutationResolver) StoryDestroy(ctx context.Context, input models.StoryDestroyInput) (bool, error) {
	storyID, err := strconv.Atoi(input.ID)
	if err != nil {
		return false, fmt.Errorf("converting id: %w", err)
	}

	fileNamingAlgo := manager.GetInstance().Config.GetVideoFileNamingAlgorithm()

	var s *models.Story
	fileDeleter := &story.FileDeleter{
		Deleter:        file.NewDeleter(),
		FileNamingAlgo: fileNamingAlgo,
		Paths:          manager.GetInstance().Paths,
	}

	deleteGenerated := utils.IsTrue(input.DeleteGenerated)
	deleteFile := utils.IsTrue(input.DeleteFile)

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Story
		var err error
		s, err = qb.Find(ctx, storyID)
		if err != nil {
			return err
		}

		if s == nil {
			return fmt.Errorf("story with id %d not found", storyID)
		}

		// kill any running encoders
		manager.KillRunningStreams(s, fileNamingAlgo)

		return r.storyService.Destroy(ctx, s, fileDeleter, deleteGenerated, deleteFile)
	}); err != nil {
		fileDeleter.Rollback()
		return false, err
	}

	// perform the post-commit actions
	fileDeleter.Commit()

	// call post hook after performing the other actions
	r.hookExecutor.ExecutePostHooks(ctx, s.ID, hook.StoryDestroyPost, plugin.StoryDestroyInput{
		StoryDestroyInput: input,
		Checksum:          s.Checksum,
		OSHash:            s.OSHash,
		Path:              s.Path,
	}, nil)

	return true, nil
}

func (r *mutationResolver) StoriesDestroy(ctx context.Context, input models.StoriesDestroyInput) (bool, error) {
	storyIDs, err := stringslice.StringSliceToIntSlice(input.Ids)
	if err != nil {
		return false, fmt.Errorf("converting ids: %w", err)
	}

	var stories []*models.Story
	fileNamingAlgo := manager.GetInstance().Config.GetVideoFileNamingAlgorithm()

	fileDeleter := &story.FileDeleter{
		Deleter:        file.NewDeleter(),
		FileNamingAlgo: fileNamingAlgo,
		Paths:          manager.GetInstance().Paths,
	}

	deleteGenerated := utils.IsTrue(input.DeleteGenerated)
	deleteFile := utils.IsTrue(input.DeleteFile)

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Story

		for _, id := range storyIDs {
			story, err := qb.Find(ctx, id)
			if err != nil {
				return err
			}
			if story == nil {
				return fmt.Errorf("story with id %d not found", id)
			}

			stories = append(stories, story)

			// kill any running encoders
			manager.KillRunningStreams(story, fileNamingAlgo)

			if err := r.storyService.Destroy(ctx, story, fileDeleter, deleteGenerated, deleteFile); err != nil {
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

	for _, story := range stories {
		// call post hook after performing the other actions
		r.hookExecutor.ExecutePostHooks(ctx, story.ID, hook.StoryDestroyPost, plugin.StoriesDestroyInput{
			StoriesDestroyInput: input,
			Checksum:            story.Checksum,
			OSHash:              story.OSHash,
			Path:                story.Path,
		}, nil)
	}

	return true, nil
}

func (r *mutationResolver) StoryAssignFile(ctx context.Context, input AssignStoryFileInput) (bool, error) {
	storyID, err := strconv.Atoi(input.StoryID)
	if err != nil {
		return false, fmt.Errorf("converting story id: %w", err)
	}

	fileID, err := strconv.Atoi(input.FileID)
	if err != nil {
		return false, fmt.Errorf("converting file id: %w", err)
	}

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		return r.Resolver.storyService.AssignFile(ctx, storyID, models.FileID(fileID))
	}); err != nil {
		return false, fmt.Errorf("assigning file to story: %w", err)
	}

	return true, nil
}

func (r *mutationResolver) StoryMerge(ctx context.Context, input StoryMergeInput) (*models.Story, error) {
	srcIDs, err := stringslice.StringSliceToIntSlice(input.Source)
	if err != nil {
		return nil, fmt.Errorf("converting source ids: %w", err)
	}

	destID, err := strconv.Atoi(input.Destination)
	if err != nil {
		return nil, fmt.Errorf("converting destination id: %w", err)
	}

	var values *models.StoryPartial
	var coverImageData []byte

	if input.Values != nil {
		translator := changesetTranslator{
			inputMap: getNamedUpdateInputMap(ctx, "input.values"),
		}

		values, err = storyPartialFromInput(*input.Values, translator)
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
		v := models.NewStoryPartial()
		values = &v
	}

	mgr := manager.GetInstance()
	fileDeleter := &story.FileDeleter{
		Deleter:        file.NewDeleter(),
		FileNamingAlgo: mgr.Config.GetVideoFileNamingAlgorithm(),
		Paths:          mgr.Paths,
	}

	var ret *models.Story
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		if err := r.Resolver.storyService.Merge(ctx, srcIDs, destID, fileDeleter, story.MergeOptions{
			StoryPartial:       *values,
			IncludePlayHistory: utils.IsTrue(input.PlayHistory),
			IncludeOHistory:    utils.IsTrue(input.OHistory),
		}); err != nil {
			return err
		}

		ret, err = r.Resolver.repository.Story.Find(ctx, destID)
		if err != nil {
			return err
		}
		if ret == nil {
			return fmt.Errorf("story with id %d not found", destID)
		}

		return r.storyUpdateCoverImage(ctx, ret, coverImageData)
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *mutationResolver) getStoryMarker(ctx context.Context, id int) (ret *models.StoryMarker, err error) {
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		ret, err = r.repository.StoryMarker.Find(ctx, id)
		return err
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *mutationResolver) StoryMarkerCreate(ctx context.Context, input StoryMarkerCreateInput) (*models.StoryMarker, error) {
	storyID, err := strconv.Atoi(input.StoryID)
	if err != nil {
		return nil, fmt.Errorf("converting story id: %w", err)
	}

	primaryTagID, err := strconv.Atoi(input.PrimaryTagID)
	if err != nil {
		return nil, fmt.Errorf("converting primary tag id: %w", err)
	}

	// Populate a new story marker from the input
	newMarker := models.NewStoryMarker()

	newMarker.Title = input.Title
	newMarker.Seconds = input.Seconds
	newMarker.PrimaryTagID = primaryTagID
	newMarker.StoryID = storyID

	tagIDs, err := stringslice.StringSliceToIntSlice(input.TagIds)
	if err != nil {
		return nil, fmt.Errorf("converting tag ids: %w", err)
	}

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.StoryMarker

		err := qb.Create(ctx, &newMarker)
		if err != nil {
			return err
		}

		// Save the marker tags
		// If this tag is the primary tag, then let's not add it.
		tagIDs = sliceutil.Exclude(tagIDs, []int{newMarker.PrimaryTagID})
		return qb.UpdateTags(ctx, newMarker.ID, tagIDs)
	}); err != nil {
		return nil, err
	}

	r.hookExecutor.ExecutePostHooks(ctx, newMarker.ID, hook.StoryMarkerCreatePost, input, nil)
	return r.getStoryMarker(ctx, newMarker.ID)
}

func (r *mutationResolver) StoryMarkerUpdate(ctx context.Context, input StoryMarkerUpdateInput) (*models.StoryMarker, error) {
	markerID, err := strconv.Atoi(input.ID)
	if err != nil {
		return nil, fmt.Errorf("converting id: %w", err)
	}

	translator := changesetTranslator{
		inputMap: getUpdateInputMap(ctx),
	}

	// Populate story marker from the input
	updatedMarker := models.NewStoryMarkerPartial()

	updatedMarker.Title = translator.optionalString(input.Title, "title")
	updatedMarker.Seconds = translator.optionalFloat64(input.Seconds, "seconds")
	updatedMarker.StoryID, err = translator.optionalIntFromString(input.StoryID, "story_id")
	if err != nil {
		return nil, fmt.Errorf("converting story id: %w", err)
	}
	updatedMarker.PrimaryTagID, err = translator.optionalIntFromString(input.PrimaryTagID, "primary_tag_id")
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

	fileDeleter := &story.FileDeleter{
		Deleter:        file.NewDeleter(),
		FileNamingAlgo: mgr.Config.GetVideoFileNamingAlgorithm(),
		Paths:          mgr.Paths,
	}

	// Start the transaction and save the story marker
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.StoryMarker
		sqb := r.repository.Story

		// check to see if timestamp was changed
		existingMarker, err := qb.Find(ctx, markerID)
		if err != nil {
			return err
		}
		if existingMarker == nil {
			return fmt.Errorf("story marker with id %d not found", markerID)
		}

		newMarker, err := qb.UpdatePartial(ctx, markerID, updatedMarker)
		if err != nil {
			return err
		}

		existingStory, err := sqb.Find(ctx, existingMarker.StoryID)
		if err != nil {
			return err
		}
		if existingStory == nil {
			return fmt.Errorf("story with id %d not found", existingMarker.StoryID)
		}

		// remove the marker preview if the story changed or if the timestamp was changed
		if existingMarker.StoryID != newMarker.StoryID || existingMarker.Seconds != newMarker.Seconds {
			seconds := int(existingMarker.Seconds)
			if err := fileDeleter.MarkMarkerFiles(existingStory, seconds); err != nil {
				return err
			}
		}

		if tagIdsIncluded {
			// Save the marker tags
			// If this tag is the primary tag, then let's not add it.
			tagIDs = sliceutil.Exclude(tagIDs, []int{newMarker.PrimaryTagID})
			if err := qb.UpdateTags(ctx, markerID, tagIDs); err != nil {
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

	r.hookExecutor.ExecutePostHooks(ctx, markerID, hook.StoryMarkerUpdatePost, input, translator.getFields())
	return r.getStoryMarker(ctx, markerID)
}

func (r *mutationResolver) StoryMarkerDestroy(ctx context.Context, id string) (bool, error) {
	markerID, err := strconv.Atoi(id)
	if err != nil {
		return false, fmt.Errorf("converting id: %w", err)
	}

	fileNamingAlgo := manager.GetInstance().Config.GetVideoFileNamingAlgorithm()

	fileDeleter := &story.FileDeleter{
		Deleter:        file.NewDeleter(),
		FileNamingAlgo: fileNamingAlgo,
		Paths:          manager.GetInstance().Paths,
	}

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.StoryMarker
		sqb := r.repository.Story

		marker, err := qb.Find(ctx, markerID)

		if err != nil {
			return err
		}

		if marker == nil {
			return fmt.Errorf("story marker with id %d not found", markerID)
		}

		s, err := sqb.Find(ctx, marker.StoryID)
		if err != nil {
			return err
		}

		if s == nil {
			return fmt.Errorf("story with id %d not found", marker.StoryID)
		}

		return story.DestroyMarker(ctx, s, marker, qb, fileDeleter)
	}); err != nil {
		fileDeleter.Rollback()
		return false, err
	}

	// perform the post-commit actions
	fileDeleter.Commit()

	r.hookExecutor.ExecutePostHooks(ctx, markerID, hook.StoryMarkerDestroyPost, id, nil)

	return true, nil
}

func (r *mutationResolver) StorySaveActivity(ctx context.Context, id string, resumeTime *float64, playDuration *float64) (ret bool, err error) {
	storyID, err := strconv.Atoi(id)
	if err != nil {
		return false, fmt.Errorf("converting id: %w", err)
	}

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Story

		ret, err = qb.SaveActivity(ctx, storyID, resumeTime, playDuration)
		return err
	}); err != nil {
		return false, err
	}

	return ret, nil
}

// deprecated
func (r *mutationResolver) StoryIncrementPlayCount(ctx context.Context, id string) (ret int, err error) {
	storyID, err := strconv.Atoi(id)
	if err != nil {
		return 0, fmt.Errorf("converting id: %w", err)
	}

	var updatedTimes []time.Time

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Story

		updatedTimes, err = qb.AddViews(ctx, storyID, nil)
		return err
	}); err != nil {
		return 0, err
	}

	return len(updatedTimes), nil
}

func (r *mutationResolver) StoryAddPlay(ctx context.Context, id string, t []*time.Time) (*HistoryMutationResult, error) {
	storyID, err := strconv.Atoi(id)
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
		qb := r.repository.Story

		updatedTimes, err = qb.AddViews(ctx, storyID, times)
		return err
	}); err != nil {
		return nil, err
	}

	return &HistoryMutationResult{
		Count:   len(updatedTimes),
		History: sliceutil.ValuesToPtrs(updatedTimes),
	}, nil
}

func (r *mutationResolver) StoryDeletePlay(ctx context.Context, id string, t []*time.Time) (*HistoryMutationResult, error) {
	storyID, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}

	var times []time.Time

	for _, tt := range t {
		times = append(times, *tt)
	}

	var updatedTimes []time.Time

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Story

		updatedTimes, err = qb.DeleteViews(ctx, storyID, times)
		return err
	}); err != nil {
		return nil, err
	}

	return &HistoryMutationResult{
		Count:   len(updatedTimes),
		History: sliceutil.ValuesToPtrs(updatedTimes),
	}, nil
}

func (r *mutationResolver) StoryResetPlayCount(ctx context.Context, id string) (ret int, err error) {
	storyID, err := strconv.Atoi(id)
	if err != nil {
		return 0, err
	}

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Story

		ret, err = qb.DeleteAllViews(ctx, storyID)
		return err
	}); err != nil {
		return 0, err
	}

	return ret, nil
}

// deprecated
func (r *mutationResolver) StoryIncrementO(ctx context.Context, id string) (ret int, err error) {
	storyID, err := strconv.Atoi(id)
	if err != nil {
		return 0, fmt.Errorf("converting id: %w", err)
	}

	var updatedTimes []time.Time

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Story

		updatedTimes, err = qb.AddO(ctx, storyID, nil)
		return err
	}); err != nil {
		return 0, err
	}

	return len(updatedTimes), nil
}

// deprecated
func (r *mutationResolver) StoryDecrementO(ctx context.Context, id string) (ret int, err error) {
	storyID, err := strconv.Atoi(id)
	if err != nil {
		return 0, fmt.Errorf("converting id: %w", err)
	}

	var updatedTimes []time.Time

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Story

		updatedTimes, err = qb.DeleteO(ctx, storyID, nil)
		return err
	}); err != nil {
		return 0, err
	}

	return len(updatedTimes), nil
}

func (r *mutationResolver) StoryResetO(ctx context.Context, id string) (ret int, err error) {
	storyID, err := strconv.Atoi(id)
	if err != nil {
		return 0, fmt.Errorf("converting id: %w", err)
	}

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Story

		ret, err = qb.ResetO(ctx, storyID)
		return err
	}); err != nil {
		return 0, err
	}

	return ret, nil
}

func (r *mutationResolver) StoryAddO(ctx context.Context, id string, t []*time.Time) (*HistoryMutationResult, error) {
	storyID, err := strconv.Atoi(id)
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
		qb := r.repository.Story

		updatedTimes, err = qb.AddO(ctx, storyID, times)
		return err
	}); err != nil {
		return nil, err
	}

	return &HistoryMutationResult{
		Count:   len(updatedTimes),
		History: sliceutil.ValuesToPtrs(updatedTimes),
	}, nil
}

func (r *mutationResolver) StoryDeleteO(ctx context.Context, id string, t []*time.Time) (*HistoryMutationResult, error) {
	storyID, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("converting id: %w", err)
	}

	var times []time.Time

	for _, tt := range t {
		times = append(times, *tt)
	}

	var updatedTimes []time.Time

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Story

		updatedTimes, err = qb.DeleteO(ctx, storyID, times)
		return err
	}); err != nil {
		return nil, err
	}

	return &HistoryMutationResult{
		Count:   len(updatedTimes),
		History: sliceutil.ValuesToPtrs(updatedTimes),
	}, nil
}

func (r *mutationResolver) StoryGenerateScreenshot(ctx context.Context, id string, at *float64) (string, error) {
	if at != nil {
		manager.GetInstance().GenerateScreenshot(ctx, id, *at)
	} else {
		manager.GetInstance().GenerateDefaultScreenshot(ctx, id)
	}

	return "todo", nil
}
