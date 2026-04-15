package src

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/zoriya/kyoo/transcoder/src/exec"
	"github.com/zoriya/kyoo/transcoder/src/utils"
)

const (
	FingerprintVersion = 1
	FpStartPercent     = 0.20
	FpStartDuration    = 10 * 60
	FpEndDuration      = 5 * 60
)

type Fingerprint struct {
	Start []uint32
	End   []uint32
}

func (s *MetadataService) ComputeFingerprint(ctx context.Context, info *MediaInfo) (*Fingerprint, error) {
	get_running, set := s.fingerprintLock.Start(info.Sha)
	if get_running != nil {
		return get_running()
	}

	var startData string
	var endData string
	err := s.Database.QueryRow(ctx,
		`select start_data, end_data from gocoder.fingerprints where id = $1`,
		info.Id,
	).Scan(&startData, &endData)
	if err == nil {
		startFingerprint, err := DecompressFingerprint(startData)
		if err != nil {
			return set(nil, fmt.Errorf("failed to decompress start fingerprint: %w", err))
		}
		endFingerprint, err := DecompressFingerprint(endData)
		if err != nil {
			return set(nil, fmt.Errorf("failed to decompress end fingerprint: %w", err))
		}
		return set(&Fingerprint{
			Start: startFingerprint,
			End:   endFingerprint,
		}, nil)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return set(nil, fmt.Errorf("failed to query fingerprint: %w", err))
	}

	defer utils.PrintExecTime(ctx, "chromaprint for %s", info.Path)()
	startFingerprint, err := computeChromaprint(
		ctx,
		info.Path,
		0,
		min(info.Duration*FpStartPercent, FpStartDuration),
	)
	if err != nil {
		return set(nil, fmt.Errorf("failed to compute start fingerprint: %w", err))
	}

	endFingerprint, err := computeChromaprint(
		ctx,
		info.Path,
		max(info.Duration-FpEndDuration, 0),
		-1,
	)
	if err != nil {
		return set(nil, fmt.Errorf("failed to compute end fingerprint: %w", err))
	}

	return set(&Fingerprint{
		Start: startFingerprint,
		End:   endFingerprint,
	}, nil)
}

func computeChromaprint(
	ctx context.Context,
	path string,
	start float64,
	duration float64,
) ([]uint32, error) {
	ctx = context.WithoutCancel(ctx)
	defer utils.PrintExecTime(ctx, "chromaprint for %s (between %f and %f)", path, start, duration)()

	args := []string{
		"-v", "error",
	}
	if start > 0 {
		args = append(args, "-ss", fmt.Sprintf("%.6f", start))
	}
	if duration > 0 {
		args = append(args, "-t", fmt.Sprintf("%.6f", duration))
	}
	args = append(args,
		"-i", path,
		"-ac", "2",
		"-f", "chromaprint",
		"-fp_format", "raw",
		"-",
	)

	cmd := exec.CommandContext(
		ctx,
		"ffmpeg",
		args...,
	)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffmpeg failed: %w", err)
	}

	if len(output)%4 != 0 {
		return nil, fmt.Errorf("invalid binary fingerprint size: %d", len(output))
	}

	result := make([]uint32, len(output)/4)
	for i := range result {
		result[i] = binary.LittleEndian.Uint32(output[i*4:])
	}
	return result, nil
}

func (s *MetadataService) StoreFingerprint(ctx context.Context, infoID int32, fingerprint *Fingerprint) error {
	startCompressed, err := CompressFingerprint(fingerprint.Start)
	if err != nil {
		return fmt.Errorf("failed to compress start fingerprint: %w", err)
	}
	endCompressed, err := CompressFingerprint(fingerprint.End)
	if err != nil {
		return fmt.Errorf("failed to compress end fingerprint: %w", err)
	}

	_, err = s.Database.Exec(ctx,
		`insert into gocoder.fingerprints(id, start_data, end_data) values ($1, $2, $3)
		 on conflict (id) do update set start_data = excluded.start_data, end_data = excluded.end_data`,
		infoID, startCompressed, endCompressed,
	)
	return err
}

func (s *MetadataService) DeleteFingerprint(ctx context.Context, infoID int32) error {
	_, err := s.Database.Exec(ctx,
		`delete from gocoder.fingerprints where id = $1`,
		infoID,
	)
	return err
}

func (s *MetadataService) GetChapterprint(ctx context.Context, id int32) ([]uint32, error) {
	var data string
	err := s.Database.QueryRow(ctx,
		`select data from gocoder.chapterprints where id = $1`,
		id,
	).Scan(&data)
	if err != nil {
		return nil, fmt.Errorf("failed to get chapterprint %d: %w", id, err)
	}

	fingerprint, err := DecompressFingerprint(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress chapterprint %d: %w", id, err)
	}
	return fingerprint, nil
}

func (s *MetadataService) StoreChapterprint(ctx context.Context, fp []uint32) (int32, error) {
	data, err := CompressFingerprint(fp)
	if err != nil {
		return 0, fmt.Errorf("failed to compress chapterprint: %w", err)
	}

	var id int32
	err = s.Database.QueryRow(ctx,
		`insert into gocoder.chapterprints(data) values ($1) returning id`,
		data,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to store chapterprint: %w", err)
	}
	return id, nil
}
