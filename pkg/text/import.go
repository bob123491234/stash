package text

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/models/json"
	"github.com/stashapp/stash/pkg/models/jsonschema"
	"github.com/stashapp/stash/pkg/sliceutil"
	"github.com/stashapp/stash/pkg/utils"
)

type ImporterReaderWriter interface {
	models.TextCreatorUpdater
	models.ReadHistoryWriter
	models.OHistoryWriter
	FindByFileID(ctx context.Context, fileID models.FileID) ([]*models.Text, error)
}

type Importer struct {
	ReaderWriter        ImporterReaderWriter
	FileFinder          models.FileFinder
	StudioWriter        models.StudioFinderCreator
	PerformerWriter     models.PerformerFinderCreator
	TagWriter           models.TagFinderCreator
	Input               jsonschema.Text
	MissingRefBehaviour models.ImportMissingRefEnum

	ID             int
	text           models.Text
	coverImageData []byte
	readHistory    []time.Time
	oHistory       []time.Time
}

func (i *Importer) PreImport(ctx context.Context) error {
	i.text = i.textJSONToText(i.Input)

	if err := i.populateFiles(ctx); err != nil {
		return err
	}

	if err := i.populateStudio(ctx); err != nil {
		return err
	}

	if err := i.populatePerformers(ctx); err != nil {
		return err
	}

	if err := i.populateTags(ctx); err != nil {
		return err
	}

	var err error
	if len(i.Input.Cover) > 0 {
		i.coverImageData, err = utils.ProcessBase64Image(i.Input.Cover)
		if err != nil {
			return fmt.Errorf("invalid cover image: %v", err)
		}
	}

	i.populateReadHistory()
	i.populateOHistory()

	return nil
}

func (i *Importer) textJSONToText(textJSON jsonschema.Text) models.Text {
	newText := models.Text{
		Title:        textJSON.Title,
		Code:         textJSON.Code,
		Details:      textJSON.Details,
		Author:       textJSON.Author,
		PerformerIDs: models.NewRelatedIDs([]int{}),
		TagIDs:       models.NewRelatedIDs([]int{}),
	}

	if len(textJSON.URLs) > 0 {
		newText.URLs = models.NewRelatedStrings(textJSON.URLs)
	}

	if textJSON.Date != "" {
		d, err := models.ParseDate(textJSON.Date)
		if err == nil {
			newText.Date = &d
		}
	}
	if textJSON.Rating != 0 {
		newText.Rating = &textJSON.Rating
	}

	newText.Organized = textJSON.Organized
	newText.CreatedAt = textJSON.CreatedAt.GetTime()
	newText.UpdatedAt = textJSON.UpdatedAt.GetTime()
	newText.ResumeLocation = textJSON.ResumeLocation
	newText.ReadDuration = textJSON.ReadDuration

	return newText
}

func getHistory(historyJSON []json.JSONTime, createdAt json.JSONTime) []time.Time {
	var ret []time.Time

	if len(historyJSON) > 0 {
		for _, d := range historyJSON {
			ret = append(ret, d.GetTime())
		}
	}

	return ret
}

func (i *Importer) populateReadHistory() {
	i.readHistory = getHistory(
		i.Input.ReadHistory,
		i.Input.CreatedAt,
	)
}

func (i *Importer) populateOHistory() {
	i.readHistory = getHistory(
		i.Input.OHistory,
		i.Input.CreatedAt,
	)
}

func (i *Importer) populateFiles(ctx context.Context) error {
	files := make([]models.File, 0)

	for _, ref := range i.Input.Files {
		path := ref
		f, err := i.FileFinder.FindByPath(ctx, path)
		if err != nil {
			return fmt.Errorf("error finding file: %w", err)
		}

		if f == nil {
			return fmt.Errorf("text file '%s' not found", path)
		} else {
			files = append(files, f.(models.File))
		}
	}

	i.text.Files = models.NewRelatedFiles(files)

	return nil
}

func (i *Importer) populateStudio(ctx context.Context) error {
	if i.Input.Studio != "" {
		studio, err := i.StudioWriter.FindByName(ctx, i.Input.Studio, false)
		if err != nil {
			return fmt.Errorf("error finding studio by name: %v", err)
		}

		if studio == nil {
			if i.MissingRefBehaviour == models.ImportMissingRefEnumFail {
				return fmt.Errorf("text studio '%s' not found", i.Input.Studio)
			}

			if i.MissingRefBehaviour == models.ImportMissingRefEnumIgnore {
				return nil
			}

			if i.MissingRefBehaviour == models.ImportMissingRefEnumCreate {
				studioID, err := i.createStudio(ctx, i.Input.Studio)
				if err != nil {
					return err
				}
				i.text.StudioID = &studioID
			}
		} else {
			i.text.StudioID = &studio.ID
		}
	}

	return nil
}

func (i *Importer) createStudio(ctx context.Context, name string) (int, error) {
	newStudio := models.NewStudio()
	newStudio.Name = name

	err := i.StudioWriter.Create(ctx, &newStudio)
	if err != nil {
		return 0, err
	}

	return newStudio.ID, nil
}

func (i *Importer) populatePerformers(ctx context.Context) error {
	if len(i.Input.Performers) > 0 {
		names := i.Input.Performers
		performers, err := i.PerformerWriter.FindByNames(ctx, names, false)
		if err != nil {
			return err
		}

		var pluckedNames []string
		for _, performer := range performers {
			if performer.Name == "" {
				continue
			}
			pluckedNames = append(pluckedNames, performer.Name)
		}

		missingPerformers := sliceutil.Filter(names, func(name string) bool {
			return !sliceutil.Contains(pluckedNames, name)
		})

		if len(missingPerformers) > 0 {
			if i.MissingRefBehaviour == models.ImportMissingRefEnumFail {
				return fmt.Errorf("text performers [%s] not found", strings.Join(missingPerformers, ", "))
			}

			if i.MissingRefBehaviour == models.ImportMissingRefEnumCreate {
				createdPerformers, err := i.createPerformers(ctx, missingPerformers)
				if err != nil {
					return fmt.Errorf("error creating text performers: %v", err)
				}

				performers = append(performers, createdPerformers...)
			}

			// ignore if MissingRefBehaviour set to Ignore
		}

		for _, p := range performers {
			i.text.PerformerIDs.Add(p.ID)
		}
	}

	return nil
}

func (i *Importer) createPerformers(ctx context.Context, names []string) ([]*models.Performer, error) {
	var ret []*models.Performer
	for _, name := range names {
		newPerformer := models.NewPerformer()
		newPerformer.Name = name

		err := i.PerformerWriter.Create(ctx, &newPerformer)
		if err != nil {
			return nil, err
		}

		ret = append(ret, &newPerformer)
	}

	return ret, nil
}

func (i *Importer) populateTags(ctx context.Context) error {
	if len(i.Input.Tags) > 0 {

		tags, err := importTags(ctx, i.TagWriter, i.Input.Tags, i.MissingRefBehaviour)
		if err != nil {
			return err
		}

		for _, p := range tags {
			i.text.TagIDs.Add(p.ID)
		}
	}

	return nil
}

func (i *Importer) addReadHistory(ctx context.Context) error {
	if len(i.readHistory) > 0 {
		_, err := i.ReaderWriter.AddReads(ctx, i.ID, i.readHistory)
		if err != nil {
			return fmt.Errorf("error adding read date: %v", err)
		}
	}

	return nil
}

func (i *Importer) addOHistory(ctx context.Context) error {
	if len(i.oHistory) > 0 {
		_, err := i.ReaderWriter.AddO(ctx, i.ID, i.oHistory)
		if err != nil {
			return fmt.Errorf("error adding o date: %v", err)
		}
	}

	return nil
}

func (i *Importer) PostImport(ctx context.Context, id int) error {
	if len(i.coverImageData) > 0 {
		if err := i.ReaderWriter.UpdateCover(ctx, id, i.coverImageData); err != nil {
			return fmt.Errorf("error setting text images: %v", err)
		}
	}

	// add histories
	if err := i.addReadHistory(ctx); err != nil {
		return err
	}

	if err := i.addOHistory(ctx); err != nil {
		return err
	}

	return nil
}

func (i *Importer) Name() string {
	if i.Input.Title != "" {
		return i.Input.Title
	}

	if len(i.Input.Files) > 0 {
		return i.Input.Files[0]
	}

	return ""
}

func (i *Importer) FindExistingID(ctx context.Context) (*int, error) {
	var existing []*models.Text
	var err error

	for _, f := range i.text.Files.List() {
		existing, err = i.ReaderWriter.FindByFileID(ctx, f.ID)
		if err != nil {
			return nil, err
		}

		if len(existing) > 0 {
			id := existing[0].ID
			return &id, nil
		}
	}

	return nil, nil
}

func (i *Importer) Create(ctx context.Context) (*int, error) {
	var fileIDs []models.FileID
	for _, f := range i.text.Files.List() {
		fileIDs = append(fileIDs, f.Base().ID)
	}
	if err := i.ReaderWriter.Create(ctx, &i.text, fileIDs); err != nil {
		return nil, fmt.Errorf("error creating text: %v", err)
	}

	id := i.text.ID
	i.ID = id
	return &id, nil
}

func (i *Importer) Update(ctx context.Context, id int) error {
	text := i.text
	text.ID = id
	i.ID = id
	if err := i.ReaderWriter.Update(ctx, &text); err != nil {
		return fmt.Errorf("error updating existing text: %v", err)
	}

	return nil
}

func importTags(ctx context.Context, tagWriter models.TagFinderCreator, names []string, missingRefBehaviour models.ImportMissingRefEnum) ([]*models.Tag, error) {
	tags, err := tagWriter.FindByNames(ctx, names, false)
	if err != nil {
		return nil, err
	}

	var pluckedNames []string
	for _, tag := range tags {
		pluckedNames = append(pluckedNames, tag.Name)
	}

	missingTags := sliceutil.Filter(names, func(name string) bool {
		return !sliceutil.Contains(pluckedNames, name)
	})

	if len(missingTags) > 0 {
		if missingRefBehaviour == models.ImportMissingRefEnumFail {
			return nil, fmt.Errorf("tags [%s] not found", strings.Join(missingTags, ", "))
		}

		if missingRefBehaviour == models.ImportMissingRefEnumCreate {
			createdTags, err := createTags(ctx, tagWriter, missingTags)
			if err != nil {
				return nil, fmt.Errorf("error creating tags: %v", err)
			}

			tags = append(tags, createdTags...)
		}

		// ignore if MissingRefBehaviour set to Ignore
	}

	return tags, nil
}

func createTags(ctx context.Context, tagWriter models.TagCreator, names []string) ([]*models.Tag, error) {
	var ret []*models.Tag
	for _, name := range names {
		newTag := models.NewTag()
		newTag.Name = name

		err := tagWriter.Create(ctx, &newTag)
		if err != nil {
			return nil, err
		}

		ret = append(ret, &newTag)
	}

	return ret, nil
}
