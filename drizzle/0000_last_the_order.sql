CREATE TABLE `playlistItems` (
	`id` text PRIMARY KEY NOT NULL,
	`playlistId` text NOT NULL,
	`path` text NOT NULL,
	`title` text NOT NULL,
	`currentTime` integer DEFAULT 0 NOT NULL,
	`isPlaying` integer DEFAULT false NOT NULL,
	`duration` real,
	`progressPercent` real DEFAULT 0 NOT NULL,
	`lastWatched` integer,
	`orderIndex` real NOT NULL,
	FOREIGN KEY (`playlistId`) REFERENCES `playlists`(`id`) ON UPDATE no action ON DELETE cascade
);
--> statement-breakpoint
CREATE INDEX `idx_playlistItems_playlistId_orderIndex` ON `playlistItems` (`playlistId`,`orderIndex`);--> statement-breakpoint
CREATE TABLE `playlists` (
	`id` text PRIMARY KEY NOT NULL,
	`name` text NOT NULL,
	`shuffle` integer DEFAULT false NOT NULL,
	`repeat` integer DEFAULT 0 NOT NULL,
	`currentItem` text,
	`currentVolume` real,
	FOREIGN KEY (`currentItem`) REFERENCES `playlistItems`(`id`) ON UPDATE no action ON DELETE set null
);
