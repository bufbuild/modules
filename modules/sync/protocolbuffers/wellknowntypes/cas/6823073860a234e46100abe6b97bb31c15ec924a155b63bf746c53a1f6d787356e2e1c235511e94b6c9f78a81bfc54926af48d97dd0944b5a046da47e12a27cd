**This is a third-party repository managed by Buf.**

This module contains the Well-Known Types. The Well-Known Types form Protobuf's mini-standard
library. They contain common types that are used throughout the Protobuf ecosystem. Specifically,
this module contains the following files:

- `google/protobuf/any.proto`
- `google/protobuf/api.proto`
- `google/protobuf/compiler/plugin.proto`
- `google/protobuf/descriptor.proto`
- `google/protobuf/duration.proto`
- `google/protobuf/empty.proto`
- `google/protobuf/field_mask.proto`
- `google/protobuf/source_context.proto`
- `google/protobuf/struct.proto`
- `google/protobuf/timestamp.proto`
- `google/protobuf/type.proto`
- `google/protobuf/wrappers.proto`

These files are part of every `buf` and `protoc` installation. Even without depending on this
module, you can use these files of the box. For example, the following file should compile right now
with both `buf` and `protoc`;

```protobuf
syntax = "proto3";
package buf.user.v1;
import "google/protobuf/timestamp.proto";
message User {
  google.protobuf.Timestamp create_time = 1;
}
```

However, by relying on the Well-Known Types built into your compiler of choice, the version of the
Well-Known Types is tied to the version of your compiler. If you'd like to instead lock the
Well-Known Types to a specific version, you can do so by depending on this module. For example:

```yaml
# buf.yaml
version: v1
deps:
  - buf.build/protocolbuffers/wellknowntypes:v21.12
```

We update the tags on this module for every release of
[protobuf](https://github.com/protocolbuffers/protobuf/releases).

## About the available BSR references

The `protocolbuffers/protobuf` repository's release & tagging history does not follow a typical
[SemVer](https://semver.org/) convention. This is briefly mentioned in the [documentation for
version support](https://protobuf.dev/support/version-support/).

Ultimately, the releases of `protocolbuffers/protobuf` are managed in the following pattern, starting from `v3.0.0`:

Semantic versioning with `v{major}.{minor}.{patch}` was adhered to:

```
v3.0.0, v3.0.2, v3.1.0, v3.2.0, v3.3.0, ...  
→
..., v3.19.6, v3.20.0, v3.20.1, v3.20.2, v3.20.3
```

> Except for `v3.4.1`, which [does not have a `protoc` attached to
> it](https://github.com/protocolbuffers/protobuf/releases/tag/v3.4.1).

From this point onward, the minor was incremented to `21` and the major was removed to follow the
following pattern `v{minor}.{patch}`:

```
v21.0, v21.1, v21.2, ...
→
..., v21.11, v21.12, v22.0
```

The version support document linked above explains this pattern as:

> In the new scheme, each language has its own major version that can be incremented independently
> of other languages. The minor and patch versions, however, remain coupled. This allows us to
> introduce breaking changes into some languages without requiring a bump of the major version in
> languages that do not experience a breaking change. For example, a single release might include
> protoc version `24.0`, Java runtime version `4.24.0` and C# runtime version `3.24.0`.

What this means is that the git repository does have many _tags_: for example it has tags that
follow the typical SemVer convention that continue from the point at which the repository moved to
the new scheme, e.g. `v3.21.0`. However, when syncing for this managed module, we follow the new
scheme/pattern by pulling in _only release tags_ from the repository, not all tags, as their release
flow adheres to the new scheme.
