\set ON_ERROR_STOP on
set timezone = 'UTC';
set datestyle = 'ISO, YMD';
set intervalstyle = 'postgres';
create temporary table upgrade_table_fingerprints (table_name text, row_count bigint, row_fingerprint text);
do $$
declare item record;
begin
  for item in select tablename from pg_tables where schemaname = 'public' order by tablename loop
    execute format('insert into upgrade_table_fingerprints select %L, count(*), md5(coalesce(string_agg(row_hash, '''' order by row_hash), '''')) from (select md5(to_jsonb(t)::text) as row_hash from public.%I t) rows', item.tablename, item.tablename);
  end loop;
end $$;
select table_name || '|' || row_count || '|' || row_fingerprint from upgrade_table_fingerprints order by table_name;
select 'sequence|' || sequencename || '|' || coalesce(last_value::text, '') || '|' || start_value || '|' || increment_by || '|' || min_value || '|' || max_value || '|' || cycle from pg_sequences where schemaname = 'public' order by sequencename;
select 'constraint|' || c.conrelid::regclass::text || '|' || c.conname || '|' || pg_get_constraintdef(c.oid) from pg_constraint c join pg_namespace n on n.oid=c.connamespace where n.nspname='public' and c.contype <> 'n' order by c.conrelid::regclass::text, c.conname;
select 'column|' || table_name || '|' || column_name || '|' || is_nullable || '|' || data_type from information_schema.columns where table_schema='public' order by table_name, ordinal_position;
select 'index|' || tablename || '|' || indexname || '|' || indexdef from pg_indexes where schemaname='public' order by tablename, indexname;
