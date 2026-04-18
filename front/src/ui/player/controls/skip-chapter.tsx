import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useEvent, type VideoPlayer } from "react-native-video";
import type { Chapter } from "~/models";
import { Button } from "~/primitives";
import { useAccount } from "~/providers/account-context";
import { useFetch } from "~/query";
import { Info } from "~/ui/info";
import { cn, useQueryState } from "~/utils";

export const SkipChapterButton = ({
	player,
	seekEnd,
	chapters,
	isVisible,
}: {
	player: VideoPlayer;
	seekEnd: () => void;
	chapters: Chapter[];
	isVisible: boolean;
}) => {
	const { t } = useTranslation();
	const account = useAccount();
	const [slug] = useQueryState<string>("slug", undefined!);
	const { data } = useFetch(Info.infoQuery(slug));
	const lastAutoSkippedChapter = useRef<number | null>(null);

	const [progress, setProgress] = useState(player.currentTime || 0);
	useEvent(player, "onProgress", ({ currentTime }) => {
		setProgress(currentTime);
	});

	const chapter = chapters.find(
		(chapter) => chapter.startTime <= progress && progress < chapter.endTime,
	);

	const behavior =
		(chapter &&
			chapter.type !== "content" &&
			account?.claims.settings.chapterSkip[chapter.type]) ||
		"showSkipButton";
	const shouldAutoSkip =
		behavior === "autoSkip" ||
		(behavior === "autoSkipExceptFirstAppearance" && !chapter!.firstAppearance);

	// delay credits appearance by a few seconds, we want to make sure it doesn't
	// show on top of the end of the serie. it's common for the end credits music
	// to start playing on top of the episode also.
	const start = chapter
		? chapter.startTime + +(chapter.type === "credits") * 4
		: Infinity;

	const skipChapter = useCallback(() => {
		if (!chapter) return;
		if (data?.durationSeconds && data.durationSeconds <= chapter.endTime + 3) {
			return seekEnd();
		}
		player.seekTo(chapter.endTime);
	}, [player, chapter, data?.durationSeconds, seekEnd]);

	useEffect(() => {
		if (
			chapter &&
			shouldAutoSkip &&
			progress >= start &&
			lastAutoSkippedChapter.current !== chapter.startTime
		) {
			lastAutoSkippedChapter.current = chapter.startTime;
			skipChapter();
		}
	}, [chapter, progress, shouldAutoSkip, start, skipChapter]);

	if (!chapter || chapter.type === "content" || behavior === "disabled")
		return null;
	if (!isVisible && progress >= start + 8) return null;

	return (
		<Button
			text={t(`player.chapters.skip`, { type: chapter.type })}
			onPress={skipChapter}
			className={cn(
				"absolute right-safe bottom-2/10 m-8",
				"z-20 bg-slate-900/70 px-4 py-2",
			)}
		/>
	);
};
