from __future__ import annotations

from concurrent.futures import ThreadPoolExecutor

from .hls_utils import (
    assert_no_large_gaps_or_overlaps,
    build_master_url,
    fetch_text,
    parse_master_playlist,
    parse_media_playlist,
    probe_segment_timeline,
)


def _load_variant_playlists_for_client(test_config, client_id: str):
    master_url = build_master_url(test_config.base_url, test_config.media_path, client_id)
    master_text = fetch_text(
        master_url,
        timeout_seconds=test_config.timeout_seconds,
        headers=test_config.headers,
    )
    variants, _ = parse_master_playlist(master_text, master_url=master_url, client_id=client_id)
    unique_variants = list({v.url: v for v in variants}.values())
    playlists = []
    for variant in unique_variants:
        text = fetch_text(
            variant.url,
            timeout_seconds=test_config.timeout_seconds,
            headers=test_config.headers,
        )
        playlists.append(parse_media_playlist(text, variant.url, client_id=client_id))
    return playlists


def test_abr_switching_has_no_timeline_holes(master_context: dict, test_config, byte_cache) -> None:
    playlists = []
    for variant in master_context["variants"]:
        text = fetch_text(
            variant.url,
            timeout_seconds=test_config.timeout_seconds,
            headers=test_config.headers,
        )
        playlists.append(parse_media_playlist(text, variant.url, master_context["client_id"]))

    if len(playlists) < 2:
        return

    selected = playlists[: min(3, len(playlists))]
    common_len = min(len(p.segment_urls) for p in selected)
    take = min(common_len, test_config.max_segments)
    assert take >= 4, "Need enough shared segments to validate ABR switching"

    timelines = []
    for seg_index in range(take):
        playlist = selected[seg_index % len(selected)]
        segment_url = playlist.segment_urls[seg_index]
        timelines.append(
            probe_segment_timeline(
                segment_url=segment_url,
                map_url=playlist.map_url,
                timeout_seconds=test_config.timeout_seconds,
                byte_cache=byte_cache,
                headers=test_config.headers,
            )
        )

    assert_no_large_gaps_or_overlaps(
        timelines=timelines,
        gap_tolerance_seconds=0.10,
        overlap_tolerance_seconds=0.10,
    )


def test_seek_storm_contiguous_windows_stay_continuous(media_playlists: dict, test_config, byte_cache) -> None:
    playlist = media_playlists["variants"][0]
    count = len(playlist.segment_urls)
    assert count >= 8, f"Need at least 8 segments for seek test, got {count}"

    mid = count // 2
    end = count - 1
    windows = [
        [0, 1, 2],
        [mid, min(mid + 1, end)],
        [3, 4],
        [max(0, end - 2), max(0, end - 1)],
        [mid],
    ]

    for window in windows:
        timelines = []
        for seg_index in window:
            timelines.append(
                probe_segment_timeline(
                    segment_url=playlist.segment_urls[seg_index],
                    map_url=playlist.map_url,
                    timeout_seconds=test_config.timeout_seconds,
                    byte_cache=byte_cache,
                    headers=test_config.headers,
                )
            )
        if len(timelines) > 1:
            assert_no_large_gaps_or_overlaps(
                timelines=timelines,
                gap_tolerance_seconds=0.08,
                overlap_tolerance_seconds=0.08,
            )


def test_concurrent_clients_can_seek_without_breaking_timeline(test_config, byte_cache) -> None:
    worker_count = 3

    def worker(idx: int) -> None:
        client_id = f"{test_config.client_prefix}-concurrent-{idx}"
        playlists = _load_variant_playlists_for_client(test_config, client_id)
        if not playlists:
            raise AssertionError("No playlists discovered for concurrent client")

        playlist = playlists[0]
        count = len(playlist.segment_urls)
        if count < 6:
            raise AssertionError(f"Need at least 6 segments for concurrent worker, got {count}")

        pattern = [0, count // 3, (2 * count) // 3, 1, 2, 3]
        timelines = []
        for seg_index in pattern:
            timelines.append(
                probe_segment_timeline(
                    segment_url=playlist.segment_urls[min(seg_index, count - 1)],
                    map_url=playlist.map_url,
                    timeout_seconds=test_config.timeout_seconds,
                    byte_cache=byte_cache,
                    headers=test_config.headers,
                )
            )

        assert_no_large_gaps_or_overlaps(
            timelines=timelines[-3:],
            gap_tolerance_seconds=0.08,
            overlap_tolerance_seconds=0.08,
        )

    with ThreadPoolExecutor(max_workers=worker_count) as pool:
        futures = [pool.submit(worker, i) for i in range(worker_count)]
        for f in futures:
            f.result()
