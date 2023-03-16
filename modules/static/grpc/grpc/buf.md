**This is a third-party repository managed by Buf.**

Updates to the [source repository](https://github.com/grpc/grpc-proto) are automatically synced on a
periodic basis, and each BSR commit is tagged with corresponding Git commits. The dependencies of
this repository are updated each time there is a new version of the source repository.

To depend on a specific Git commit, you can use it as your reference in your dependencies:

```
deps:
  - buf.build/grpc/grpc:<GIT_COMMIT_TAG>
```

For more information, see the [documentation](https://docs.buf.build/bsr/overview).
