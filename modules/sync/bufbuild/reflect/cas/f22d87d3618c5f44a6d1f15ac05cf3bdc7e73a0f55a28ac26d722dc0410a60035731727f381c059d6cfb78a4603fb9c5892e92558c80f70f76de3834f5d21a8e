Buf Reflection API
==================
[![CI](https://github.com/bufbuild/reflect-proto/workflows/buf/badge.svg)](https://github.com/bufbuild/reflect-proto/actions?workflow=buf)

This repo contains Protobuf sources for APIs related to runtime reflection backed by a schema
registry.

For now, this consists solely of a single RPC endpoint which can be used to download Protobuf
schemas from a server. The BSR (Buf Schema Registry) _implements_ this service, so you can use
it to download schemas that you have pushed to the BSR.

This API differs from [gRPC Server Reflection](https://github.com/grpc/grpc/blob/master/doc/server-reflection.md)
in that it is intended to be provided by a separate registry of schemas. Unlike gRPC Server
Reflection, this API supports the notion of a registry having multiple schemas, identified by
name, and even multiple versions of each schema. The gRPC Server Reflection is only useful for
certain RPC-related use cases. But a separate registry is needed for other use cases, such as
interpreting Protobuf data in a persistent store (or message queue system). It is also useful
for introspecting on RPC details when gRPC Server Reflection is unavailable.

_This Protobuf module is available on the Buf Schema Registry at [buf.build/bufbuild/reflect](https://buf.build/bufbuild/reflect)._

### Background

The Protobuf binary format is compact and efficient, and it has clever features that allow for a
wide variety of schema changes to be both backward- and forward-compatible.

However, it is not possible to make meaningful sense of the data without a schema. Not only is it
not human-friendly, since all fields are identified by an integer instead of a semantic name, but
it also uses a very simple wire format which re-uses various value encoding strategies for
different value types. This means it is not even possible to usefully interpret encoded values
without a schema — for example, one cannot know (with certainty) if a value is a text string, a
binary blob, or a nested message structure.

But there exists a category of systems and use cases where it is necessary or useful to decode the
data at runtime, by a process or user agent that does not have prior (compile-time) knowledge of
the schemas:

1. **RPC debugging**. It is useful for a human to be able to meaningfully interpret/examine/modify
   RPC requests and responses (with tools like tcpdump, Wireshark, or Charles proxy). But without
   the schema, these payloads are inscrutable byte sequences.
2. **Persistent store debugging** (includes message queues): This is similar to the above use case,
   but the human is looking at data blobs in a database or durable queue.
3. **Data pipeline schemas and transformations**: This is less for human interaction and more for
   data validation and transformation. A producer may be pushing binary blobs of encoded protos
   into a queue or publish/subscribe system. The system may want to verify that the blob is
   actually valid for the expected type of data, which requires a schema. The consumer may need
   the data in an alternate format; the only way to transform the binary data into an alternate
   format is to have the schema. Further, the only way to avoid dropping data is to have a version
   of the schema that is no older than the version used by the publisher. (Otherwise, newly added
   fields may not be recognized and then silently dropped during a format transformation.)

All of these cases call for a mechanism by which the schema for a particular message type can be
downloaded on demand, for interpreting the binary data.

Enter the `FileDescriptorSetService` service...

### buf.reflect.v1beta1.FileDescriptorSetService

This service provides a single method that allows callers to query for a schema. A schema
is identified by a "module name" and an optional version.

In addition to querying for the schema by module name and version, this API also allows the
caller to signal what part of the schema in which they are interested, such as a specific
message type or a specific service or method. This is used to filter the schema, allowing
the client to ignore parts of a module that it does not need. Depending on how modules
are scoped and exactly what parts of the schema the client needs, this often will greatly
reduce the amount of data that a client needs to download.

### Other Resources

There is currently a client library for the `FileDescriptorSetService` for Go that
enables users to convert messages between formats (such as converting binary data to
JSON) and even potentially convert to different message types or modify message data
(such as redacting sensitive data). This could be used in a data pipeline subscriber
to manipulate/transform Protobuf message data before pushing it into a data warehouse.

That library is called `prototransform` and it is available on
[GitHub](https://github.com/bufbuild/prototransform).

## Status: Beta

This API is currently designated as a Beta version. While it is unlikely to change in a
significant way before a "v1", we're looking for users to put some miles on the API and
provide feedback (via a [GitHub issue](https://github.com/bufbuild/reflect-proto/issues)).

## Legal

Offered under the [Apache 2 license][license].

[license]: https://github.com/bufbuild/reflect-proto/blob/main/LICENSE
