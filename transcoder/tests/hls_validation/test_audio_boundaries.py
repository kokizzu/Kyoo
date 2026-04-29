from __future__ import annotations

from .hls_utils import fetch_text, parse_media_playlist, probe_segment_timeline


def test_audio_segment_100_boundary_is_continuous(master_context: dict, test_config, byte_cache) -> None:
    if not master_context["audios"]:
        return

    max_allowed_discontinuity = 0.06
    failures: list[str] = []

    for audio in master_context["audios"]:
        text = fetch_text(
            audio.url,
            timeout_seconds=test_config.timeout_seconds,
            headers=test_config.headers,
        )
        playlist = parse_media_playlist(text, audio.url, master_context["client_id"])

        if len(playlist.segment_urls) <= 100:
            continue

        # Audio is transcoded lazily in 100-segment windows.
        # Verify continuity at each window boundary (99->100, 199->200, ...).
        boundaries = [i for i in range(100, len(playlist.segment_urls), 100)]
        for boundary in boundaries:
            before = probe_segment_timeline(
                segment_url=playlist.segment_urls[boundary - 1],
                map_url=playlist.map_url,
                timeout_seconds=test_config.timeout_seconds,
                byte_cache=byte_cache,
                headers=test_config.headers,
            )
            after = probe_segment_timeline(
                segment_url=playlist.segment_urls[boundary],
                map_url=playlist.map_url,
                timeout_seconds=test_config.timeout_seconds,
                byte_cache=byte_cache,
                headers=test_config.headers,
            )

            delta = after.start - before.end
            if abs(delta) > max_allowed_discontinuity:
                failures.append(
                    f"{audio.attrs.get('GROUP-ID', audio.url)}"
                    f" boundary={boundary - 1}->{boundary}"
                    f" delta={delta:.6f}s"
                )

    assert not failures, "audio discontinuity around lazy head boundary: " + ", ".join(failures)
