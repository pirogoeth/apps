# docker-hermes

## goals

docker-hermes is a sort of service registration system. the goal is to create an agent along with a server.

the agent should:
- run on each docker node/instance and report labels to a central database/system.
- labels should be kept fresh in the database as long as the container they are associated with is still running
- information about the host and ports associated with the labelled container should also be reported to the central database/system.
- expose a prometheus endpoint detailing resource usage (this should be relatively built-in to the apps framework)
- expose traces for the gathering and emitting process
- support specifying a label prefix/"namespacing"

the server should:
- run in a central location, attached to the database. 
- serve a Prometheus HTTP SD endpoint containing any services that have a proper set of prometheus labels.
- have an API that allows for lookup of containers/services based on label contents
- have an API that allows for lookup of labels by container information
