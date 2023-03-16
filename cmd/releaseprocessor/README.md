# Release Tag Sync 

Release tag sync will pull all _release_ tags and return them as a list to stdout. Release tags
differ from _tags_ in that one GitHub release can only have one associated tag.

## Special case for WKT

See the `protocolbuffers/wellknowntypes` [document][wkt-doc] to understand this module's special case.

[wkt-doc]: https://github.com/bufbuild/modules/blob/main/modules/static/protocolbuffers/wellknowntypes/buf.md
