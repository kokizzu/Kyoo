begin;

-- chapters
alter table gocoder.chapters add column sha varchar(40);
update gocoder.chapters c set sha = i.sha from gocoder.info i where c.id = i.id;
alter table gocoder.chapters alter column sha set not null;

alter table gocoder.chapters drop constraint chapter_pk;
alter table gocoder.chapters drop constraint chapters_info_fk;
alter table gocoder.chapters drop column id;
alter table gocoder.chapters add constraint chapter_pk primary key (sha, start_time);
alter table gocoder.chapters add foreign key (sha) references gocoder.info(sha) on delete cascade;

-- subtitles
alter table gocoder.subtitles add column sha varchar(40);
update gocoder.subtitles s set sha = i.sha from gocoder.info i where s.id = i.id;
alter table gocoder.subtitles alter column sha set not null;

alter table gocoder.subtitles drop constraint subtitle_pk;
alter table gocoder.subtitles drop constraint subtitles_info_fk;
alter table gocoder.subtitles drop column id;
alter table gocoder.subtitles add constraint subtitle_pk primary key (sha, idx);
alter table gocoder.subtitles add foreign key (sha) references gocoder.info(sha) on delete cascade;

-- audios
alter table gocoder.audios add column sha varchar(40);
update gocoder.audios a set sha = i.sha from gocoder.info i where a.id = i.id;
alter table gocoder.audios alter column sha set not null;

alter table gocoder.audios drop constraint audios_pk;
alter table gocoder.audios drop constraint audios_info_fk;
alter table gocoder.audios drop column id;
alter table gocoder.audios add constraint audios_pk primary key (sha, idx);
alter table gocoder.audios add foreign key (sha) references gocoder.info(sha) on delete cascade;

-- videos
alter table gocoder.videos add column sha varchar(40);
update gocoder.videos v set sha = i.sha from gocoder.info i where v.id = i.id;
alter table gocoder.videos alter column sha set not null;

alter table gocoder.videos drop constraint videos_pk;
alter table gocoder.videos drop constraint videos_info_fk;
alter table gocoder.videos drop column id;
alter table gocoder.videos add constraint videos_pk primary key (sha, idx);
alter table gocoder.videos add foreign key (sha) references gocoder.info(sha) on delete cascade;

alter table gocoder.info drop column id;

commit;
