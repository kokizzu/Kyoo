import type { Property } from "csstype";
import type { ReactNode } from "react";
import { Platform, type TextStyle } from "react-native";
import {
	type Theme,
	useAutomaticTheme,
	ThemeProvider as WebThemeProvider,
} from "yoshiki";
import "yoshiki";
import { ThemeProvider, useTheme, useYoshiki } from "yoshiki/native";
import "yoshiki/native";
import { catppuccin } from "./catppuccin";

type FontList = Partial<
	Record<Exclude<TextStyle["fontWeight"], null | undefined | number>, string>
>;

type Mode = {
	mode: "light" | "dark" | "auto";
	overlay0: Property.Color;
	overlay1: Property.Color;
	lightOverlay: Property.Color;
	darkOverlay: Property.Color;
	themeOverlay: Property.Color;
	link: Property.Color;
	contrast: Property.Color;
	variant: Variant;
	colors: {
		red: Property.Color;
		green: Property.Color;
		blue: Property.Color;
		yellow: Property.Color;
		white: Property.Color;
		black: Property.Color;
	};
};

type Variant = {
	background: Property.Color;
	accent: Property.Color;
	divider: Property.Color;
	heading: Property.Color;
	paragraph: Property.Color;
	subtext: Property.Color;
};

declare module "yoshiki" {
	export interface Theme extends Mode, Variant {
		light: Mode & Variant;
		dark: Mode & Variant;
		user: Mode & Variant;
		alternate: Mode & Variant;
		font: FontList;
	}
}

export type { Theme } from "yoshiki";
export type ThemeBuilder = {
	light: Omit<Mode, "contrast" | "mode" | "themeOverlay"> & {
		default: Variant;
	};
	dark: Omit<Mode, "contrast" | "mode" | "themeOverlay"> & { default: Variant };
};

const selectMode = (
	theme: ThemeBuilder & { font: FontList },
	mode: "light" | "dark" | "auto",
): Theme => {
	const { light: lightBuilder, dark: darkBuilder, ...options } = theme;
	const light: Mode & Variant = {
		...lightBuilder,
		...lightBuilder.default,
		contrast: lightBuilder.colors.black,
		themeOverlay: lightBuilder.lightOverlay,
		mode: "light",
	};
	const dark: Mode & Variant = {
		...darkBuilder,
		...darkBuilder.default,
		contrast: darkBuilder.colors.white,
		themeOverlay: darkBuilder.darkOverlay,
		mode: "dark",
	};
	if (Platform.OS !== "web" || mode !== "auto") {
		const value = mode === "light" ? light : dark;
		const alternate = mode === "light" ? dark : light;
		return {
			...options,
			...value,
			light,
			dark,
			user: value,
			alternate,
		};
	}

	// biome-ignore lint/correctness/useHookAtTopLevel: const
	const auto = useAutomaticTheme("theme", { light, dark });
	// biome-ignore lint/correctness/useHookAtTopLevel: const
	const alternate = useAutomaticTheme("alternate", {
		dark: light,
		light: dark,
	});
	return {
		...options,
		...auto,
		mode: "auto",
		light,
		dark,
		user: { ...auto, mode: "auto" },
		alternate: { ...alternate, mode: "auto" },
	};
};

const switchVariant = (theme: Theme) => {
	return {
		...theme,
		...theme.variant,
		variant: {
			background: theme.background,
			accent: theme.accent,
			divider: theme.divider,
			heading: theme.heading,
			paragraph: theme.paragraph,
			subtext: theme.subtext,
		},
	};
};

export const ThemeSelector = ({
	children,
	theme,
	font,
}: {
	children: ReactNode;
	theme: "light" | "dark" | "auto";
	font: FontList;
}) => {
	const newTheme = selectMode({ ...catppuccin, font }, theme);

	return (
		<ThemeProvider theme={newTheme}>
			<WebThemeProvider theme={newTheme}>{children as any}</WebThemeProvider>
		</ThemeProvider>
	);
};

export type YoshikiFunc<T> = (props: ReturnType<typeof useYoshiki>) => T;

const YoshikiProvider = ({
	children,
}: {
	children: YoshikiFunc<ReactNode>;
}) => {
	const yoshiki = useYoshiki();
	return <>{children(yoshiki)}</>;
};

export const SwitchVariant = ({
	children,
}: {
	children: ReactNode | YoshikiFunc<ReactNode>;
}) => {
	const theme = useTheme();

	return (
		<ThemeProvider theme={switchVariant(theme)}>
			{typeof children === "function" ? (
				<YoshikiProvider>{children}</YoshikiProvider>
			) : (
				(children as any)
			)}
		</ThemeProvider>
	);
};

export const ContrastArea = ({
	children,
	mode = "dark",
	contrastText,
}: {
	children: ReactNode | YoshikiFunc<ReactNode>;
	mode?: "light" | "dark" | "user" | "alternate";
	contrastText?: boolean;
}) => {
	const oldTheme = useTheme();
	const theme: Theme = { ...oldTheme, ...oldTheme[mode] };

	return (
		<ThemeProvider
			theme={
				contrastText
					? {
							...theme,
							// Keep the same skeletons, it looks weird otherwise.
							overlay0: theme.user.overlay0,
							overlay1: theme.user.overlay1,
							heading: theme.contrast,
							paragraph: theme.heading,
						}
					: theme
			}
		>
			{typeof children === "function" ? (
				<YoshikiProvider>{children}</YoshikiProvider>
			) : (
				(children as any)
			)}
		</ThemeProvider>
	);
};

export const alpha = (color: Property.Color, alpha: number) => {
	return color + Math.round(alpha * 255).toString(16);
};
