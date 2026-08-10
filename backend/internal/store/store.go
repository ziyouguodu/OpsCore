package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"opscore/backend/internal/auth"
	secretcrypto "opscore/backend/internal/crypto"
	"opscore/backend/internal/models"
)

type Store struct {
	pool          *pgxpool.Pool
	credentialBox secretcrypto.SecretBox
}

var ErrForbiddenAssetDelete = errors.New("only super admin or asset creator can delete asset")
var ErrLastSuperAdmin = errors.New("at least one super admin is required")
var ErrUserAssignedToDuty = errors.New("user is assigned to the duty roster and must be removed there first")
var ErrStatusConflict = errors.New("status changed concurrently or transition is invalid")

const credentialVerificationPasswordKey = "credential_verification_password_hash"

func Open(ctx context.Context, databaseURL string, credentialBox secretcrypto.SecretBox) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Store{pool: pool, credentialBox: credentialBox}, nil
}

func (s *Store) Close() {
	s.pool.Close()
}

func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

func (s *Store) SeedDefaults(ctx context.Context, adminPassword string) error {
	if err := s.runMigrations(ctx); err != nil {
		return err
	}

	passwordHash, err := auth.HashPassword(adminPassword)
	if err != nil {
		return err
	}

	if _, err := s.pool.Exec(ctx, `
		insert into roles(code, name) values
			('super_admin', '超级管理员'),
			('ops_engineer', '运维工程师')
		on conflict (code) do nothing
	`); err != nil {
		return err
	}
	if _, err := s.pool.Exec(ctx, `
		insert into users(username, display_name, password_hash, must_change_password)
		values ('admin', '超级管理员', $1, true)
		on conflict (username) do nothing
	`, passwordHash); err != nil {
		return err
	}
	if _, err := s.pool.Exec(ctx, `
		insert into user_roles(user_id, role_id)
		select u.id, r.id from users u, roles r
		where u.username = 'admin' and r.code = 'super_admin'
		on conflict do nothing
	`); err != nil {
		return err
	}
	if err := s.ensureDutyRelations(ctx); err != nil {
		return err
	}
	if err := s.encryptLegacyAssetCredentials(ctx); err != nil {
		return err
	}
	return s.encryptLegacyMiddlewareCredentials(ctx)
}

func (s *Store) Authenticate(ctx context.Context, username, password string) (models.User, bool, error) {
	row := s.pool.QueryRow(ctx, `select id, username, display_name, password_hash, must_change_password, created_at from users where username=$1`, username)
	var user models.User
	var passwordHash string
	if err := row.Scan(&user.ID, &user.Username, &user.DisplayName, &passwordHash, &user.MustChangePassword, &user.CreatedAt); err != nil {
		return models.User{}, false, err
	}
	if !auth.VerifyPassword(passwordHash, password) {
		return models.User{}, false, nil
	}
	roles, err := s.UserRoles(ctx, user.ID)
	if err != nil {
		return models.User{}, false, err
	}
	user.Roles = roles
	return user, true, nil
}

func (s *Store) GetUser(ctx context.Context, userID int64) (models.User, error) {
	row := s.pool.QueryRow(ctx, `select id, username, display_name, must_change_password, created_at from users where id=$1`, userID)
	var user models.User
	if err := row.Scan(&user.ID, &user.Username, &user.DisplayName, &user.MustChangePassword, &user.CreatedAt); err != nil {
		return models.User{}, err
	}
	roles, err := s.UserRoles(ctx, user.ID)
	if err != nil {
		return models.User{}, err
	}
	user.Roles = roles
	return user, nil
}

func (s *Store) ChangePassword(ctx context.Context, userID int64, currentPassword string, newPassword string) (models.User, error) {
	row := s.pool.QueryRow(ctx, `select username, password_hash from users where id=$1`, userID)
	var username string
	var passwordHash string
	if err := row.Scan(&username, &passwordHash); err != nil {
		return models.User{}, err
	}
	if !auth.VerifyPassword(passwordHash, currentPassword) {
		return models.User{}, errors.New("current password is invalid")
	}
	nextHash, err := auth.HashPassword(newPassword)
	if err != nil {
		return models.User{}, err
	}
	if _, err := s.pool.Exec(ctx, `update users set password_hash=$2, must_change_password=false, updated_at=now() where id=$1`, userID, nextHash); err != nil {
		return models.User{}, err
	}
	return s.GetUser(ctx, userID)
}

func (s *Store) ResetUserPassword(ctx context.Context, username string, newPassword string, mustChangePassword bool) (models.User, error) {
	passwordHash, err := auth.HashPassword(newPassword)
	if err != nil {
		return models.User{}, err
	}
	row := s.pool.QueryRow(ctx, `
		update users
		set password_hash=$2, must_change_password=$3, updated_at=now()
		where username=$1
		returning id
	`, username, passwordHash, mustChangePassword)
	var userID int64
	if err := row.Scan(&userID); err != nil {
		return models.User{}, err
	}
	return s.GetUser(ctx, userID)
}

func (s *Store) UserRoles(ctx context.Context, userID int64) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		select r.code from roles r
		join user_roles ur on ur.role_id = r.id
		where ur.user_id = $1
		order by r.code
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var roles []string
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}

func (s *Store) ListUsers(ctx context.Context) ([]models.UserListItem, error) {
	rows, err := s.pool.Query(ctx, `
		select
			u.id,
			u.username,
			u.display_name,
			u.must_change_password,
			coalesce(array_agg(r.code order by r.code) filter (where r.code is not null), '{}') as roles,
			u.created_at,
			u.updated_at
		from users u
		left join user_roles ur on ur.user_id = u.id
		left join roles r on r.id = ur.role_id
		group by u.id
		order by u.created_at desc
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []models.UserListItem{}
	for rows.Next() {
		var item models.UserListItem
		if err := rows.Scan(&item.ID, &item.Username, &item.DisplayName, &item.MustChangePassword, &item.Roles, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) ListUserDirectory(ctx context.Context) ([]models.UserDirectoryItem, error) {
	rows, err := s.pool.Query(ctx, `select id, username, display_name from users order by display_name, username`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]models.UserDirectoryItem, 0)
	for rows.Next() {
		var item models.UserDirectoryItem
		if err := rows.Scan(&item.ID, &item.Username, &item.DisplayName); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CreateUser(ctx context.Context, item models.UserMutation) (models.UserListItem, error) {
	if item.Username == "" || item.DisplayName == "" || item.Password == "" || len(item.Roles) == 0 {
		return models.UserListItem{}, errors.New("username, displayName, password and roles are required")
	}
	passwordHash, err := auth.HashPassword(item.Password)
	if err != nil {
		return models.UserListItem{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return models.UserListItem{}, err
	}
	defer tx.Rollback(ctx)

	var id int64
	if err := tx.QueryRow(ctx, `
		insert into users(username, display_name, password_hash, must_change_password)
		values ($1,$2,$3,$4)
		returning id
	`, item.Username, item.DisplayName, passwordHash, item.MustChangePassword).Scan(&id); err != nil {
		return models.UserListItem{}, err
	}
	if err := setUserRoles(ctx, tx, id, item.Roles); err != nil {
		return models.UserListItem{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return models.UserListItem{}, err
	}
	return s.userListItem(ctx, id)
}

func (s *Store) UpdateUser(ctx context.Context, id int64, item models.UserMutation) (models.UserListItem, error) {
	if item.Username == "" || item.DisplayName == "" || len(item.Roles) == 0 {
		return models.UserListItem{}, errors.New("username, displayName and roles are required")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return models.UserListItem{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `select pg_advisory_xact_lock(hashtext('opscore:super-admin-role'))`); err != nil {
		return models.UserListItem{}, err
	}
	if !hasRoleCode(item.Roles, auth.RoleSuperAdmin) {
		var targetIsAdmin bool
		var adminCount int
		if err := tx.QueryRow(ctx, `select exists(select 1 from user_roles ur join roles r on r.id=ur.role_id where ur.user_id=$1 and r.code=$2)`, id, auth.RoleSuperAdmin).Scan(&targetIsAdmin); err != nil {
			return models.UserListItem{}, err
		}
		if targetIsAdmin {
			if err := tx.QueryRow(ctx, `select count(*) from user_roles ur join roles r on r.id=ur.role_id where r.code=$1`, auth.RoleSuperAdmin).Scan(&adminCount); err != nil {
				return models.UserListItem{}, err
			}
			if adminCount <= 1 {
				return models.UserListItem{}, ErrLastSuperAdmin
			}
		}
	}

	tag, err := tx.Exec(ctx, `
		update users set username=$2, display_name=$3, must_change_password=$4, updated_at=now()
		where id=$1
	`, id, item.Username, item.DisplayName, item.MustChangePassword)
	if err != nil {
		return models.UserListItem{}, err
	}
	if tag.RowsAffected() == 0 {
		return models.UserListItem{}, errors.New("user not found")
	}
	if _, err := tx.Exec(ctx, `update duty_members set username=$2, name=$3 where user_id=$1`, id, item.Username, item.DisplayName); err != nil {
		return models.UserListItem{}, err
	}
	for _, query := range []string{
		`update tasks set assignee=$2 where assignee_user_id=$1`,
		`update incidents set owner=$2 where owner_user_id=$1`,
		`update oncall_schedules set primary_user=$2 where primary_user_id=$1`,
		`update oncall_schedules set backup_user=$2 where backup_user_id=$1`,
	} {
		if _, err := tx.Exec(ctx, query, id, item.DisplayName); err != nil {
			return models.UserListItem{}, err
		}
	}
	if item.Password != "" {
		passwordHash, err := auth.HashPassword(item.Password)
		if err != nil {
			return models.UserListItem{}, err
		}
		if _, err := tx.Exec(ctx, `update users set password_hash=$2, updated_at=now() where id=$1`, id, passwordHash); err != nil {
			return models.UserListItem{}, err
		}
	}
	if err := setUserRoles(ctx, tx, id, item.Roles); err != nil {
		return models.UserListItem{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return models.UserListItem{}, err
	}
	return s.userListItem(ctx, id)
}

func (s *Store) DeleteUser(ctx context.Context, id int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `select pg_advisory_xact_lock(hashtext('opscore:super-admin-role'))`); err != nil {
		return err
	}
	var targetIsAdmin bool
	var adminCount int
	if err := tx.QueryRow(ctx, `select exists(select 1 from user_roles ur join roles r on r.id=ur.role_id where ur.user_id=$1 and r.code=$2)`, id, auth.RoleSuperAdmin).Scan(&targetIsAdmin); err != nil {
		return err
	}
	if targetIsAdmin {
		if err := tx.QueryRow(ctx, `select count(*) from user_roles ur join roles r on r.id=ur.role_id where r.code=$1`, auth.RoleSuperAdmin).Scan(&adminCount); err != nil {
			return err
		}
		if adminCount <= 1 {
			return ErrLastSuperAdmin
		}
	}
	var assignedToDuty bool
	if err := tx.QueryRow(ctx, `select exists(select 1 from duty_members where user_id=$1)`, id).Scan(&assignedToDuty); err != nil {
		return err
	}
	if assignedToDuty {
		return ErrUserAssignedToDuty
	}
	tag, err := tx.Exec(ctx, `delete from users where id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("user not found")
	}
	return tx.Commit(ctx)
}

func hasRoleCode(roles []string, expected string) bool {
	for _, role := range roles {
		if role == expected {
			return true
		}
	}
	return false
}

type roleSetter interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func setUserRoles(ctx context.Context, tx roleSetter, userID int64, roles []string) error {
	if _, err := tx.Exec(ctx, `delete from user_roles where user_id=$1`, userID); err != nil {
		return err
	}
	for _, role := range roles {
		tag, err := tx.Exec(ctx, `
			insert into user_roles(user_id, role_id)
			select $1, id from roles where code=$2
			on conflict do nothing
		`, userID, role)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("role %s not found", role)
		}
	}
	return nil
}

func (s *Store) userListItem(ctx context.Context, id int64) (models.UserListItem, error) {
	rows, err := s.pool.Query(ctx, `
		select
			u.id,
			u.username,
			u.display_name,
			u.must_change_password,
			coalesce(array_agg(r.code order by r.code) filter (where r.code is not null), '{}') as roles,
			u.created_at,
			u.updated_at
		from users u
		left join user_roles ur on ur.user_id = u.id
		left join roles r on r.id = ur.role_id
		where u.id = $1
		group by u.id
	`, id)
	if err != nil {
		return models.UserListItem{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		return models.UserListItem{}, errors.New("user not found")
	}
	var item models.UserListItem
	if err := rows.Scan(&item.ID, &item.Username, &item.DisplayName, &item.MustChangePassword, &item.Roles, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return models.UserListItem{}, err
	}
	return item, rows.Err()
}

func (s *Store) ListAssets(ctx context.Context) ([]models.Asset, error) {
	rows, err := s.pool.Query(ctx, `select id, coalesce(created_by, 0), asset_no, type, vendor, cpu_arch, sn, location, business, ipv4, ipv6, environment, os, hostname, network_zone, cpu, memory, disk, deployment_info, owner, status, host_machine, created_at, updated_at from assets order by updated_at desc`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []models.Asset{}
	for rows.Next() {
		var item models.Asset
		if err := rows.Scan(&item.ID, &item.CreatedBy, &item.AssetNo, &item.Type, &item.Vendor, &item.CPUArch, &item.SN, &item.Location, &item.Business, &item.IPv4, &item.IPv6, &item.Environment, &item.OS, &item.Hostname, &item.NetworkZone, &item.CPU, &item.Memory, &item.Disk, &item.DeploymentInfo, &item.Owner, &item.Status, &item.HostMachine, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) UpsertAsset(ctx context.Context, item models.Asset) (models.Asset, error) {
	if item.Type == "" || item.CPUArch == "" || item.Business == "" || item.IPv4 == "" || item.Environment == "" || item.OS == "" || item.NetworkZone == "" || item.CPU == "" || item.Memory == "" || item.Disk == "" || item.DeploymentInfo == "" || item.Owner == "" {
		return models.Asset{}, errors.New("type, cpuArch, business, ipv4, environment, os, networkZone, cpu, memory, disk, deploymentInfo and owner are required")
	}
	if item.ID > 0 {
		return s.updateAsset(ctx, item.ID, item)
	}
	if item.AssetNo == "" {
		assetNo, err := s.nextAssetNo(ctx)
		if err != nil {
			return models.Asset{}, err
		}
		item.AssetNo = assetNo
	}
	if item.Status == "" {
		item.Status = "运行中"
	}
	row := s.pool.QueryRow(ctx, `
		insert into assets(created_by, asset_no, type, vendor, cpu_arch, sn, location, business, ipv4, ipv6, environment, os, hostname, network_zone, cpu, memory, disk, deployment_info, owner, status, host_machine)
		values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21)
		on conflict (asset_no) do update set
			type=excluded.type, vendor=excluded.vendor, cpu_arch=excluded.cpu_arch, sn=excluded.sn, location=excluded.location,
			business=excluded.business, ipv4=excluded.ipv4, ipv6=excluded.ipv6, environment=excluded.environment, os=excluded.os,
			hostname=excluded.hostname, network_zone=excluded.network_zone, cpu=excluded.cpu, memory=excluded.memory, disk=excluded.disk,
			deployment_info=excluded.deployment_info, owner=excluded.owner, status=excluded.status,
			host_machine=excluded.host_machine, updated_at=now()
		returning id, created_at, updated_at
	`, nullableUserID(item.CreatedBy), item.AssetNo, item.Type, item.Vendor, item.CPUArch, item.SN, item.Location, item.Business, item.IPv4, item.IPv6, item.Environment, item.OS, item.Hostname, item.NetworkZone, item.CPU, item.Memory, item.Disk, item.DeploymentInfo, item.Owner, item.Status, item.HostMachine)
	if err := row.Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return models.Asset{}, err
	}
	return item, nil
}

func (s *Store) updateAsset(ctx context.Context, id int64, item models.Asset) (models.Asset, error) {
	if item.Status == "" {
		item.Status = "运行中"
	}
	row := s.pool.QueryRow(ctx, `
		update assets set
			asset_no=$2, type=$3, vendor=$4, cpu_arch=$5, sn=$6, location=$7, business=$8, ipv4=$9, ipv6=$10,
			environment=$11, os=$12, hostname=$13, network_zone=$14, cpu=$15, memory=$16, disk=$17,
			deployment_info=$18, owner=$19, status=$20, host_machine=$21, updated_at=now()
		where id=$1
		returning id, created_at, updated_at
	`, id, item.AssetNo, item.Type, item.Vendor, item.CPUArch, item.SN, item.Location, item.Business, item.IPv4, item.IPv6, item.Environment, item.OS, item.Hostname, item.NetworkZone, item.CPU, item.Memory, item.Disk, item.DeploymentInfo, item.Owner, item.Status, item.HostMachine)
	if err := row.Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return models.Asset{}, err
	}
	return item, nil
}

func (s *Store) DeleteAsset(ctx context.Context, id int64, actorUserID int64, actorIsSuperAdmin bool) error {
	var createdBy int64
	if err := s.pool.QueryRow(ctx, `select coalesce(created_by, 0) from assets where id=$1`, id).Scan(&createdBy); err != nil {
		return errors.New("asset not found")
	}
	if !actorIsSuperAdmin && (createdBy == 0 || createdBy != actorUserID) {
		return ErrForbiddenAssetDelete
	}
	tag, err := s.pool.Exec(ctx, `delete from assets where id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("asset not found")
	}
	return nil
}

func nullableUserID(id int64) any {
	if id == 0 {
		return nil
	}
	return id
}

func (s *Store) nextAssetNo(ctx context.Context) (string, error) {
	var seq int64
	if err := s.pool.QueryRow(ctx, `select nextval('assets_id_seq')`).Scan(&seq); err != nil {
		return "", err
	}
	return fmt.Sprintf("ASSET-%s-%04d", time.Now().Format("200601"), seq), nil
}

func (s *Store) ListMiddleware(ctx context.Context) ([]models.MiddlewareInstance, error) {
	rows, err := s.pool.Query(ctx, `select m.id, m.name, m.kind, m.version, m.environment, m.network_zone, m.endpoint, m.business, m.owner, m.status, m.asset_id, coalesce(a.asset_no, ''), m.created_at, m.updated_at from middleware_instances m left join assets a on a.id=m.asset_id order by m.updated_at desc`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []models.MiddlewareInstance{}
	for rows.Next() {
		var item models.MiddlewareInstance
		if err := rows.Scan(&item.ID, &item.Name, &item.Kind, &item.Version, &item.Environment, &item.NetworkZone, &item.Endpoint, &item.Business, &item.Owner, &item.Status, &item.AssetID, &item.AssetNo, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CreateMiddleware(ctx context.Context, item models.MiddlewareInstance) (models.MiddlewareInstance, error) {
	if item.Name == "" || item.Kind == "" || item.Environment == "" || item.NetworkZone == "" || item.Endpoint == "" || item.Business == "" || item.Owner == "" {
		return models.MiddlewareInstance{}, errors.New("name, kind, environment, networkZone, endpoint, business and owner are required")
	}
	if item.Status == "" {
		item.Status = "运行中"
	}
	row := s.pool.QueryRow(ctx, `insert into middleware_instances(name, kind, version, environment, network_zone, endpoint, business, owner, status, asset_id) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) returning id, created_at, updated_at`, item.Name, item.Kind, item.Version, item.Environment, item.NetworkZone, item.Endpoint, item.Business, item.Owner, item.Status, item.AssetID)
	if err := row.Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return models.MiddlewareInstance{}, err
	}
	return item, nil
}

func (s *Store) UpdateMiddleware(ctx context.Context, id int64, item models.MiddlewareInstance) (models.MiddlewareInstance, error) {
	if item.Name == "" || item.Kind == "" || item.Environment == "" || item.NetworkZone == "" || item.Endpoint == "" || item.Business == "" || item.Owner == "" {
		return models.MiddlewareInstance{}, errors.New("name, kind, environment, networkZone, endpoint, business and owner are required")
	}
	if item.Status == "" {
		item.Status = "运行中"
	}
	row := s.pool.QueryRow(ctx, `
		update middleware_instances set
			name=$2, kind=$3, version=$4, environment=$5, network_zone=$6, endpoint=$7,
			business=$8, owner=$9, status=$10, asset_id=$11, updated_at=now()
		where id=$1
		returning id, created_at, updated_at
	`, id, item.Name, item.Kind, item.Version, item.Environment, item.NetworkZone, item.Endpoint, item.Business, item.Owner, item.Status, item.AssetID)
	if err := row.Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return models.MiddlewareInstance{}, err
	}
	return item, nil
}

func (s *Store) DeleteMiddleware(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `delete from middleware_instances where id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("middleware instance not found")
	}
	return nil
}

func (s *Store) ListOnCalls(ctx context.Context) ([]models.OnCallSchedule, error) {
	rows, err := s.pool.Query(ctx, `select id, rule_type, coalesce(date_value::text, date_value_legacy), week_value, primary_user, primary_user_id, backup_user, backup_user_id, swap_from, swap_to, notes, created_at, updated_at from oncall_schedules order by date_value desc nulls last, updated_at desc`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []models.OnCallSchedule{}
	for rows.Next() {
		var item models.OnCallSchedule
		if err := rows.Scan(&item.ID, &item.RuleType, &item.Date, &item.Week, &item.Primary, &item.PrimaryUserID, &item.Backup, &item.BackupUserID, &item.SwapFrom, &item.SwapTo, &item.Notes, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CreateOnCall(ctx context.Context, item models.OnCallSchedule) (models.OnCallSchedule, error) {
	var err error
	if item.Primary, err = resolveUserDisplayName(ctx, s.pool, item.PrimaryUserID, item.Primary); err != nil {
		return models.OnCallSchedule{}, err
	}
	if item.Backup, err = resolveUserDisplayName(ctx, s.pool, item.BackupUserID, item.Backup); err != nil {
		return models.OnCallSchedule{}, err
	}
	row := s.pool.QueryRow(ctx, `insert into oncall_schedules(rule_type, date_value, date_value_legacy, week_value, primary_user, primary_user_id, backup_user, backup_user_id, swap_from, swap_to, notes) values ($1,nullif($2,'')::date,$2,$3,$4,$5,$6,$7,$8,$9,$10) returning id, created_at, updated_at`, item.RuleType, item.Date, item.Week, item.Primary, item.PrimaryUserID, item.Backup, item.BackupUserID, item.SwapFrom, item.SwapTo, item.Notes)
	if err := row.Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return models.OnCallSchedule{}, err
	}
	return item, nil
}

func (s *Store) UpdateOnCall(ctx context.Context, id int64, item models.OnCallSchedule) (models.OnCallSchedule, error) {
	var err error
	if item.Primary, err = resolveUserDisplayName(ctx, s.pool, item.PrimaryUserID, item.Primary); err != nil {
		return models.OnCallSchedule{}, err
	}
	if item.Backup, err = resolveUserDisplayName(ctx, s.pool, item.BackupUserID, item.Backup); err != nil {
		return models.OnCallSchedule{}, err
	}
	row := s.pool.QueryRow(ctx, `
		update oncall_schedules set
			rule_type=$2, date_value=nullif($3,'')::date, date_value_legacy=$3, week_value=$4,
			primary_user=$5, primary_user_id=$6, backup_user=$7, backup_user_id=$8,
			swap_from=$9, swap_to=$10, notes=$11, updated_at=now()
		where id=$1
		returning id, created_at, updated_at
	`, id, item.RuleType, item.Date, item.Week, item.Primary, item.PrimaryUserID, item.Backup, item.BackupUserID, item.SwapFrom, item.SwapTo, item.Notes)
	if err := row.Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return models.OnCallSchedule{}, err
	}
	return item, nil
}

func (s *Store) DeleteOnCall(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `delete from oncall_schedules where id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("oncall schedule not found")
	}
	return nil
}

type userDisplayNameQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func resolveUserDisplayName(ctx context.Context, querier userDisplayNameQuerier, userID *int64, fallback string) (string, error) {
	if userID == nil {
		return fallback, nil
	}
	var displayName string
	if err := querier.QueryRow(ctx, `select display_name from users where id=$1`, *userID).Scan(&displayName); err != nil {
		return "", err
	}
	return displayName, nil
}
