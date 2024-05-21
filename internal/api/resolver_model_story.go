package api

import (
	"context"
	"time"

	"github.com/stashapp/stash/internal/api/loaders"
	"github.com/stashapp/stash/internal/api/urlbuilders"
	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/pkg/models"
)

func (r *storyResolver) getPrimaryFile(ctx context.Context, obj *models.Story) (*models.VideoFile, error) {
	if obj.PrimaryFileID != nil {
		f, err := loaders.From(ctx).FileByID.Load(*obj.PrimaryFileID)
		if err != nil {
			return nil, err
		}

		ret, err := convertVideoFile(f)
		if err != nil {
			return nil, err
		}

		obj.Files.SetPrimary(ret)

		return ret, nil
	} else {
		_ = obj.LoadPrimaryFile(ctx, r.repository.File)
	}

	return nil, nil
}

func (r *storyResolver) getFiles(ctx context.Context, obj *models.Story) ([]*models.VideoFile, error) {
	fileIDs, err := loaders.From(ctx).StoryFiles.Load(obj.ID)
	if err != nil {
		return nil, err
	}

	files, errs := loaders.From(ctx).FileByID.LoadAll(fileIDs)
	err = firstError(errs)
	if err != nil {
		return nil, err
	}

	ret := make([]*models.VideoFile, len(files))
	for i, f := range files {
		ret[i], err = convertVideoFile(f)
		if err != nil {
			return nil, err
		}
	}

	obj.Files.Set(ret)

	return ret, nil
}

func (r *storyResolver) Date(ctx context.Context, obj *models.Story) (*string, error) {
	if obj.Date != nil {
		result := obj.Date.String()
		return &result, nil
	}
	return nil, nil
}

func (r *storyResolver) Files(ctx context.Context, obj *models.Story) ([]*VideoFile, error) {
	files, err := r.getFiles(ctx, obj)
	if err != nil {
		return nil, err
	}

	ret := make([]*VideoFile, len(files))

	for i, f := range files {
		ret[i] = &VideoFile{
			VideoFile: f,
		}
	}

	return ret, nil
}

func (r *storyResolver) Rating(ctx context.Context, obj *models.Story) (*int, error) {
	if obj.Rating != nil {
		rating := models.Rating100To5(*obj.Rating)
		return &rating, nil
	}
	return nil, nil
}

func (r *storyResolver) Rating100(ctx context.Context, obj *models.Story) (*int, error) {
	return obj.Rating, nil
}

func (r *storyResolver) Paths(ctx context.Context, obj *models.Story) (*StoryPathsType, error) {
	baseURL, _ := ctx.Value(BaseURLCtxKey).(string)
	config := manager.GetInstance().Config
	builder := urlbuilders.NewStoryURLBuilder(baseURL, obj)
	screenshotPath := builder.GetScreenshotURL()

	return &StoryPathsType{
		Screenshot: &screenshotPath,
	}, nil
}

func (r *storyResolver) StoryBookmarks(ctx context.Context, obj *models.Story) (ret []*models.StoryMarker, err error) {
	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		ret, err = r.repository.StoryBookmark.FindByStoryID(ctx, obj.ID)
		return err
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *storyResolver) Studio(ctx context.Context, obj *models.Story) (ret *models.Studio, err error) {
	if obj.StudioID == nil {
		return nil, nil
	}

	return loaders.From(ctx).StudioByID.Load(*obj.StudioID)
}

func (r *storyResolver) Tags(ctx context.Context, obj *models.Story) (ret []*models.Tag, err error) {
	if !obj.TagIDs.Loaded() {
		if err := r.withReadTxn(ctx, func(ctx context.Context) error {
			return obj.LoadTagIDs(ctx, r.repository.Story)
		}); err != nil {
			return nil, err
		}
	}

	var errs []error
	ret, errs = loaders.From(ctx).TagByID.LoadAll(obj.TagIDs.List())
	return ret, firstError(errs)
}

func (r *storyResolver) Performers(ctx context.Context, obj *models.Story) (ret []*models.Performer, err error) {
	if !obj.PerformerIDs.Loaded() {
		if err := r.withReadTxn(ctx, func(ctx context.Context) error {
			return obj.LoadPerformerIDs(ctx, r.repository.Story)
		}); err != nil {
			return nil, err
		}
	}

	var errs []error
	ret, errs = loaders.From(ctx).PerformerByID.LoadAll(obj.PerformerIDs.List())
	return ret, firstError(errs)
}

func (r *storyResolver) Urls(ctx context.Context, obj *models.Story) ([]string, error) {
	if !obj.URLs.Loaded() {
		if err := r.withReadTxn(ctx, func(ctx context.Context) error {
			return obj.LoadURLs(ctx, r.repository.Story)
		}); err != nil {
			return nil, err
		}
	}

	return obj.URLs.List(), nil
}

func (r *storyResolver) OCounter(ctx context.Context, obj *models.Story) (*int, error) {
	ret, err := loaders.From(ctx).StoryOCount.Load(obj.ID)
	if err != nil {
		return nil, err
	}

	return &ret, nil
}

func (r *storyResolver) LastReadAt(ctx context.Context, obj *models.Story) (*time.Time, error) {
	ret, err := loaders.From(ctx).StoryLastRead.Load(obj.ID)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *storyResolver) ReadCount(ctx context.Context, obj *models.Story) (*int, error) {
	ret, err := loaders.From(ctx).StoryReadCount.Load(obj.ID)
	if err != nil {
		return nil, err
	}

	return &ret, nil
}

func (r *storyResolver) ReadHistory(ctx context.Context, obj *models.Story) ([]*time.Time, error) {
	ret, err := loaders.From(ctx).StoryReadHistory.Load(obj.ID)
	if err != nil {
		return nil, err
	}

	// convert to pointer slice
	ptrRet := make([]*time.Time, len(ret))
	for i, t := range ret {
		tt := t
		ptrRet[i] = &tt
	}

	return ptrRet, nil
}

func (r *storyResolver) OHistory(ctx context.Context, obj *models.Story) ([]*time.Time, error) {
	ret, err := loaders.From(ctx).StoryOHistory.Load(obj.ID)
	if err != nil {
		return nil, err
	}

	// convert to pointer slice
	ptrRet := make([]*time.Time, len(ret))
	for i, t := range ret {
		tt := t
		ptrRet[i] = &tt
	}

	return ptrRet, nil
}
