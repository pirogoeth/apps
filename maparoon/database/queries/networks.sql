-- name: GetNetworkById :one
select * from networks
where id = ? limit 1;

-- name: GetNetworkByAddress :one
select * from networks
where address = ? limit 1;

-- name: DeleteNetwork :exec
delete from networks where id = ?;

-- name: CreateNetwork :one
insert into networks (
    name, address, cidr, comments
) values (
    ?, ?, ?, ?
)
returning *;

-- name: UpdateNetwork :one
update networks
set
    name = ?,
    comments = ?
where id = ?
returning *;

-- name: ListNetworks :many
select * from networks;
