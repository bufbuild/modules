# bufbuild/confluent

[![Build](https://github.com/bufbuild/confluent-proto/actions/workflows/ci.yaml/badge.svg?branch=main)](https://github.com/bufbuild/confluent-proto/actions/workflows/ci.yaml)
![GitHub release (with filter)](https://img.shields.io/github/v/release/bufbuild/confluent-proto)
![GitHub](https://img.shields.io/github/license/bufbuild/confluent-proto)

This module contains Protobuf extensions for integration with the Confluent Schema Registry.

## Usage

To integrate with the Confluent Schema Registry, extend the
`buf.confluent.v1.subject` extension on a protobuf Message.
For example, see the `demo/analytics/event.proto` message below, which defines
a mapping between the message `demo.analytics.MyEvent` and the Confluent Schema
Registry subject `my-event-value`.

```proto
syntax = "proto3";

package demo.analytics;

import "buf/confluent/v1/extensions.proto";

message MyEvent {
  string a_string = 1;
  bytes some_data = 2;

  // Define one or more options to associate protobuf messages with Kafka topics.
  option (buf.confluent.v1.subject) = {
    // The BSR's instance name for its Confluent Schema Registry integration.
    // These are provisioned in the Admin Settings of the BSR.
    instance_name: "prod",
    // The name of the subject to associate the message.
    // The '-value' suffix is used for messages used as Kafka topic values.
    // The '-key' suffix is used for Kafka topic keys.
    name: "my-event-value",
  };
}
```

When a Buf module is pushed to the BSR with the Confluent integration enabled,
this will automatically create a subject named `my-event-value` associated with
the `demo.analytics.MyEvent` message on the instance name `prod`.
