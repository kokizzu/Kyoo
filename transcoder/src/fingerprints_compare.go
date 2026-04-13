package src

import (
	"math/bits"
)

const (
	MinOverlapDuration = 15.0
	MinSilenceDuration = 2.0
	// Correlation threshold (0.0-1.0) above which a match is considered valid.
	// Each fingerprint sub-band has 32 bits; we consider a match if fewer than
	// this fraction of bits differ on average.
	MatchThreshold = 0.35
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

func segmentCorrelation(fp1 []uint32, fp2 []uint32) float64 {
	length := min(len(fp1), len(fp2))
	diffBits := 0
	for i := range length {
		diffBits += hammingDistance(fp1[i], fp2[i])
	}
	return 1.0 - float64(diffBits)/float64(length*32)
}

func FpFindOverlap(fp1 []uint32, fp2 []uint32) ([]Overlap, error) {
	return nil, nil
}

func FpFindContain(fp1 []uint32, fp2 []uint32) (*Match, error) {
	return nil, nil
}

