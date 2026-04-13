begin;

alter table gocoder.chapters drop column match_accuracy;
alter table gocoder.chapters drop column fingerprint_id;
drop table gocoder.chapterprints;
drop table gocoder.fingerprints;
alter table gocoder.info drop column ver_fingerprint;

commit;
