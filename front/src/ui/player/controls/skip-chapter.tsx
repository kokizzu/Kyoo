import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useEvent, type VideoPlayer } from "react-native-video";
import type { Chapter } from "~/models";
import { Button } from "~/primitives";
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
	const [slug] = useQueryState<string>("slug", undefined!);
	const { data } = useFetch(Info.infoQuery(slug));

	const [progress, setProgress] = useState(player.currentTime || 0);
	useEvent(player, "onProgress", ({ currentTime }) => {
		setProgress(currentTime);
	});

	const chapter = chapters.find(
		(chapter) => chapter.startTime <= progress && progress < chapter.endTime,
	);

	if (!chapter || chapter.type === "content") return null;

	// delay credits appearance by a few seconds, we want to make sure it doesn't
	// show on top of the end of the serie. it's common for the end credits music
	// to start playing on top of the episode also.
	const start = chapter.startTime + +(chapter.type === "credits") * 4;
	if (!isVisible && progress >= start + 8) return null;

	return (
		<Button
			text={t(`player.skip-${chapter.type}`)}
			onPress={() => {
				if (data?.durationSeconds && data.durationSeconds <= chapter.endTime) {
					return seekEnd();
				}
				player.seekTo(chapter.endTime);
			}}
			className={cn(
				"absolute right-safe bottom-2/10 m-8",
				"z-20 bg-slate-900/70 px-4 py-2",
			)}
		/>
	);
};
