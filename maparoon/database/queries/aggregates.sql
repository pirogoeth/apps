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
