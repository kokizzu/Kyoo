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

import { Platform } from "react-native";
import { useMMKVString } from "react-native-mmkv";
import { z } from "zod";
import { storage } from "./account-internal";
import type { Movie, Show } from "./resources";

export const zdate = z.coerce.date;

export const useLocalSetting = (setting: string, def: string) => {
	if (Platform.OS === "web" && typeof window === "undefined") return [def, null!] as const;
	// eslint-disable-next-line react-hooks/rules-of-hooks
	const [val, setter] = useMMKVString(`settings.${setting}`, storage);
	return [val ?? def, setter] as const;
};

export const getLocalSetting = (setting: string, def: string) => {
	if (Platform.OS === "web" && typeof window === "undefined") return def;
	return storage.getString(`settings.${setting}`) ?? setting;
};
