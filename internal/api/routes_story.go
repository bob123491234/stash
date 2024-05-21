package api

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/pkg/ffmpeg"
	"github.com/stashapp/stash/pkg/file/video"
	"github.com/stashapp/stash/pkg/fsutil"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/utils"
)

type StoryFinder interface {
	models.StoryGetter

	FindByChecksum(ctx context.Context, checksum string) ([]*models.Story, error)
	FindByOSHash(ctx context.Context, oshash string) ([]*models.Story, error)
	GetCover(ctx context.Context, storyID int) ([]byte, error)
}

type StoryMarkerFinder interface {
	models.StoryMarkerGetter
	FindByStoryID(ctx context.Context, storyID int) ([]*models.StoryMarker, error)
}

type StoryMarkerTagFinder interface {
	models.TagGetter
	FindByStoryMarkerID(ctx context.Context, storyMarkerID int) ([]*models.Tag, error)
}

type CaptionFinder interface {
	GetCaptions(ctx context.Context, fileID models.FileID) ([]*models.VideoCaption, error)
}

type storyRoutes struct {
	routes
	storyFinder       StoryFinder
	fileGetter        models.FileGetter
	captionFinder     CaptionFinder
	storyMarkerFinder StoryMarkerFinder
	tagFinder         StoryMarkerTagFinder
}

func (rs storyRoutes) Routes() chi.Router {
	r := chi.NewRouter()

	r.Route("/{storyId}", func(r chi.Router) {
		r.Use(rs.StoryCtx)

		// streaming endpoints
		r.Get("/stream", rs.StreamDirect)
		r.Get("/stream.mp4", rs.StreamMp4)
		r.Get("/stream.webm", rs.StreamWebM)
		r.Get("/stream.mkv", rs.StreamMKV)
		r.Get("/stream.m3u8", rs.StreamHLS)
		r.Get("/stream.m3u8/{segment}.ts", rs.StreamHLSSegment)
		r.Get("/stream.mpd", rs.StreamDASH)
		r.Get("/stream.mpd/{segment}_v.webm", rs.StreamDASHVideoSegment)
		r.Get("/stream.mpd/{segment}_a.webm", rs.StreamDASHAudioSegment)

		r.Get("/screenshot", rs.Screenshot)
		r.Get("/preview", rs.Preview)
		r.Get("/webp", rs.Webp)
		r.Get("/vtt/chapter", rs.VttChapter)
		r.Get("/vtt/thumbs", rs.VttThumbs)
		r.Get("/vtt/sprite", rs.VttSprite)
		r.Get("/funscript", rs.Funscript)
		r.Get("/interactive_csv", rs.InteractiveCSV)
		r.Get("/interactive_heatmap", rs.InteractiveHeatmap)
		r.Get("/caption", rs.CaptionLang)

		r.Get("/story_marker/{storyMarkerId}/stream", rs.StoryMarkerStream)
		r.Get("/story_marker/{storyMarkerId}/preview", rs.StoryMarkerPreview)
		r.Get("/story_marker/{storyMarkerId}/screenshot", rs.StoryMarkerScreenshot)
	})
	r.Get("/{storyHash}_thumbs.vtt", rs.VttThumbs)
	r.Get("/{storyHash}_sprite.jpg", rs.VttSprite)

	return r
}

func (rs storyRoutes) StreamDirect(w http.ResponseWriter, r *http.Request) {
	story := r.Context().Value(storyKey).(*models.Story)
	ss := manager.StoryServer{
		TxnManager:       rs.txnManager,
		StoryCoverGetter: rs.storyFinder,
	}
	ss.StreamStoryDirect(story, w, r)
}

func (rs storyRoutes) StreamMp4(w http.ResponseWriter, r *http.Request) {
	rs.streamTranscode(w, r, ffmpeg.StreamTypeMP4)
}

func (rs storyRoutes) StreamWebM(w http.ResponseWriter, r *http.Request) {
	rs.streamTranscode(w, r, ffmpeg.StreamTypeWEBM)
}

func (rs storyRoutes) StreamMKV(w http.ResponseWriter, r *http.Request) {
	// only allow mkv streaming if the story container is an mkv already
	story := r.Context().Value(storyKey).(*models.Story)

	pf := story.Files.Primary()
	if pf == nil {
		return
	}

	container, err := manager.GetVideoFileContainer(pf)
	if err != nil {
		logger.Errorf("[transcode] error getting container: %v", err)
	}

	if container != ffmpeg.Matroska {
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte("not an mkv file")); err != nil {
			logger.Warnf("[stream] error writing to stream: %v", err)
		}
		return
	}

	rs.streamTranscode(w, r, ffmpeg.StreamTypeMKV)
}

func (rs storyRoutes) streamTranscode(w http.ResponseWriter, r *http.Request, streamType ffmpeg.StreamFormat) {
	story := r.Context().Value(storyKey).(*models.Story)

	streamManager := manager.GetInstance().StreamManager
	if streamManager == nil {
		http.Error(w, "Live transcoding disabled", http.StatusServiceUnavailable)
		return
	}

	f := story.Files.Primary()
	if f == nil {
		return
	}

	if err := r.ParseForm(); err != nil {
		logger.Warnf("[transcode] error parsing query form: %v", err)
	}

	startTime := r.Form.Get("start")
	ss, _ := strconv.ParseFloat(startTime, 64)
	resolution := r.Form.Get("resolution")

	options := ffmpeg.TranscodeOptions{
		StreamType: streamType,
		VideoFile:  f,
		Resolution: resolution,
		StartTime:  ss,
	}

	logger.Debugf("[transcode] streaming story %d as %s", story.ID, streamType.MimeType)
	streamManager.ServeTranscode(w, r, options)
}

func (rs storyRoutes) StreamHLS(w http.ResponseWriter, r *http.Request) {
	rs.streamManifest(w, r, ffmpeg.StreamTypeHLS, "HLS")
}

func (rs storyRoutes) StreamDASH(w http.ResponseWriter, r *http.Request) {
	rs.streamManifest(w, r, ffmpeg.StreamTypeDASHVideo, "DASH")
}

func (rs storyRoutes) streamManifest(w http.ResponseWriter, r *http.Request, streamType *ffmpeg.StreamType, logName string) {
	story := r.Context().Value(storyKey).(*models.Story)

	streamManager := manager.GetInstance().StreamManager
	if streamManager == nil {
		http.Error(w, "Live transcoding disabled", http.StatusServiceUnavailable)
		return
	}

	f := story.Files.Primary()
	if f == nil {
		return
	}

	if err := r.ParseForm(); err != nil {
		logger.Warnf("[transcode] error parsing query form: %v", err)
	}

	resolution := r.Form.Get("resolution")

	logger.Debugf("[transcode] returning %s manifest for story %d", logName, story.ID)
	streamManager.ServeManifest(w, r, streamType, f, resolution)
}

func (rs storyRoutes) StreamHLSSegment(w http.ResponseWriter, r *http.Request) {
	rs.streamSegment(w, r, ffmpeg.StreamTypeHLS)
}

func (rs storyRoutes) StreamDASHVideoSegment(w http.ResponseWriter, r *http.Request) {
	rs.streamSegment(w, r, ffmpeg.StreamTypeDASHVideo)
}

func (rs storyRoutes) StreamDASHAudioSegment(w http.ResponseWriter, r *http.Request) {
	rs.streamSegment(w, r, ffmpeg.StreamTypeDASHAudio)
}

func (rs storyRoutes) streamSegment(w http.ResponseWriter, r *http.Request, streamType *ffmpeg.StreamType) {
	story := r.Context().Value(storyKey).(*models.Story)

	streamManager := manager.GetInstance().StreamManager
	if streamManager == nil {
		http.Error(w, "Live transcoding disabled", http.StatusServiceUnavailable)
		return
	}

	f := story.Files.Primary()
	if f == nil {
		return
	}

	if err := r.ParseForm(); err != nil {
		logger.Warnf("[transcode] error parsing query form: %v", err)
	}

	storyHash := story.GetHash(config.GetInstance().GetVideoFileNamingAlgorithm())

	segment := chi.URLParam(r, "segment")
	resolution := r.Form.Get("resolution")

	options := ffmpeg.StreamOptions{
		StreamType: streamType,
		VideoFile:  f,
		Resolution: resolution,
		Hash:       storyHash,
		Segment:    segment,
	}

	streamManager.ServeSegment(w, r, options)
}

func (rs storyRoutes) Screenshot(w http.ResponseWriter, r *http.Request) {
	story := r.Context().Value(storyKey).(*models.Story)

	ss := manager.StoryServer{
		TxnManager:       rs.txnManager,
		StoryCoverGetter: rs.storyFinder,
	}
	ss.ServeScreenshot(story, w, r)
}

func (rs storyRoutes) Preview(w http.ResponseWriter, r *http.Request) {
	story := r.Context().Value(storyKey).(*models.Story)
	storyHash := story.GetHash(config.GetInstance().GetVideoFileNamingAlgorithm())
	filepath := manager.GetInstance().Paths.Story.GetVideoPreviewPath(storyHash)

	utils.ServeStaticFile(w, r, filepath)
}

func (rs storyRoutes) Webp(w http.ResponseWriter, r *http.Request) {
	story := r.Context().Value(storyKey).(*models.Story)
	storyHash := story.GetHash(config.GetInstance().GetVideoFileNamingAlgorithm())
	filepath := manager.GetInstance().Paths.Story.GetWebpPreviewPath(storyHash)

	utils.ServeStaticFile(w, r, filepath)
}

func (rs storyRoutes) getChapterVttTitle(r *http.Request, marker *models.StoryMarker) (*string, error) {
	if marker.Title != "" {
		return &marker.Title, nil
	}

	var title string
	if err := rs.withReadTxn(r, func(ctx context.Context) error {
		qb := rs.tagFinder
		primaryTag, err := qb.Find(ctx, marker.PrimaryTagID)
		if err != nil {
			return err
		}

		title = primaryTag.Name

		tags, err := qb.FindByStoryMarkerID(ctx, marker.ID)
		if err != nil {
			return err
		}

		for _, t := range tags {
			title += ", " + t.Name
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return &title, nil
}

func (rs storyRoutes) VttChapter(w http.ResponseWriter, r *http.Request) {
	story := r.Context().Value(storyKey).(*models.Story)
	var storyMarkers []*models.StoryMarker
	readTxnErr := rs.withReadTxn(r, func(ctx context.Context) error {
		var err error
		storyMarkers, err = rs.storyMarkerFinder.FindByStoryID(ctx, story.ID)
		return err
	})
	if errors.Is(readTxnErr, context.Canceled) {
		return
	}
	if readTxnErr != nil {
		logger.Warnf("read transaction error on fetch story markers: %v", readTxnErr)
		http.Error(w, readTxnErr.Error(), http.StatusInternalServerError)
		return
	}

	vttLines := []string{"WEBVTT", ""}
	for i, marker := range storyMarkers {
		vttLines = append(vttLines, strconv.Itoa(i+1))
		time := utils.GetVTTTime(marker.Seconds)
		vttLines = append(vttLines, time+" --> "+time)

		vttTitle, err := rs.getChapterVttTitle(r, marker)
		if errors.Is(err, context.Canceled) {
			return
		}
		if err != nil {
			logger.Warnf("read transaction error on fetch story marker title: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		vttLines = append(vttLines, *vttTitle)
		vttLines = append(vttLines, "")
	}
	vtt := strings.Join(vttLines, "\n")

	w.Header().Set("Content-Type", "text/vtt")
	utils.ServeStaticContent(w, r, []byte(vtt))
}

func (rs storyRoutes) VttThumbs(w http.ResponseWriter, r *http.Request) {
	story, ok := r.Context().Value(storyKey).(*models.Story)
	var storyHash string
	if ok && story != nil {
		storyHash = story.GetHash(config.GetInstance().GetVideoFileNamingAlgorithm())
	} else {
		storyHash = chi.URLParam(r, "storyHash")
	}
	filepath := manager.GetInstance().Paths.Story.GetSpriteVttFilePath(storyHash)

	w.Header().Set("Content-Type", "text/vtt")
	utils.ServeStaticFile(w, r, filepath)
}

func (rs storyRoutes) VttSprite(w http.ResponseWriter, r *http.Request) {
	story, ok := r.Context().Value(storyKey).(*models.Story)
	var storyHash string
	if ok && story != nil {
		storyHash = story.GetHash(config.GetInstance().GetVideoFileNamingAlgorithm())
	} else {
		storyHash = chi.URLParam(r, "storyHash")
	}
	filepath := manager.GetInstance().Paths.Story.GetSpriteImageFilePath(storyHash)

	utils.ServeStaticFile(w, r, filepath)
}

func (rs storyRoutes) Funscript(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(storyKey).(*models.Story)
	filepath := video.GetFunscriptPath(s.Path)

	utils.ServeStaticFile(w, r, filepath)
}

func (rs storyRoutes) InteractiveCSV(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(storyKey).(*models.Story)
	filepath := video.GetFunscriptPath(s.Path)

	// TheHandy directly only accepts interactive CSVs
	csvBytes, err := manager.ConvertFunscriptToCSV(filepath)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	utils.ServeStaticContent(w, r, csvBytes)
}

func (rs storyRoutes) InteractiveHeatmap(w http.ResponseWriter, r *http.Request) {
	story := r.Context().Value(storyKey).(*models.Story)
	storyHash := story.GetHash(config.GetInstance().GetVideoFileNamingAlgorithm())
	filepath := manager.GetInstance().Paths.Story.GetInteractiveHeatmapPath(storyHash)

	utils.ServeStaticFile(w, r, filepath)
}

func (rs storyRoutes) Caption(w http.ResponseWriter, r *http.Request, lang string, ext string) {
	s := r.Context().Value(storyKey).(*models.Story)

	var captions []*models.VideoCaption
	readTxnErr := rs.withReadTxn(r, func(ctx context.Context) error {
		var err error
		primaryFile := s.Files.Primary()
		if primaryFile == nil {
			return nil
		}

		captions, err = rs.captionFinder.GetCaptions(ctx, primaryFile.Base().ID)

		return err
	})
	if errors.Is(readTxnErr, context.Canceled) {
		return
	}
	if readTxnErr != nil {
		logger.Warnf("read transaction error on fetch story captions: %v", readTxnErr)
		http.Error(w, readTxnErr.Error(), http.StatusInternalServerError)
		return
	}

	for _, caption := range captions {
		if lang != caption.LanguageCode || ext != caption.CaptionType {
			continue
		}

		sub, err := video.ReadSubs(caption.Path(s.Path))
		if err != nil {
			logger.Warnf("error while reading subs: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var buf bytes.Buffer

		err = sub.WriteToWebVTT(&buf)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/vtt")
		utils.ServeStaticContent(w, r, buf.Bytes())
		return
	}
}

func (rs storyRoutes) CaptionLang(w http.ResponseWriter, r *http.Request) {
	// serve caption based on lang query param, if provided
	if err := r.ParseForm(); err != nil {
		logger.Warnf("[caption] error parsing query form: %v", err)
	}

	l := r.Form.Get("lang")
	ext := r.Form.Get("type")
	rs.Caption(w, r, l, ext)
}

func (rs storyRoutes) StoryMarkerStream(w http.ResponseWriter, r *http.Request) {
	story := r.Context().Value(storyKey).(*models.Story)
	storyHash := story.GetHash(config.GetInstance().GetVideoFileNamingAlgorithm())
	storyMarkerID, _ := strconv.Atoi(chi.URLParam(r, "storyMarkerId"))
	var storyMarker *models.StoryMarker
	readTxnErr := rs.withReadTxn(r, func(ctx context.Context) error {
		var err error
		storyMarker, err = rs.storyMarkerFinder.Find(ctx, storyMarkerID)
		return err
	})
	if errors.Is(readTxnErr, context.Canceled) {
		return
	}
	if readTxnErr != nil {
		logger.Warnf("read transaction error on fetch story marker: %v", readTxnErr)
		http.Error(w, readTxnErr.Error(), http.StatusInternalServerError)
		return
	}

	if storyMarker == nil {
		http.Error(w, http.StatusText(404), 404)
		return
	}

	filepath := manager.GetInstance().Paths.StoryMarkers.GetVideoPreviewPath(storyHash, int(storyMarker.Seconds))
	utils.ServeStaticFile(w, r, filepath)
}

func (rs storyRoutes) StoryMarkerPreview(w http.ResponseWriter, r *http.Request) {
	story := r.Context().Value(storyKey).(*models.Story)
	storyHash := story.GetHash(config.GetInstance().GetVideoFileNamingAlgorithm())
	storyMarkerID, _ := strconv.Atoi(chi.URLParam(r, "storyMarkerId"))
	var storyMarker *models.StoryMarker
	readTxnErr := rs.withReadTxn(r, func(ctx context.Context) error {
		var err error
		storyMarker, err = rs.storyMarkerFinder.Find(ctx, storyMarkerID)
		return err
	})
	if errors.Is(readTxnErr, context.Canceled) {
		return
	}
	if readTxnErr != nil {
		logger.Warnf("read transaction error on fetch story marker preview: %v", readTxnErr)
		http.Error(w, readTxnErr.Error(), http.StatusInternalServerError)
		return
	}

	if storyMarker == nil {
		http.Error(w, http.StatusText(404), 404)
		return
	}

	filepath := manager.GetInstance().Paths.StoryMarkers.GetWebpPreviewPath(storyHash, int(storyMarker.Seconds))

	// If the image doesn't exist, send the placeholder
	exists, _ := fsutil.FileExists(filepath)
	if !exists {
		w.Header().Set("Content-Type", "image/png")
		utils.ServeStaticContent(w, r, utils.PendingGenerateResource)
	} else {
		utils.ServeStaticFile(w, r, filepath)
	}
}

func (rs storyRoutes) StoryMarkerScreenshot(w http.ResponseWriter, r *http.Request) {
	story := r.Context().Value(storyKey).(*models.Story)
	storyHash := story.GetHash(config.GetInstance().GetVideoFileNamingAlgorithm())
	storyMarkerID, _ := strconv.Atoi(chi.URLParam(r, "storyMarkerId"))
	var storyMarker *models.StoryMarker
	readTxnErr := rs.withReadTxn(r, func(ctx context.Context) error {
		var err error
		storyMarker, err = rs.storyMarkerFinder.Find(ctx, storyMarkerID)
		return err
	})
	if errors.Is(readTxnErr, context.Canceled) {
		return
	}
	if readTxnErr != nil {
		logger.Warnf("read transaction error on fetch story marker screenshot: %v", readTxnErr)
		http.Error(w, readTxnErr.Error(), http.StatusInternalServerError)
		return
	}

	if storyMarker == nil {
		http.Error(w, http.StatusText(404), 404)
		return
	}

	filepath := manager.GetInstance().Paths.StoryMarkers.GetScreenshotPath(storyHash, int(storyMarker.Seconds))

	// If the image doesn't exist, send the placeholder
	exists, _ := fsutil.FileExists(filepath)
	if !exists {
		w.Header().Set("Content-Type", "image/png")
		utils.ServeStaticContent(w, r, utils.PendingGenerateResource)
	} else {
		utils.ServeStaticFile(w, r, filepath)
	}
}

func (rs storyRoutes) StoryCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		storyID, err := strconv.Atoi(chi.URLParam(r, "storyId"))
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		var story *models.Story
		_ = rs.withReadTxn(r, func(ctx context.Context) error {
			qb := rs.storyFinder
			story, _ = qb.Find(ctx, storyID)

			if story != nil {
				if err := story.LoadPrimaryFile(ctx, rs.fileGetter); err != nil {
					if !errors.Is(err, context.Canceled) {
						logger.Errorf("error loading primary file for story %d: %v", storyID, err)
					}
					// set story to nil so that it doesn't try to use the primary file
					story = nil
				}
			}

			return nil
		})
		if story == nil {
			http.Error(w, http.StatusText(404), 404)
			return
		}

		ctx := context.WithValue(r.Context(), storyKey, story)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
