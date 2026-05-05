from __future__ import annotations

from .hls_utils import fetch_text, parse_media_playlist, probe_segment_timeline


def test_copied_audio_has_no_gaps_at_lazy_window_boundaries(
    master_context: dict, test_config, byte_cache
) -> None:
    """For -c:a copy, each lazy window (every ~100 segments) can drop a
    boundary-crossing packet. Verify there are no audible gaps."""
    max_allowed_gap_seconds = 0.06
    failures: list[str] = []

    for audio in master_context["audios"]:
        # Only test the "original" (copied) audio rendition.
        if "original" not in audio.url:
            continue

        text = fetch_text(
            audio.url,
            timeout_seconds=test_config.timeout_seconds,
            headers=test_config.headers,
        )
        playlist = parse_media_playlist(text, audio.url, master_context["client_id"])
        if len(playlist.segment_urls) < 102:
            # Not enough segments to cross a boundary
            continue

        # Probe segments around each 100-segment boundary.
        boundaries = [i for i in range(99, len(playlist.segment_urls), 100)]
        for boundary in boundaries:
            before = probe_segment_timeline(
                segment_url=playlist.segment_urls[boundary],
                map_url=playlist.map_url,
                timeout_seconds=test_config.timeout_seconds,
                byte_cache=byte_cache,
                headers=test_config.headers,
            )
            after = probe_segment_timeline(
                segment_url=playlist.segment_urls[boundary + 1],
                map_url=playlist.map_url,
                timeout_seconds=test_config.timeout_seconds,
                byte_cache=byte_cache,
                headers=test_config.headers,
            )
            gap = after.start - before.end
            if gap > max_allowed_gap_seconds:
                failures.append(
                    f"{audio.attrs.get('GROUP-ID', audio.url)}"
                    f" boundary={boundary}->{boundary + 1}"
                    f" gap={gap:.6f}s"
                )

    assert not failures, "copied audio gap at lazy window boundary: " + ", ".join(
        failures
    )


def test_copied_audio_has_no_large_gaps_anywhere(
    master_context: dict, test_config, byte_cache
) -> None:
    """Check every consecutive segment pair in the copied audio for gaps."""
    max_allowed_gap_seconds = 0.06
    failures: list[str] = []

    for audio in master_context["audios"]:
        if "original" not in audio.url:
            continue

        text = fetch_text(
            audio.url,
            timeout_seconds=test_config.timeout_seconds,
            headers=test_config.headers,
        )
        playlist = parse_media_playlist(text, audio.url, master_context["client_id"])
        if len(playlist.segment_urls) < 2:
            continue

        # Check up to 50 consecutive pairs (enough to cross a window boundary).
        sample_count = min(len(playlist.segment_urls) - 1, 50)
        for idx in range(sample_count):
            before = probe_segment_timeline(
                segment_url=playlist.segment_urls[idx],
                map_url=playlist.map_url,
                timeout_seconds=test_config.timeout_seconds,
                byte_cache=byte_cache,
                headers=test_config.headers,
            )
            after = probe_segment_timeline(
                segment_url=playlist.segment_urls[idx + 1],
                map_url=playlist.map_url,
                timeout_seconds=test_config.timeout_seconds,
                byte_cache=byte_cache,
                headers=test_config.headers,
            )
            gap = after.start - before.end
            if gap > max_allowed_gap_seconds:
                failures.append(
                    f"{audio.attrs.get('GROUP-ID', audio.url)}"
                    f" seg={idx}->{idx + 1} gap={gap:.6f}s"
                )
                # Fail fast on the first large gap
                break

    assert not failures, "copied audio gap detected: " + ", ".join(failures)
