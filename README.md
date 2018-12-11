# gen3-client
`gen3-client` is a command-line tool for downloading, uploading, and submitting data files to and from a Gen3 data commons. 

Read more about what it does and how to use it in the `gen3-client` [user guide](https://gen3.org/resources/user/gen3-client/).

`gen3-client` is built on Cobra, a library providing a simple interface to create powerful modern CLI interfaces similar to git & go tools. Read more about Cobra [here](https://github.com/spf13/cobra).


## Installation

First, [install Go and the Go tools](https://golang.org/doc/install) if you have not already done so. [Set up your workspace and your GOPATH.](https://golang.org/doc/code.html)


Then: 
```
go get -d github.com/uc-cdis/gen3-client
go install
```


*TODO: Remove after GitHub repo is renamed*
**_For now, the above actually won't work because the GitHub repo needs to be renamed. Do this instead:_**

```
mkdir -p $GOPATH/src/github.com/uc-cdis
cd $GOPATH/src/github.com/uc-cdis
git clone git@github.com:uc-cdis/cdis-data-client.git
mv /cdis-data-client /gen3-client
cd gen3-client
go get -d ./...
go install .
```
