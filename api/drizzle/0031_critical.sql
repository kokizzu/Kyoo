ALTER TABLE "kyoo"."entries" ADD COLUMN "critical_to_story" boolean;--> statement-breakpoint
UPDATE "kyoo"."entries" SET "critical_to_story" = true;--> statement-breakpoint
ALTER TABLE "kyoo"."entries" ALTER COLUMN "critical_to_story" SET NOT NULl;--> statement-breakpoint
