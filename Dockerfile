ARG ALPINE_VERSION=3.17

FROM rust:alpine${ALPINE_VERSION} AS base

RUN apk add --no-cache \
        # musl-dev is needed so the rust compiler can link against musl's libc implementation
        musl-dev \
        # libressl is needed for SSL support (for our HTTP client)
        libressl-dev 

FROM base AS builder

WORKDIR /app
COPY ./ /app

# This flag is needed so the compiler understands how to link against musl
ENV RUSTFLAGS="-C target-feature=-crt-static"

# This greatly speeds up the build because it uses a new HTTP based system to download the crate metadata
ENV CARGO_REGISTRIES_CRATES_IO_PROTOCOL=sparse

RUN cargo build --release

FROM alpine:${ALPINE_VERSION} AS dist

RUN apk add --no-cache libressl libgcc

COPY --from=builder /app/target/release/dsmr-adapter-rust .

EXPOSE 8080

ENTRYPOINT ["/dsmr-adapter-rust"]