package src

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

// Number of fingerprint items per second (chromaprint default sample rate).
// Chromaprint uses ~8000 Hz sample rate with 4096-sample frames and 4096/3 overlap,
// producing roughly 7.8 items/s. We use the conventional approximation.
const FingerprintSampleRate = 7.8125

func secToSamples(sec float64) int {
	return int(math.Round(sec * FingerprintSampleRate))
}

func samplesToSec(samples int) float64 {
	return float64(samples) / FingerprintSampleRate
}

func CompressFingerprint(fp []uint32) (string, error) {
	if len(fp) == 0 {
		return "", nil
	}

	raw := make([]byte, len(fp)*4)
	for i, v := range fp {
		binary.LittleEndian.PutUint32(raw[i*4:], v)
	}

	var compressed bytes.Buffer
	zw := zlib.NewWriter(&compressed)
	if _, err := zw.Write(raw); err != nil {
		_ = zw.Close()
		return "", fmt.Errorf("failed to compress fingerprint: %w", err)
	}
	if err := zw.Close(); err != nil {
		return "", fmt.Errorf("failed to finalize compressed fingerprint: %w", err)
	}

	return base64.StdEncoding.EncodeToString(compressed.Bytes()), nil
}

func DecompressFingerprint(compressed string) ([]uint32, error) {
	data, err := base64.StdEncoding.DecodeString(compressed)
	if err != nil {
		return nil, fmt.Errorf("failed to base64 decode fingerprint: %w", err)
	}

	zr, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create zlib reader: %w", err)
	}
	defer zr.Close()

	raw, err := io.ReadAll(zr)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress fingerprint: %w", err)
	}

	if len(raw)%4 != 0 {
		return nil, fmt.Errorf("invalid raw fingerprint size: %d", len(raw))
	}

	numItems := len(raw) / 4
	result := make([]uint32, numItems)
	for i := range numItems {
		result[i] = binary.LittleEndian.Uint32(raw[i*4:])
	}

	return result, nil
}

func ExtractSegment(fp []uint32, startSec, endSec float64) ([]uint32, error) {
	startIdx := secToSamples(startSec)
	endIdx := secToSamples(endSec)

	if startIdx < 0 {
		startIdx = 0
	}
	if endIdx > len(fp) {
		endIdx = len(fp)
	}
	if startIdx >= endIdx {
		return nil, fmt.Errorf("invalid segment range: %f-%f", startSec, endSec)
	}

	segment := make([]uint32, endIdx-startIdx)
	copy(segment, fp[startIdx:endIdx])
	return segment, nil
}
