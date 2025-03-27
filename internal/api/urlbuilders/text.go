package urlbuilders

import (
	"strconv"

	"github.com/stashapp/stash/pkg/models"
)

type TextURLBuilder struct {
	BaseURL   string
	TextID    string
	Checksum  string
	UpdatedAt string
}

func NewTextURLBuilder(baseURL string, text *models.Text) TextURLBuilder {
	return TextURLBuilder{
		BaseURL:   baseURL,
		TextID:    strconv.Itoa(text.ID),
		Checksum:  text.Checksum,
		UpdatedAt: strconv.FormatInt(text.UpdatedAt.Unix(), 10),
	}
}

func (b ImageURLBuilder) GetTextURL() string {
	return b.BaseURL + "/text/" + b.ImageID + "/text?t=" + b.UpdatedAt
}
