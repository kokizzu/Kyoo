import { ScrollView } from "react-native";
import { useAccount } from "~/providers/account-context";
import { AccountSettings } from "./account";
import { About, GeneralSettings } from "./general";
import { OidcSettings } from "./oidc";
import { ChapterSkipSettings, PlaybackSettings } from "./playback";
import { SessionsSettings } from "./sessions";

export const SettingsPage = () => {
	const account = useAccount();
	return (
		<ScrollView contentContainerClassName="gap-8 pb-8">
			<GeneralSettings />
			{account && <PlaybackSettings />}
			{account && <ChapterSkipSettings />}
			{account && <AccountSettings />}
			{account && <SessionsSettings />}
			{account && <OidcSettings />}
			<About />
		</ScrollView>
	);
};
