package text

import (
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/models/paths"
	"github.com/stashapp/stash/pkg/plugin"
)

type Service struct {
	File               models.FileReaderWriter
	Repository         models.TextReaderWriter
	BookmarkRepository models.TextBookmarkReaderWriter
	PluginCache        *plugin.Cache

	Paths *paths.Paths
}
