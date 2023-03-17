#!/usr/bin/env bash

set -eo pipefail

echo "${BUF_TOKEN}" | buf registry login --debug --token-stdin --username "${BUF_USER}"

repo_root="$(cd "$(dirname "${0}")/.." && pwd)"
all_mods_sync_path="${repo_root}/modules/sync"
all_mods_static_path="${repo_root}/modules/static"
cd "${repo_root}"

log() {
  >&2 echo "$@"
}

# process_ref should be called within the appropriate proto src directory
# where files will be copied from. Rsync file should be relative to this
# dir. This func process a reference, checks out to it, copies relevant
# files and stores them in CAS format. It updates references in state
# files.
process_ref() {
  local -r mod_ref="${1}"
  local -r mod_tmp_path="$(mktemp -d)"
  dirs_to_delete+=("${mod_tmp_path}")
  local -r module_static_path="${repo_root}/modules/static/${owner}/${repo}"
  git -c advice.detachedHead=false checkout "${mod_ref}" -q
  git clean -df

  # Copy only curated subset of files from that repo into a tmp module dir.
  rsync -amR --include-from="${module_static_path}/rsync.incl" . "${mod_tmp_path}"
  cp "${module_static_path}/buf.md" "${mod_tmp_path}"
  cp "${module_static_path}/buf.yaml" "${mod_tmp_path}"

  # Go into the copied files, make sure it has right files and is
  # buildable. Then remove `buf.lock` file since it's no longer
  # relevant: each BSR cluster will sync itself from the base files and
  # regenerate its own `buf.lock`.
  pushd "${mod_tmp_path}" > /dev/null
  buf mod update
  buf build
  rm buf.lock
  popd > /dev/null

  # process the prepared module: convert it to CAS from the tmp mod
  # directory and put blob files in the cas path in the repo, and update
  # the state file.
  pushd "${repo_root}" > /dev/null
  go run "${repo_root}/cmd/modprocessor" \
    --root-sync-dir="${all_mods_sync_path}" \
    --src-dir="${mod_tmp_path}" \
    --owner="${owner}" \
    --repo="${repo}" \
    --ref="${mod_ref}"
  popd > /dev/null
}

# sync_references ${sync_strategy} ${owner} ${repo} ${git_remote} ${opt_proto_subdir}
#
# iterates over all git references, and syncs their content to the sync
# dir.
sync_references() {
  local -r sync_strategy="${1}"
  local -r owner="${2}"
  local -r repo="${3}"
  local -r git_remote="${4}"
  local -r proto_subdir="${5:-.}"
  local -r mod_static_path="${all_mods_static_path}/${owner}/${repo}"
  local -r mod_sync_path="${all_mods_sync_path}/${owner}/${repo}"
  local -r mod_state_file="${mod_sync_path}/state.json"
  local -r mod_initref_file="${mod_static_path}/initref"

  re="^(https|git)(:\/\/|@)([^\/:]+)[\/:]([^\/:]+)\/(.+)(.git)*$"
  if [[ $git_remote =~ $re ]]; then
    local -r git_owner=${BASH_REMATCH[4]}
    local -r git_repo=${BASH_REMATCH[5]}
  else
    log "git remote ${git_remote} is malformed, cannot recognize owner/repo"
    exit 2
  fi
  git clone --single-branch "${git_remote}" "${git_owner}/${git_repo}"
  pushd "${git_owner}/${git_repo}/${proto_subdir}" > /dev/null

  local rev_list
  if [ "${sync_strategy}" == "releases" ]; then
    rev_list=$(get_release_revlist)
  else
    rev_list=$(get_commit_revlist)
  fi

  if [ -z "${rev_list}" ]; then
    echo "skipping module ${owner}/${repo}, no references to sync"
  else
    for reference in $(echo "${rev_list}"); do
      echo "processing reference ${owner}/${repo}:${reference}"
      process_ref "${reference}"
    done
  fi
  popd > /dev/null
}

get_commit_revlist() {
  local commit_rev_list
  if [ -f "${mod_state_file}" ]; then
    mod_latest_ref="$(cat "${mod_state_file}" | jq -r '.references | last.name')"
    log "latest reference for module ${owner}/${repo}: ${mod_latest_ref}"
    # revisions from initial latest_ref...HEAD (excluding latest_ref)
    commit_rev_list=$(git rev-list "${mod_latest_ref}"...HEAD --first-parent --reverse)
  elif [ -f "${mod_initref_file}" ]; then
    log "state file not found: ${mod_state_file}"
    mod_init_ref="$(cat "${mod_initref_file}")"
    log "falling back to initializing reference for module ${owner}/${repo}: ${mod_init_ref}"
    # revisions from initial init_ref...HEAD (inclusive)
    commit_rev_list=$(git rev-list ^"${mod_init_ref}"~ HEAD --first-parent --reverse)
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

# sync_references ${sync_strategy} ${owner} ${repo} ${git_remote} ${opt_proto_subdir}
sync_references commits cncf xds https://github.com/cncf/xds # depends on [envoyproxy/protoc-gen-validate, googleapis/googleapis]
# sync_references commits envoyproxy envoy https://github.com/envoyproxy/envoy api # depends on [cncf/xds, googleapis/googleapis, opencensus/opencensus, opentelemetry/opentelemetry, prometheus/client-model]
sync_references commits envoyproxy protoc-gen-validate https://github.com/envoyproxy/protoc-gen-validate
sync_references commits gogo protobuf https://github.com/gogo/protobuf
sync_references commits googleapis googleapis https://github.com/googleapis/googleapis
sync_references commits grpc grpc https://github.com/grpc/grpc-proto # depends on [envoyproxy/envoy, googleapis/googleapis]
sync_references commits opencensus opencensus https://github.com/census-instrumentation/opencensus-proto src
sync_references commits opentelemetry opentelemetry https://github.com/open-telemetry/opentelemetry-proto
sync_references releases protocolbuffers wellknowntypes https://github.com/protocolbuffers/protobuf src

popd > /dev/null
