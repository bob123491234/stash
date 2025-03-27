package jsonschema

import (
	"fmt"
	"os"
	"strconv"

	jsoniter "github.com/json-iterator/go"
	"github.com/stashapp/stash/pkg/fsutil"
	"github.com/stashapp/stash/pkg/models/json"
)

type TextBookmark struct {
	Title      string        `json:"title,omitempty"`
	Location   string        `json:"location"`
	PrimaryTag string        `json:"primary_tag,omitempty"`
	Tags       []string      `json:"tags,omitempty"`
	CreatedAt  json.JSONTime `json:"created_at,omitempty"`
	UpdatedAt  json.JSONTime `json:"updated_at,omitempty"`
}

type TextFile struct {
	ModTime json.JSONTime `json:"mod_time,omitempty"`
	Size    string        `json:"size"`
}

type Text struct {
	Title        string         `json:"title,omitempty"`
	TagLine      string         `json:"tag_line,omitempty"`
	Code         string         `json:"code,omitempty"`
	Studio       string         `json:"studio,omitempty"`
	URLs         []string       `json:"urls,omitempty"`
	Date         string         `json:"date,omitempty"`
	Rating       int            `json:"rating,omitempty"`
	Organized    bool           `json:"organized,omitempty"`
	Details      string         `json:"details,omitempty"`
	Author       string         `json:"author,omitempty"`
	LanguageCode string         `json:"language_code,omitempty"`
	Performers   []string       `json:"performers,omitempty"`
	Tags         []string       `json:"tags,omitempty"`
	Bookmarks    []TextBookmark `json:"bookmarks,omitempty"`
	Files        []string       `json:"files,omitempty"`
	Cover        string         `json:"cover,omitempty"`
	CreatedAt    json.JSONTime  `json:"created_at,omitempty"`
	UpdatedAt    json.JSONTime  `json:"updated_at,omitempty"`

	ResumeLocation string          `json:"resume_location,omitempty"`
	ReadHistory    []json.JSONTime `json:"read_history,omitempty"`
	OHistory       []json.JSONTime `json:"o_history,omitempty"`

	ReadDuration float64 `json:"read_duration,omitempty"`
}

func (s Text) Filename(id int, basename string, hash string) string {
	ret := fsutil.SanitiseBasename(s.Title)
	if ret == "" {
		ret = basename
	}

	if hash != "" {
		ret += "." + hash
	} else {
		// texts may have no file and therefore no hash
		ret += "." + strconv.Itoa(id)
	}

	return ret + ".json"
}

func LoadTextFile(filePath string) (*Text, error) {
	var text Text
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	jsonParser := json.NewDecoder(file)
	err = jsonParser.Decode(&text)
	if err != nil {
		return nil, err
	}
	return &text, nil
}

func SaveTextFile(filePath string, text *Text) error {
	if text == nil {
		return fmt.Errorf("text must not be nil")
	}
	return marshalToFile(filePath, text)
}
