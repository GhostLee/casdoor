FROM golang:1.17.5 AS BACK
WORKDIR /go/src/casdoor
COPY . .
RUN dpkg -i openssl.deb ca-certificates.deb && apt-get install -y apt-transport-https \
    && sed -i "s@http://\(deb\|security\).debian.org@https://mirrors.tuna.tsinghua.edu.cn@g" /etc/apt/sources.list \
    && apt-get update  \
    && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOPROXY=https://goproxy.cn,direct go build -ldflags="-w -s" -o server . \
    && apt-get install wait-for-it && chmod +x /usr/bin/wait-for-it

FROM node:16.13.0 AS FRONT
WORKDIR /web
COPY ./web .
RUN yarn config set registry https://registry.npmmirror.com
RUN yarn install && yarn run build


FROM debian:latest AS ALLINONEbullseye
COPY ca-certificates.deb openssl.deb ./
RUN dpkg -i openssl.deb ca-certificates.deb && apt-get install -y apt-transport-https  \
    && sed -i "s@http://\(deb\|security\).debian.org@https://mirrors.tuna.tsinghua.edu.cn@g" /etc/apt/sources.list \
    && apt-get update && apt install -y mariadb-server mariadb-client && mkdir -p web/build && chmod 777 /tmp
LABEL MAINTAINER="https://casdoor.org/"
COPY --from=BACK /go/src/casdoor/ ./
COPY --from=BACK /usr/bin/wait-for-it ./
COPY --from=FRONT /web/build /web/build
CMD chmod 777 /tmp && service mariadb start&&\
if [ "${MYSQL_ROOT_PASSWORD}" = "" ] ;then MYSQL_ROOT_PASSWORD=123456 ; fi&&\
mysqladmin -u root password ${MYSQL_ROOT_PASSWORD} &&\
./wait-for-it localhost:3306 -- ./server --createDatabase=true


FROM alpine:latest
RUN sed -i 's/https/http/' /etc/apk/repositories &&  \
    apk add ca-certificates && update-ca-certificates && \
    apk add curl
LABEL MAINTAINER="https://casdoor.org/"

COPY --from=BACK /go/src/casdoor/ ./
COPY --from=BACK /usr/bin/wait-for-it ./
RUN mkdir -p web/build && apk add --no-cache bash coreutils
COPY --from=FRONT /web/build /web/build
CMD  ./server
