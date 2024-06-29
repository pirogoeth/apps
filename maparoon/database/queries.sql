-- name: GetHostWithPortsByAddress :many
select
    net.name as network_name,
    net.address as network_address,
    net.cidr as network_cidr_size,
    h.comments as host_comments,
    h.address as host_address,
    hp.port as port_number,
    hp.protocol as port_protocol,
    hp.comments as port_comments
from hosts h
left join host_ports hp
on hp.address = h.address
left join networks net
on net.id = h.network_id
where h.address = ?;
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

