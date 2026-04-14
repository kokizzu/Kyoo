import { getLogger } from "@logtape/logtape";
import { eq, and } from "drizzle-orm";
import { Elysia, t } from "elysia";
import slugify from "slugify";
import { auth } from "~/auth";
import { db } from "~/db";
import { entries, entryVideoJoin, videos } from "~/db/schema";
import { KError } from "~/models/error";
import { isUuid } from "~/models/utils";
import { Video } from "~/models/video";

const logger = getLogger();

export const videosMetadata = new Elysia({
	prefix: "/videos",
	tags: ["videos"],
})
	.model({
		video: Video,
		error: t.Object({}),
	})
	.use(auth)
	.get(
		":id/info",
		async ({ params: { id }, status, redirect }) => {
			const [video] = await db
				.select({
					path: videos.path,
				})
				.from(videos)
				.leftJoin(entryVideoJoin, eq(videos.pk, entryVideoJoin.videoPk))
				.where(isUuid(id) ? eq(videos.id, id) : eq(entryVideoJoin.slug, id))
				.limit(1);

			if (!video) {
				return status(404, {
					status: 404,
					message: `No video found with id or slug '${id}'`,
				});
			}
			const path = Buffer.from(video.path, "utf8").toString("base64url");
			return redirect(`/video/${path}/info`);
		},
		{
			detail: { description: "Get a video's metadata informations" },
			params: t.Object({
				id: t.String({
					description: "The id or slug of the video to retrieve.",
					example: "made-in-abyss-s1e13",
				}),
			}),
			response: {
				302: t.Void({
					description:
						"Redirected to the [/video/{path}/info](?api=transcoder#tag/metadata/get/:path/info) route (of the transcoder)",
				}),
				404: {
					...KError,
					description: "No video found with the given id or slug.",
				},
			},
		},
	)
	.get(
		":id/thumbnails.vtt",
		async ({ params: { id }, status, redirect }) => {
			const [video] = await db
				.select({
					path: videos.path,
				})
				.from(videos)
				.leftJoin(entryVideoJoin, eq(videos.pk, entryVideoJoin.videoPk))
				.where(isUuid(id) ? eq(videos.id, id) : eq(entryVideoJoin.slug, id))
				.limit(1);

			if (!video) {
				return status(404, {
					status: 404,
					message: `No video found with id or slug '${id}'`,
				});
			}
			const path = Buffer.from(video.path, "utf8").toString("base64url");
			return redirect(`/video/${path}/thumbnails.vtt`);
		},
		{
			detail: {
				description: "Get redirected to the direct stream of the video",
			},
			params: t.Object({
				id: t.String({
					description: "The id or slug of the video to watch.",
					example: "made-in-abyss-s1e13",
				}),
			}),
			response: {
				302: t.Void({
					description:
						"Redirected to the [/video/{path}/direct](?api=transcoder#tag/metadata/get/:path/direct) route (of the transcoder)",
				}),
				404: {
					...KError,
					description: "No video found with the given id or slug.",
				},
			},
		},
	)
	.get(
		":id/direct",
		async ({ params: { id }, status, redirect }) => {
			const [video] = await db
				.select({
					path: videos.path,
				})
				.from(videos)
				.leftJoin(entryVideoJoin, eq(videos.pk, entryVideoJoin.videoPk))
				.where(isUuid(id) ? eq(videos.id, id) : eq(entryVideoJoin.slug, id))
				.limit(1);

			if (!video) {
				return status(404, {
					status: 404,
					message: `No video found with id or slug '${id}'`,
				});
			}
			const path = Buffer.from(video.path, "utf8").toString("base64url");
			const filename = video.path.substring(video.path.lastIndexOf("/") + 1);
			return redirect(
				`/video/${path}/direct/${slugify(filename, { lower: true })}`,
			);
		},
		{
			detail: {
				description: "Get redirected to the direct stream of the video",
			},
			params: t.Object({
				id: t.String({
					description: "The id or slug of the video to watch.",
					example: "made-in-abyss-s1e13",
				}),
			}),
			response: {
				302: t.Void({
					description:
						"Redirected to the [/video/{path}/direct](?api=transcoder#tag/metadata/get/:path/direct) route (of the transcoder)",
				}),
				404: {
					...KError,
					description: "No video found with the given id or slug.",
				},
			},
		},
	)
	.get(
		":id/master.m3u8",
		async ({ params: { id }, request, status, redirect }) => {
			const [video] = await db
				.select({
					path: videos.path,
				})
				.from(videos)
				.leftJoin(entryVideoJoin, eq(videos.pk, entryVideoJoin.videoPk))
				.where(isUuid(id) ? eq(videos.id, id) : eq(entryVideoJoin.slug, id))
				.limit(1);

			if (!video) {
				return status(404, {
					status: 404,
					message: `No video found with id or slug '${id}'`,
				});
			}
			const path = Buffer.from(video.path, "utf8").toString("base64url");
			const query = request.url.substring(request.url.indexOf("?"));
			return redirect(`/video/${path}/master.m3u8${query}`);
		},
		{
			detail: { description: "Get redirected to the master.m3u8 of the video" },
			params: t.Object({
				id: t.String({
					description: "The id or slug of the video to watch.",
					example: "made-in-abyss-s1e13",
				}),
			}),
			response: {
				302: t.Void({
					description:
						"Redirected to the [/video/{path}/master.m3u8](?api=transcoder#tag/metadata/get/:path/master.m3u8) route (of the transcoder)",
				}),
				404: {
					...KError,
					description: "No video found with the given id or slug.",
				},
			},
		},
	)
	.get(
		":id/prepare",
		async ({ params: { id }, headers: { authorization } }) => {
			await prepareVideo(id, authorization!);
		},
		{
			detail: { description: "Prepare a video for playback" },
			params: t.Object({
				id: t.String({
					description: "The id or slug of the video to watch.",
					example: "made-in-abyss-s1e13",
				}),
			}),
			response: {
				302: t.Void({
					description:
						"Prepare said video for playback (compute everything possible and cache it)",
				}),
				404: {
					...KError,
					description: "No video found with the given id or slug.",
				},
			},
		},
	);

export const prepareVideo = async (slug: string, auth: string) => {
	logger.info("Preparing next video {slug}", { slug });
	const [vid] = await db
		.select({ path: videos.path, show: entries.showPk, order: entries.order })
		.from(videos)
		.innerJoin(entryVideoJoin, eq(videos.pk, entryVideoJoin.videoPk))
		.leftJoin(entries, eq(entries.pk, entryVideoJoin.entryPk))
		.where(eq(entryVideoJoin.slug, slug))
		.limit(1);

	const related = vid.show
		? await db
				.select({ order: entries.order, path: videos.path })
				.from(entries)
				.innerJoin(entryVideoJoin, eq(entries.pk, entryVideoJoin.entryPk))
				.innerJoin(videos, eq(videos.pk, entryVideoJoin.videoPk))
				.where(and(eq(entries.showPk, vid.show), eq(entries.kind, "episode")))
				.orderBy(entries.order)
		: [];
	const idx = related.findIndex((x) => x.order === vid.order);

	const path = Buffer.from(vid.path, "utf8").toString("base64url");
	await fetch(
		new URL(
			`/video/${path}/prepare`,
			process.env.TRANSCODER_SERVER ?? "http://transcoder:7666",
		),
		{
			headers: {
				authorization: auth,
				"content-type": "application/json",
			},
			method: "POST",
			body: JSON.stringify({
				nearEpisodes: [related[idx - 1], related[idx + 1]]
					.filter((x) => x)
					.map((x) => x.path),
			}),
		},
	);
};
