ARCH="$(uname -m)"

release_url="https://github.com/firecracker-microvm/firecracker/releases"

latest=$(basename $(curl -fsSLI -o /dev/null -w %{url_effective} ${release_url}/latest))

curl -L ${release_url}/download/${latest}/firecracker-${latest}-${ARCH}.tgz |
  tar -xz

# Rename the binary to "firecracker"
mv release-${latest}-$(uname -m)/firecracker-${latest}-${ARCH} firecracker
