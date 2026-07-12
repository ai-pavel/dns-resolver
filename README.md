# phonebook

[![CI](https://github.com/ai-pavel/phonebook/actions/workflows/ci.yml/badge.svg)](https://github.com/ai-pavel/phonebook/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/ai-pavel/phonebook/branch/main/graph/badge.svg)](https://codecov.io/gh/ai-pavel/phonebook)

A recursive DNS resolver implemented in Go using only the standard library.

## Features

- Full DNS message parsing and serialization (header, question, answer, authority, additional sections)
- Recursive resolution starting from root DNS servers, following referrals
- Response caching with TTL-based expiry
- UDP and TCP transport (automatic TCP fallback on truncation)
- Supported record types: A, AAAA, CNAME, MX, NS, TXT

## Project Structure

```
cmd/resolve/main.go      - CLI entry point
pkg/dns/message.go        - DNS message parsing and serialization
pkg/dns/resolver.go       - Recursive resolution algorithm
pkg/dns/cache.go          - Response caching with TTL expiry
pkg/dns/transport.go      - UDP/TCP transport layer
```

## Build

```bash
make build
```

## Usage

```bash
# Resolve A records (default)
./bin/resolve example.com

# Resolve specific record type
./bin/resolve -type AAAA example.com
./bin/resolve -type MX example.com
./bin/resolve -type TXT example.com
./bin/resolve -type NS example.com
./bin/resolve -type CNAME www.example.com
```

## Options

- `-type` - Record type to query (A, AAAA, CNAME, MX, NS, TXT). Default: A
- `-verbose` - Enable verbose output showing resolution steps

## Clean

```bash
make clean
```

## Test

```bash
make test
```
