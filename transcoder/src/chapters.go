package src

import (
	"context"
	"fmt"
	"math"

	"github.com/zoriya/kyoo/transcoder/src/utils"
)

const (
	// MergeWindowSec is the maximum gap (in seconds) between a detected chapter
	// boundary and an existing chapter for them to be merged.
	MergeWindowSec float32 = 3.0
)

func (s *MetadataService) IdentifyChapters(ctx context.Context, info *MediaInfo, nearEpisodes []string) {
	defer utils.PrintExecTime("identify chapters for %s", info.Path)()

	if info.Versions.Fingerprint >= FingerprintVersion {
		return
	}

	fingerprint, err := s.ComputeFingerprint(ctx, info)
	if err != nil {
		fmt.Printf("failed to compute fingerprint for %s: %v\n", info.Path, err)
		return
	}

	candidates := make([]Chapter, 0)

	for _, otherPath := range nearEpisodes {
		otherCandidates, err := s.compareWithOther(ctx, info, fingerprint, otherPath)
		if err != nil {
			fmt.Printf("failed to compare %s with %s: %v\n", info.Path, otherPath, err)
			continue
		}
		candidates = append(candidates, otherCandidates...)
	}

	chapters := mergeChapters(info, candidates)
	if err := s.saveChapters(ctx, info.Id, chapters); err != nil {
		fmt.Printf("failed to save chapters for %s: %v\n", info.Path, err)
		return
	}

	if err := s.DeleteFingerprint(ctx, info.Id); err != nil {
		fmt.Printf("failed to delete fingerprint for %s: %v\n", info.Path, err)
	}

	_, err = s.Database.Exec(ctx,
		`update gocoder.info set ver_fingerprint = $2 where id = $1`,
		info.Id, FingerprintVersion,
	)
	if err != nil {
		fmt.Printf("failed to update fingerprint version for %s: %v\n", info.Path, err)
	}
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

	hasChapterprints := false
	for _, c := range otherInfo.Chapters {
		if c.FingerprintId != nil {
			hasChapterprints = true
			break
		}
	}

	if hasChapterprints {
		return s.matchByChapterprints(ctx, info, fingerprint, otherInfo)
	}

	return s.matchByOverlap(ctx, info, fingerprint, otherInfo)
}

func (s *MetadataService) matchByChapterprints(
	ctx context.Context,
	info *MediaInfo,
	fingerprint *Fingerprint,
	otherInfo *MediaInfo,
) ([]Chapter, error) {
	var candidates []Chapter

	for _, ch := range otherInfo.Chapters {
		if ch.FingerprintId == nil {
			continue
		}
		if ch.Type == Content {
			continue
		}

		needle, err := s.GetChapterprint(ctx, *ch.FingerprintId)
		if err != nil {
			fmt.Printf("failed to get chapterprint %d: %v\n", *ch.FingerprintId, err)
			continue
		}

		fp := fingerprint.Start
		startOffset := 0.0
		if ch.Type == Credits {
			fp = fingerprint.End
			startOffset = max(info.Duration-FpEndDuration, 0)
		}

		match, err := FpFindContain(fp, needle)
		if err != nil {
			fmt.Printf("failed to find chapterprint in fingerprint: %v\n", err)
			continue
		}
		if match == nil {
			continue
		}

		candidates = append(candidates, Chapter{
			Id:            info.Id,
			StartTime:     float32(startOffset + match.Start),
			EndTime:       float32(startOffset + match.Start + match.Duration),
			Name:          "",
			Type:          ch.Type,
			FingerprintId: ch.FingerprintId,
			MatchAccuracy: new(int32(match.Accuracy)),
		})
	}

	return candidates, nil
}

func (s *MetadataService) matchByOverlap(
	ctx context.Context,
	info *MediaInfo,
	fingerprint *Fingerprint,
	otherInfo *MediaInfo,
) ([]Chapter, error) {
	otherPrint, err := s.ComputeFingerprint(ctx, otherInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to compute fingerprint for %s: %w", otherInfo.Path, err)
	}

	if err := s.StoreFingerprint(ctx, otherInfo.Id, otherPrint); err != nil {
		fmt.Printf("failed to store fingerprint for %s: %v\n", otherInfo.Path, err)
	}

	intros, err := FpFindOverlap(fingerprint.Start, otherPrint.Start)
	if err != nil {
		return nil, fmt.Errorf("failed to find intro overlaps: %w", err)
	}
	credits, err := FpFindOverlap(fingerprint.End, otherPrint.End)
	if err != nil {
		return nil, fmt.Errorf("failed to find credit overlaps: %w", err)
	}

	var candidates []Chapter
	for _, intro := range intros {
		fp, err := ExtractSegment(fingerprint.Start, intro.StartFirst, intro.StartFirst+intro.Duration)
		if err != nil {
			fmt.Printf("failed to extract segment: %v\n", err)
			continue
		}

		fpId, err := s.StoreChapterprint(ctx, fp)
		if err != nil {
			fmt.Printf("failed to store chapterprint: %v\n", err)
			continue
		}

		candidates = append(candidates, Chapter{
			Id:            info.Id,
			StartTime:     float32(intro.StartFirst),
			EndTime:       float32(intro.StartFirst + intro.Duration),
			Name:          "",
			Type:          Intro,
			FingerprintId: &fpId,
			MatchAccuracy: new(int32(intro.Accuracy)),
		})
	}

	endOffset := max(info.Duration-FpEndDuration, 0)
	for _, ov := range credits {
		segData, err := ExtractSegment(fingerprint.End, ov.StartFirst, ov.StartFirst+ov.Duration)
		if err != nil {
			fmt.Printf("failed to extract segment: %v\n", err)
			continue
		}

		fpId, err := s.StoreChapterprint(ctx, segData)
		if err != nil {
			fmt.Printf("failed to store chapterprint: %v\n", err)
			continue
		}

		candidates = append(candidates, Chapter{
			Id:            info.Id,
			StartTime:     float32(endOffset + ov.StartFirst),
			EndTime:       float32(endOffset + ov.StartFirst + ov.Duration),
			Name:          "",
			Type:          Credits,
			FingerprintId: &fpId,
			MatchAccuracy: new(int32(ov.Accuracy)),
		})
	}

	return candidates, nil
}

func mergeChapters(info *MediaInfo, candidates []Chapter) []Chapter {
	if len(candidates) == 0 {
		return info.Chapters
	}

	chapters := make([]Chapter, len(info.Chapters))
	copy(chapters, info.Chapters)

	for _, cand := range candidates {
		if cand.Type == Content {
			continue
		}

		merged := false
		for i := range chapters {
			if absF32(chapters[i].StartTime-cand.StartTime) < MergeWindowSec {
				if chapters[i].Type == Content {
					chapters[i].Type = cand.Type
				}
				chapters[i].FingerprintId = cand.FingerprintId
				chapters[i].MatchAccuracy = cand.MatchAccuracy
				merged = true
				break
			}
		}

		if !merged {
			chapters = insertChapter(chapters, Chapter{
				Id:            info.Id,
				StartTime:     cand.StartTime,
				EndTime:       cand.EndTime,
				Name:          "",
				Type:          cand.Type,
				FingerprintId: cand.FingerprintId,
				MatchAccuracy: cand.MatchAccuracy,
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

	// Delete existing chapters
	_, err = tx.Exec(ctx, `delete from gocoder.chapters where id = $1`, infoId)
	if err != nil {
		return fmt.Errorf("failed to delete existing chapters: %w", err)
	}

	// Insert new chapters
	for _, c := range chapters {
		_, err = tx.Exec(ctx,
			`insert into gocoder.chapters(id, start_time, end_time, name, type, fingerprint_id, match_accuracy)
			 values ($1, $2, $3, $4, $5, $6, $7)`,
			infoId, c.StartTime, c.EndTime, c.Name, c.Type, c.FingerprintId, c.MatchAccuracy,
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
