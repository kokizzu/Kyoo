package src

import (
	"math/bits"
)

/// See how acoustid handles comparision:
//// https://bitbucket.org/acoustid/acoustid-server/src/cb303c2a3588ff055b7669cf6f1711a224ab9183/postgresql/acoustid_compare.c?at=master

const (
	MinOverlapDuration = 15.0
	MinSilenceDuration = 2.0

	// Correlation threshold (0.0-1.0) above which a match is considered valid.
	// Uses the AcoustID-style formula: 1.0 - 2.0 * biterror / (32 * length),
	// where random noise scores ~0.0 and identical audio scores 1.0.
	MatchThreshold = 0.1

	// Number of most-significant bits used as a hash key for offset voting.
	// Matches AcoustID's MATCH_BITS. The top bits of a chromaprint value are
	// the most discriminative (classifiers are ordered by importance).
	MatchBits = 14

	// Chromaprint encodes silence as this specific value.
	// We skip it during offset voting to avoid false matches.
	SilenceValue = 627964279

	// Number of samples per correlation block (~2 seconds at 7.8125 samples/s).
	// Segments are evaluated in blocks of this size to find contiguous matching runs.
	CorrBlockSize = 16
)

type Overlap struct {
	StartFirst  float64
	StartSecond float64
	Duration    float64
	Accuracy    int
}

type Match struct {
	Start    float64
	Duration float64
	Accuracy int
}

func hammingDistance(a, b uint32) int {
	return bits.OnesCount32(a ^ b)
}

// segmentCorrelation computes a similarity score between two aligned
// fingerprint slices using the AcoustID formula.
// Returns a value in [0.0, 1.0] where 0.0 means completely different
// (or random noise) and 1.0 means identical.
func segmentCorrelation(fp1 []uint32, fp2 []uint32) float64 {
	length := min(len(fp1), len(fp2))
	if length == 0 {
		return 0
	}
	biterror := 0
	for i := range length {
		biterror += hammingDistance(fp1[i], fp2[i])
	}
	score := 1.0 - 2.0*float64(biterror)/float64(32*length)
	return max(0, score)
}

func matchStrip(v uint32) uint16 {
	return uint16(v >> (32 - MatchBits))
}

// findBestOffset discovers the time offset that best aligns two fingerprints.
//
// It follows AcoustID's match_fingerprints2 approach:
//  1. Hash each fingerprint value by its top 14 bits into a fixed-size table,
//     storing the last seen position for each hash bucket.
//  2. For each hash bucket present in both tables, vote for the offset
//     (position_in_fp1 - position_in_fp2).
//  3. The offset with the most votes wins.
//  4. A diversity check rejects matches caused by repetitive/silent audio.
func findBestOffset(fp1, fp2 []uint32) *int {
	offsets1 := make(map[uint16]int)
	offsets2 := make(map[uint16]int)

	for i, v := range fp1 {
		if v == SilenceValue {
			continue
		}
		key := matchStrip(v)
		offsets1[key] = i + 1
	}

	for i, v := range fp2 {
		if v == SilenceValue {
			continue
		}
		key := matchStrip(v)
		offsets2[key] = i + 1
	}

	if len(offsets1) == 0 || len(offsets2) == 0 {
		return nil
	}

	votes := make(map[int]int)
	topCount := 0
	topOffset := 0

	for key, a := range offsets1 {
		b, ok := offsets2[key]
		if !ok {
			continue
		}
		offset := a - b
		votes[offset]++
		if votes[offset] > topCount {
			topCount = votes[offset]
			topOffset = offset
		}
	}

	// Diversity check: reject if the top offset got very few votes relative
	// to the number of unique values. This filters out repetitive audio
	// (silence, static noise) that would produce spurious matches.
	// (at least 2% of values must match with said offset)
	if topCount < max(len(offsets1), len(offsets2))*2/100 {
		return nil
	}
	return new(topOffset)
}

// alignFingerprints returns the sub-slices of fp1 and fp2 that overlap
// when fp1 is shifted by `offset` positions relative to fp2.
// offset = position_in_fp1 - position_in_fp2.
// Also returns the starting indices in fp1 and fp2.
func alignFingerprints(fp1, fp2 []uint32, offset int) ([]uint32, []uint32, int, int) {
	start1 := 0
	start2 := 0
	if offset > 0 {
		start1 = offset
	} else {
		start2 = -offset
	}

	length := min(len(fp1)-start1, len(fp2)-start2)
	if length <= 0 {
		return nil, nil, 0, 0
	}
	return fp1[start1 : start1+length], fp2[start2 : start2+length], start1, start2
}

// findMatchingRuns divides the aligned fingerprints into fixed-size blocks,
// computes the correlation of each block, and finds contiguous runs of
// blocks whose correlation exceeds MatchThreshold. Each run that is at least
// MinOverlapDuration long is returned as an Overlap.
func findMatchingRuns(fp1, fp2 []uint32, start1, start2 int) []Overlap {
	length := min(len(fp1), len(fp2))
	minSamples := secToSamples(MinOverlapDuration)
	if length < minSamples {
		return nil
	}

	nblocks := length / CorrBlockSize
	blockCorr := make([]float64, nblocks)
	for b := range nblocks {
		lo := b * CorrBlockSize
		hi := lo + CorrBlockSize
		blockCorr[b] = segmentCorrelation(fp1[lo:hi], fp2[lo:hi])
	}

	// Find contiguous runs of blocks above threshold.
	var overlaps []Overlap
	inRun := false
	runStart := 0

	// Handle a run that extends to the last block.
	nblocks++
	blockCorr = append(blockCorr, MatchThreshold)

	for b := range nblocks {
		if blockCorr[b] >= MatchThreshold {
			inRun = true
			runStart = min(runStart, b)
			continue
		}
		if !inRun {
			continue
		}

		inRun = false
		start := runStart * CorrBlockSize
		end := b * CorrBlockSize
		if end-start >= minSamples {
			corr := segmentCorrelation(fp1[start:end], fp2[start:end])
			overlaps = append(overlaps, Overlap{
				StartFirst:  samplesToSec(start1 + start),
				StartSecond: samplesToSec(start2 + start),
				Duration:    samplesToSec(end - start),
				Accuracy:    max(0, min(int(corr*100), 100)),
			})
		}
	}

	return overlaps
}

// FpFindOverlap finds all similar segments (like shared intro music) between
// two chromaprint fingerprints.
//
//  1. Hash each fingerprint value by its top 14 bits to find the best
//     time-offset alignment between the two fingerprints (like
//     AcoustID's match_fingerprints2)
//  2. Align the fingerprints at that offset.
//  3. Divide the aligned region into ~2-second blocks and compute correlation
//     per block using the AcoustID scoring formula.
//  4. Find contiguous runs of high-correlation blocks that are at least
//     MinOverlapDuration long.
func FpFindOverlap(fp1 []uint32, fp2 []uint32) ([]Overlap, error) {
	offset := findBestOffset(fp1, fp2)
	if offset == nil {
		return nil, nil
	}

	a1, a2, s1, s2 := alignFingerprints(fp1, fp2, *offset)
	if len(a1) == 0 {
		return nil, nil
	}

	runs := findMatchingRuns(a1, a2, s1, s2)
	return runs, nil
}

func FpFindContain(haystack []uint32, needle []uint32) (*Match, error) {
	offset := findBestOffset(haystack, needle)
	if offset == nil || *offset < 0 || *offset+len(needle) < len(haystack) {
		return nil, nil
	}

	corr := segmentCorrelation(haystack[*offset:*offset+len(needle)], needle)
	if corr < MatchThreshold {
		return nil, nil
	}

	accuracy := min(int(corr*100), 100)
	return &Match{
		Start:    samplesToSec(*offset),
		Duration: samplesToSec(len(needle)),
		Accuracy: accuracy,
	}, nil
}
