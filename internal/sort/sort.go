package sort

import (
	"sort"

	"github.com/chapmanjacobd/discoteca/internal/models"
	"github.com/chapmanjacobd/discoteca/internal/utils"
)

type Method string

const (
	ByPath         Method = "path"
	ByTitle        Method = "title"
	ByDuration     Method = "duration"
	BySize         Method = "size"
	ByTimeCreated  Method = "time_created"
	ByTimeModified Method = "time_modified"
	ByTimePlayed   Method = "time_last_played"
	ByPlayCount    Method = "play_count"
)

func Apply(media []models.Media, method Method, reverse, natural bool) {
	less := makeLessFunc(media, method, natural)

	if reverse {
		sort.Slice(media, func(i, j int) bool { return !less(i, j) })
	} else {
		sort.Slice(media, less)
	}
}

func makeLessFunc(media []models.Media, method Method, natural bool) func(i, j int) bool {
	switch method {
	case ByPath:
		if natural {
			return func(i, j int) bool {
				return utils.NaturalLess(media[i].Path, media[j].Path)
			}
		}
		return func(i, j int) bool { return media[i].Path < media[j].Path }
	case ByTitle:
		return func(i, j int) bool { return utils.StringValue(media[i].Title) < utils.StringValue(media[j].Title) }
	case ByDuration:
		return func(i, j int) bool { return utils.Int64Value(media[i].Duration) < utils.Int64Value(media[j].Duration) }
	case BySize:
		return func(i, j int) bool { return utils.Int64Value(media[i].Size) < utils.Int64Value(media[j].Size) }
	case ByTimeCreated:
		return func(i, j int) bool {
			return utils.Int64Value(media[i].TimeCreated) < utils.Int64Value(media[j].TimeCreated)
		}
	case ByTimeModified:
		return func(i, j int) bool {
			return utils.Int64Value(media[i].TimeModified) < utils.Int64Value(media[j].TimeModified)
		}
	case ByTimePlayed:
		return func(i, j int) bool {
			return utils.Int64Value(media[i].TimeLastPlayed) < utils.Int64Value(media[j].TimeLastPlayed)
		}
	case ByPlayCount:
		return func(i, j int) bool {
			return utils.Int64Value(media[i].PlayCount) < utils.Int64Value(media[j].PlayCount)
		}
	default:
		return func(i, j int) bool { return media[i].Path < media[j].Path }
	}
}
