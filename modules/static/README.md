# modules static files

Files in this directory are mapped to the synced managed modules. Each new fetch action for these
modules expects to copy `buf.yaml` and `buf.md` files into their corresponding sync module
directories, so each BSR instance can keep those modules in sync.

Once merged in `main`, updates to `buf.yaml` or `buf.md` are synced via `fetch` action to the new
module references.
