CREATE TABLE texts (
  `id` integer not null primary key autoincrement,
  `title` varchar(255),
  `tag_line` text,
  `details` text,
  `date` date,
  `rating` tinyint,
  `studio_id` integer,
  `organized` boolean not null default '0',
  `created_at` datetime not null,
  `updated_at` datetime not null,
  `code` text,
  `author` text,
  `language_code` varchar(255),
  `resume_location` text,
  `play_duration` float not null default 0,
  `cover_blob` varchar (255) references `blobs` (checksum),
  foreign key (`studio_id`) references `studios` (`id`) on delete set null
);

CREATE TABLE `performers_texts` (
  `performer_id` integer,
  `text_id` integer,
  foreign key(`performer_id`) references `performers`(`id`),
  foreign key(`text_id`) references `texts`(`id`)
);

CREATE INDEX `index_performers_texts_on_text_id` on `performers_texts` (`text_id`);
CREATE INDEX `index_performers_texts_on_performer_id` on `performers_texts` (`performer_id`);

CREATE TABLE `texts_tags` (
  `text_id` integer,
  `tag_id` integer,
  foreign key(`text_id`) references `texts`(`id`) on delete CASCADE,
  foreign key(`tag_id`) references `tags`(`id`)
);

CREATE INDEX `index_texts_tags_on_tag_id` on `texts_tags` (`tag_id`);
CREATE INDEX `index_texts_tags_on_text_id` on `texts_tags` (`text_id`);

CREATE TABLE `text_urls` (
  `text_id` integer NOT NULL,
  `position` integer NOT NULL,
  `url` varchar(255) NOT NULL,
  foreign key(`text_id`) references `texts`(`id`) on delete CASCADE,
  PRIMARY KEY(`text_id`, `position`, `url`)
);

CREATE INDEX `text_urls_url` on `text_urls` (`url`);

CREATE TABLE `texts_read_dates` (
  `text_id` integer,
  `read_date` datetime not null,
  `read_duration` tinyint
  foreign key(`text_id`) references `texts`(`id`) on delete CASCADE
);

CREATE TABLE `texts_o_dates` (
  `text_id` integer,
  `o_date` datetime not null,
  foreign key(`text_id`) references `texts`(`id`) on delete CASCADE
);

CREATE TABLE `texts_files` (
    `text_id` integer NOT NULL,
    `file_id` integer NOT NULL,
    `primary` boolean NOT NULL,
    foreign key(`text_id`) references `text`(`id`) on delete CASCADE,
    foreign key(`file_id`) references `files`(`id`) on delete CASCADE,
    PRIMARY KEY(`text_id`, `file_id`)
);

CREATE INDEX `index_texts_files_file_id` ON `texts_files` (`file_id`);
CREATE UNIQUE INDEX `unique_index_texts_files_on_primary` on `texts_files` (`text_id`) WHERE `primary` = 1;

CREATE TABLE text_chapters (
  `id` integer not null primary key autoincrement,
  `text_id` integer,
  `chapter_index` integer,
  `title` varchar(255),
  `details` text,
  `content` text,
  `date` date,
  `rating` tinyint,
  `organized` boolean not null default '0',
  `created_at` datetime not null,
  `updated_at` datetime not null,
  `code` text,
  `author` text,
  `resume_location` text,
  `play_duration` float not null default 0,
  `cover_blob` varchar (255) references `blobs` (checksum),
  foreign key (`text_id`) references `texts` (`id`) on delete CASCADE
);

CREATE TABLE `text_bookmarks` (
  `id` integer not null primary key autoincrement,
  `text_id` integer,
  `chapter_id` integer,
  `location` text,
  `created_at` datetime not null,
  `updated_at` datetime not null,
  foreign key(`text_id`) references `texts`(`id`)
);

CREATE TABLE `text_bookmarks_tags` (
  `text_bookmark_id` integer,
  `tag_id` integer,
  foreign key(`text_bookmark_id`) references `text_bookmarks`(`id`) on delete CASCADE,
  foreign key(`tag_id`) references `tags`(`id`)
);