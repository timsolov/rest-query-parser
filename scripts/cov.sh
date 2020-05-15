#!/bin/bash -e
# Run from directory above via ./scripts/cov.sh

go get github.com/mattn/goveralls
go get github.com/wadey/gocovmerge

rm -rf ./cov
mkdir cov
go test -v -covermode=atomic -coverprofile=./cov/rqp.out
gocovmerge ./cov/*.out > acc.out
rm -rf ./cov

# If we have an arg, assume travis run and push to coveralls. Otherwise launch browser results
if [[ -n $1 ]]; then
    $HOME/gopath/bin/goveralls -coverprofile=acc.out -service=travis-ci
    rm -rf ./acc.out
else
    go tool cover -html=acc.out
fi