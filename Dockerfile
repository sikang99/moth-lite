##---------------------------------------------------------------------------------
FROM golang:1.22.12-alpine3.20
##---------------------------------------------------------------------------------
LABEL maintainer="Stoney Kang, sikang@teamgrit.kr"
##=================================================================================
ENV GO111MODULE=on
RUN mkdir -p /moth
##--------------------------------------------------------------------------
RUN mkdir -p /go/src/github.com/sikang99/moth/server
ADD ./server /go/src/github.com/sikang99/moth/server
RUN cd /go/src/github.com/sikang99/moth/server \
    && go build -ldflags="-s -w" -trimpath -o moth-server
RUN mv /go/src/github.com/sikang99/moth/server/moth-server /moth
##--------------------------------------------------------------------------
RUN mkdir -p /moth/cmd
##=================================================================================
ENV CGO_ENABLED=1
RUN mkdir -p /moth/tools
##--------------------------------------------------------------------------
RUN apk add --no-cache ca-certificates \
    musl-dev make gcc curl

RUN rm -rf /go/src/*
ADD ./server/cert /moth/cert/
ADD ./server/conf /moth/conf/
ADD ./server/html /moth/html/
ENV PATH="/moth:/moth/tools:${PATH}"
WORKDIR /moth
#EXPOSE 8276-8277/tcp
#EXPOSE 8276-8277/udp
##==========================================================================
COPY ./build/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
##==========================================================================

