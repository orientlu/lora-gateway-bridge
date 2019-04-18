FROM golang:1.11-alpine AS development

ENV http_proxy=http://web-proxy.tencent.com:8080
ENV PROJECT_PATH=/lora-gateway-bridge
ENV PATH=$PATH:$PROJECT_PATH/build
ENV CGO_ENABLED=0
ENV GO_EXTRA_BUILD_ARGS="-a -installsuffix cgo"

RUN apk add --no-cache ca-certificates make git bash\
    && mkdir -p $PROJECT_PATH\
    && git config --global http.proxy "http://web-proxy.tencent.com:8080"
WORKDIR $PROJECT_PATH

COPY ./go.mod .
RUN go mod download

COPY . .
RUN make dev-requirements
RUN make

# ----
FROM alpine:latest AS production
WORKDIR /root/
RUN export http_proxy=http://web-proxy.tencent.com:8080\
    && apk --no-cache add ca-certificates\
    && unset http_proxy
COPY --from=development /lora-gateway-bridge/build/lora-gateway-bridge .
ENTRYPOINT ["./lora-gateway-bridge"]
