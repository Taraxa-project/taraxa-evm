FROM ubuntu:18.10

VOLUME /mnt
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
# Python
RUN apt-get -y install python3.7 python3.7-dev python3-pip
RUN pip3 install --upgrade pip virtualenv
# RocksDB & LevelDB
RUN apt-get -y install liblz4-dev libgflags-dev libsnappy-dev zlib1g-dev libbz2-dev libzstd-dev
RUN apt-get -y install librocksdb-dev libleveldb-dev
# The Project
WORKDIR /mnt/workspace/taraxa-evm/tests_perf
#RUN rm -rf venv && virtualenv --no-site-packages --python=python3.7 venv
#RUN . venv/bin/activate && python install.py
ENTRYPOINT venv/bin/python apps "collect_blocks('/mnt/xvdf/perf_test')"