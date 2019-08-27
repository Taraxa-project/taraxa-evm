FROM ubuntu:19.04

VOLUME /workspace
# General
RUN apt-get -y upgrade
RUN apt-get -y update --fix-missing
RUN apt-get -y install wget git gnupg2
# RocksDB & LevelDB
RUN apt-get -y install liblz4-dev libgflags-dev libsnappy-dev zlib1g-dev libbz2-dev libzstd-dev
RUN apt-get -y install librocksdb-dev libleveldb-dev
# Go
RUN wget -qO- --show-progress --progress=bar:force \
    https://dl.google.com/go/go1.12.1.linux-amd64.tar.gz | tar xvz -C /usr/local
ENV GOROOT=/usr/local/go
ENV GOPATH=$HOME/.go
ENV PATH=$GOPATH/bin:$GOROOT/bin:$PATH
# Python
RUN apt-get -y install python3.7 python3.7-dev python3-pip
RUN pip3 install --upgrade pip virtualenv
# Python env
RUN virtualenv --no-site-packages --python=python3.7 /venv
RUN cat << EOF > /entrypoint.sh \
#!/bin/sh \
. /venv/bin/activate \
exec "\$@" \
EOF
RUN apt-get -y autoremove
WORKDIR /workspace
#ENTRYPOINT ["/entrypoint.sh"]