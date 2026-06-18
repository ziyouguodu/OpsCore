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
	connected_status text not null default '已并网',
	host_machine text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

alter table if exists assets add column if not exists created_by bigint references users(id) on delete set null;
alter table if exists assets add column if not exists connected_status text not null default '已并网';
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
