# PostgreSQL 16 → 18.6 升级与回退

## 当前部署

- 目标镜像：`postgres:18.6-alpine`。
- PostgreSQL 18 数据卷：`deploy_postgres18-data`，作为外部卷挂载 `/var/lib/postgresql`。
- 实际数据目录：`/var/lib/postgresql/18/docker`。
- 保留旧 PostgreSQL 16 卷：`deploy_postgres-data`，旧挂载 `/var/lib/postgresql/data`。
- 保留本次运行的旧镜像：`opscore-postgres16-rollback:20261014`，来自原运行镜像，而不是再次拉取可变的 `16-alpine` 标签。
- 新部署须先创建目标外部卷；不同卷名使用 `POSTGRES18_VOLUME`、`POSTGRES16_VOLUME`。不要仅改变 Compose 项目名而假设旧数据会自动找到。

本次采用逻辑备份/恢复，不直接让 PostgreSQL 18 读取 PostgreSQL 16 的物理目录。

## 本次备份与验证

受限备份在仓库 `backups/postgres-16-to-18-20261014-155034/`，目录已忽略 Git。数据库 dump 和全局角色文件包含敏感业务数据及哈希，不能提交、公开或作为普通交付文件分发。文件创建使用 `umask 077`。

- `rehearsal.dump`、`rehearsal-globals.sql`：隔离恢复演练备份。
- `final.dump`、`final-globals.sql`：前后端停写后的最终备份。
- `final-fingerprint.txt`、`final-restored-fingerprint.txt`、`compose18-fingerprint.txt`：恢复前后校验记录。
- `scripts/postgres-upgrade-fingerprint.sql`：比较 public 表行数及按行哈希排序的内容指纹、序列、索引、可比约束和列类型/非空属性，不输出业务明文。

PostgreSQL 18 将 NOT NULL 作为独立系统约束记录；为跨版本比较，指纹查询排除 `pg_constraint.contype='n'`，同时通过 `information_schema.columns.is_nullable` 比较非空属性。不能把系统目录新增约束记录误认为丢失数据。

恢复新集群使用独立初始化角色以避免与原应用角色冲突。`upgrade_restore_bootstrap` 是新集群初始化角色，持有系统对象，不能安全删除或取消 bootstrap superuser 属性；已设 `NOLOGIN` 并清除密码，应用连接继续使用恢复后的原应用角色。没有改动原 JWT、凭据加密密钥或应用密码。

备份目前保留在本机，尚未创建异机/对象存储副本。旧卷和本机 dump 不等于灾难恢复方案，应按运维策略复制到受保护的独立故障域。

## 后续大版本升级的安全顺序

1. 记录实际服务器版本、镜像身份、卷、数据库清单、扩展、角色、编码/排序规则、表空间和复制状态；验证目标镜像可获取且支持当前架构。
2. 先做逻辑备份并在隔离的新容器/卷中恢复演练，恢复遇错即停止；核对所有应用数据与结构。
3. 在批准的维护窗口停止全部写入者，保留数据库运行以完成最终 `pg_dump -Fc` 和所需全局角色备份。凭据文件受限保存，不能把内容打印到日志。
4. 使用全新数据卷初始化目标数据库；后端保持停止，先恢复全局角色再恢复应用库，避免后端自动迁移与恢复冲突。
5. 对比表内容指纹、序列、约束、索引、列属性、扩展、`schema_migrations`；执行 `ANALYZE`，确认恢复命令无错误。
6. 干净停止新临时恢复容器和旧数据库，再把 Compose 指向新卷。保留旧卷和旧镜像，禁止 `down -v` 或卷 prune。
7. 核对实际 `server_version`、`data_directory`、卷挂载；启动后端与前端，验证健康、登录、RBAC、搜索、值班、事件及凭据解密能力。
8. 开放写入后记录验收点，并按保留期管理旧卷和备份。不要自动删除旧数据。

## 回退

**开放新库写入前**，原卷未被修改时，可以回到停写点。先停止应用写入与当前数据库，再使用同一 Compose 项目和回退覆盖：

```bash
cd deploy
docker compose stop frontend backend postgres
docker compose -f docker-compose.yml -f docker-compose.postgres16-rollback.yml up -d postgres --wait
docker compose -f docker-compose.yml -f docker-compose.postgres16-rollback.yml up -d backend frontend --wait
```

生产部署还应保留 `docker-compose.production.yml` 覆盖与原安全设置，不暴露数据库或后端端口。`!override` 需要支持该标记的 Compose 版本；本次使用 Compose 5.5.0。

**当前应用已经恢复写入**。旧 16 卷只包含切换前数据；直接执行上面的回退会丢失切换后的业务和审计变化。须先再次停写并备份新库，评估差异恢复/补偿方案；不能假设 PostgreSQL 18 的 dump 可以无条件恢复到 16。

回退配置仅准备并静态核查；没有在切换后实际执行回退，以免覆盖新写入。

## 验收注意事项

`pg_isready` 与 API Ping 只证明连接可用，不证明数据完整。还须核对业务记录、迁移版本、序列、扩展和权限。

升级当时未运行旧 `scripts/smoke-api.sh`，因为旧版会永久改写统一凭据校验密码。后续安全修复版已加入容器内0600快照及退出事务还原，恢复原值和原更新时间，原先无配置则删除测试设置；成功/失败退出须单独验证，SIGKILL或容器丢失不能承诺自动还原。Smoke仍会生成审计，完整Playwright回归也会创建清理临时业务记录，均不是完全无副作用的只读检查。

## 验收结果

结果完成后记录在 `WORKLOG.md`；可分享的升级摘要放在 `deliverables/postgres-18-upgrade-report.md`。其中不包含备份内容、角色密码哈希或明文凭据。
