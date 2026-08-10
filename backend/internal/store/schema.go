package store

const schemaSQL = `
create table if not exists roles (
	id bigserial primary key,
	code text not null unique,
	name text not null
);

create table if not exists users (
	id bigserial primary key,
	username text not null unique,
	display_name text not null,
	password_hash text not null,
	must_change_password boolean not null default true,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create table if not exists user_roles (
	user_id bigint not null references users(id) on delete cascade,
	role_id bigint not null references roles(id) on delete cascade,
	primary key(user_id, role_id)
);

create table if not exists system_settings (
	key text primary key,
	value text not null,
	updated_at timestamptz not null default now()
);

create table if not exists assets (
	id bigserial primary key,
	created_by bigint references users(id) on delete set null,
	asset_no text not null unique,
	type text not null,
	vendor text not null default '',
	cpu_arch text not null default '',
	sn text not null default '',
	location text not null default '',
	business text not null default '',
	ipv4 text not null default '',
	ipv6 text not null default '',
	environment text not null default '',
	os text not null default '',
	hostname text not null,
	network_zone text not null default '',
	cpu text not null default '',
	memory text not null default '',
	disk text not null default '',
	deployment_info text not null default '',
	owner text not null default '',
	status text not null default '运行中',
	host_machine text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

alter table if exists assets add column if not exists created_by bigint references users(id) on delete set null;
alter table if exists assets add column if not exists host_machine text not null default '';

create table if not exists asset_credentials (
	asset_id bigint primary key references assets(id) on delete cascade,
	login_url text not null default '',
	username text not null default '',
	secret text not null default '',
	notes text not null default '',
	updated_at timestamptz not null default now()
);

create table if not exists middleware_instances (
	id bigserial primary key,
	name text not null,
	kind text not null check (kind in ('MySQL','Redis','Kafka','PostgreSQL','达梦','Nginx','ElasticSearch','Nacos','RocketMQ','MinIO')),
	version text not null default '',
	environment text not null default '',
	network_zone text not null default '',
	endpoint text not null default '',
	business text not null default '',
	owner text not null default '',
	status text not null default '运行中',
	asset_id bigint references assets(id) on delete set null,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

alter table if exists middleware_instances add column if not exists network_zone text not null default '';
alter table if exists middleware_instances drop constraint if exists middleware_instances_kind_check;
alter table if exists middleware_instances add constraint middleware_instances_kind_check check (kind in ('MySQL','Redis','Kafka','PostgreSQL','达梦','Nginx','ElasticSearch','Nacos','RocketMQ','MinIO'));

create table if not exists middleware_credentials (
	middleware_id bigint primary key references middleware_instances(id) on delete cascade,
	login_url text not null default '',
	username text not null default '',
	secret text not null default '',
	notes text not null default '',
	updated_at timestamptz not null default now()
);

create table if not exists oncall_schedules (
	id bigserial primary key,
	rule_type text not null check (rule_type in ('daily','weekly')),
	date_value text not null default '',
	week_value text not null default '',
	primary_user text not null,
	backup_user text not null default '',
	swap_from text not null default '',
	swap_to text not null default '',
	notes text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create table if not exists tasks (
	id bigserial primary key,
	title text not null,
	type text not null default '任务',
	assignee text not null default '',
	status text not null check (status in ('待处理','处理中','待确认','已完成','已关闭')),
	due_at text not null default '',
	description text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create table if not exists incidents (
	id bigserial primary key,
	title text not null,
	level text not null check (level in ('P1','P2','P3','P4')),
	status text not null check (status in ('新建','处理中','已恢复','已关闭')),
	owner text not null default '',
	business text not null default '',
	started_at text not null default '',
	recovered_at text not null default '',
	summary text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);
`

const dutyCenterSchemaSQL = `
create table if not exists duty_center_state (
	singleton boolean primary key default true check (singleton),
	revision bigint not null default 0,
	state jsonb not null default '{}'::jsonb,
	updated_by bigint references users(id) on delete set null,
	updated_at timestamptz not null default now()
);
`

const removeConnectedStatusSchemaSQL = `
alter table if exists assets drop column if exists connected_status;
`

const auditEventsSchemaSQL = `
create table if not exists audit_events (
	id bigserial primary key,
	actor_user_id bigint references users(id) on delete set null,
	actor_username text not null default '',
	action text not null,
	resource_type text not null,
	resource_id text not null default '',
	outcome text not null check (outcome in ('success','failure')),
	detail jsonb not null default '{}'::jsonb,
	ip_address text not null default '',
	user_agent text not null default '',
	created_at timestamptz not null default now()
);

create index if not exists audit_events_created_at_idx on audit_events(created_at desc);
create index if not exists audit_events_actor_idx on audit_events(actor_user_id, created_at desc);
create index if not exists audit_events_resource_idx on audit_events(resource_type, resource_id, created_at desc);
`

const listIndexesSchemaSQL = `
create index if not exists assets_updated_at_idx on assets(updated_at desc, id desc);
create index if not exists assets_filter_idx on assets(type, environment, business, network_zone, status);
create index if not exists middleware_updated_at_idx on middleware_instances(updated_at desc, id desc);
create index if not exists middleware_filter_idx on middleware_instances(kind, environment, business, network_zone, status);
create index if not exists tasks_updated_at_idx on tasks(updated_at desc, id desc);
create index if not exists tasks_status_idx on tasks(status, updated_at desc);
create index if not exists incidents_updated_at_idx on incidents(updated_at desc, id desc);
create index if not exists incidents_filter_idx on incidents(level, status, business, updated_at desc);
`

const typedDatesAndUserReferencesSchemaSQL = `
alter table tasks rename column due_at to due_at_legacy;
alter table tasks add column due_at timestamptz;
alter table tasks add column assignee_user_id bigint references users(id) on delete set null;
update tasks set due_at = case when pg_input_is_valid(due_at_legacy, 'timestamptz') then due_at_legacy::timestamptz else null end;
update tasks t set assignee_user_id=u.id from users u where t.assignee <> '' and (t.assignee=u.display_name or t.assignee=u.username);

alter table incidents rename column started_at to started_at_legacy;
alter table incidents rename column recovered_at to recovered_at_legacy;
alter table incidents add column started_at timestamptz;
alter table incidents add column recovered_at timestamptz;
alter table incidents add column owner_user_id bigint references users(id) on delete set null;
update incidents set started_at = case when pg_input_is_valid(started_at_legacy, 'timestamptz') then started_at_legacy::timestamptz else null end;
update incidents set recovered_at = case when pg_input_is_valid(recovered_at_legacy, 'timestamptz') then recovered_at_legacy::timestamptz else null end;
update incidents i set owner_user_id=u.id from users u where i.owner <> '' and (i.owner=u.display_name or i.owner=u.username);

alter table oncall_schedules rename column date_value to date_value_legacy;
alter table oncall_schedules add column date_value date;
alter table oncall_schedules add column primary_user_id bigint references users(id) on delete set null;
alter table oncall_schedules add column backup_user_id bigint references users(id) on delete set null;
update oncall_schedules set date_value = case when pg_input_is_valid(date_value_legacy, 'date') then date_value_legacy::date else null end;
update oncall_schedules o set primary_user_id=u.id from users u where o.primary_user <> '' and (o.primary_user=u.display_name or o.primary_user=u.username);
update oncall_schedules o set backup_user_id=u.id from users u where o.backup_user <> '' and (o.backup_user=u.display_name or o.backup_user=u.username);

create index if not exists tasks_due_at_idx on tasks(due_at) where due_at is not null;
create index if not exists tasks_assignee_user_idx on tasks(assignee_user_id, status);
create index if not exists incidents_started_at_idx on incidents(started_at desc) where started_at is not null;
create index if not exists incidents_owner_user_idx on incidents(owner_user_id, status);
create index if not exists oncall_date_idx on oncall_schedules(date_value);
`

const relationalDutyCenterSchemaSQL = `
alter table duty_center_state add column relations_revision bigint not null default 0;

create table duty_teams (
	name text primary key,
	sort_order integer not null
);

create table duty_members (
	id text primary key,
	user_id bigint not null unique references users(id) on delete restrict,
	username text not null,
	name text not null unique,
	team text not null references duty_teams(name) on update cascade on delete restrict,
	role text not null default '',
	duty_count integer not null default 0 check (duty_count >= 0),
	next_value text not null default '',
	status text not null default '',
	sort_order integer not null
);

create table duty_schedule_templates (
	id text primary key,
	name text not null,
	team text not null references duty_teams(name) on update cascade on delete restrict,
	rotation text not null check (rotation in ('daily','weekly')),
	time_window text not null default '',
	active boolean not null default true,
	sort_order integer not null
);

create table duty_schedule_members (
	schedule_id text not null references duty_schedule_templates(id) on delete cascade,
	member_id text not null references duty_members(id) on delete restrict,
	sort_order integer not null,
	primary key(schedule_id, member_id)
);

create table duty_assignments (
	date_value date primary key,
	primary_member_id text references duty_members(id) on delete restrict,
	backup_member_id text references duty_members(id) on delete restrict,
	check (primary_member_id is not null or backup_member_id is not null)
);

create table duty_current_people (
	id text primary key,
	member_id text not null references duty_members(id) on delete restrict,
	role text not null default '',
	team text not null references duty_teams(name) on update cascade on delete restrict,
	since_value text not null default '',
	until_value text not null default '',
	phone text not null default '',
	status text not null default '',
	sort_order integer not null
);

create table duty_handovers (
	id text primary key,
	from_member_id text not null references duty_members(id) on delete restrict,
	to_member_id text not null references duty_members(id) on delete restrict,
	time_value text not null default '',
	content text not null,
	complete boolean not null default false,
	sort_order integer not null
);

create table duty_escalation_policies (
	singleton boolean primary key default true check (singleton),
	name text not null,
	team text not null references duty_teams(name) on update cascade on delete restrict,
	severity text not null check (severity in ('P1','P2','P3','P4'))
);

create table duty_escalation_levels (
	singleton boolean not null references duty_escalation_policies(singleton) on delete cascade,
	level integer not null check (level > 0),
	target text not null,
	delay_value text not null,
	channel text not null,
	primary key(singleton, level)
);

create index duty_members_team_idx on duty_members(team, sort_order);
create index duty_schedule_team_idx on duty_schedule_templates(team, active);
create index duty_assignments_date_idx on duty_assignments(date_value);
create index duty_handovers_complete_idx on duty_handovers(complete, sort_order);
`

const trigramSearchIndexesSchemaSQL = `
create extension if not exists pg_trgm;
create index assets_search_trgm_idx on assets using gin ((asset_no || ' ' || business || ' ' || ipv4 || ' ' || ipv6 || ' ' || owner || ' ' || deployment_info || ' ' || hostname) gin_trgm_ops);
create index middleware_search_trgm_idx on middleware_instances using gin ((name || ' ' || kind || ' ' || endpoint || ' ' || business || ' ' || owner) gin_trgm_ops);
create index tasks_search_trgm_idx on tasks using gin ((title || ' ' || assignee || ' ' || description) gin_trgm_ops);
create index incidents_search_trgm_idx on incidents using gin ((title || ' ' || owner || ' ' || business || ' ' || summary) gin_trgm_ops);
`

const copilotModelProfilesSchemaSQL = `
create table copilot_model_configs (
	id bigserial primary key,
	name text not null,
	provider text not null check (provider in ('local','openai','anthropic','google','compatible')),
	endpoint text not null default '',
	model text not null default '',
	api_key_encrypted text not null default '',
	local_endpoint text not null default '',
	local_model text not null default '',
	temperature double precision not null default 0.2 check (temperature >= 0 and temperature <= 2),
	max_tokens integer not null default 2048 check (max_tokens >= 1 and max_tokens <= 4096),
	enable_asset_context boolean not null default true,
	enable_incident_context boolean not null default true,
	enable_task_context boolean not null default true,
	enable_oncall_context boolean not null default true,
	audit_enabled boolean not null default true check (audit_enabled),
	is_active boolean not null default false,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create unique index copilot_model_configs_name_unique_idx on copilot_model_configs(lower(name));
create unique index copilot_model_configs_active_unique_idx on copilot_model_configs(is_active) where is_active;

insert into copilot_model_configs(
	name, provider, endpoint, model, api_key_encrypted, local_endpoint, local_model,
	temperature, max_tokens, enable_asset_context, enable_incident_context,
	enable_task_context, enable_oncall_context, audit_enabled, is_active
)
select
	'默认模型配置',
	case when value::jsonb->>'provider' in ('local','openai','anthropic','google','compatible') then value::jsonb->>'provider' else 'compatible' end,
	coalesce(value::jsonb->>'endpoint', ''),
	coalesce(value::jsonb->>'model', ''),
	coalesce(value::jsonb->>'apiKeyEncrypted', ''),
	coalesce(value::jsonb->>'localEndpoint', ''),
	coalesce(value::jsonb->>'localModel', ''),
	case when coalesce(value::jsonb->>'temperature', '') ~ '^[0-9]+([.][0-9]+)?$' then least(2, greatest(0, (value::jsonb->>'temperature')::double precision)) else 0.2 end,
	case when coalesce(value::jsonb->>'maxTokens', '') ~ '^[0-9]+$' then least(4096, greatest(1, (value::jsonb->>'maxTokens')::integer)) else 2048 end,
	coalesce((value::jsonb->>'enableAssetContext')::boolean, true),
	coalesce((value::jsonb->>'enableIncidentContext')::boolean, true),
	coalesce((value::jsonb->>'enableTaskContext')::boolean, true),
	coalesce((value::jsonb->>'enableOncallContext')::boolean, true),
	true,
	true
from system_settings
where key='copilot_config'
on conflict do nothing;
`
