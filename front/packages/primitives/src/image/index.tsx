/*
 * Kyoo - A portable and vast media library solution.
 * Copyright (c) Kyoo.
 *
 * See AUTHORS.md and LICENSE file in the project root for full license information.
 *
 * Kyoo is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * any later version.
 *
 * Kyoo is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with Kyoo. If not, see <https://www.gnu.org/licenses/>.
 */

import { LinearGradient, type LinearGradientProps } from "expo-linear-gradient";
import type { ComponentProps, ComponentType, ReactElement, ReactNode } from "react";
import { type ImageStyle, View, type ViewProps, type ViewStyle } from "react-native";
import { percent } from "yoshiki/native";
import { useYoshiki } from "yoshiki/native";
import { imageBorderRadius } from "../constants";
import { ContrastArea } from "../themes";
import type { ImageLayout, Props, YoshikiEnhanced } from "./base-image";
import { Image } from "./image";

export { Sprite } from "./sprite";
export { BlurhashContainer } from "./blurhash";
export { type Props as ImageProps, Image };

export const Poster = ({
	alt,
	layout,
	...props
}: Props & { style?: ImageStyle } & {
	layout: YoshikiEnhanced<{ width: ImageStyle["width"] } | { height: ImageStyle["height"] }>;
}) => <Image alt={alt!} layout={{ aspectRatio: 2 / 3, ...layout }} {...props} />;

Poster.Loader = ({
	layout,
	...props
}: {
	children?: ReactElement;
	layout: YoshikiEnhanced<{ width: ImageStyle["width"] } | { height: ImageStyle["height"] }>;
}) => <Image.Loader layout={{ aspectRatio: 2 / 3, ...layout }} {...props} />;

export const PosterBackground = ({
	alt,
	layout,
	...props
}: Omit<ComponentProps<typeof ImageBackground>, "layout"> & { style?: ImageStyle } & {
	layout: YoshikiEnhanced<{ width: ImageStyle["width"] } | { height: ImageStyle["height"] }>;
}) => {
	const { css } = useYoshiki();

	return (
		<ImageBackground
			alt={alt!}
			layout={{ aspectRatio: 2 / 3, ...layout }}
			{...css({ borderRadius: imageBorderRadius }, props)}
		/>
	);
};

type ImageBackgroundProps = {
	children?: ReactNode;
	containerStyle?: YoshikiEnhanced<ViewStyle>;
	imageStyle?: YoshikiEnhanced<ImageStyle>;
	layout?: ImageLayout;
	contrast?: "light" | "dark" | "user";
};

export const ImageBackground = <AsProps = ViewProps>({
	src,
	alt,
	quality,
	as,
	children,
	containerStyle,
	imageStyle,
	layout,
	contrast = "dark",
	imageSibling,
	...asProps
}: {
	as?: ComponentType<AsProps>;
	imageSibling?: ReactElement;
} & AsProps &
	ImageBackgroundProps &
	Props) => {
	const Container = as ?? View;

	return (
		<ContrastArea contrastText mode={contrast}>
			{({ css }) => (
				<Container {...(css([layout, { overflow: "hidden" }], asProps) as AsProps)}>
					<View
						{...css([
							{
								position: "absolute",
								top: 0,
								bottom: 0,
								left: 0,
								right: 0,
								zIndex: -1,
								bg: (theme) => theme.background,
							},
							containerStyle,
						])}
					>
						{src && (
							<Image
								src={src}
								quality={quality}
								alt={alt!}
								layout={{ width: percent(100), height: percent(100) }}
								{...(css([{ borderWidth: 0, borderRadius: 0 }, imageStyle]) as {
									style: ImageStyle;
								})}
							/>
						)}
						{imageSibling}
					</View>
					{children}
				</Container>
			)}
		</ContrastArea>
	);
};

export const GradientImageBackground = <AsProps = ViewProps>({
	contrast = "dark",
	gradient,
	...props
}: {
	as?: ComponentType<AsProps>;
	gradient?: Partial<LinearGradientProps>;
} & AsProps &
	ImageBackgroundProps &
	Props) => {
	const { css, theme } = useYoshiki();

	return (
		<ImageBackground
			contrast={contrast}
			imageSibling={
				<LinearGradient
					start={{ x: 0, y: 0.25 }}
					end={{ x: 0, y: 1 }}
					colors={["transparent", theme[contrast].darkOverlay]}
					{...css(
						{
							position: "absolute",
							top: 0,
							bottom: 0,
							left: 0,
							right: 0,
						},
						typeof gradient === "object" ? gradient : undefined,
					)}
				/>
			}
			{...(props as any)}
		/>
	);
};
