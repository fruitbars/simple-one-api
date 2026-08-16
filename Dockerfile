FROM alpine:3.22

WORKDIR /app

ARG TARGETARCH=amd64
RUN addgroup -S app && adduser -S -G app app && chown app:app /app
COPY --chown=app:app build/linux-${TARGETARCH}/simple-one-api /app/simple-one-api

EXPOSE 9090

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -q -O /dev/null http://127.0.0.1:9090/healthz || exit 1

USER app

CMD ["./simple-one-api"]
