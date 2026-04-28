package src

import (
	"context"
	"fmt"
	"log/slog"
	"math"

	"github.com/zoriya/kyoo/transcoder/src/utils"
)

const (
	// MergeWindowSec is the maximum gap (in seconds) between a detected chapter
	// boundary and an existing chapter for them to be merged.
	MergeWindowSec float32 = 3.0
)

func (s *MetadataService) IdentifyChapters(
	ctx context.Context,
	info *MediaInfo,
	prev string,
	next string,
) error {
	defer utils.PrintExecTime(ctx, "identify chapters for %s", info.Path)()

	fingerprint, err := s.ComputeFingerprint(ctx, info)
	if err != nil {
		slog.ErrorContext(ctx, "failed to compute fingerprint", "path", info.Path, "err", err)
		return err
	}

	candidates := make([]Chapter, 0)

	for _, otherPath := range []string{prev, next} {
		if otherPath == "" {
			continue
		}
		nc, err := s.compareWithOther(ctx, info, fingerprint, otherPath)
		if err != nil {
			slog.WarnContext(ctx, "failed to compare episodes", "path", info.Path, "otherPath", otherPath, "err", err)
			continue
		}
		if otherPath == next {
			for i := range nc {
				nc[i].FirstAppearance = new(true)
			}
		}
		candidates = append(candidates, nc...)
	}

	chapters := mergeChapters(info, candidates)
	err = s.saveChapters(ctx, info.Id, chapters)
	if err != nil {
		slog.ErrorContext(ctx, "failed to save chapters", "path", info.Path, "err", err)
		return err
	}
	return nil
}

func (s *MetadataService) compareWithOther(
	ctx context.Context,
	info *MediaInfo,
	fingerprint *Fingerprint,
	otherPath string,
) ([]Chapter, error) {
	otherSha, err := ComputeSha(otherPath)
	if err != nil {
		return nil, fmt.Errorf("failed to compute sha for %s: %w", otherPath, err)
	}
	otherInfo, err := s.GetMetadata(ctx, otherPath, otherSha)
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata for %s: %w", otherPath, err)
	}

	otherPrint, err := s.ComputeFingerprint(ctx, otherInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to compute fingerprint for %s: %w", otherInfo.Path, err)
	}

	if err := s.StoreFingerprint(ctx, otherInfo.Id, otherPrint); err != nil {
		slog.WarnContext(ctx, "failed to store fingerprint", "path", otherInfo.Path, "err", err)
	}

	intros, err := FpFindOverlap(ctx, fingerprint.Start, otherPrint.Start)
	if err != nil {
		return nil, fmt.Errorf("failed to find intro overlaps: %w", err)
	}
	credits, err := FpFindOverlap(ctx, fingerprint.End, otherPrint.End)
	if err != nil {
		return nil, fmt.Errorf("failed to find credit overlaps: %w", err)
	}

	var candidates []Chapter
	for _, intro := range intros {
		slog.InfoContext(ctx, "Identified intro", "start", intro.StartFirst, "duration", intro.Duration)
		candidates = append(candidates, Chapter{
			Id:            info.Id,
			StartTime:     float32(intro.StartFirst),
			EndTime:       float32(intro.StartFirst + intro.Duration),
			Name:          "",
			Type:          Intro,
			MatchAccuracy: new(int32(intro.Accuracy)),
		})
	}

	for _, cred := range credits {
		endOffset := info.Duration - samplesToSec(len(fingerprint.End))
		slog.InfoContext(ctx, "Identified credits", "start", endOffset+cred.StartFirst, "duration", cred.Duration, "end_offset", endOffset)
		candidates = append(candidates, Chapter{
			Id:            info.Id,
			StartTime:     float32(endOffset + cred.StartFirst),
			EndTime:       float32(endOffset + cred.StartFirst + cred.Duration),
			Name:          "",
			Type:          Credits,
			MatchAccuracy: new(int32(cred.Accuracy)),
		})
	}

	return candidates, nil
}

func mergeChapters(info *MediaInfo, candidates []Chapter) []Chapter {
	if len(candidates) == 0 {
		return info.Chapters
	}

	chapters := make([]Chapter, 0, len(info.Chapters))
	for _, c := range info.Chapters {
		// ignore pre-generated chapters
		if c.Name != "" {
			chapters = append(chapters, c)
		}
	}

	for _, cand := range candidates {
		if cand.Type == Content {
			continue
		}

		merged := false
		for i := range chapters {
			if absF32(chapters[i].StartTime-cand.StartTime) < MergeWindowSec &&
				absF32(chapters[i].EndTime-cand.EndTime) < MergeWindowSec {
				if chapters[i].Type == Content {
					chapters[i].Type = cand.Type
				}
				if chapters[i].MatchAccuracy != nil {
					chapters[i].MatchAccuracy = new(max(*chapters[i].MatchAccuracy, *cand.MatchAccuracy))
				} else {
					chapters[i].MatchAccuracy = cand.MatchAccuracy
					chapters[i].FirstAppearance = cand.FirstAppearance
				}
				if chapters[i].Name != "" {
					chapters[i].FirstAppearance = cand.FirstAppearance
				}
				merged = true
				break
			}

			if cand.StartTime-MergeWindowSec < chapters[i].StartTime &&
				cand.EndTime+MergeWindowSec > chapters[i].EndTime &&
				cand.Type == chapters[i].Type {
				// prefer the existing match instead of splitting it into two.
				merged = true
			}
		}

		if !merged {
			if cand.StartTime < MergeWindowSec {
				cand.StartTime = 0
			}
			if absF32(float32(info.Duration)-cand.EndTime) < MergeWindowSec {
				cand.EndTime = float32(info.Duration)
			}
			chapters = insertChapter(chapters, Chapter{
				Id:              info.Id,
				StartTime:       cand.StartTime,
				EndTime:         cand.EndTime,
				Name:            "",
				Type:            cand.Type,
				MatchAccuracy:   cand.MatchAccuracy,
				FirstAppearance: cand.FirstAppearance,
			}, info.Duration)
		}
	}

	return chapters
}

// insertChapter adds a new chapter into the chapter list, adjusting adjacent
// chapters so there are no gaps or overlaps.
func insertChapter(chapters []Chapter, ch Chapter, duration float64) []Chapter {
	var ret []Chapter
	if len(chapters) == 0 {
		if ch.StartTime > 0 {
			ret = append(ret, Chapter{
				Id:        ch.Id,
				StartTime: 0,
				EndTime:   ch.StartTime,
				Name:      "",
				Type:      Content,
			})
		}
		ret = append(ret, ch)
		if ch.EndTime < float32(duration) {
			ret = append(ret, Chapter{
				Id:        ch.Id,
				StartTime: ch.EndTime,
				EndTime:   float32(duration),
				Name:      "",
				Type:      Content,
			})
		}
		return ret
	}

	inserted := false
	for _, existing := range chapters {
		if !inserted && ch.StartTime < existing.EndTime {
			if ch.StartTime > existing.StartTime {
				before := existing
				before.EndTime = ch.StartTime
				ret = append(ret, before)
			}
			ret = append(ret, ch)
			inserted = true

			if ch.EndTime < existing.EndTime {
				after := existing
				after.StartTime = ch.EndTime
				ret = append(ret, after)
			}
			continue
		}

		if inserted && existing.StartTime < ch.EndTime {
			if existing.EndTime > ch.EndTime {
				existing.StartTime = ch.EndTime
				ret = append(ret, existing)
			}
			continue
		}

		ret = append(ret, existing)
	}

	if !inserted {
		ret = append(ret, ch)
	}

	return ret
}

func (s *MetadataService) saveChapters(ctx context.Context, infoId int32, chapters []Chapter) error {
	tx, err := s.Database.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `delete from gocoder.chapters where id = $1`, infoId)
	if err != nil {
		return fmt.Errorf("failed to delete existing chapters: %w", err)
	}

	for _, c := range chapters {
		_, err = tx.Exec(ctx,
			`insert into gocoder.chapters(id, start_time, end_time, name, type, match_accuracy, first_appearance)
			 values ($1, $2, $3, $4, $5, $6, $7)`,
			infoId, c.StartTime, c.EndTime, c.Name, c.Type, c.MatchAccuracy, c.FirstAppearance,
		)
		if err != nil {
			return fmt.Errorf("failed to insert chapter: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func absF32(v float32) float32 {
	return float32(math.Abs(float64(v)))
}
