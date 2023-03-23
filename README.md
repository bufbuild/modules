# modules

## Description

This repository contains third-party modules synced and published to the [Buf Schema Registry][bsr].

If you'd like a common third-party module to be managed by Buf, open an issue using the [Managed Module Request for Buf 
Schema Registry][issue-template] issue template and our team will follow up.

## Managed modules

We currently sync automatically the following modules:

| Module | Depends on | Source Community Repository |
|---|---|---|
| cncf/xds | - envoyproxy/protoc-gen-validate<br>- googleapis/googleapis | https://github.com/cncf/xds |
| envoyproxy/envoy | - cncf/xds<br>- googleapis/googleapis<br>- opencensus/opencensus<br>- opentelemetry/opentelemetry<br>- prometheus/client-model | https://github.com/envoyproxy/envoy |
| envoyproxy/protoc-gen-validate |  | https://github.com/envoyproxy/protoc-gen-validate |
| gogo/protobuf |  | https://github.com/gogo/protobuf |
| googleapis/googleapis |  | https://github.com/googleapis/googleapis |
| grpc/grpc | - envoyproxy/envoy<br>- googleapis/googleapis | https://github.com/grpc/grpc-proto |
| opencensus/opencensus |  | https://github.com/census-instrumentation/opencensus-proto |
| opentelemetry/opentelemetry |  | https://github.com/open-telemetry/opentelemetry-proto |
| protocolbuffers/wellknowntypes |  | https://github.com/protocolbuffers/protobuf |

## Community

For help and discussion regarding Protobuf managed modules, join us on
[Slack][slack].

For feature requests, bugs, or technical questions, email us at [dev@buf.build](dev@buf.build).

## Legal

Offered under the [Apache 2 license][license].

[bsr]: https://buf.build/explore 
[issue-template]: https://github.com/bufbuild/modules/issues/new?assignees=&labels=Feature&template=managed-module-request-for-buf-schema-registry.md&title=Managed+Module+request+for+Buf+Schema+Registry
[license]: https://github.com/bufbuild/modules/blob/main/LICENSE
[slack]: https://buf.build/links/slack
