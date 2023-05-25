---
name: Managed Module request for Buf Schema Registry
about: Request for Managed Module
title: Managed Module request for Buf Schema Registry
labels: Feature
assignees: ''

---

Not ready to open an issue, but want to chat about your module? Come find us on our Public Slack channel:

https://buf.build/links/slack

## Mandatory

**Where is the source code for the Managed Module?**

Example, the source code for the `googleapis/googleapis` module is found here:

https://github.com/googleapis/googleapis/tree/master/google

**Do the proto files declare a package?**

For example, we will accept modules that [declare a package](https://github.com/googleapis/googleapis/blob/master/google/api/annotations.proto#L15-L27), 
but will reject those that [do not](https://github.com/GoogleChrome/lighthouse/blob/main/proto/lighthouse-result.proto#L1-L10).

## Optional

**Does this module have other module dependencies/imports?**

For example, [cndf/xds](https://github.com/cncf/xds) depends on `googleapis/googleapis` and `envoyproxy/protoc-gen-validate` 

https://github.com/bufbuild/modules/blob/main/modules/static/cncf/xds/buf.yaml

**Based on the repository's release process, do you prefer syncing by releases (semver) or by git commit SHA?**

Please suggest a reasonable sync method for your proposed target.

**Additionally, is there a sensible initial reference to sync from?**

Along with a sync method, please propose a reference to commence syncing from. Consider commonly used versions and whether there are major breaking changes across versions that segment the user base.

**Do you think this module is widely used by the community, and is not already provided on the BSR by the owners?**

Unsure? Let's chat!
