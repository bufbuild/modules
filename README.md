# modules

## Description

This repository contains third-party modules synced and published to the [Buf Schema Registry][bsr].

If you'd like a common third-party module to be managed by Buf, open an issue using the [Managed Module Request for Buf 
Schema Registry][issue-template] issue template and our team will follow up.

## Managed modules

We currently sync automatically the following modules:

| Module | Source Community Repository | Depends on |
|---|---|---|
| cncf/xds | https://github.com/cncf/xds | - envoyproxy/protoc-gen-validate<br>- googleapis/googleapis |
| envoyproxy/envoy | https://github.com/envoyproxy/envoy | - cncf/xds<br>- googleapis/googleapis<br>- opencensus/opencensus<br>- opentelemetry/opentelemetry<br>- prometheus/client-model |
| envoyproxy/protoc-gen-validate | https://github.com/envoyproxy/protoc-gen-validate |  |
| gogo/protobuf | https://github.com/gogo/protobuf |  |
| googleapis/googleapis | https://github.com/googleapis/googleapis |  |
| grpc/grpc | https://github.com/grpc/grpc-proto | - envoyproxy/envoy<br>- googleapis/googleapis |
| opencensus/opencensus | https://github.com/census-instrumentation/opencensus-proto |  |
| opentelemetry/opentelemetry | https://github.com/open-telemetry/opentelemetry-proto |  |
| prometheus/client-model | https://github.com/prometheus/client_model |  |
| protocolbuffers/wellknowntypes | https://github.com/protocolbuffers/protobuf |  |

### How we handle dependencies

Dependencies are an essential part of these community modules as they help developers reuse well known Protobuf types, reduce errors and speed up the development process. However, we at Buf do not have control over the source of these modules, and managing and pinning dependencies to their exact commit can be difficult, especially when multiple modules sources and build systems are involved.

Our current approach for BSR instances is as follows; first we sort the order in which we sync these modules such that standalone modules are synced first, then after they succeed sync the ones that depends on them, which would use the latest pushed dependency commit. As long as the dependencies don’t have any breaking change in the source code, this should be sufficient and stable for upstream modules.

We know this is not ideal, and that’s why we’ve started to get more involved in those source repositories, trying to steer them into using Buf tools to manage their protos (e.g. [envoyproxy/envoy](https://github.com/envoyproxy/envoy/pull/17515)), so we can better interpret their `buf.yaml` file(s), and can do a better job pinning their dependencies.

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
