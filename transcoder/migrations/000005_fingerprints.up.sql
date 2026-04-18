begin;

alter table gocoder.info add column ver_fingerprint integer not null default 0;
alter table gocoder.info add column ver_fp_with text[] not null default '{}';

create table gocoder.fingerprints(
	id integer not null primary key references gocoder.info(id) on delete cascade,
	start_data text not null,
	end_data text not null
);

alter table gocoder.chapters add column match_accuracy integer;
alter table gocoder.chapters add column first_appearance boolean;

commit;
