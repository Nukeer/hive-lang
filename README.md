<p align="center">
  <img src="assets/hive-logo.svg" alt="The Hive logo: a bee over a honeycomb" width="160">
</p>

<h1 align="center">Hive, in Hive</h1>

<p align="center">
  <a href="https://github.com/R0DR160HM/hive-lang/actions/workflows/build.yml"><img src="https://github.com/R0DR160HM/hive-lang/actions/workflows/build.yml/badge.svg" alt="build"></a>
</p>

Hive is a compiled, memory-managed language with **no runtime exceptions**: there
is no null, nothing is thrown, and every vector index is proved in bounds before
the program runs. It is built for distributed systems, secure by default, and
small enough to learn in a day.

This repository holds the [specification](spec/) and the compiler, written in Hive.

## Install

Put the `hivec` binary from a [release](../../releases) on your `PATH`, and install
[Go 1.24+](https://go.dev/dl/) to build and run programs.
[INSTALL.md](INSTALL.md) has the details, and how to build it from source.

## Use

```
hivec run       <entrypoint.hive>    compile and run
hivec test      <entrypoint.hive>    run its tests, with coverage
hivec check     <entrypoint.hive>    report errors, build nothing
hivec build     <entrypoint.hive>    compile to a native executable
hivec export    <entrypoint.hive>    package it as an Android app
hivec analyze   <entrypoint.hive>    score what it will cost to run
hivec emit      <entrypoint.hive>    print the generated Go
hivec container <entrypoint.hive>    write a Dockerfile that builds and runs it
hivec agents                         write .hivedocs/ for a coding agent to read
hivec version                        which release this compiler is
```

`build` and `export` take `--target <goos>/<goarch>`.

## Learn

* [The tour](https://hive-tour.fly.dev): the whole language in fifty-five pages.
* [examples/](examples): thirty-three programs, from `echo` to a multiplayer
  shooter, every one compiled and run by `./examples/run`.
* [spec/](spec/): the specification. [18 — Conformance](spec/18-conformance.md)
  says where this compiler departs from it.
* [CHANGELOG.md](CHANGELOG.md): what changed, release by release.

## Develop

```
./bootstrap        build the compiler with itself
./test/run         every test the compiler has
./examples/run     every example, compiled and run
./selfhost         build it twice more, and check the two builds emit the same Go
```

`./hive` (`hive.cmd` on Windows) runs the compiler in `src/`.

```
spec/                the specification, in 18 chapters
examples/            the examples
src/                 the compiler, in the order the pipeline runs
  lexer token ast parser         source text -> tokens -> one module's tree
  regex show                     a string pattern's regex; a tree as one line
  loader fetch goffi             the import graph: files, repositories, Go files
  mono                           one copy of a generic per set of type arguments
  types infer stdlib             what a type means, what an expression is, what hive.* is
  check ranges writes            the rules; every index proved; what a copy copies
  emit runtime                   one program -> one Go file, and the Go it runs on
  project vendor progress        the Go module and toolchain, the one download, progress
  analyze analyzereport          what hive analyze scores, and the page it writes
  container agents icon docs/    a Dockerfile, .hivedocs/, the program's icon, the pages
  text paths naming diag         strings, paths, name shapes, one error
  version testreport hivec       the release, the test report, the command line
test/                one suite per module, and whole programs in e2e/
seed/                which release a clean build starts from, and its digest
```
