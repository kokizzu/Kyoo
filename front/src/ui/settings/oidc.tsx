import Badge from "@material-symbols/svg-400/outlined/badge.svg";
import Remove from "@material-symbols/svg-400/outlined/close.svg";
import OpenProfile from "@material-symbols/svg-400/outlined/open_in_new.svg";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Image } from "react-native";
import { type KyooError, User } from "~/models";
import { AuthInfo } from "~/models/auth-info";
import { Button, IconButton, Link, P, Skeleton, tooltip } from "~/primitives";
import { type QueryIdentifier, useFetch, useMutation } from "~/query";
import { Preference, SettingsContainer } from "./base";

export const OidcSettings = () => {
	const { t } = useTranslation();
	const [unlinkError, setUnlinkError] = useState<string | null>(null);
	const { data } = useFetch(OidcSettings.authQuery());
	const { data: user } = useFetch(OidcSettings.query());
	const { mutateAsync: unlinkAccount } = useMutation({
		method: "DELETE",
		compute: (provider: string) => ({
			path: ["auth", "oidc", "login", provider],
		}),
		invalidate: ["auth", "users", "me"],
	});

	if (data && Object.keys(data.oidc).length === 0) return null;

	return (
		<SettingsContainer title={t("settings.oidc.label")}>
			{unlinkError && <P className="text-red-500">{unlinkError}</P>}
			{data && user
				? Object.entries(data.oidc).map(([id, x]) => {
						const acc = user.oidc[id];
						return (
							<Preference
								key={id}
								icon={Badge}
								label={x.name}
								description={
									acc
										? t("settings.oidc.connected", { username: acc.username })
										: t("settings.oidc.not-connected")
								}
								customIcon={
									x.logo != null && (
										<Image
											source={{ uri: x.logo }}
											className="mr-4 h-6 w-6"
											resizeMode="contain"
										/>
									)
								}
							>
								{acc ? (
									<>
										{acc.profileUrl && (
											<IconButton
												icon={OpenProfile}
												as={Link}
												href={acc.profileUrl}
												{...tooltip(
													t("settings.oidc.open-profile", { provider: x.name }),
												)}
											/>
										)}
										<IconButton
											icon={Remove}
											onPress={async () => {
												setUnlinkError(null);
												try {
													await unlinkAccount(id);
												} catch (e) {
													setUnlinkError((e as KyooError).message);
												}
											}}
											{...tooltip(
												t("settings.oidc.delete", { provider: x.name }),
											)}
										/>
									</>
								) : (
									<Button
										text={t("settings.oidc.link")}
										as={Link}
										href={x.link}
										replace
									/>
								)}
							</Preference>
						);
					})
				: [...Array(3)].map((_, i) => (
						<Preference
							key={i}
							customIcon={<Skeleton className="h-6 w-6" />}
							icon={null!}
							label={<Skeleton className="w-24" />}
							description={<Skeleton className="h-4 w-28" />}
						/>
					))}
		</SettingsContainer>
	);
};

OidcSettings.query = (): QueryIdentifier<User> => ({
	path: ["auth", "users", "me"],
	parser: User,
});

OidcSettings.authQuery = (): QueryIdentifier<AuthInfo> => ({
	path: ["auth", "info"],
	parser: AuthInfo,
});
