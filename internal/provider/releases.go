package provider

import (
	"context"
	"encoding/json"
	"errors"
)

const MaxReleases = 50

type Release struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	TagName     string `json:"tag_name"`
	URL         string `json:"html_url"`
	Draft       bool   `json:"draft"`
	Prerelease  bool   `json:"prerelease"`
	PublishedAt string `json:"published_at"`
}

type ReleaseClient interface {
	ListReleases(context.Context, Repository, int, int) ([]Release, error)
}

func ParseReleases(data []byte) ([]Release, error) {
	var values []Release
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, err
	}
	if len(values) > MaxReleases {
		return nil, errors.New("release page exceeds bound")
	}
	for _, value := range values {
		if value.ID < 1 || value.TagName == "" {
			return nil, errors.New("invalid release")
		}
	}
	return values, nil
}
