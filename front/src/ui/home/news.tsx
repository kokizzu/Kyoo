import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { EntryBox, entryDisplayNumber } from "~/components/entries";
import { EntrySelect } from "~/components/entries/select";
import { Entry } from "~/models";
import { usePopup } from "~/primitives";
import { InfiniteFetch, type QueryIdentifier } from "~/query";
import { EmptyView } from "~/ui/empty-view";
import { Header } from "./genre";

export const NewsList = () => {
	const { t } = useTranslation();
	const [setPopup, closePopup] = usePopup();

	const openEntrySelect = useCallback(
		(entry: {
			displayNumber: string;
			name: string | null;
			videos: Entry["videos"];
		}) => {
			setPopup(
				<EntrySelect
					displayNumber={entry.displayNumber}
					name={entry.name ?? ""}
					videos={entry.videos}
					close={closePopup}
				/>,
			);
		},
		[setPopup, closePopup],
	);

	return (
		<>
			<Header title={t("home.news")} />
			<InfiniteFetch
				query={NewsList.query()}
				layout={{ ...EntryBox.layout, layout: "horizontal" }}
				Empty={<EmptyView message={t("home.none")} />}
				Render={({ item }) => (
					<EntryBox
						kind={item.kind}
						slug={item.slug}
						serieSlug={item.show!.slug}
						name={`${item.show!.name} ${entryDisplayNumber(item)}`}
						description={item.name}
						thumbnail={item.thumbnail ?? item.show!.thumbnail}
						href={item.href}
						watchedPercent={item.progress.percent}
						videos={item.videos}
						onSelectVideos={() =>
							openEntrySelect({
								displayNumber: entryDisplayNumber(item),
								name: item.name,
								videos: item.videos,
							})
						}
					/>
				)}
				Loader={EntryBox.Loader}
			/>
		</>
	);
};

NewsList.query = (): QueryIdentifier<Entry> => ({
	parser: Entry,
	infinite: true,
	path: ["api", "news"],
	params: {
		limit: 10,
	},
});
