## ImplementationLanguage

### Context

The tool performs file I/O, directory traversal, and markdown structure analysis. It must distribute as a single standalone binary on macOS, Linux, and Windows.

### Requirement

A language that supports single-binary distribution, fast cross-compilation, and a straightforward development experience for a small CLI tool with no concurrency or unsafe memory requirements.

### Decision

Go — the tool is implemented in Go. Cross-compilation to all target platforms is built into the standard toolchain. `cobra` handles CLI structure; the standard library covers file I/O and directory traversal. No external runtime is required.
