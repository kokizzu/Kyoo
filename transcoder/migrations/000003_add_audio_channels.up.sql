begin;

alter table gocoder.audios add column channels int not null default 2;

commit;
