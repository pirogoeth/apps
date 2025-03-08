# `worker/emailscanner.go`

Instead of always fetching, we can try to defer this!
The scanner should just gather all of the messages according
to the inboxes/mailboxes configuration, and pass that off, but
support a future pipeline step being able to "backfill" all of the message data.

So, an initial fetch of the envelope, UID, internal date, etc should _always_ happen, as we need that 
information to be able to archive the messages appropriately. Body structure fetching should only
happen if the message is not archived to the storage or present in the search index.

# `worker` package, generally

Maybe this is a huge premature optimization, but it seems like the workers could be refactored into a pipeline of functionality,
where a pair of channels are passed to each "pipeline stage" for message data to flow through. This could make it simpler
to introduce new stages or generally experiment with the pipeline in a more straightforward way. 

Attempting to lay out this idea in `../pkg/pipeline`...

But also not entirely sure this is the right idea since it doesn't really allow for modification of the data between stages
_outside of the instantiated type_. So the pipeline would really be oriented on a stage-based enrichment/modification type of action
