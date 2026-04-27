import { TypeCompiler } from "@sinclair/typebox/compiler";
import { t } from "elysia";
import { KError } from "./error";

export const User = t.Object({
	id: t.String({ format: "uuid" }),
	username: t.String(),
	email: t.String({ format: "email" }),
	createdDate: t.Date(),
	lastSeen: t.Date(),
	claims: t.Record(t.String(), t.Any()),
	oidc: t.Record(
		t.String(),
		t.Object({
			id: t.String(),
			username: t.String(),
			profileUrl: t.Nullable(t.String({ format: "url" })),
		}),
	),
});

export type User = typeof User.static;

export const UserC = TypeCompiler.Compile(t.Union([User, KError]));
