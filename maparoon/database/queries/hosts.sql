-- name: ListHosts :many
select * from hosts;

-- name: GetHost :one
select * from hosts
where address = ? limit 1;

-- name: GetHostWithNetwork :one
select * from hosts
where address = ? and network_id = ? limit 1;

-- name: CreateHost :one
insert into hosts (
    network_id, address, comments
) values (
    ?, ?, ?
)
returning *;

-- name: UpdateHost :one
update hosts
set
    comments = ?
where address = ?
returning *;

-- name: DeleteHost :exec
delete from hosts
where address = ?;
