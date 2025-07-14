##==========================================================================
FROM golang:1.22.12-alpine3.20
LABEL maintainer="Stoney Kang, sikang@teamgrit.kr
##-------------------------------------------------------------------------
#ENV GO111MODULE=on
#ENV CGO_ENABLED=0
##--------------------------------------------------------------------------
ARG SRC_HOME=/Users/stoney/coding/go/src/github.com/sikang99/moth-lite
RUN mkdir -p /go/src/github.com/sikang99/moth-lite/server
COPY $SRC_HOME/server/* /go/src/github.com/sikang99/moth-lite/server/*
RUN cd /go/src/github.com/sikang99/moth-lite/server \
    && go build -mod=mod -ldflags "-s -w" -o moth-lite
##--------------------------------------------------------------------------
RUN mkdir /moth
RUN mv /go/src/github.com/sikang99/moth-lite/server/moth /moth-lite
RUN rm -rf /go/src/*
ADD ./cert /moth-lite/cert/
ADD ./conf /moth-lite/conf/
ADD ./html /moth-lite/html/
WORKDIR /moth-lite
#EXPOSE 3478/tcp 3478/udp 50000-55000/udp
##==========================================================================
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
##==========================================================================

