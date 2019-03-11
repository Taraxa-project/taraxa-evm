FROM ubuntu:18.04

COPY / /project_root

RUN apt-get -y upgrade
RUN apt-get -y update
RUN apt-get -y install curl wget git gnupg2
RUN curl -sL https://deb.nodesource.com/setup_10.x | bash -
RUN apt-get -y install nodejs python3-pip
RUN pip3 install cmake
RUN wget -qO- --show-progress --progress=bar:force \
    https://dl.google.com/go/go1.11.5.linux-amd64.tar.gz | tar xvz -C /usr/local
ENV GOROOT=/usr/local/go
ENV GOPATH=$HOME/.go
ENV PATH=$GOPATH/bin:$GOROOT/bin:$PATH

WORKDIR /project_root
RUN make tests_integration