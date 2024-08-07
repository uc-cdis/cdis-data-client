  if [ "$GOOS" == "linux" ]
  then
    set -e
    go build -o cdis-data-client
    ls -al
    if [ "$GITHUB_PULL_REQUEST" == "false" ]; then
      mv gen3-client files && mv cdis-data-client gen3-client
      zip dataclient_linux.zip gen3-client && mv dataclient_linux.zip ~/shared/.
      aws s3 sync ~/shared s3://cdis-dc-builds/$GITHUB_BRANCH
    fi
    set +e
  elif [ "$GOOS" == "darwin" ]
  then
    set -e
    go build -o cdis-data-client
    ls -al
    if [ "$GITHUB_PULL_REQUEST" == "false" ]; then
      mv gen3-client files && mv cdis-data-client gen3-client
      zip dataclient_osx.zip gen3-client && mv dataclient_osx.zip ~/shared/.
      aws s3 sync ~/shared s3://cdis-dc-builds/$GITHUB_BRANCH
    fi
    set +e
  elif [ "$GOOS" == "windows" ]
  then
    set -e
    go build -o gen3-client.exe
    ls -al
    if [ "$GITHUB_PULL_REQUEST" == "false" ]; then
      zip dataclient_win64.zip gen3-client.exe && mv dataclient_win64.zip ~/shared/.
      aws s3 sync ~/shared s3://cdis-dc-builds/$GITHUB_BRANCH
    fi
    set +e
  fi
