**Dependencies**

- cmake

**Build**

cd [Project Root]

make tests_integration

**Local run**

_evm tests_

cd [Project Root]

cd tests_integration

./tests_integration vargs...

- vargs - see google test dev doc

_grpc tests_

cd [Project Root]

cd tests_integration

starting test server:
./test_server

./tests_grpc vargs...

- vargs - see google test dev doc
