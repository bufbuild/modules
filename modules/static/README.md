# Modules static files

Files in this directory are mapped to the synced managed modules. If the modules originally have
buf-specific files like `buf.yaml, buf.lock, buf.md...`, then those files can be synced via the
`rsync.incl` rules. If not, then a static copy can be put in the module directory, for it to be
copied on each git reference.

## Note on syncing `buf.yaml` and `buf.lock`

Regardless if you're syncing or copying static files for configuration and dependencies, take into
account that those files will be modified and/or overriten on each BSR instance. Files copied here
might be ok to work with the public BSR instance `buf.build`, but for private instances we use a
diferent hostname, and recreate/pull again the dependencies, so the `buf.lock` matches with the
right dependencies FQNs and commit names.

So the `buf.lock` is regenerated when syncing, why do we have it here? Having a copy of the source
`buf.lock` helps make evident when the module content changes (it results in a different digest), so
even if that change was just a `buf mod update` pointing to a new version of dependencies, BSR
instances know it's not exactly the same module.
