CREATE TYPE "kyoo"."entry_content" AS ENUM('story', 'recap', 'filler', 'ova');--> statement-breakpoint
ALTER TABLE "kyoo"."entries" ADD COLUMN "content" "kyoo"."entry_content";--> statement-breakpoint
UPDATE "kyoo"."entries" SET content = 'story';-->statement-breakpoint
ALTER TABLE "kyoo"."entries" ALTER COLUMN "content" SET NOT NULl;--> statement-breakpoint
