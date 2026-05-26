#!/usr/bin/env bash
# Manual release script for terraform-provider-devin
# Usage: ./scripts/release.sh v0.1.0
set -euo pipefail

VERSION="${1:?Usage: release.sh <version-tag, e.g. v0.1.0>}"
VERSION_NUM="${VERSION#v}"
PROVIDER_NAME="terraform-provider-devin"
DIST_DIR="dist"

PLATFORMS=(
  "linux/amd64"
  "linux/arm64"
  "darwin/amd64"
  "darwin/arm64"
  "windows/amd64"
)

echo "==> Building ${PROVIDER_NAME} ${VERSION} for all platforms..."
rm -rf "${DIST_DIR}"
mkdir -p "${DIST_DIR}"

for platform in "${PLATFORMS[@]}"; do
  GOOS="${platform%/*}"
  GOARCH="${platform#*/}"
  output_name="${PROVIDER_NAME}_${VERSION_NUM}"
  if [ "$GOOS" = "windows" ]; then
    output_name="${output_name}.exe"
  fi

  echo "  Building ${GOOS}/${GOARCH}..."
  CGO_ENABLED=0 GOOS="${GOOS}" GOARCH="${GOARCH}" \
    go build -trimpath \
    -ldflags="-s -w -X main.version=${VERSION_NUM}" \
    -o "${DIST_DIR}/${output_name}" .

  # Create zip archive following Terraform Registry conventions
  archive_name="${PROVIDER_NAME}_${VERSION_NUM}_${GOOS}_${GOARCH}.zip"
  (cd "${DIST_DIR}" && zip "${archive_name}" "${output_name}" && rm "${output_name}")
  echo "    -> ${archive_name}"
done

echo "==> Generating SHA256SUMS..."
(cd "${DIST_DIR}" && shasum -a 256 *.zip > "${PROVIDER_NAME}_${VERSION_NUM}_SHA256SUMS")
echo "    -> ${PROVIDER_NAME}_${VERSION_NUM}_SHA256SUMS"

echo "==> Signing SHA256SUMS with GPG..."
if [ -n "${GPG_FINGERPRINT:-}" ]; then
  gpg --batch --local-user "${GPG_FINGERPRINT}" \
    --detach-sign "${DIST_DIR}/${PROVIDER_NAME}_${VERSION_NUM}_SHA256SUMS"
  echo "    -> ${PROVIDER_NAME}_${VERSION_NUM}_SHA256SUMS.sig"
else
  echo "    SKIPPED (set GPG_FINGERPRINT to enable signing)"
fi

echo "==> Generating Terraform Registry manifest..."
cat > "${DIST_DIR}/terraform-registry-manifest.json" <<EOF
{
  "version": 1,
  "metadata": {
    "protocol_versions": ["6.0"]
  }
}
EOF
echo "    -> terraform-registry-manifest.json"

echo "==> Creating GitHub release ${VERSION}..."
if command -v gh &>/dev/null; then
  RELEASE_FILES=("${DIST_DIR}"/*.zip "${DIST_DIR}"/*SHA256SUMS*)
  if [ -f "${DIST_DIR}/terraform-registry-manifest.json" ]; then
    RELEASE_FILES+=("${DIST_DIR}/terraform-registry-manifest.json")
  fi

  gh release create "${VERSION}" \
    --title "${VERSION}" \
    --generate-notes \
    "${RELEASE_FILES[@]}"
  echo "==> Release ${VERSION} created!"
else
  echo "    SKIPPED (gh CLI not found — upload files manually)"
  echo "    Files ready in ${DIST_DIR}/"
fi

echo "==> Done!"
ls -lh "${DIST_DIR}/"
