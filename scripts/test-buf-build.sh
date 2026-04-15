#!/usr/bin/env bash

set -euo pipefail

BUF="${BUF:-buf}"
echo "Testing with buf version: $("${BUF}" --version)"

repo_root="$(cd "$(dirname "${0}")/.." && pwd)"
sync_dir="${repo_root}/modules/sync"
failures=()
successes=0
total=0

for state_file in "${sync_dir}"/*/state.json "${sync_dir}"/*/*/state.json; do
  [[ -f "${state_file}" ]] || continue
  mod_dir="$(dirname "${state_file}")"
  cas_dir="${mod_dir}/cas"
  mod_name="${mod_dir#"${sync_dir}/"}"

  # Get the latest reference's manifest digest from state.json
  manifest_digest="$(python3 -c "
import json, sys
with open(sys.argv[1]) as f:
    d = json.load(f)
refs = d.get('references', [])
if not refs:
    sys.exit(1)
print(refs[-1]['digest'])
" "${state_file}" 2>/dev/null)" || continue

  manifest_path="${cas_dir}/${manifest_digest}"
  [[ -f "${manifest_path}" ]] || continue

  total=$((total + 1))
  echo "--- Building ${mod_name} ---"

  # Reconstruct module files from CAS into a temp directory.
  # Manifest format: "shake256:<hex>  <filepath>"
  tmp_dir="$(mktemp -d)"
  while IFS= read -r line; do
    # Extract the hex digest and file path
    digest_hex="${line%%  *}"          # "shake256:<hex>"
    digest_hex="${digest_hex#shake256:}" # "<hex>"
    file_path="${line#*  }"            # "<filepath>"
    blob_path="${cas_dir}/${digest_hex}"
    if [[ -f "${blob_path}" ]]; then
      mkdir -p "${tmp_dir}/$(dirname "${file_path}")"
      cp "${blob_path}" "${tmp_dir}/${file_path}"
    else
      echo "  WARN: missing blob for ${file_path}"
    fi
  done < "${manifest_path}"

  pushd "${tmp_dir}" > /dev/null
  if "${BUF}" dep update 2>&1 && "${BUF}" build 2>&1; then
    echo "OK: ${mod_name}"
    successes=$((successes + 1))
  else
    echo "FAIL: ${mod_name}"
    failures+=("${mod_name}")
  fi
  popd > /dev/null
  rm -rf "${tmp_dir}"
done

echo ""
echo "=== Results ==="
echo "Total: ${total}  Passed: ${successes}  Failed: ${#failures[@]}"
if [[ ${#failures[@]} -eq 0 ]]; then
  echo "All modules built successfully."
else
  printf 'Failures:\n'
  printf '  - %s\n' "${failures[@]}"
  exit 1
fi
