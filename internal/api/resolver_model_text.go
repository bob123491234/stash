package api

import (
	"context"
	"fmt"
	"time"

	"github.com/stashapp/stash/internal/api/loaders"
	"github.com/stashapp/stash/internal/api/urlbuilders"
	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/pkg/models"
)

func (r *textResolver) getPrimaryFile(ctx context.Context, obj *models.Text) (*models.VideoFile, error) {
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

func (r *textResolver) getFiles(ctx context.Context, obj *models.Text) ([]*models.BaseFile, error) {
	fileIDs, err := loaders.From(ctx).TextFiles.Load(obj.ID)
	if err != nil {
		return nil, err
	}

	files, errs := loaders.From(ctx).FileByID.LoadAll(fileIDs)
	err = firstError(errs)
	if err != nil {
		return nil, err
	}

	ret := make([]*models.BaseFile, len(files))
	for i, f := range files {
		ret[i], err = convertBaseFile(f)
		if err != nil {
			return nil, err
		}
	}

	obj.Files.Set(ret)

	return ret, nil
}

func (r *textResolver) Date(ctx context.Context, obj *models.Text) (*string, error) {
	if obj.Date != nil {
		result := obj.Date.String()
		return &result, nil
	}
	return nil, nil
}

func (r *textResolver) Files(ctx context.Context, obj *models.Text) ([]BaseFile, error) {
	files, err := r.getFiles(ctx, obj)
	if err != nil {
		return nil, err
	}

	ret := make([]*BaseFile, len(files))

	for i, f := range files {
		ret[i] = &BaseFile{
			BaseFile: f,
		}
	}

	return ret, nil
}

func (r *textResolver) Rating(ctx context.Context, obj *models.Text) (*int, error) {
	if obj.Rating != nil {
		rating := models.Rating100To5(*obj.Rating)
		return &rating, nil
	}
	return nil, nil
}

func (r *textResolver) Rating100(ctx context.Context, obj *models.Text) (*int, error) {
	return obj.Rating, nil
}

func (r *textResolver) Paths(ctx context.Context, obj *models.Text) (*TextPathsType, error) {
	baseURL, _ := ctx.Value(BaseURLCtxKey).(string)
	config := manager.GetInstance().Config
	builder := urlbuilders.NewTextURLBuilder(baseURL, obj)

	return &TextPathsType{}, nil
}

func (r *textResolver) TextBookmarks(ctx context.Context, obj *models.Text) (ret []*models.TextBookmark, err error) {
	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		ret, err = r.repository.TextBookmark.FindByTextID(ctx, obj.ID)
		return err
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *textResolver) Studio(ctx context.Context, obj *models.Text) (ret *models.Studio, err error) {
	if obj.StudioID == nil {
		return nil, nil
	}

	return loaders.From(ctx).StudioByID.Load(*obj.StudioID)
}

func (r *textResolver) Tags(ctx context.Context, obj *models.Text) (ret []*models.Tag, err error) {
	if !obj.TagIDs.Loaded() {
		if err := r.withReadTxn(ctx, func(ctx context.Context) error {
			return obj.LoadTagIDs(ctx, r.repository.Text)
		}); err != nil {
			return nil, err
		}
	}

	var errs []error
	ret, errs = loaders.From(ctx).TagByID.LoadAll(obj.TagIDs.List())
	return ret, firstError(errs)
}

func (r *textResolver) Performers(ctx context.Context, obj *models.Text) (ret []*models.Performer, err error) {
	if !obj.PerformerIDs.Loaded() {
		if err := r.withReadTxn(ctx, func(ctx context.Context) error {
			return obj.LoadPerformerIDs(ctx, r.repository.Text)
		}); err != nil {
			return nil, err
		}
	}

	var errs []error
	ret, errs = loaders.From(ctx).PerformerByID.LoadAll(obj.PerformerIDs.List())
	return ret, firstError(errs)
}

func (r *textResolver) URL(ctx context.Context, obj *models.Text) (*string, error) {
	if !obj.URLs.Loaded() {
		if err := r.withReadTxn(ctx, func(ctx context.Context) error {
			return obj.LoadURLs(ctx, r.repository.Text)
		}); err != nil {
			return nil, err
		}
	}

	urls := obj.URLs.List()
	if len(urls) == 0 {
		return nil, nil
	}

	return &urls[0], nil
}

func (r *textResolver) Urls(ctx context.Context, obj *models.Text) ([]string, error) {
	if !obj.URLs.Loaded() {
		if err := r.withReadTxn(ctx, func(ctx context.Context) error {
			return obj.LoadURLs(ctx, r.repository.Text)
		}); err != nil {
			return nil, err
		}
	}

	return obj.URLs.List(), nil
}

func (r *textResolver) OCounter(ctx context.Context, obj *models.Text) (*int, error) {
	ret, err := loaders.From(ctx).TextOCount.Load(obj.ID)
	if err != nil {
		return nil, err
	}

	return &ret, nil
}

func (r *textResolver) LastReadAt(ctx context.Context, obj *models.Text) (*time.Time, error) {
	ret, err := loaders.From(ctx).TextLastRead.Load(obj.ID)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *textResolver) ReadCount(ctx context.Context, obj *models.Text) (*int, error) {
	ret, err := loaders.From(ctx).TextReadCount.Load(obj.ID)
	if err != nil {
		return nil, err
	}

	return &ret, nil
}

func (r *textResolver) ReadHistory(ctx context.Context, obj *models.Text) ([]*time.Time, error) {
	ret, err := loaders.From(ctx).TextReadHistory.Load(obj.ID)
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

func (r *textResolver) OHistory(ctx context.Context, obj *models.Text) ([]*time.Time, error) {
	ret, err := loaders.From(ctx).TextOHistory.Load(obj.ID)
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
