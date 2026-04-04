import { useTranslation } from "react-i18next";
import { SideMenu } from "~/primitives";
import { EntryList } from "~/ui/details/season";

export const EntriesMenu = ({
	isOpen,
	onClose,
	showSlug,
	season,
	currentEntrySlug,
}: {
	isOpen: boolean;
	onClose: () => void;
	showSlug: string;
	season: string | number;
	currentEntrySlug?: string;
}) => {
	return (
		<SideMenu isOpen={isOpen} onClose={onClose} containerClassName="bg-card">
			<EntryList
				slug={showSlug}
				season={season}
				currentEntrySlug={currentEntrySlug}
			/>
		</SideMenu>
	);
};
