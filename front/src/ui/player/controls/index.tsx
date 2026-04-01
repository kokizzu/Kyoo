import { useCallback, useState } from "react";
import type { ViewProps } from "react-native";
import { View } from "react-native";
import type { VideoPlayer } from "react-native-video";
import type { Chapter, KImage, Show } from "~/models";
import { useIsTouch } from "~/primitives";
import { Back } from "./back";
import { BottomControls } from "./bottom-controls";
import { MiddleControls } from "./middle-controls";
import { TouchControls } from "./touch";

export const Controls = ({
	player,
	showHref,
	name,
	poster,
	showKind,
	showLogo,
	subName,
	chapters,
	playPrev,
	playNext,
}: {
	player: VideoPlayer;
	showHref?: string;
	name?: string;
	poster?: KImage | null;
	showKind?: Show["kind"];
	showLogo?: KImage | null;
	subName?: string;
	chapters: Chapter[];
	playPrev: (() => boolean) | null;
	playNext: (() => boolean) | null;
}) => {
	const isTouch = useIsTouch();

	const [hover, setHover] = useState(false);
	const [menuOpened, setMenuOpened] = useState(false);

	const hoverControls = {
		onPointerEnter: (e) => {
			if (e.nativeEvent.pointerType === "mouse") setHover(true);
		},
		onPointerLeave: (e) => {
			if (e.nativeEvent.pointerType === "mouse") setHover(false);
		},
	} satisfies ViewProps;

	const setMenu = useCallback((val: boolean) => {
		setMenuOpened(val);
		// Disable hover since the menu overlay makes the pointer leave unreliable.
		if (!val) setHover(false);
	}, []);

	return (
		<View className="absolute inset-0">
			<TouchControls
				player={player}
				forceShow={hover || menuOpened}
				className="absolute inset-0"
			>
				<Back
					showHref={showHref}
					name={name}
					kind={showKind}
					logo={showLogo}
					className="absolute top-0 w-full bg-slate-900/50 px-safe pt-safe"
					{...hoverControls}
				/>
				{isTouch && (
					<MiddleControls
						player={player}
						playPrev={playPrev}
						playNext={playNext}
					/>
				)}
				<BottomControls
					player={player}
					name={subName}
					poster={poster}
					chapters={chapters}
					playPrev={playPrev}
					playNext={playNext}
					setMenu={setMenu}
					className="absolute bottom-0 w-full bg-slate-900/50 px-safe pt-safe"
					{...hoverControls}
				/>
			</TouchControls>
		</View>
	);
};

export { LoadingIndicator } from "./misc";
