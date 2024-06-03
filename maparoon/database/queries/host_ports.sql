-- name: ListHostPorts :many
select * from host_ports;

-- name: ListHostPortsByHostAddress :many
select * from host_ports
where address = ?;

-- name: CreateHostPort :one
insert into host_ports (
    address, port, protocol, comments
) values (
    ?, ?, ?, ?
)
returning *;

-- name: GetHostPort :one
select * from host_ports
where address = ?
    and port = ?
    and protocol = ?;

-- name: UpdateHostPort :one
update host_ports
set
    comments = ?
where
    address = ?
    and port = ?
    and protocol = ?
returning *;

-- name: DeleteHostPort :exec
delete from host_ports
where
    address = ?
    and port = ?
    and protocol = ?;
