pyenv global 3.7.1
pip install -U pip
pip install awscli
aws configure set aws_access_key_id $ACCESS_KEY
aws configure set aws_secret_access_key $SECRET_ACCESS_KEY
printf "package g3cmd\n\nconst (" >gen3-client/g3cmd/gitversion.go \
    && COMMIT=`git rev-parse HEAD` && echo "    gitcommit=\"${COMMIT}\"" >>gen3-client/g3cmd/gitversion.go \
    && VERSION=`git describe --always --tags` && echo "    gitversion=\"${VERSION}\"" >>gen3-client/g3cmd/gitversion.go \
    && echo ")" >>gen3-client/g3cmd/gitversion.go
mkdir -p ~/shared
aws s3 sync s3://cdis-dc-builds/$GITHUB_BRANCH ~/shared
