##==========================================================================
FROM golang:1.22.12-alpine3.20
LABEL maintainer="Stoney Kang, sikang99@gmail.com"
##-------------------------------------------------------------------------
#RUN apk add --no-cache \
#    ffmpeg ffmpeg-libs ffmpeg-dev ffmpeg-doc \
#    ffmpegthumbnailer ffmpegthumbnailer-dev ffmpegthumbnailer-doc
##--------------------------------------------------------------------------
#RUN apk add --no-cache \
#    gstreamer gstreamer-dev gstreamer-tools gstreamer-doc gstreamer-lang \
#    gst-libav \
#    gst-plugins-base gst-plugins-good gst-plugins-bad gst-plugins-ugly \
#    gst-plugins-base-dev gst-plugins-bad-dev gst-plugins-base-doc
##--------------------------------------------------------------------------
#RUN apk add --no-cache \
#    musl-dev make gcc curl
##--------------------------------------------------------------------------
#ENV GO111MODULE=on
#ENV CGO_ENABLED=0
##--------------------------------------------------------------------------
ARG SRC_HOME=/Users/stoney/coding/go/src/github.com/sikang99/moth
RUN mkdir -p /go/src/github.com/sikang99/moth/server
COPY $SRC_HOME/server/* /go/src/github.com/sikang99/moth/server/*
RUN cd /go/src/github.com/sikang99/moth/server \
    && go build -mod=mod -ldflags "-s -w" -o moth
##--------------------------------------------------------------------------
RUN mkdir -p /go/src/github.com/sikang99/moth/tools/gst-ws/med-record
COPY $SRC_HOME/tools/gst-ws/med-record/* /go/src/github.com/sikang99/moth/tools/gst-ws/med-record
##--------------------------------------------------------------------------
RUN mkdir /moth
RUN mv /go/src/github.com/sikang99/moth/server/moth /moth
RUN rm -rf /go/src/*
ADD ./cert /moth/cert/
ADD ./conf /moth/conf/
ADD ./html /moth/html/
WORKDIR /moth
#EXPOSE 3478/tcp 3478/udp 50000-55000/udp
##==========================================================================
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
##==========================================================================

