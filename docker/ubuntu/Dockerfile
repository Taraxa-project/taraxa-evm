FROM ubuntu:18.10

RUN apt-get -y upgrade
RUN apt-get -y update
RUN apt-get -y install curl wget git gnupg2
RUN apt-get -y install python3-pip
RUN pip3 install cmake
RUN wget -qO- --show-progress --progress=bar:force \
    https://dl.google.com/go/go1.12.1.linux-amd64.tar.gz | tar xvz -C /usr/local
ENV GOROOT=/usr/local/go
ENV GOPATH=$HOME/.go
ENV PATH=$GOPATH/bin:$GOROOT/bin:$PATH

COPY / /project_root

ENV LIB_CPP_BUILD_DIR=/project_root/lib_cpp/build
RUN mkdir $LIB_CPP_BUILD_DIR && cd $LIB_CPP_BUILD_DIR && \
    cmake /project_root/lib_cpp && \
    cmake --build . --target taraxa_evm_tests -j $(nproc --all)
RUN $LIB_CPP_BUILD_DIR/taraxa_evm_tests
WORKDIR /project_root