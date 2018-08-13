#!/bin/bash
echo "Pulling latest lit from github (github.com/mit-dci/lit)"
cd $GOPATH/src/github.com/mit-dci/lit
git pull
echo "Building binaries"
go build
cd cmd/lit-af
go build
echo "Copy binaries to dlctest (github.com/navybluesilver/lit-trader-test/bin)"
cp lit-af $GOPATH/src/github.com/navybluesilver/lit-trader-test/bin/lit-af
cd $GOPATH/src/github.com/mit-dci/lit
cp lit $GOPATH/src/github.com/navybluesilver/lit-trader-test/bin/lit
cd $GOPATH/src/github.com/navybluesilver/lit-trader-test
