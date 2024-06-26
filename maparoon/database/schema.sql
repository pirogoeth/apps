pragma foreign_keys = on;

create table if not exists networks (
    id integer primary key,
    name text not null,
    address text not null,
    cidr integer not null,
    comments text not null,

    unique (address, cidr)
);
create index if not exists idx_networks_address on networks(address);
create index if not exists idx_networks_address_cidr on networks(address, cidr);

create table if not exists network_attributes (
    network_id integer not null,
    key text not null,
    value text not null,

    primary key (network_id, key),
    foreign key (network_id) references networks(id) on delete cascade on update cascade
);

create table if not exists hosts (
    address text primary key,
    network_id integer not null,
    comments text not null,

    foreign key (network_id) references networks(id) on delete cascade on update cascade
);
create index if not exists idx_hosts_address on hosts(address);

create table if not exists host_attributes (
    address text not null,
    key text not null,
    value text not null,

    primary key (address, key),
    foreign key (address) references hosts(address) on delete cascade on update cascade
);

create table if not exists host_ports (
    address text not null,
    port integer not null,
    protocol text not null,
    comments text not null,

    primary key (address, port, protocol),
    foreign key (address) references hosts(address) on delete cascade on update cascade
);
create index if not exists idx_host_ports_address on host_ports(address);
create index if not exists idx_host_ports_address_port on host_ports(address, port);
create index if not exists idx_host_ports_address_port_proto on host_ports(address, port, protocol);

create table if not exists host_port_attributes (
    address text not null,
    port integer not null,
    protocol text not null,
    key text not null,
    value text not null,

    primary key (address, port, protocol, key),
    foreign key (address, port, protocol) references host_ports(address, port, protocol) on delete cascade on update cascade
);

create table if not exists network_scans (
    id integer primary key,
    network_id integer not null,
    started_at integer not null default -1,
    finished_at integer not null default -1,
    hosts_found integer not null default 0,
    ports_found integer not null default 0,

    foreign key (network_id) references networks(id) on delete cascade on update cascade
);
create index if not exists idx_network_scans_network_id on network_scans(network_id);

create trigger if not exists trg_network_scans_insert
after insert on network_scans
for each row
begin
    update network_scans
    set started_at = strftime('%s', 'now')
    where id = new.id;
end;