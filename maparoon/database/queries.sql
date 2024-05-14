-- name: GetNetworkById :one
select * from networks
where id=? limit 1;

-- name: GetNetworkByAddress :one
select * from networks
where address=? limit 1;

-- name: DeleteNetwork :exec
delete from networks where id=?;

-- name: CreateNetwork :one
insert into networks (
    name, address, cidr, comments, attributes
) values (
    ?, ?, ?, ?, ?
)
returning *;

-- name: UpdateNetwork :one
update networks
set
    name = ?,
    comments = ?,
    attributes = ?
where id = ?
returning *;

-- name: ListNetworks :many
select * from networks;

-- name: ListHosts :many
select * from hosts;

-- name: ListHostPorts :many
select * from host_ports;

-- name: GetHostWithPortsById :many
select
    h.id as host_id,
    net.name as network_name,
    net.address as network_address,
    net.cidr as network_cidr_size,
    h.comments as host_comments,
    h.attributes as host_attributes,
    h.address as host_address,
    hp.port as port_number,
    hp.protocol as port_protocol,
    hp.comments as port_comments,
    hp.attributes as port_attributes
from hosts h
left join host_ports hp
on hp.address=h.address
left join networks net
on net.id=h.network_id
where h.id=?;