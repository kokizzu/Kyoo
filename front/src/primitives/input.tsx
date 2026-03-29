import type { ReactNode, Ref } from "react";
import { TextInput, type TextInputProps, View } from "react-native";
import { cn } from "~/utils";

export const Input = ({
	left,
	right,
	containerClassName,
	ref,
	className,
	...props
}: {
	left?: ReactNode;
	right?: ReactNode;
	containerClassName?: string;
	ref?: Ref<TextInput>;
} & TextInputProps) => {
	return (
		<View
			className={cn(
				"shrink flex-row content-center items-center rounded-xl border border-accent p-2",
				"ring-accent focus-within:ring-2",
				containerClassName,
			)}
		>
			{left}
			<TextInput
				ref={ref}
				textAlignVertical="center"
				className={cn(
					"min-h-6 flex-1 font-sans text-base text-slate-600 outline-0 dark:text-slate-400",
					className,
				)}
				{...props}
			/>
			{right}
		</View>
	);
};
