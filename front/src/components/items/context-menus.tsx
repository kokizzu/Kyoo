// import Download from "@material-symbols/svg-400/rounded/download.svg";
import Refresh from "@material-symbols/svg-400/rounded/autorenew.svg";
import Info from "@material-symbols/svg-400/rounded/info.svg";
import MoreVert from "@material-symbols/svg-400/rounded/more_vert.svg";
import MovieInfo from "@material-symbols/svg-400/rounded/movie_info.svg";
import type { ComponentProps } from "react";
import { useTranslation } from "react-i18next";
import { WatchStatusV } from "~/models";
import { HR, IconButton, Menu, tooltip } from "~/primitives";
import { useAccount } from "~/providers/account-context";
import { useMutation } from "~/query";
import { cn } from "~/utils";
import { watchListIcon } from "./watchlist-info";
// import { useDownloader } from "../../packages/ui/src/downloads/ui/src/downloads";

export const EntryContext = ({
	kind,
	slug,
	serieSlug,
	videoSlug,
	className,
	...props
}: {
	kind: "movie" | "episode" | "special";
	serieSlug: string | null;
	slug: string;
	videoSlug: string | null;
	className?: string;
} & Partial<ComponentProps<typeof Menu>> &
	Partial<ComponentProps<typeof IconButton>>) => {
	// const downloader = useDownloader();
	const account = useAccount();
	const { t } = useTranslation();

	const markAsSeenMutation = useMutation({
		method: "POST",
		path: ["api", "profiles", "me", "history"],
		body: [
			{
				percent: 100,
				entry: slug,
				videoId: null,
				time: 0,
				playedDate: null,
				external: true,
			},
		],
		invalidate: null,
	});

	return (
		<Menu
			Trigger={IconButton}
			icon={MoreVert}
			className={cn("not:web:hidden", className)}
			{...tooltip(t("misc.more"))}
			{...(props as any)}
		>
			{serieSlug && (
				<Menu.Item
					label={t("home.episodeMore.goToShow")}
					icon={Info}
					href={`/${kind === "movie" ? "movies" : "series"}/${serieSlug}`}
				/>
			)}
			{account && (
				<Menu.Item
					label={t("show.watchlistMark.completed")}
					icon={watchListIcon("completed")}
					onSelect={() => markAsSeenMutation.mutate()}
				/>
			)}
			{/* <Menu.Item */}
			{/* 	label={t("home.episodeMore.download")} */}
			{/* 	icon={Download} */}
			{/* 	onSelect={() => downloader(type, slug)} */}
			{/* /> */}
			{videoSlug && (
				<Menu.Item
					label={t("home.episodeMore.mediainfo")}
					icon={MovieInfo}
					href={`/info/${videoSlug}`}
				/>
			)}
		</Menu>
	);
};

export const ItemContext = ({
	kind,
	slug,
	videoSlug,
	status,
	className,
	...props
}: {
	kind: "movie" | "serie";
	slug: string;
	videoSlug: string | null;
	status: WatchStatusV | null;
	className?: string;
} & Partial<ComponentProps<typeof Menu>> &
	Partial<ComponentProps<typeof IconButton>>) => {
	const account = useAccount();
	// const downloader = useDownloader();
	const { t } = useTranslation();

	const mutation = useMutation({
		path: ["api", `${kind}s`, slug, "watchstatus"],
		compute: (newStatus: WatchStatusV | null) => ({
			method: newStatus ? "POST" : "DELETE",
			body: newStatus ? { status: newStatus } : undefined,
		}),
		invalidate: [kind, slug],
	});

	const metadataRefreshMutation = useMutation({
		method: "POST",
		path: ["scanner", `${kind}s`, slug, "refresh"],
		invalidate: null,
	});

	return (
		<Menu
			Trigger={IconButton}
			icon={MoreVert}
			className={cn("not:web:hidden", className)}
			{...tooltip(t("misc.more"))}
			{...(props as any)}
		>
			<Menu.Sub
				label={account ? t("show.watchlistEdit") : t("show.watchlistLogin")}
				disabled={!account}
				icon={watchListIcon(status)}
			>
				{Object.values(WatchStatusV).map((x) => (
					<Menu.Item
						key={x}
						label={t(
							`show.watchlistMark.${x.toLowerCase() as Lowercase<WatchStatusV>}`,
						)}
						onSelect={() => mutation.mutate(x)}
						selected={x === status}
					/>
				))}
				{status !== null && (
					<Menu.Item
						label={t("show.watchlistMark.null")}
						onSelect={() => mutation.mutate(null)}
					/>
				)}
			</Menu.Sub>
			{videoSlug && (
				<>
					{/* <Menu.Item */}
					{/* 	label={t("home.episodeMore.download")} */}
					{/* 	icon={Download} */}
					{/* 	onSelect={() => downloader(type, slug)} */}
					{/* /> */}
					<Menu.Item
						label={t("home.episodeMore.mediainfo")}
						icon={MovieInfo}
						href={`/info/${videoSlug}`}
					/>
				</>
			)}
			{account?.isAdmin === true && (
				<>
					<HR />
					<Menu.Item
						label={t("home.refreshMetadata")}
						icon={Refresh}
						onSelect={() => metadataRefreshMutation.mutate()}
					/>
				</>
			)}
		</Menu>
	);
};
