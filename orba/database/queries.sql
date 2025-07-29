-- name: CreateUser :one
insert into users (name) values (?) returning *;

-- name: GetUserById :one
select * from users where id=?;

-- name: CreateSource :one
insert into sources (
  id, user_id, description, protected
) values (?, ?, ?, ?)
returning *;

-- name: ListSourcesForUser :many
select * from sources where user_id=?;

-- name: ProtectSource :exec
update sources
set protected=true
where id=?;

-- name: UnprotectSource :exec
update sources
set protected=false
where id=?;

-- name: CreateMemory :one
insert into memories (
  source_id, user_id, memory
) values (?, ?, ?)
returning *;

-- name: ListMemories :many
select * from memories
where user_id=?
order by created_at desc;

-- name: GetRelevantMemoriesForToday :many
select * from memories
where
  created_date=date('now')
  and user_id=?
order by created_at desc;

-- name: DeleteMemoryById :exec
delete from memories where id=?;
