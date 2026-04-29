from __future__ import annotations

from .hls_utils import assert_no_large_gaps_or_overlaps, probe_segment_timeline


def test_variant_playlists_have_continuous_timeline(media_playlists: dict, test_config, byte_cache) -> None:
    playlists = media_playlists["variants"] + media_playlists["audios"]

    for playlist in playlists:
        segment_urls = playlist.segment_urls[: test_config.max_segments]
        assert len(segment_urls) >= 3, f"Not enough segments in playlist {playlist.url}"

        timelines = [
            probe_segment_timeline(
                segment_url=url,
                map_url=playlist.map_url,
                timeout_seconds=test_config.timeout_seconds,
                byte_cache=byte_cache,
                headers=test_config.headers,
            )
            for url in segment_urls
        ]

        stream_kind = timelines[0].stream_kind
        gap_tol = 0.06 if stream_kind == "audio" else 0.05
        overlap_tol = 0.02 if stream_kind == "audio" else 0.04
        assert_no_large_gaps_or_overlaps(timelines, gap_tol, overlap_tol)
