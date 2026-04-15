import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useEvent, type VideoPlayer } from "react-native-video";
import type { Chapter } from "~/models";
import { Button } from "~/primitives";
import { cn } from "~/utils";

export const SkipChapterButton = ({
	player,
	chapters,
	isVisible,
}: {
	player: VideoPlayer;
	chapters: Chapter[];
	isVisible: boolean;
}) => {
	const { t } = useTranslation();

	const [progress, setProgress] = useState(player.currentTime || 0);
	useEvent(player, "onProgress", ({ currentTime }) => {
		setProgress(currentTime);
	});

	const chapter = chapters.find(
		(chapter) => chapter.startTime <= progress && progress < chapter.endTime,
	);

	if (!chapter || chapter.type === "content") return null;

	if (!isVisible && progress >= chapter.startTime + 8) return null;

	return (
		<Button
			text={t(`player.skip-${chapter.type}`)}
			onPress={() => player.seekTo(chapter.endTime)}
			className={cn(
				"absolute right-safe bottom-2/10 m-8",
				"z-20 bg-slate-900/70 px-4 py-2",
			)}
		/>
	);
};
