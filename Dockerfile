FROM golang:1.14 as builder

WORKDIR /builddir
ADD . /builddir

RUN make build && rm -rf pkg vendor cmd

FROM gcr.io/distroless/base-debian10
# Copy the binary to a standard location where it will run.
COPY --from=builder /builddir/bin/pinta-scheduler /bin/
COPY --from=builder /builddir/bin/pinta-controller /bin/

# This image doesn't need to run as root user.
USER 1001

EXPOSE 8080
