FROM ubuntu:18.10
# General
RUN apt-get -y upgrade
RUN apt-get -y update --fix-missing
RUN apt-get -y install wget git gnupg2
# Go
RUN wget -qO- --show-progress --progress=bar:force \
    https://dl.google.com/go/go1.12.1.linux-amd64.tar.gz | tar xvz -C /usr/local
ENV GOROOT=/usr/local/go
ENV GOPATH=$HOME/.go
ENV PATH=$GOPATH/bin:$GOROOT/bin:$PATH
# Project files
COPY / /taraxa_evm
WORKDIR /taraxa_evm