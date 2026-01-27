# Publishing

Github Actions only supports three methods of running actions today:

* Composite (run a shell script)
* Node
* Docker

None of these approaches work very well for a Go Github Action.
Composite allows you to JIT build the action, but you rely on the host
having a specific version of Go. Node, well, can't run Go. Lastly,
Docker is the heavier (more reproducible) approach, but it only supports
JIT build the binary as well, which means every action will be slower.

**Note**: You can also use `- uses: docker://...`, but compared to other
actions that's pretty non-standard. Also, Docker executors only work on
Linux, so macOS & Windows can't use that method.

## How we publish our action

Since Github Actions _does_ support Node, we can actually pretty easily
write a node.js shim to execute a Go binary (which we did!
[shim/shim.js](../shim/shim.js)). Given a directory full of binaries, we
can check which OS/Arch pairing we're on and execute the binary.

However, we still have the problem of how we get the binary. Blend
[solved this problem by commiting the binaries to
source](https://full-stack.blend.com/how-we-write-github-actions-in-go.html#just-go).
This approach works, but it suffers from the traditional binaries-in-git
problem:

Repository size will bloat over time. Each new binary increases the
size. This can be mitigated by moving binaries into their own repo, but
now you have two repos when you really don't need two.

Thankfully, Git has support for [orphaned
branches](https://graphite.com/guides/git-orphan-branches), which are
essentially a new set of history within the same branch. This is perfect
for storing these binaries, as they're not part of the default branch's
history. Through our [release.sh](../scripts/release.sh) script, we
create a commit on an orphaned branch, tag it with the conventional
`v$major`, `v$major.$minor` and `v$major.$minor.$patch` tags and push
them up. The actions are still discoverable through the tag view, github
releases, and most importantly: Github Actions! While we're not able to
restrict which binaries we download (we download every platform's), the
download time is still incredibly fast, especially compared to the
Docker & composite JIT build.

## Examples

* [`v0.1.3`](https://github.com/rgst-io/stencil-action-go/releases/tag/v0.1.3)
	```yml
	- uses: rgst-io/stencil-action-go@v0.1.3
	```
