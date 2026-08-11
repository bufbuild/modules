#!/usr/bin/env bash

set -eo pipefail

if [[ -n "${BUF_TOKEN}" ]]; then
  echo "${BUF_TOKEN}" | buf registry login --token-stdin
fi

repo_root="$(cd "$(dirname "${0}")/.." && pwd)"
all_mods_sync_path="${repo_root}/modules/sync"
all_mods_static_path="${repo_root}/modules/static"
cd "${repo_root}"

# Build the module processor once up front. It runs at least once per reference walked, and `go run`
# relinks it on every invocation.
modprocessor_bin="${repo_root}/.tmp/bin/modprocessor"
go build -o "${modprocessor_bin}" "${repo_root}/cmd/modprocessor"

log() {
  >&2 echo "$@"
}

# prepare_ref should be called within the appropriate proto src directory where files will be copied
# from. Rsync file should be relative to this dir. This func checks out to a reference and copies the
# relevant files to a fresh temporary directory, setting its path in ${prepared_ref_path}.
prepare_ref() {
  local -r mod_ref="${1}"
  local -r mod_tmp_path="$(mktemp -d)"
  dirs_to_delete+=("${mod_tmp_path}")
  local -r module_static_path="${repo_root}/modules/static/${owner}/${repo}"
  git -c advice.detachedHead=false checkout "${mod_ref}" -q
  git clean -df

  # If there was a proto_subdir set, and there is no LICENSE file in the current path (subdir), and
  # there is a LICENSE at root of the repo, copy it here (subdir) so that it's included in the
  # module reference.
  if [ "${proto_subdir}" != "." ] && [ ! -e "LICENSE" ] && [ -s "${module_root}/${git_owner}/${git_repo}/LICENSE" ]; then
    cp "${module_root}/${git_owner}/${git_repo}/LICENSE" .
  fi

  if [ -e "${module_static_path}/pre-sync.sh" ]; then
    . "${module_static_path}/pre-sync.sh"
  fi

  # rsync flags: https://linux.die.net/man/1/rsync
  rsync_args=(
    # Archive mode is a shortcut to preserve files modification times, permissions, recursive
    # folders, among others: https://serverfault.com/questions/141773/what-is-archive-mode-in-rsync
    --archive

    # Prune empty directory chains from file-list
    --prune-empty-dirs

    # Use relative path names
    --relative

    # Copy symlinks as content, not as symlinks, it overrides the --links option embedded in
    # --archive that copies symlinks as symlinks (eg LICENSE for planetscale/vitess)
    --copy-links

    # Copy only curated subset of files from that repo into a tmp module dir using `rsync.incl` file
    # on each module's static dir
    --include-from="${module_static_path}/rsync.incl"
  )
  rsync "${rsync_args[@]}" . "${mod_tmp_path}"
  [ ! -e "${module_static_path}/buf.md" ] || cp "${module_static_path}/buf.md" "${mod_tmp_path}"
  [ ! -e "${module_static_path}/buf.yaml" ] || cp "${module_static_path}/buf.yaml" "${mod_tmp_path}"
  prepared_ref_path="${mod_tmp_path}"
}

# ref_digest_status prints "changed" or "unchanged" depending on whether the contents prepared in the
# given directory differ from the digest of the module's latest reference in its state file.
ref_digest_status() {
  local -r mod_tmp_path="${1}"
  local -r mod_ref="${2}"
  "${modprocessor_bin}" \
    --dry-run \
    --root-sync-dir="${all_mods_sync_path}" \
    --src-dir="${mod_tmp_path}" \
    --owner="${owner}" \
    --repo="${repo}" \
    --ref="${mod_ref}"
}

# process_ref should be called within the appropriate proto src directory where files will be copied
# from. This func processes a reference, checks it out, copies relevant files in CAS format, and
# updates references in state files.
process_ref() {
  local -r mod_ref="${1}"
  prepare_ref "${mod_ref}"
  local -r mod_tmp_path="${prepared_ref_path}"

  # Modules synced by commit mirror every commit of their source repository, but we only sync a
  # curated subset of its files. Appending a reference whose contents are identical to the previous
  # one publishes a BSR label pointing at identical content, so drop it.
  if [ "${sync_strategy}" == "commits" ]; then
    local digest_status
    digest_status="$(ref_digest_status "${mod_tmp_path}" "${mod_ref}")"
    if [ "${digest_status}" == "unchanged" ]; then
      echo "skipping reference ${owner}/${repo}:${mod_ref}, contents unchanged"
      rm -rf "${mod_tmp_path}"
      return
    fi
  fi

  # If the source of the module that we are syncing is from a v2 workspace with another module
  # we are syncing, e.g. protovalidate and protovalidate-testing, then it is possible that
  # there are local dependencies between the modules (e.g. protovalidate-testing@local_version
  # depends on protovalidate@local_version).
  # In our example, this can cause the `buf build` validation step to fail for protovalidate-testing,
  # since it cannot resolve a version of protovalidate that does not exist yet through `buf dep update`.

  # To validate the build in these cases, we will instead build the entire workspace. This
  # step is used to validate that the module we are syncing is from a buildable workspace state.
  if [ "${source_config_version}" == "v2" ] && [ "${proto_subdir}" != "." ]; then
    pushd "${module_root}/${git_owner}/${git_repo}" > /dev/null
    buf build
    popd > /dev/null
  else
    # For all other modules (e.g. v1 modules and individual v2 modules), we go into the
    # copied files, make sure it has right files and is buildable. Then remove `buf.lock`
    # file since it's no longer relevant: each BSR cluster will sync itself from the base files and
    # regenerate its own `buf.lock`.
    pushd "${mod_tmp_path}" > /dev/null
    buf dep update
    buf build
    rm -f buf.lock
    popd > /dev/null
  fi

  # process the prepared module: convert it to CAS from the tmp mod directory and put blob files in
  # the cas path in the repo, and update the state file.
  "${modprocessor_bin}" \
    --root-sync-dir="${all_mods_sync_path}" \
    --src-dir="${mod_tmp_path}" \
    --owner="${owner}" \
    --repo="${repo}" \
    --ref="${mod_ref}"
}

# sync_references ${sync_strategy} ${owner} ${repo} ${git_remote} ${opt_proto_subdir}
#
# iterates over all git references, and syncs their content to the sync dir.
sync_references() {
  local -r sync_strategy="${1}"
  local -r owner="${2}"
  local -r repo="${3}"
  local -r git_remote="${4}"
  local -r source_config_version="${5}"
  local -r proto_subdir="${6:-.}"
  local -r mod_static_path="${all_mods_static_path}/${owner}/${repo}"
  local -r mod_sync_path="${all_mods_sync_path}/${owner}/${repo}"
  local -r mod_state_file="${mod_sync_path}/state.json"
  local -r mod_initref_file="${mod_static_path}/initref"
  local -r mod_skip_refs_file="${mod_static_path}/skip_refs.json"

  re="^(https|git)(:\/\/|@)([^\/:]+)[\/:]([^\/:]+)\/(.+)(.git)*$"
  if [[ $git_remote =~ $re ]]; then
    local -r git_owner=${BASH_REMATCH[4]}
    local -r git_repo=${BASH_REMATCH[5]}
  else
    log "git remote ${git_remote} is malformed, cannot recognize owner/repo"
    exit 2
  fi
  [ -d "${git_owner}/${git_repo}" ] || git clone --single-branch "${git_remote}" "${git_owner}/${git_repo}"

  local -r module_root=$(pwd)
  pushd "${git_owner}/${git_repo}/${proto_subdir}" > /dev/null

  # Resolve the tip of the cloned branch from the remote ref instead of HEAD: processing a reference
  # leaves the work tree on a detached HEAD, and a single clone can back more than one managed module
  # (googleapis/googleapis and googleapis/cloud-run).
  local git_origin_head_ref="refs/remotes/origin/HEAD"
  if ! git rev-parse --verify --quiet "${git_origin_head_ref}" > /dev/null; then
    git_origin_head_ref="$(git for-each-ref --count=1 --format='%(refname)' refs/remotes/origin)"
  fi
  local git_origin_head
  git_origin_head="$(git rev-parse "${git_origin_head_ref}")"

  local rev_list
  if [ "${sync_strategy}" == "releases" ]; then
    rev_list=$(get_release_revlist)
  else
    rev_list=$(get_commit_revlist)
  fi

  if [ -z "${rev_list}" ]; then
    echo "skipping module ${owner}/${repo}, no references to sync"
  else
    skippable_refs="{}"
    if [ -f "${mod_skip_refs_file}" ]; then
      skippable_refs="$(cat "${mod_skip_refs_file}")"
    fi
    for reference in $(echo "${rev_list}"); do
      if [ "$(echo "${skippable_refs}" | jq --arg jqref "${reference}" 'has($jqref)')" == "true" ]; then
        echo "skipping reference ${owner}/${repo}:${reference}"
      else
        echo "processing reference ${owner}/${repo}:${reference}"
        process_ref "${reference}"
      fi
    done
  fi
  popd > /dev/null
}

get_commit_revlist() {
  local commit_rev_list
  if [ -f "${mod_state_file}" ]; then
    mod_latest_ref="$(cat "${mod_state_file}" | jq -r '.references | last.name')"
    log "latest reference for module ${owner}/${repo}: ${mod_latest_ref}"
    # revisions from initial latest_ref...git_origin_head (excluding latest_ref)
    commit_rev_list=$(git rev-list "${mod_latest_ref}..${git_origin_head}" --first-parent --reverse)
  elif [ -f "${mod_initref_file}" ]; then
    log "state file not found: ${mod_state_file}"
    mod_init_ref="$(cat "${mod_initref_file}")"
    log "falling back to initializing reference for module ${owner}/${repo}: ${mod_init_ref}"
    # Prints revisions on the main branch, stopping when ${mod_init_ref} is
    # encountered, and using tac to reverse the revisions (includes init_ref).
    commit_rev_list=$(git rev-list "${git_origin_head}" --first-parent | sed "/${mod_init_ref}/q" | tac)
  else
    log "module ${owner}/${repo} has no initializing reference"
    exit 2
  fi
  echo "${commit_rev_list}"
}

get_release_revlist() {
  local mod_ref
  local inclusive
  if [ -f "${mod_state_file}" ]; then
    mod_ref="$(cat "${mod_state_file}" | jq -r '.references | last.name')"
    inclusive=false
    log "latest reference for module ${owner}/${repo}: ${mod_ref}"
  elif [ -f "${mod_initref_file}" ]; then
    log "state file not found: ${mod_state_file}"
    mod_ref="$(cat "${mod_initref_file}")"
    inclusive=true
    log "falling back to initializing reference for module ${owner}/${repo}: ${mod_ref}"
  else
    log "module ${owner}/${repo} has no initializing reference"
    exit 2
  fi
  pushd "${repo_root}" > /dev/null
  # releaseprocessor needs git's owner+repo to query Github's API.
  release_rev_list=$(go run "${repo_root}/cmd/releaseprocessor" \
    --owner="${git_owner}" \
    --repo="${git_repo}" \
    --reference="${mod_ref}" \
    --inclusive="${inclusive}")
  popd > /dev/null
  if [ -n "${release_rev_list}" ]; then
    # fetch tags so we're able to checkout to them
    git fetch --tags
  fi
  echo "${release_rev_list}"
}

cleanup() {
  for dir in "${dirs_to_delete[@]}"; do
    rm -rf "${dir}"
  done
}

all_mods_tmp_dir="$(mktemp -d)"
pushd "${all_mods_tmp_dir}" > /dev/null
dirs_to_delete=("${all_mods_tmp_dir}")
trap cleanup EXIT

# TODO: use matrix to parallelize per module via GHA. Separate PRs per
# module would be nice.
# https://tomasvotruba.com/blog/2020/11/16/how-to-make-dynamic-matrix-in-github-actions/

# These modules are sorted A-Z. Order of these modules syncs is not truly necessary, even though
# some of these modules depend on each other, this workflow is not pushing new commits or tags to
# any BSR cluster, so we are not immediately receiving the updated versions of the dependencies in
# the dependent modules.

# Modules from source repositories that do not have a buf.yaml configuration are defaulted to v1.

# Keep this module list synced with README.md
# sync_references ${sync_strategy} ${owner} ${repo} ${git_remote} ${source_config_version} ${opt_proto_subdir}
sync_references releases bufbuild confluent https://github.com/bufbuild/confluent-proto v2
sync_references releases bufbuild protovalidate https://github.com/bufbuild/protovalidate v2 proto/protovalidate
sync_references releases bufbuild protovalidate-testing https://github.com/bufbuild/protovalidate v2 proto/protovalidate-testing
sync_references commits bufbuild reflect https://github.com/bufbuild/reflect v1
sync_references commits cncf xds https://github.com/cncf/xds v1
sync_references releases envoyproxy envoy https://github.com/envoyproxy/envoy v1 api
sync_references releases envoyproxy protoc-gen-validate https://github.com/envoyproxy/protoc-gen-validate
sync_references commits envoyproxy ratelimit https://github.com/envoyproxy/ratelimit v1 api
sync_references releases gogo protobuf https://github.com/gogo/protobuf v1
sync_references releases google cel-spec https://github.com/google/cel-spec v1 proto
sync_references commits googleapis cloud-run https://github.com/googleapis/googleapis v1
sync_references commits googleapis googleapis https://github.com/googleapis/googleapis v1
sync_references releases googlechrome lighthouse https://github.com/GoogleChrome/lighthouse v1 proto
sync_references releases googlecloudplatform bq-schema-api https://github.com/GoogleCloudPlatform/protoc-gen-bq-schema v1
sync_references commits grpc grpc https://github.com/grpc/grpc-proto v1
sync_references releases grpc-ecosystem grpc-gateway https://github.com/grpc-ecosystem/grpc-gateway v1
sync_references releases opencensus opencensus https://github.com/census-instrumentation/opencensus-proto v1 src
sync_references releases opentelemetry opentelemetry https://github.com/open-telemetry/opentelemetry-proto v1
sync_references releases prometheus client-model https://github.com/prometheus/client_model v1
sync_references releases protocolbuffers wellknowntypes https://github.com/protocolbuffers/protobuf v1 src

popd > /dev/null
