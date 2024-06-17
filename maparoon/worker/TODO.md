Inside of the worker, it seems like we're relatively quickly ending up with a few different concurrent mechanisms that are fetching information in different ways, between the SNMP gatherer, `nmap` result gatherer, naabu gatherer, and it feels like this could end up all strapped together in a pretty nasty way fairly quickly. 

How can this be structure in a better way to make it easier to maintain while also being able to collate this information together fairly simply? 

The notable exception is the naabu gatherer, since it effectively "unrolls" the entire network into a series of hosts to scan, so it always needs to be the base process. But, we've got the `nmap` results following the naabu scans inside of a FIFO, and the SNMP gatherer triggering off of the naabu results, but returning separately into the top level worker. 

It seems like both the `nmap` results and the SNMP gather results should ideally be returned into the top-level worker and then sent into the database from a single location, instead of the `nmap` results POSTing directly to the database and the SNMP gather ostensibly doing the same. 

Ship the two(/`n`) sets of results back to the top-level process, collate, and "batch" into the database? Should there be an analogue to `/v1/hostscans` for the SNMP data? Or is that effectively considered part of the scan and sent to the hostscans index?
