import Close from "@material-symbols/svg-400/rounded/close-fill.svg";
import { useTranslation } from "react-i18next";
import { View } from "react-native";
import { Heading, IconButton, P } from "~/primitives";
import { cn } from "~/utils";

export const ErrorPopup = ({
	message,
	dismiss,
}: {
	message: string;
	dismiss: () => void;
}) => {
	const { t } = useTranslation();
	return (
		<View
			className={cn(
				"absolute inset-x-6 top-1/2 flex-1 -translate-y-1/2 flex-row justify-between",
				"rounded-xl border border-slate-700 bg-background p-5",
			)}
		>
			<View className="flex-1 flex-wrap">
				<Heading className="my-2">{t("player.fatal")}</Heading>
				<P className="mt-2 flex-1">{message}</P>
			</View>
			<IconButton icon={Close} onPress={dismiss} />
		</View>
	);
};
