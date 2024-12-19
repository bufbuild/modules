**This is a third-party repository managed by Buf.**

> This module has been **deprecated**. Use instead the `protocolbuffers/wellknowntypes` module v29.2
> or higher.

This module contains Protobuf sources that are defined in the
https://github.com/protocolbuffers/protobuf-go repo. This is currently
limited to custom features used with [Editions](https://protobuf.dev/editions/overview/)
to provide perfect backwards compatibility in generated Go code when
migrating "proto2" sources to Editions.

Updates to the [source repository](https://github.com/protocolbuffers/protobuf-go)
are automatically synced on a periodic basis, and each BSR commit is
tagged with corresponding Git semver releases.

To depend on a specific version, you can use it as your reference in your dependencies:

```
deps:
  - buf.build/protocolbuffers/gofeatures:<SEMVER_RELEASE_VERSION>
```

For more information, see the [documentation](https://buf.build/docs/bsr/overview).
